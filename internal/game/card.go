package game

import (
	"encoding/json"
	"fmt"
)

type Suit uint8

const (
	Yellow Suit = iota
	Red
	Green
	Black
	NoSuit // Rook bird card
)

func (s Suit) String() string {
	switch s {
	case Yellow:
		return "yellow"
	case Red:
		return "red"
	case Green:
		return "green"
	case Black:
		return "black"
	default:
		return "none"
	}
}

func (s Suit) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *Suit) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	switch str {
	case "yellow":
		*s = Yellow
	case "red":
		*s = Red
	case "green":
		*s = Green
	case "black":
		*s = Black
	default:
		*s = NoSuit
	}
	return nil
}

type Card struct {
	Suit  Suit `json:"suit"`
	Value int  `json:"value"` // 1–14; 0 = Rook bird card
}

func RookCard() Card { return Card{Suit: NoSuit, Value: 0} }

func (c Card) IsRook() bool { return c.Value == 0 }

func (c Card) String() string {
	if c.IsRook() {
		return "Rook"
	}
	return fmt.Sprintf("%d-%s", c.Value, c.Suit)
}

// PointValue returns the scoring value of a card.
// Standard Rook scoring: 5s=5, 10s=10, 1s=15, 14s=10, Rook=20, all others=0.
func (c Card) PointValue() int {
	if c.IsRook() {
		return 20
	}
	switch c.Value {
	case 5:
		return 5
	case 10, 14:
		return 10
	case 1:
		return 15
	default:
		return 0
	}
}
