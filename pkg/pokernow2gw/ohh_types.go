package pokernow2gw

import "time"

// OHHFormat represents the Open Hand History JSON format
// This is a JSON-based format for poker hand histories
type OHHFormat struct {
	Version string    `json:"version"`
	Hands   []OHHHand `json:"hands"`
}

// OHHHand represents a single hand in OHH format
type OHHHand struct {
	HandID     string         `json:"handId"`
	HandNumber string         `json:"handNumber"`
	GameType   string         `json:"gameType"` // e.g., "No Limit Texas Hold'em"
	TableName  string         `json:"tableName"`
	StartTime  time.Time      `json:"startTime"`
	Blinds     OHHBlinds      `json:"blinds"`
	Ante       int            `json:"ante,omitempty"`
	Players    []OHHPlayer    `json:"players"`
	Dealer     OHHSeatRef     `json:"dealer"`
	HeroCards  []string       `json:"heroCards,omitempty"`
	Board      OHHBoard       `json:"board,omitempty"`
	Actions    []OHHAction    `json:"actions"`
	Winners    []OHHWinner    `json:"winners,omitempty"`
}

// OHHBlinds represents the blind structure
type OHHBlinds struct {
	SmallBlind int `json:"smallBlind"`
	BigBlind   int `json:"bigBlind"`
}

// OHHPlayer represents a player at the table
type OHHPlayer struct {
	SeatNumber int    `json:"seatNumber"`
	Name       string `json:"name"`
	Stack      int    `json:"stack"`
}

// OHHSeatRef references a player by seat number
type OHHSeatRef struct {
	SeatNumber int `json:"seatNumber"`
}

// OHHBoard represents community cards
type OHHBoard struct {
	Flop  []string `json:"flop,omitempty"`
	Turn  string   `json:"turn,omitempty"`
	River string   `json:"river,omitempty"`
}

// OHHAction represents a player action
type OHHAction struct {
	Player     string `json:"player"`
	ActionType string `json:"actionType"` // fold, check, call, bet, raise, postSB, postBB, postAnte, show, collect, uncalled
	Amount     int    `json:"amount,omitempty"`
	Street     string `json:"street"` // preflop, flop, turn, river, showdown
	IsAllIn    bool   `json:"isAllIn,omitempty"`
}

// OHHWinner represents a pot winner
type OHHWinner struct {
	Player    string   `json:"player"`
	Amount    int      `json:"amount"`
	HandCards []string `json:"handCards,omitempty"`
}
