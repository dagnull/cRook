package game

// RuleSet encodes all variant-specific game logic.
// Implementations live in the game/rules sub-package.
type RuleSet interface {
	ValidateAction(s GameState, a Action) error
	DealCards(deck []Card, players []PlayerState, opts GameOptions) (hands map[string][]Card, nest []Card)
	NestSize(opts GameOptions) int
	MinBid(s GameState) int
	ApplyBid(s GameState, a Action) (GameState, []Event, error)
	ResolveTrick(s GameState, t Trick) (winnerID string, points int)
	ScoreRound(s GameState) (GameState, []Event, error)
}
