package game

type ActionKind string

const (
	ActionBid       ActionKind = "bid"
	ActionPass      ActionKind = "pass"
	ActionNameTrump ActionKind = "name_trump"
	ActionSetNest   ActionKind = "set_nest" // bidder discards to nest
	ActionPlayCard  ActionKind = "play_card"
)

type Action struct {
	Kind     ActionKind
	PlayerID string

	// Bid / Pass
	Amount int

	// NameTrump
	Trump Suit

	// SetNest: cards the bidder puts back into the nest
	Discards []Card

	// PlayCard
	Card Card
}
