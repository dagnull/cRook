package game

type Phase uint8

const (
	PhaseWaiting Phase = iota // room open, waiting for players
	PhaseDealing              // cards being distributed
	PhaseBidding              // players bidding for trump
	PhaseNesting              // winning bidder picks up nest, discards
	PhasePlaying              // trick-taking
	PhaseScoring              // round ended, scores tallied
)

func (p Phase) String() string {
	switch p {
	case PhaseWaiting:
		return "waiting"
	case PhaseDealing:
		return "dealing"
	case PhaseBidding:
		return "bidding"
	case PhaseNesting:
		return "nesting"
	case PhasePlaying:
		return "playing"
	case PhaseScoring:
		return "scoring"
	default:
		return "unknown"
	}
}
