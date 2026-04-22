package game_test

import (
	"math/rand"
	"testing"

	"github.com/dagnull/cRook/internal/game"
	"github.com/dagnull/cRook/internal/game/rules"
)

const winningScore = 300

// TestFullGame_PlayUntilWinner simulates complete rounds until one team reaches
// the winning score. Every phase transition is verified along the way.
func TestFullGame_PlayUntilWinner(t *testing.T) {
	opts := game.DefaultOptions()
	s := game.GameState{
		ID:      "full-game",
		Options: opts,
		Players: []game.PlayerState{
			{ID: "p1", Name: "Alice", Team: 0},
			{ID: "p2", Name: "Bob", Team: 1},
			{ID: "p3", Name: "Carol", Team: 0},
			{ID: "p4", Name: "Dave", Team: 1},
		},
	}
	eng := game.NewEngine(rules.Standard{}, rand.New(rand.NewSource(42)))

	for round := 1; round <= 50; round++ {
		t.Logf("--- Round %d (dealer seat %d) ---", round, s.Dealer)

		s = playOneRound(t, s, eng, round)

		// Log scores after each round.
		team0, team1 := teamTotals(s)
		t.Logf("  Totals → Team 0: %d  Team 1: %d", team0, team1)

		if team0 >= winningScore || team1 >= winningScore {
			winner := 0
			if team1 > team0 {
				winner = 1
			}
			t.Logf("Team %d wins after %d rounds!", winner, round)
			return
		}

		// Rotate dealer for next round.
		s.Dealer = (s.Dealer + 1) % len(s.Players)
	}

	t.Fatal("game did not produce a winner within 50 rounds")
}

// TestFullGame_BidderMakesPoints verifies that when the bidder wins enough
// tricks to cover their bid, their team scores positively and the total
// points across both teams equals TotalPoints().
func TestFullGame_BidderMakesPoints(t *testing.T) {
	// Try seeds until we find a round where the bidder makes their bid.
	for seed := int64(0); seed < 100; seed++ {
		opts := game.DefaultOptions()
		s := game.GameState{
			ID:      "test",
			Options: opts,
			Players: []game.PlayerState{
				{ID: "p1", Name: "Alice", Team: 0},
				{ID: "p2", Name: "Bob", Team: 1},
				{ID: "p3", Name: "Carol", Team: 0},
				{ID: "p4", Name: "Dave", Team: 1},
			},
		}
		eng := game.NewEngine(rules.Standard{}, rand.New(rand.NewSource(seed)))
		s = playOneRound(t, s, eng, 1)

		bidderIdx := s.PlayerIndex(s.CurrentBidder)
		if bidderIdx < 0 {
			continue
		}
		bidderTeam := s.Players[bidderIdx].Team
		bidderScore := s.Players[bidderIdx].RoundScore

		if bidderScore > 0 {
			// Bidder's team made their bid — verify totals sum to TotalPoints.
			team0, team1 := roundScores(s)
			if team0+team1 != game.TotalPoints() {
				t.Errorf("seed %d: round scores %d + %d != %d", seed, team0, team1, game.TotalPoints())
			}
			t.Logf("seed %d: bidder (team %d) made bid — scores %d / %d ✓", seed, bidderTeam, team0, team1)
			return
		}
	}
	t.Error("could not find a round where the bidder made their bid in 100 seeds")
}

// TestFullGame_BidderIsSet verifies that when the bidder fails to cover their
// bid, their team receives a negative round score equal to the bid amount.
func TestFullGame_BidderIsSet(t *testing.T) {
	// Use a high bid (120) to force a set.
	for seed := int64(0); seed < 200; seed++ {
		opts := game.DefaultOptions()
		s := game.GameState{
			ID:      "test",
			Options: opts,
			Players: []game.PlayerState{
				{ID: "p1", Name: "Alice", Team: 0},
				{ID: "p2", Name: "Bob", Team: 1},
				{ID: "p3", Name: "Carol", Team: 0},
				{ID: "p4", Name: "Dave", Team: 1},
			},
		}
		eng := game.NewEngine(rules.Standard{}, rand.New(rand.NewSource(seed)))

		var err error
		s, _, err = eng.Deal(s)
		if err != nil {
			t.Fatal(err)
		}

		firstBidder := s.Players[s.BidTurn].ID
		// Bid the maximum — very hard to make.
		s, _ = mustApply(t, eng, s, game.Action{Kind: game.ActionBid, PlayerID: firstBidder, Amount: 120})
		for s.Phase == game.PhaseBidding {
			s, _ = mustApply(t, eng, s, game.Action{Kind: game.ActionPass, PlayerID: s.Players[s.BidTurn].ID})
		}

		s, _ = mustApply(t, eng, s, game.Action{Kind: game.ActionNameTrump, PlayerID: firstBidder, Trump: game.Yellow})
		bidderIdx := s.PlayerIndex(firstBidder)
		discards := make([]game.Card, 5)
		copy(discards, s.Players[bidderIdx].Hand[:5])
		s, _ = mustApply(t, eng, s, game.Action{Kind: game.ActionSetNest, PlayerID: firstBidder, Discards: discards})

		s = playFullRound(t, s, eng)

		bidderTeam := s.Players[bidderIdx].Team
		teamScore := s.Players[bidderIdx].RoundScore

		if teamScore < 0 {
			// Bidder was set — score must be exactly -bid.
			if teamScore != -120 {
				t.Errorf("seed %d: expected set score -120, got %d", seed, teamScore)
			}
			// Verify teammates have the same round score.
			for _, p := range s.Players {
				if p.Team == bidderTeam && p.RoundScore != teamScore {
					t.Errorf("seed %d: teammate %s has round score %d, want %d", seed, p.Name, p.RoundScore, teamScore)
				}
			}
			t.Logf("seed %d: bidder set correctly (score %d) ✓", seed, teamScore)
			return
		}
	}
	t.Error("could not produce a set in 200 seeds with max bid — something is wrong with scoring")
}

// TestFullGame_AllCardsPlayedEachRound verifies that after each round,
// every player's hand is empty.
func TestFullGame_AllCardsPlayedEachRound(t *testing.T) {
	opts := game.DefaultOptions()
	s := game.GameState{
		ID:      "test",
		Options: opts,
		Players: []game.PlayerState{
			{ID: "p1", Name: "Alice", Team: 0},
			{ID: "p2", Name: "Bob", Team: 1},
			{ID: "p3", Name: "Carol", Team: 0},
			{ID: "p4", Name: "Dave", Team: 1},
		},
	}
	eng := game.NewEngine(rules.Standard{}, rand.New(rand.NewSource(7)))

	for round := 1; round <= 5; round++ {
		s = playOneRound(t, s, eng, round)

		for _, p := range s.Players {
			if len(p.Hand) != 0 {
				t.Errorf("round %d: player %s still has %d cards after round", round, p.Name, len(p.Hand))
			}
		}

		if len(s.CompletedTricks) != 13 {
			t.Errorf("round %d: expected 13 completed tricks, got %d", round, len(s.CompletedTricks))
		}

		s.Dealer = (s.Dealer + 1) % len(s.Players)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// playOneRound runs a full round from PhaseWaiting/PhaseScoring through PhaseScoring.
func playOneRound(t *testing.T, s game.GameState, eng *game.Engine, roundNum int) game.GameState {
	t.Helper()

	// Deal.
	var err error
	s, _, err = eng.Deal(s)
	if err != nil {
		t.Fatalf("round %d Deal: %v", roundNum, err)
	}
	if s.Phase != game.PhaseBidding {
		t.Fatalf("round %d: expected bidding after deal, got %s", roundNum, s.Phase)
	}
	assertAllCardsDealt(t, s, roundNum)

	// Bidding: first player bids MinBid, the rest pass.
	firstBidder := s.Players[s.BidTurn].ID
	bidAmount := s.Options.MinBid
	s, _ = mustApply(t, eng, s, game.Action{Kind: game.ActionBid, PlayerID: firstBidder, Amount: bidAmount})
	for s.Phase == game.PhaseBidding {
		s, _ = mustApply(t, eng, s, game.Action{Kind: game.ActionPass, PlayerID: s.Players[s.BidTurn].ID})
	}
	if s.Phase != game.PhaseNesting {
		t.Fatalf("round %d: expected nesting after bidding, got %s", roundNum, s.Phase)
	}
	if s.CurrentBidder != firstBidder {
		t.Errorf("round %d: winning bidder = %s, want %s", roundNum, s.CurrentBidder, firstBidder)
	}

	// Name trump.
	s, _ = mustApply(t, eng, s, game.Action{Kind: game.ActionNameTrump, PlayerID: firstBidder, Trump: game.Green})

	// Set nest: bidder discards their first 5 cards.
	bidderIdx := s.PlayerIndex(firstBidder)
	discards := make([]game.Card, 5)
	copy(discards, s.Players[bidderIdx].Hand[:5])
	s, _ = mustApply(t, eng, s, game.Action{Kind: game.ActionSetNest, PlayerID: firstBidder, Discards: discards})
	if s.Phase != game.PhasePlaying {
		t.Fatalf("round %d: expected playing after nest, got %s", roundNum, s.Phase)
	}

	// Play all 13 tricks.
	tricksBefore := len(s.CompletedTricks)
	s = playFullRound(t, s, eng)
	if s.Phase != game.PhaseScoring {
		t.Fatalf("round %d: expected scoring after play, got %s", roundNum, s.Phase)
	}
	if len(s.CompletedTricks) != tricksBefore+13 {
		t.Errorf("round %d: completed %d tricks, want 13", roundNum, len(s.CompletedTricks)-tricksBefore)
	}

	// Verify round scores are internally consistent.
	assertRoundScoresConsistent(t, s, roundNum)

	return s
}

// assertAllCardsDealt checks that all 57 cards appear exactly once across
// all player hands and the nest.
func assertAllCardsDealt(t *testing.T, s game.GameState, round int) {
	t.Helper()
	seen := make(map[game.Card]int)
	for _, p := range s.Players {
		for _, c := range p.Hand {
			seen[c]++
		}
	}
	for _, c := range s.Nest {
		seen[c]++
	}
	if len(seen) != 57 {
		t.Errorf("round %d: %d unique cards after deal, want 57", round, len(seen))
	}
	for c, n := range seen {
		if n > 1 {
			t.Errorf("round %d: card %s appears %d times", round, c, n)
		}
	}
}

// assertRoundScoresConsistent checks that:
//   - Both teams have the same round score within each team.
//   - Either the two team scores sum to TotalPoints, or the losing team scored
//     exactly -bid (set case).
func assertRoundScoresConsistent(t *testing.T, s game.GameState, round int) {
	t.Helper()

	// All players on the same team must have the same round score.
	teamScore := map[int]int{}
	for _, p := range s.Players {
		if prev, ok := teamScore[p.Team]; ok {
			if prev != p.RoundScore {
				t.Errorf("round %d: team %d members have differing round scores (%d vs %d)",
					round, p.Team, prev, p.RoundScore)
			}
		}
		teamScore[p.Team] = p.RoundScore
	}

	t0, t1 := teamScore[0], teamScore[1]
	total := game.TotalPoints()

	// If both scores are positive they must sum to TotalPoints.
	if t0 > 0 && t1 > 0 && t0+t1 != total {
		t.Errorf("round %d: positive scores %d + %d != %d", round, t0, t1, total)
	}

	// If one team was set their score is exactly negative bid amount.
	bidderIdx := s.PlayerIndex(s.CurrentBidder)
	if bidderIdx >= 0 {
		bidderTeam := s.Players[bidderIdx].Team
		if teamScore[bidderTeam] < 0 && teamScore[bidderTeam] != -s.CurrentBid {
			t.Errorf("round %d: set score %d != -%d", round, teamScore[bidderTeam], s.CurrentBid)
		}
	}

	t.Logf("  round %d: team 0 = %+d  team 1 = %+d  (bid %d by %s)",
		round, t0, t1, s.CurrentBid, s.CurrentBidder)
}

func teamTotals(s game.GameState) (team0, team1 int) {
	for _, p := range s.Players {
		if p.Team == 0 {
			team0 = p.TotalScore
		} else {
			team1 = p.TotalScore
		}
	}
	return
}

func roundScores(s game.GameState) (team0, team1 int) {
	for _, p := range s.Players {
		if p.Team == 0 {
			team0 = p.RoundScore
		} else {
			team1 = p.RoundScore
		}
	}
	return
}
