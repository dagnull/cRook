package web

import (
	"encoding/json"
	"fmt"

	"github.com/dagnull/cRook/internal/game"
)

// wsIncoming is the JSON envelope received from a browser client.
type wsIncoming struct {
	Kind     game.ActionKind `json:"kind"`
	Amount   int             `json:"amount"`
	Trump    string          `json:"trump"`
	Card     *cardJSON       `json:"card"`
	Discards []cardJSON      `json:"discards"`
}

type cardJSON struct {
	Suit  string `json:"suit"`
	Value int    `json:"value"`
}

func parseSuit(s string) (game.Suit, error) {
	switch s {
	case "yellow":
		return game.Yellow, nil
	case "red":
		return game.Red, nil
	case "green":
		return game.Green, nil
	case "black":
		return game.Black, nil
	default:
		return game.NoSuit, fmt.Errorf("unknown suit: %q", s)
	}
}

func (c cardJSON) toCard() (game.Card, error) {
	suit, err := parseSuit(c.Suit)
	if err != nil {
		return game.Card{}, err
	}
	if c.Value == 0 {
		return game.RookCard(), nil
	}
	if c.Value < 1 || c.Value > 14 {
		return game.Card{}, fmt.Errorf("invalid card value: %d", c.Value)
	}
	return game.Card{Suit: suit, Value: c.Value}, nil
}

// parseAction converts a raw WebSocket message into a game.Action.
func parseAction(data []byte, playerID string) (game.Action, bool, error) {
	var msg wsIncoming
	if err := json.Unmarshal(data, &msg); err != nil {
		return game.Action{}, false, err
	}

	a := game.Action{Kind: msg.Kind, PlayerID: playerID}

	switch msg.Kind {
	case game.ActionBid:
		a.Amount = msg.Amount

	case game.ActionPass:
		// no extra fields

	case game.ActionNameTrump:
		suit, err := parseSuit(msg.Trump)
		if err != nil {
			return a, false, err
		}
		a.Trump = suit

	case game.ActionSetNest:
		for _, cj := range msg.Discards {
			c, err := cj.toCard()
			if err != nil {
				return a, false, err
			}
			a.Discards = append(a.Discards, c)
		}

	case game.ActionPlayCard:
		if msg.Card == nil {
			return a, false, fmt.Errorf("play_card requires a card field")
		}
		c, err := msg.Card.toCard()
		if err != nil {
			return a, false, err
		}
		a.Card = c

	case "start_game":
		// Special non-Action command handled by the caller.
		return a, true, nil

	default:
		return a, false, fmt.Errorf("unknown action kind: %q", msg.Kind)
	}

	return a, false, nil
}
