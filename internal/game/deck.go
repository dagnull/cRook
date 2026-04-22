package game

import "math/rand"

// NewDeck returns the standard 57-card Rook deck:
// values 1–14 in each of 4 suits, plus the Rook bird card.
func NewDeck() []Card {
	deck := make([]Card, 0, 57)
	for _, suit := range []Suit{Yellow, Red, Green, Black} {
		for v := 1; v <= 14; v++ {
			deck = append(deck, Card{Suit: suit, Value: v})
		}
	}
	deck = append(deck, RookCard())
	return deck
}

func Shuffle(deck []Card, rng *rand.Rand) []Card {
	out := make([]Card, len(deck))
	copy(out, deck)
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}
