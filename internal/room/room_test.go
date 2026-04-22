package room_test

import (
	"testing"

	"github.com/dagnull/cRook/internal/game"
	"github.com/dagnull/cRook/internal/room"
)

func TestRoom_StartAndDispatch(t *testing.T) {
	r := room.New("r1", "Test Room")

	players := []room.Player{
		{ID: "p1", Name: "Alice"},
		{ID: "p2", Name: "Bob"},
		{ID: "p3", Name: "Carol"},
		{ID: "p4", Name: "Dave"},
	}
	for _, p := range players {
		if !r.Join(p) {
			t.Fatalf("Join(%s) failed", p.Name)
		}
	}

	if err := r.StartGame(); err != nil {
		t.Fatalf("StartGame: %v", err)
	}

	sync := r.StateSyncFor("p1")
	if sync.Phase != "bidding" {
		t.Errorf("phase = %q, want bidding", sync.Phase)
	}
	if len(sync.YourHand) != 13 {
		t.Errorf("p1 hand size = %d, want 13", len(sync.YourHand))
	}

	// Other player's hand should not be visible.
	sync2 := r.StateSyncFor("p2")
	if len(sync2.YourHand) != 13 {
		t.Errorf("p2 hand size = %d, want 13", len(sync2.YourHand))
	}
}

func TestRoom_StartGame_TooFewPlayers(t *testing.T) {
	r := room.New("r2", "Solo")
	r.Join(room.Player{ID: "p1", Name: "Lone"})
	if err := r.StartGame(); err == nil {
		t.Error("expected error starting with 1 player")
	}
}

func TestRoom_DispatchBeforeStart(t *testing.T) {
	r := room.New("r3", "Not Started")
	r.Join(room.Player{ID: "p1", Name: "Alice"})
	r.Join(room.Player{ID: "p2", Name: "Bob"})

	err := r.Dispatch(game.Action{Kind: game.ActionBid, PlayerID: "p1", Amount: 70})
	if err == nil {
		t.Error("expected error dispatching before game starts")
	}
}

func TestRoom_FullBidFlow(t *testing.T) {
	r := room.New("r4", "Bid Test")
	for i, p := range []room.Player{
		{ID: "p1", Name: "A"},
		{ID: "p2", Name: "B"},
		{ID: "p3", Name: "C"},
		{ID: "p4", Name: "D"},
	} {
		_ = i
		r.Join(p)
	}
	if err := r.StartGame(); err != nil {
		t.Fatal(err)
	}

	sync := r.StateSyncFor("p1")
	bidTurnID := sync.BidTurn

	// First active player bids.
	if err := r.Dispatch(game.Action{Kind: game.ActionBid, PlayerID: bidTurnID, Amount: 70}); err != nil {
		t.Fatalf("bid: %v", err)
	}

	// Remaining players pass.
	for i := 0; i < 3; i++ {
		s := r.StateSyncFor("p1")
		pid := s.BidTurn
		if pid == "" {
			break
		}
		if err := r.Dispatch(game.Action{Kind: game.ActionPass, PlayerID: pid}); err != nil {
			t.Fatalf("pass: %v", err)
		}
	}

	s := r.StateSyncFor("p1")
	if s.Phase != "nesting" {
		t.Errorf("phase = %q after all-but-one bid, want nesting", s.Phase)
	}
}
