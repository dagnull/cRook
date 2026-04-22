package room

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/dagnull/cRook/internal/game"
	"github.com/dagnull/cRook/internal/game/rules"
	"github.com/dagnull/cRook/internal/hub"
)

type Status string

const (
	StatusWaiting  Status = "waiting"
	StatusPlaying  Status = "playing"
	StatusFinished Status = "finished"
)

type Player struct {
	ID   string
	Name string
}

// PlayerView is the per-player public view of a PlayerState (hand hidden).
type PlayerView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Team      int    `json:"team"`
	CardCount int    `json:"card_count"`
	Score     int    `json:"score"`
}

// StateSyncPayload is sent to a client on (re)connect.
type StateSyncPayload struct {
	Phase         string       `json:"phase"`
	Players       []PlayerView `json:"players"`
	YourHand      []game.Card  `json:"your_hand"`
	Trump         game.Suit    `json:"trump"`
	CurrentBid    int          `json:"current_bid"`
	BidderID      string       `json:"bidder_id"`
	BidTurn       string       `json:"bid_turn"`        // player ID whose turn it is to bid
	TrickLeaderID string       `json:"trick_leader_id"` // player ID who led the current trick
	CurrentTrick  game.Trick   `json:"current_trick"`
	NestSize      int          `json:"nest_size"`
}

type Room struct {
	ID        string
	Name      string
	CreatedAt time.Time
	Hub       *hub.Hub

	mu      sync.Mutex
	players []Player
	status  Status
	state   game.GameState
	engine  *game.Engine
}

func New(id, name string) *Room {
	return &Room{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now(),
		status:    StatusWaiting,
		Hub:       hub.New(),
		engine:    game.NewEngine(rules.Standard{}, rand.New(rand.NewSource(time.Now().UnixNano()))),
	}
}

// ── Player management ─────────────────────────────────────────────────────────

func (r *Room) Join(p Player) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.players {
		if existing.ID == p.ID {
			return true
		}
	}
	if len(r.players) >= 4 {
		return false
	}
	r.players = append(r.players, p)
	return true
}

func (r *Room) Leave(playerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.players {
		if p.ID == playerID {
			r.players = append(r.players[:i], r.players[i+1:]...)
			return
		}
	}
}

func (r *Room) Players() []Player {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Player, len(r.players))
	copy(out, r.players)
	return out
}

func (r *Room) PlayerCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.players)
}

func (r *Room) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

// ── Game control ──────────────────────────────────────────────────────────────

func (r *Room) StartGame() error {
	r.mu.Lock()
	if r.status == StatusPlaying {
		r.mu.Unlock()
		return errors.New("game already in progress")
	}
	if len(r.players) < 2 {
		r.mu.Unlock()
		return errors.New("need at least 2 players to start")
	}

	// Build initial GameState from current player list.
	opts := game.DefaultOptions()
	opts.NumPlayers = len(r.players)
	// Adjust nest/deal for non-4-player counts (simple fallback).
	if len(r.players) == 2 {
		opts.NestSize = 5 // 2 players get 26 cards each
	}

	ps := make([]game.PlayerState, len(r.players))
	for i, p := range r.players {
		ps[i] = game.PlayerState{
			ID:   p.ID,
			Name: p.Name,
			Team: i % 2, // alternating teams: 0,1,0,1
		}
	}

	r.state = game.GameState{
		ID:      r.ID,
		Options: opts,
		Players: ps,
	}

	var events []game.Event
	var err error
	r.state, events, err = r.engine.Deal(r.state)
	if err != nil {
		r.mu.Unlock()
		return fmt.Errorf("deal failed: %w", err)
	}
	r.status = StatusPlaying
	r.mu.Unlock()

	for _, evt := range events {
		r.broadcastEvent(evt)
	}
	r.broadcastStateSyncs()
	return nil
}

// Dispatch validates and applies a player action, then broadcasts the resulting events.
func (r *Room) Dispatch(a game.Action) error {
	r.mu.Lock()
	if r.status != StatusPlaying {
		r.mu.Unlock()
		return errors.New("game is not in progress")
	}
	newState, events, err := r.engine.Apply(r.state, a)
	if err != nil {
		r.mu.Unlock()
		return err
	}
	r.state = newState
	if r.state.Phase == game.PhaseScoring {
		r.status = StatusFinished
	}
	r.mu.Unlock()

	for _, evt := range events {
		r.broadcastEvent(evt)
	}
	r.broadcastStateSyncs()
	return nil
}

// StateSyncFor returns a state-sync payload tailored to one player.
func (r *Room) StateSyncFor(playerID string) StateSyncPayload {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := r.state
	views := make([]PlayerView, len(s.Players))
	var yourHand []game.Card
	var bidTurnID string

	if s.BidTurn >= 0 && s.BidTurn < len(s.Players) {
		bidTurnID = s.Players[s.BidTurn].ID
	}

	for i, p := range s.Players {
		views[i] = PlayerView{
			ID:        p.ID,
			Name:      p.Name,
			Team:      p.Team,
			CardCount: len(p.Hand),
			Score:     p.TotalScore,
		}
		if p.ID == playerID {
			yourHand = p.Hand
		}
	}

	return StateSyncPayload{
		Phase:        s.Phase.String(),
		Players:      views,
		YourHand:     yourHand,
		Trump:        s.Trump,
		CurrentBid:   s.CurrentBid,
		BidderID:     s.CurrentBidder,
		BidTurn:      bidTurnID,
		CurrentTrick: s.CurrentTrick,
		NestSize:     len(s.Nest),
	}
}

// ── Event broadcasting ────────────────────────────────────────────────────────

func (r *Room) broadcastEvent(evt game.Event) {
	if evt.Kind == game.EventCardDealt {
		// Each player only sees their own cards.
		r.Hub.BroadcastFiltered(toHubMsg(evt), func(playerID string, msg hub.Message) (hub.Message, bool) {
			p, ok := evt.Payload.(game.CardDealtPayload)
			if !ok {
				return msg, true
			}
			if p.PlayerID == playerID {
				return msg, true
			}
			// Send a version with an empty hand — client knows cards were dealt but not what they are.
			return hub.Message{
				Kind: string(game.EventCardDealt),
				Payload: game.CardDealtPayload{
					PlayerID: p.PlayerID,
					Cards:    []game.Card{},
				},
			}, true
		})
		return
	}
	r.Hub.Broadcast(toHubMsg(evt))
}

func toHubMsg(evt game.Event) hub.Message {
	return hub.Message{Kind: string(evt.Kind), Payload: evt.Payload}
}

// broadcastStateSyncs sends a personalised state_sync to every connected client.
func (r *Room) broadcastStateSyncs() {
	r.Hub.BroadcastFiltered(hub.Message{Kind: "state_sync"}, func(playerID string, _ hub.Message) (hub.Message, bool) {
		payload := r.StateSyncFor(playerID)
		return hub.Message{Kind: "state_sync", Payload: payload}, true
	})
}

// RunHub starts the hub goroutine. Called by the manager.
func (r *Room) RunHub(ctx context.Context) {
	r.Hub.Run(ctx)
}
