package game

import (
	"errors"
	"math/rand"
)

// Engine is stateless — callers own GameState and pass it in with each action.
// The Room goroutine serializes all calls so no locking is needed here.
type Engine struct {
	Rules RuleSet
	Rng   *rand.Rand
}

func NewEngine(r RuleSet, rng *rand.Rand) *Engine {
	return &Engine{Rules: r, Rng: rng}
}

// Deal shuffles the deck, distributes cards to players, and advances to bidding.
func (e *Engine) Deal(s GameState) (GameState, []Event, error) {
	if s.Phase != PhaseWaiting && s.Phase != PhaseScoring {
		return s, nil, errors.New("can only deal from waiting or scoring phase")
	}
	if len(s.Players) < 2 {
		return s, nil, errors.New("need at least 2 players to deal")
	}

	deck := Shuffle(NewDeck(), e.Rng)
	hands, nest := e.Rules.DealCards(deck, s.Players, s.Options)

	s.Nest = nest
	for i := range s.Players {
		s.Players[i].Hand = hands[s.Players[i].ID]
		s.Players[i].HasPassed = false
		s.Players[i].RoundScore = 0
	}

	s.CurrentBid = 0
	s.CurrentBidder = ""
	s.CompletedTricks = nil
	s.CurrentTrick = Trick{}
	s.Trump = NoSuit

	// First bid goes to the player left of the dealer.
	s.BidTurn = (s.Dealer + 1) % len(s.Players)
	s.Phase = PhaseBidding

	var events []Event
	events = append(events, Event{Kind: EventGameStarted})
	for _, p := range s.Players {
		events = append(events, Event{
			Kind:    EventCardDealt,
			Payload: CardDealtPayload{PlayerID: p.ID, Cards: p.Hand},
		})
	}
	return s, events, nil
}

// Apply validates and applies a single player action, returning the new state and events.
func (e *Engine) Apply(s GameState, a Action) (GameState, []Event, error) {
	if err := e.Rules.ValidateAction(s, a); err != nil {
		return s, nil, err
	}

	switch a.Kind {
	case ActionBid, ActionPass:
		return e.Rules.ApplyBid(s, a)

	case ActionNameTrump:
		return e.applyNameTrump(s, a)

	case ActionSetNest:
		return e.applySetNest(s, a)

	case ActionPlayCard:
		return e.applyPlayCard(s, a)
	}
	return s, nil, errors.New("unknown action")
}

func (e *Engine) applyNameTrump(s GameState, a Action) (GameState, []Event, error) {
	s.Trump = a.Trump

	// Give the bidder the nest cards to look at.
	idx := s.PlayerIndex(a.PlayerID)
	s.Players[idx].Hand = append(s.Players[idx].Hand, s.Nest...)

	events := []Event{{
		Kind:    EventTrumpNamed,
		Payload: TrumpNamedPayload{Trump: a.Trump},
	}}
	return s, events, nil
}

func (e *Engine) applySetNest(s GameState, a Action) (GameState, []Event, error) {
	idx := s.PlayerIndex(a.PlayerID)

	// Remove discards from bidder's hand.
	s.Players[idx].Hand = removeCards(s.Players[idx].Hand, a.Discards)
	s.Nest = a.Discards

	// The bidder leads the first trick.
	s.TrickLeader = idx
	s.CurrentTrick = Trick{}
	s.Phase = PhasePlaying

	events := []Event{{Kind: EventNestSet}}
	return s, events, nil
}

func (e *Engine) applyPlayCard(s GameState, a Action) (GameState, []Event, error) {
	idx := s.PlayerIndex(a.PlayerID)
	s.Players[idx].Hand = removeCards(s.Players[idx].Hand, []Card{a.Card})

	s.CurrentTrick.Plays = append(s.CurrentTrick.Plays, Play{
		PlayerID: a.PlayerID,
		Card:     a.Card,
	})

	events := []Event{{
		Kind: EventCardPlayed,
		Payload: CardPlayedPayload{
			PlayerID:   a.PlayerID,
			Card:       a.Card,
			TrickIndex: len(s.CompletedTricks),
		},
	}}

	if len(s.CurrentTrick.Plays) == len(s.Players) {
		// Trick complete — resolve it.
		winnerID, points := e.Rules.ResolveTrick(s, s.CurrentTrick)
		s.CurrentTrick.WonBy = winnerID
		s.CurrentTrick.Points = points
		s.CompletedTricks = append(s.CompletedTricks, s.CurrentTrick)
		s.CurrentTrick = Trick{}
		s.TrickLeader = s.PlayerIndex(winnerID)

		events = append(events, Event{
			Kind:    EventTrickWon,
			Payload: TrickWonPayload{PlayerID: winnerID, Points: points},
		})

		// Check if the round is over (no cards left in any hand).
		if len(s.Players[0].Hand) == 0 {
			var scoreEvents []Event
			var err error
			s, scoreEvents, err = e.Rules.ScoreRound(s)
			if err != nil {
				return s, events, err
			}
			events = append(events, scoreEvents...)
		}
	}

	return s, events, nil
}

func removeCards(hand, toRemove []Card) []Card {
	counts := make(map[Card]int, len(toRemove))
	for _, c := range toRemove {
		counts[c]++
	}
	out := hand[:0:0] // preserve underlying array capacity is not needed; fresh slice
	out = make([]Card, 0, len(hand))
	for _, c := range hand {
		if counts[c] > 0 {
			counts[c]--
		} else {
			out = append(out, c)
		}
	}
	return out
}
