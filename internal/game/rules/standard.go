package rules

import (
	"errors"
	"fmt"

	"github.com/dagnull/cRook/internal/game"
)

// Standard implements the canonical 4-player partnership Rook rules.
type Standard struct{}

var _ game.RuleSet = Standard{}

// ── Validation ────────────────────────────────────────────────────────────────

func (Standard) ValidateAction(s game.GameState, a game.Action) error {
	idx := s.PlayerIndex(a.PlayerID)
	if idx < 0 {
		return errors.New("player not in game")
	}

	switch a.Kind {
	case game.ActionBid, game.ActionPass:
		if s.Phase != game.PhaseBidding {
			return errors.New("not in bidding phase")
		}
		if s.BidTurn != idx {
			return fmt.Errorf("not your turn to bid")
		}
		if a.Kind == game.ActionBid {
			min := s.CurrentBid + s.Options.BidIncrement
			if s.CurrentBid == 0 {
				min = s.Options.MinBid
			}
			if a.Amount < min {
				return fmt.Errorf("bid must be at least %d", min)
			}
			if a.Amount > s.Options.MaxBid {
				return fmt.Errorf("bid cannot exceed %d", s.Options.MaxBid)
			}
			if a.Amount%s.Options.BidIncrement != 0 {
				return fmt.Errorf("bid must be a multiple of %d", s.Options.BidIncrement)
			}
		}

	case game.ActionNameTrump:
		if s.Phase != game.PhaseNesting {
			return errors.New("not in nesting phase")
		}
		if s.CurrentBidder != a.PlayerID {
			return errors.New("only the winning bidder names trump")
		}
		if a.Trump == game.NoSuit {
			return errors.New("must name a valid suit as trump")
		}

	case game.ActionSetNest:
		if s.Phase != game.PhaseNesting {
			return errors.New("not in nesting phase")
		}
		if s.CurrentBidder != a.PlayerID {
			return errors.New("only the winning bidder sets the nest")
		}
		if s.Trump == game.NoSuit {
			return errors.New("name trump before setting the nest")
		}
		if len(a.Discards) != s.Options.NestSize {
			return fmt.Errorf("must discard exactly %d cards", s.Options.NestSize)
		}
		// verify bidder actually holds all discards
		hand := s.Players[idx].Hand
		if err := validateHoldsCards(hand, a.Discards); err != nil {
			return err
		}

	case game.ActionPlayCard:
		if s.Phase != game.PhasePlaying {
			return errors.New("not in playing phase")
		}
		if s.TrickLeader != idx && len(s.CurrentTrick.Plays) != idx-s.TrickLeader {
			// simple turn check: it's idx's turn when Plays count equals distance from leader
			if !isPlayerTurn(s, idx) {
				return errors.New("not your turn to play")
			}
		}
		hand := s.Players[idx].Hand
		if !holdsCard(hand, a.Card) {
			return errors.New("card not in hand")
		}
		if err := validateFollowSuit(s, idx, a.Card); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown action kind: %s", a.Kind)
	}
	return nil
}

func isPlayerTurn(s game.GameState, idx int) bool {
	n := len(s.Players)
	expected := (s.TrickLeader + len(s.CurrentTrick.Plays)) % n
	return expected == idx
}

func validateFollowSuit(s game.GameState, playerIdx int, played game.Card) error {
	if len(s.CurrentTrick.Plays) == 0 {
		return nil // leading the trick, any card is fine
	}
	ledSuit := s.CurrentTrick.Plays[0].Card.Suit
	hand := s.Players[playerIdx].Hand
	if hasSuit(hand, ledSuit) && played.Suit != ledSuit {
		return fmt.Errorf("must follow suit (%s)", ledSuit)
	}
	return nil
}

func hasSuit(hand []game.Card, suit game.Suit) bool {
	for _, c := range hand {
		if !c.IsRook() && c.Suit == suit {
			return true
		}
	}
	return false
}

func holdsCard(hand []game.Card, c game.Card) bool {
	for _, h := range hand {
		if h == c {
			return true
		}
	}
	return false
}

func validateHoldsCards(hand, cards []game.Card) error {
	counts := make(map[game.Card]int)
	for _, c := range hand {
		counts[c]++
	}
	for _, c := range cards {
		if counts[c] == 0 {
			return fmt.Errorf("card not in hand: %s", c)
		}
		counts[c]--
	}
	return nil
}

// ── Dealing ───────────────────────────────────────────────────────────────────

func (Standard) NestSize(opts game.GameOptions) int { return opts.NestSize }

func (Standard) DealCards(deck []game.Card, players []game.PlayerState, opts game.GameOptions) (map[string][]game.Card, []game.Card) {
	nestSize := opts.NestSize
	hands := make(map[string][]game.Card, len(players))
	for _, p := range players {
		hands[p.ID] = []game.Card{}
	}

	nest := make([]game.Card, nestSize)
	copy(nest, deck[:nestSize])

	remaining := deck[nestSize:]
	n := len(players)
	for i, card := range remaining {
		pid := players[i%n].ID
		hands[pid] = append(hands[pid], card)
	}
	return hands, nest
}

// ── Bidding ───────────────────────────────────────────────────────────────────

func (Standard) MinBid(s game.GameState) int { return s.Options.MinBid }

func (Standard) ApplyBid(s game.GameState, a game.Action) (game.GameState, []game.Event, error) {
	var events []game.Event
	n := len(s.Players)

	if a.Kind == game.ActionPass {
		s.Players[s.BidTurn].HasPassed = true
		events = append(events, game.Event{
			Kind:    game.EventPlayerPassed,
			Payload: game.PassPayload{PlayerID: a.PlayerID},
		})
	} else {
		s.CurrentBid = a.Amount
		s.CurrentBidder = a.PlayerID
		events = append(events, game.Event{
			Kind:    game.EventBidPlaced,
			Payload: game.BidPayload{PlayerID: a.PlayerID, Amount: a.Amount},
		})
	}

	// advance to next non-passed player
	active := s.ActivePlayers()
	if len(active) == 1 {
		// bidding over
		s.Phase = game.PhaseNesting
		events = append(events, game.Event{
			Kind:    game.EventBiddingWon,
			Payload: game.BiddingWonPayload{PlayerID: s.CurrentBidder, Amount: s.CurrentBid},
		})
		return s, events, nil
	}

	// find next active player after current BidTurn
	next := (s.BidTurn + 1) % n
	for s.Players[next].HasPassed {
		next = (next + 1) % n
	}
	s.BidTurn = next
	return s, events, nil
}

// ── Trick resolution ──────────────────────────────────────────────────────────

func (Standard) ResolveTrick(s game.GameState, t game.Trick) (string, int) {
	if len(t.Plays) == 0 {
		return "", 0
	}

	ledSuit := t.Plays[0].Card.Suit

	winnerIdx := 0
	winnerCard := t.Plays[0].Card

	for i := 1; i < len(t.Plays); i++ {
		c := t.Plays[i].Card
		if beats(c, winnerCard, ledSuit, s.Trump) {
			winnerCard = c
			winnerIdx = i
		}
	}

	points := 0
	for _, p := range t.Plays {
		points += p.Card.PointValue()
	}

	return t.Plays[winnerIdx].PlayerID, points
}

// beats reports whether challenger beats current winner given led suit and trump.
func beats(challenger, current game.Card, ledSuit, trump game.Suit) bool {
	// Rook always loses to everything except when it's the only card
	// (handled by starting with index 0 as winner; Rook at index 0 can still be beaten)
	if challenger.IsRook() {
		return false
	}
	if current.IsRook() {
		// anything non-Rook beats the Rook
		return true
	}

	chalTrump := challenger.Suit == trump
	curTrump := current.Suit == trump
	chalLed := challenger.Suit == ledSuit
	curLed := current.Suit == ledSuit

	switch {
	case chalTrump && !curTrump:
		return true
	case !chalTrump && curTrump:
		return false
	case chalTrump && curTrump:
		return challenger.Value > current.Value
	case chalLed && !curLed:
		return true
	case !chalLed && curLed:
		return false
	default:
		// neither trump nor led: only matters if same suit
		if challenger.Suit == current.Suit {
			return challenger.Value > current.Value
		}
		return false
	}
}

// ── Scoring ───────────────────────────────────────────────────────────────────

func (Standard) ScoreRound(s game.GameState) (game.GameState, []game.Event, error) {
	// Tally points per team from completed tricks.
	// The nest points go to whichever team won the last trick.
	teamPoints := make(map[int]int, 2)

	lastWinner := ""
	for _, t := range s.CompletedTricks {
		idx := s.PlayerIndex(t.WonBy)
		if idx >= 0 {
			teamPoints[s.Players[idx].Team] += t.Points
			lastWinner = t.WonBy
		}
	}

	// Nest points go to the team that won the last trick.
	if lastWinner != "" {
		nestPoints := 0
		for _, c := range s.Nest {
			nestPoints += c.PointValue()
		}
		idx := s.PlayerIndex(lastWinner)
		if idx >= 0 {
			teamPoints[s.Players[idx].Team] += nestPoints
		}
	}

	// Determine bidder's team and whether they met their bid.
	bidderIdx := s.PlayerIndex(s.CurrentBidder)
	bidderTeam := -1
	if bidderIdx >= 0 {
		bidderTeam = s.Players[bidderIdx].Team
	}
	bidderMet := teamPoints[bidderTeam] >= s.CurrentBid

	// Apply scores.
	var scores []game.PlayerScore
	for i := range s.Players {
		team := s.Players[i].Team
		pts := teamPoints[team]

		// Bidding team gets set (loses bid amount) if they didn't meet it.
		if team == bidderTeam && !bidderMet {
			pts = -s.CurrentBid
		}

		s.Players[i].RoundScore = pts
		s.Players[i].TotalScore += pts
		scores = append(scores, game.PlayerScore{
			PlayerID: s.Players[i].ID,
			Points:   pts,
		})
	}

	events := []game.Event{{
		Kind: game.EventRoundScored,
		Payload: game.RoundScoredPayload{
			Scores:    scores,
			BidderID:  s.CurrentBidder,
			BidAmount: s.CurrentBid,
			BidderMet: bidderMet,
		},
	}}

	s.Phase = game.PhaseScoring
	return s, events, nil
}
