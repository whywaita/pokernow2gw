package pokernow2gw

import (
	"errors"
	"time"
)

// ErrSpectatorLog is returned when the log is from a spectator (no "Your hand is" entries)
var ErrSpectatorLog = errors.New("spectator log detected: no hero cards found in any hand")

// LogEntry represents a single row from the PokerNow CSV log
type LogEntry struct {
	Entry string    // ログ内容
	At    time.Time // ISO8601 UTC タイムスタンプ
	Order int64     // 並び順キー（昇順で処理）
}

// ConvertOptions contains options for conversion
type ConvertOptions struct {
	HeroName          string            // 表示名（例: "whywaita"）
	SiteName          string            // "GTO Wizard" (default)
	TimeLocation      *time.Location    // "UTC", "Asia/Tokyo" 等
	TournamentName    string            // optional
	TournamentID      string            // optional (will use first hand's ID if not specified)
	PlayerCountFilter PlayerCountFilter // Player count filter for GTO Wizard plans (default: PlayerCountAll)
	RakePercent       float64           // Rake percentage for cash games (e.g., 5.0 for 5%)
	RakeCapBB         float64           // Rake cap in big blinds (e.g., 4.0 for 4BB)
}

// SkipReason represents why a hand was skipped
type SkipReason string

const (
	SkipReasonIncomplete     SkipReason = "incomplete_hand"
	SkipReasonTooManyPlayers SkipReason = "too_many_players"
	SkipReasonFilteredOut    SkipReason = "filtered_out"
)

// SkippedHandInfo contains details about a skipped hand
type SkippedHandInfo struct {
	HandID      string     `json:"hand_id"`
	HandNumber  string     `json:"hand_number"`
	Reason      SkipReason `json:"reason"`
	Detail      string     `json:"detail"`
	PlayerCount int        `json:"player_count,omitempty"`
	RawInput    []string   `json:"raw_input,omitempty"` // 元のCSVエントリ
}

// ConvertResult contains the result of conversion
type ConvertResult struct {
	HH               []byte            // GTO Wizard HH text
	SkippedHands     int               // パースに失敗したハンド数
	SkippedHandsInfo []SkippedHandInfo // スキップされたハンドの詳細情報
}

// Hand represents a parsed poker hand
type Hand struct {
	HandNumber string
	HandID     string
	Dealer     string
	Players    []Player
	Actions    []Action
	Board      Board
	StartTime  time.Time
	SmallBlind int
	BigBlind   int
	Ante       int
	Winners    []Winner
	HeroCards  []string // Heroのハンド（"Your hand is" から取得）
}

// Player represents a player in a hand
type Player struct {
	SeatNumber  int
	Name        string
	DisplayName string // "@" で分割した左側
	Stack       int
}

// Action represents a player action
type Action struct {
	Player     string
	ActionType ActionType
	Amount     int
	Street     Street
	IsAllIn    bool
}

// ActionType represents the type of action
type ActionType int

const (
	ActionFold ActionType = iota
	ActionCheck
	ActionCall
	ActionBet
	ActionRaise
	ActionPostSB
	ActionPostBB
	ActionPostAnte
	ActionShow
	ActionCollect
	ActionUncalled
)

// Street represents the betting round
type Street int

const (
	StreetPreflop Street = iota
	StreetFlop
	StreetTurn
	StreetRiver
	StreetShowdown
)

// Board represents the community cards
type Board struct {
	Flop  []string // 3 cards
	Turn  string
	River string
}

// Winner represents a pot winner
type Winner struct {
	Player    string
	Amount    int
	HandCards []string // ショウダウンで見せたカード
	HandName  string   // optional
}

// PlayerCountFilter represents the player count filter for GTO Wizard plans
// Multiple filters can be combined using bitwise OR (|)
type PlayerCountFilter int

const (
	// PlayerCountAll includes all hands (default, 2-10 players)
	PlayerCountAll PlayerCountFilter = 0
	// PlayerCountHU includes hands with 2 players (Heads-Up)
	PlayerCountHU PlayerCountFilter = 1 << 0
	// PlayerCountSpinAndGo includes hands with 3 players
	PlayerCountSpinAndGo PlayerCountFilter = 1 << 1
	// PlayerCountMTT includes hands with 4-9 players
	PlayerCountMTT PlayerCountFilter = 1 << 2
)

// isPlayerCountAllowed checks if the given player count matches the filter
func (f PlayerCountFilter) isPlayerCountAllowed(playerCount int) bool {
	// If filter is All (0), accept all player counts (2-10)
	if f == PlayerCountAll {
		return true
	}

	// Check each flag
	if f&PlayerCountHU != 0 && playerCount == 2 {
		return true
	}
	if f&PlayerCountSpinAndGo != 0 && playerCount == 3 {
		return true
	}
	if f&PlayerCountMTT != 0 && playerCount >= 4 && playerCount <= 9 {
		return true
	}

	return false
}
