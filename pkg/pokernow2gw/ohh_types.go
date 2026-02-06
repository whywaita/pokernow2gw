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

// OHHSpecFormat represents the official OHH specification format from hh-specs.handhistory.org
type OHHSpecFormat struct {
	ID        string       `json:"id"`
	OHH       OHHSpec      `json:"ohh"`
	Profits   map[string]float64 `json:"_profits,omitempty"`
	EVProfits map[string]float64 `json:"_ev_profits,omitempty"`
	Format    string       `json:"_format,omitempty"`
	CreatedAt time.Time    `json:"createdAt,omitempty"`
}

// OHHSpec represents the OHH specification structure
type OHHSpec struct {
	SpecVersion      string        `json:"spec_version"`
	InternalVersion  string        `json:"internal_version"`
	NetworkName      string        `json:"network_name"`
	SiteName         string        `json:"site_name"`
	GameType         string        `json:"game_type"`
	TableName        string        `json:"table_name"`
	TableSize        int           `json:"table_size"`
	GameNumber       string        `json:"game_number"`
	StartDateUTC     time.Time     `json:"start_date_utc"`
	Currency         string        `json:"currency"`
	AnteAmount       float64       `json:"ante_amount"`
	SmallBlindAmount float64       `json:"small_blind_amount"`
	BigBlindAmount   float64       `json:"big_blind_amount"`
	BetLimit         OHHBetLimit   `json:"bet_limit"`
	DealerSeat       int           `json:"dealer_seat"`
	HeroPlayerID     int           `json:"hero_player_id"`
	Players          []OHHSpecPlayer `json:"players"`
	Rounds           []OHHRound    `json:"rounds"`
	Pots             []OHHPot      `json:"pots"`
}

// OHHBetLimit represents bet limit information
type OHHBetLimit struct {
	BetCap  float64 `json:"bet_cap"`
	BetType string  `json:"bet_type"` // NL, PL, FL
}

// OHHSpecPlayer represents a player in the OHH spec format
type OHHSpecPlayer struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	Seat          int      `json:"seat"`
	StartingStack float64  `json:"starting_stack"`
	Cards         []string `json:"cards,omitempty"`
	UID           string   `json:"_uid,omitempty"`
}

// OHHRound represents a betting round
type OHHRound struct {
	ID      int              `json:"id"`
	Street  string           `json:"street"` // Preflop, Flop, Turn, River
	Cards   []string         `json:"cards"`
	Actions []OHHRoundAction `json:"actions"`
}

// OHHRoundAction represents an action in a round
type OHHRoundAction struct {
	ActionNumber int     `json:"action_number"`
	PlayerID     int     `json:"player_id"`
	Action       string  `json:"action"` // Post SB, Post BB, Fold, Check, Call, Bet, Raise
	Amount       float64 `json:"amount,omitempty"`
	IsAllIn      bool    `json:"is_allin,omitempty"`
}

// OHHPot represents a pot in the hand
type OHHPot struct {
	Number     int             `json:"number"`
	Amount     float64         `json:"amount"`
	Rake       float64         `json:"rake"`
	PlayerWins []OHHPlayerWin  `json:"player_wins"`
}

// OHHPlayerWin represents a player's win in a pot
type OHHPlayerWin struct {
	PlayerID  int     `json:"player_id"`
	WinAmount float64 `json:"win_amount"`
}
