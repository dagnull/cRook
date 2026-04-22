package game

type GameState struct {
	ID     string
	Phase  Phase
	Round  int
	Dealer int // index into Players

	Players []PlayerState

	Nest  []Card
	Trump Suit

	// Bidding
	CurrentBid    int
	CurrentBidder string // PlayerID of current high bidder
	BidTurn       int    // index into Players whose turn it is to bid
	Passed        []bool // parallel to Players

	// Playing
	CurrentTrick    Trick
	CompletedTricks []Trick
	TrickLeader     int // index into Players who leads the current trick

	// Options are set once at game creation and don't change.
	Options GameOptions
}

type PlayerState struct {
	ID         string
	Name       string
	Team       int // 0 or 1
	Hand       []Card
	RoundScore int
	TotalScore int
	HasPassed  bool
}

type Bid struct {
	PlayerID string
	Amount   int
}

type Play struct {
	PlayerID string `json:"player_id"`
	Card     Card   `json:"card"`
}

type Trick struct {
	Plays  []Play `json:"plays"`
	WonBy  string `json:"won_by"`
	Points int    `json:"points"`
}

type PlayerScore struct {
	PlayerID string
	Points   int
	Met      bool // bidder met their bid
}

type GameOptions struct {
	NestSize     int // default 5
	MinBid       int // default 70
	MaxBid       int // default 120
	BidIncrement int // default 5
	WinningScore int // game ends when a team reaches this; 0 = single round
	NumPlayers   int // default 4
}

func DefaultOptions() GameOptions {
	return GameOptions{
		NestSize:     5,
		MinBid:       70,
		MaxBid:       120,
		BidIncrement: 5,
		NumPlayers:   4,
	}
}

// ActivePlayers returns the indices of players who have not yet passed.
func (s *GameState) ActivePlayers() []int {
	var out []int
	for i, p := range s.Players {
		if !p.HasPassed {
			out = append(out, i)
		}
	}
	return out
}

// PlayerIndex returns the index of the player with the given ID, or -1.
func (s *GameState) PlayerIndex(id string) int {
	for i, p := range s.Players {
		if p.ID == id {
			return i
		}
	}
	return -1
}

// TotalPoints returns the total point value of all cards in the deck.
func TotalPoints() int {
	total := 0
	for _, c := range NewDeck() {
		total += c.PointValue()
	}
	return total
}
