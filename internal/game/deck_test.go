package game

import (
	"testing"
)

func TestNewDeck(t *testing.T) {
	deck := NewDeck()
	if len(deck) != 57 {
		t.Fatalf("deck length = %d, want 57", len(deck))
	}

	seen := make(map[Card]int)
	for _, c := range deck {
		seen[c]++
	}

	// Each card should appear exactly once.
	for c, n := range seen {
		if n != 1 {
			t.Errorf("card %s appears %d times, want 1", c, n)
		}
	}

	// Rook card must be present.
	if _, ok := seen[RookCard()]; !ok {
		t.Error("Rook card missing from deck")
	}
}

func TestTotalPoints(t *testing.T) {
	// 4 suits × (5+10+15+10) = 4×40 = 160, plus Rook=20 → 180
	want := 180
	got := TotalPoints()
	if got != want {
		t.Errorf("TotalPoints = %d, want %d", got, want)
	}
}

func TestCardPointValues(t *testing.T) {
	cases := []struct {
		card Card
		want int
	}{
		{Card{Yellow, 5}, 5},
		{Card{Red, 10}, 10},
		{Card{Green, 14}, 10},
		{Card{Black, 1}, 15},
		{RookCard(), 20},
		{Card{Yellow, 2}, 0},
		{Card{Red, 9}, 0},
		{Card{Green, 13}, 0},
	}
	for _, tc := range cases {
		if got := tc.card.PointValue(); got != tc.want {
			t.Errorf("%s.PointValue() = %d, want %d", tc.card, got, tc.want)
		}
	}
}
