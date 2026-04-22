package game

type EventKind string

const (
	EventGameStarted  EventKind = "game_started"
	EventCardDealt    EventKind = "card_dealt" // per-player; filtered before broadcast
	EventBidPlaced    EventKind = "bid_placed"
	EventPlayerPassed EventKind = "player_passed"
	EventBiddingWon   EventKind = "bidding_won"
	EventTrumpNamed   EventKind = "trump_named"
	EventNestSet      EventKind = "nest_set"
	EventCardPlayed   EventKind = "card_played"
	EventTrickWon     EventKind = "trick_won"
	EventRoundScored  EventKind = "round_scored"
	EventGameOver     EventKind = "game_over"
)

type Event struct {
	Kind    EventKind `json:"kind"`
	Payload any       `json:"payload,omitempty"`
}

// Payload structs for each event kind.

type CardDealtPayload struct {
	PlayerID string `json:"player_id"`
	Cards    []Card `json:"cards"` // filtered to empty slice for other players
}

type BidPayload struct {
	PlayerID string `json:"player_id"`
	Amount   int    `json:"amount"`
}

type PassPayload struct {
	PlayerID string `json:"player_id"`
}

type BiddingWonPayload struct {
	PlayerID string `json:"player_id"`
	Amount   int    `json:"amount"`
}

type TrumpNamedPayload struct {
	Trump Suit `json:"trump"`
}

type CardPlayedPayload struct {
	PlayerID   string `json:"player_id"`
	Card       Card   `json:"card"`
	TrickIndex int    `json:"trick_index"`
}

type TrickWonPayload struct {
	PlayerID string `json:"player_id"`
	Points   int    `json:"points"`
}

type RoundScoredPayload struct {
	Scores    []PlayerScore `json:"scores"`
	BidderID  string        `json:"bidder_id"`
	BidAmount int           `json:"bid_amount"`
	BidderMet bool          `json:"bidder_met"`
}

type GameOverPayload struct {
	WinnerTeam int           `json:"winner_team"`
	Scores     []PlayerScore `json:"scores"`
}
