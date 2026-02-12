package pokernow2gw

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestConvertCard(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Basic suit conversions
		{name: "heart ace", input: "A♥", want: "Ah"},
		{name: "diamond king", input: "K♦", want: "Kd"},
		{name: "club queen", input: "Q♣", want: "Qc"},
		{name: "spade jack", input: "J♠", want: "Js"},
		// Number cards
		{name: "2 of spades", input: "2♠", want: "2s"},
		{name: "3 of hearts", input: "3♥", want: "3h"},
		{name: "9 of diamonds", input: "9♦", want: "9d"},
		// 10 should become T
		{name: "10 of spades", input: "10♠", want: "Ts"},
		{name: "10 of hearts", input: "10♥", want: "Th"},
		{name: "10 of diamonds", input: "10♦", want: "Td"},
		{name: "10 of clubs", input: "10♣", want: "Tc"},
		// Already-converted cards should pass through
		{name: "already converted Ah", input: "Ah", want: "Ah"},
		{name: "already converted 2s", input: "2s", want: "2s"},
		{name: "already converted Td", input: "Td", want: "Td"},
		// Whitespace handling
		{name: "leading whitespace", input: "  A♥", want: "Ah"},
		{name: "trailing whitespace", input: "A♥  ", want: "Ah"},
		{name: "both whitespace", input: "  K♦  ", want: "Kd"},
		// Empty string
		{name: "empty string", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertCard(tt.input)
			if got != tt.want {
				t.Errorf("convertCard(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseCards(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "two cards with unicode suits",
			input: "A♥, K♦",
			want:  []string{"Ah", "Kd"},
		},
		{
			name:  "three flop cards",
			input: "2♥, 3♦, 4♠",
			want:  []string{"2h", "3d", "4s"},
		},
		{
			name:  "cards with 10",
			input: "10♦, 10♣",
			want:  []string{"Td", "Tc"},
		},
		{
			name:  "single card",
			input: "J♣",
			want:  []string{"Jc"},
		},
		{
			name:  "already converted format",
			input: "Ah, Kd",
			want:  []string{"Ah", "Kd"},
		},
		{
			name:  "mixed spacing",
			input: "A♥,K♦, Q♠",
			want:  []string{"Ah", "Kd", "Qs"},
		},
		{
			name:  "showdown hand format",
			input: "A♠, 7♥",
			want:  []string{"As", "7h"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCards(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("parseCards(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestExtractDisplayName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard player with ID",
			input: "whywaita @ DtjzvbAuKs",
			want:  "whywaita",
		},
		{
			name:  "name with internal @ symbol",
			input: "spa @ ces @ ZQfm6ZDMPO",
			want:  "spa@ces",
		},
		{
			name:  "name with multiple @ symbols",
			input: "@atsymbols@@@ @ 3mid0aO0hZ",
			want:  "@atsymbols@@@",
		},
		{
			name:  "name with quotes (CSV escaping)",
			input: `""quotes""' @ us2R6psQVF`,
			want:  `quotes'`,
		},
		{
			name:  "simple name without @",
			input: "simplename",
			want:  "simplename",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "name with spaces",
			input: "player name @ abc123",
			want:  "playername",
		},
		{
			name:  "single character name",
			input: "x @ id1",
			want:  "x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDisplayName(tt.input)
			if got != tt.want {
				t.Errorf("extractDisplayName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCalculateRake(t *testing.T) {
	tests := []struct {
		name        string
		totalPot    float64
		bigBlind    float64
		rakePercent float64
		rakeCapBB   float64
		want        float64
	}{
		{
			name:        "no rake (zero percent)",
			totalPot:    1000,
			bigBlind:    100,
			rakePercent: 0,
			rakeCapBB:   0,
			want:        0,
		},
		{
			name:        "negative rake percent",
			totalPot:    1000,
			bigBlind:    100,
			rakePercent: -5.0,
			rakeCapBB:   0,
			want:        0,
		},
		{
			name:        "5% rake no cap",
			totalPot:    1000,
			bigBlind:    100,
			rakePercent: 5.0,
			rakeCapBB:   0,
			want:        50,
		},
		{
			name:        "5% rake with cap not reached",
			totalPot:    200,
			bigBlind:    100,
			rakePercent: 5.0,
			rakeCapBB:   4.0,
			want:        10, // 5% of 200 = 10, cap = 400, not reached
		},
		{
			name:        "5% rake with cap reached",
			totalPot:    10000,
			bigBlind:    100,
			rakePercent: 5.0,
			rakeCapBB:   4.0,
			want:        400, // 5% of 10000 = 500, cap = 4*100 = 400
		},
		{
			name:        "5% rake at exactly cap",
			totalPot:    8000,
			bigBlind:    100,
			rakePercent: 5.0,
			rakeCapBB:   4.0,
			want:        400, // 5% of 8000 = 400, cap = 400
		},
		{
			name:        "small pot",
			totalPot:    10,
			bigBlind:    100,
			rakePercent: 5.0,
			rakeCapBB:   4.0,
			want:        0.5, // 5% of 10 = 0.5
		},
		{
			name:        "zero pot",
			totalPot:    0,
			bigBlind:    100,
			rakePercent: 5.0,
			rakeCapBB:   4.0,
			want:        0,
		},
		{
			name:        "10% rake",
			totalPot:    1000,
			bigBlind:    50,
			rakePercent: 10.0,
			rakeCapBB:   0,
			want:        100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateRake(tt.totalPot, tt.bigBlind, tt.rakePercent, tt.rakeCapBB)
			if got != tt.want {
				t.Errorf("calculateRake(%f, %f, %f, %f) = %f, want %f",
					tt.totalPot, tt.bigBlind, tt.rakePercent, tt.rakeCapBB, got, tt.want)
			}
		})
	}
}

func TestCalculateTotalPot(t *testing.T) {
	tests := []struct {
		name string
		hand Hand
		want float64
	}{
		{
			name: "single winner",
			hand: Hand{
				Winners: []Winner{
					{Player: "player1", Amount: 100},
				},
			},
			want: 100,
		},
		{
			name: "multiple winners (split pot)",
			hand: Hand{
				Winners: []Winner{
					{Player: "player1", Amount: 170},
					{Player: "player2", Amount: 170},
				},
			},
			want: 340,
		},
		{
			name: "no winners",
			hand: Hand{
				Winners: nil,
			},
			want: 0,
		},
		{
			name: "winner with zero amount (showed but lost)",
			hand: Hand{
				Winners: []Winner{
					{Player: "player1", Amount: 200},
					{Player: "player2", Amount: 0},
				},
			},
			want: 200,
		},
		{
			name: "three winners (side pots)",
			hand: Hand{
				Winners: []Winner{
					{Player: "player1", Amount: 300},
					{Player: "player2", Amount: 200},
					{Player: "player3", Amount: 100},
				},
			},
			want: 600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateTotalPot(tt.hand)
			if got != tt.want {
				t.Errorf("calculateTotalPot() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestNormalizePlayerSeats(t *testing.T) {
	tests := []struct {
		name string
		in   []Player
		want []Player
	}{
		{
			name: "already normalized (1,2,3)",
			in: []Player{
				{SeatNumber: 1, DisplayName: "a", Stack: 100},
				{SeatNumber: 2, DisplayName: "b", Stack: 200},
				{SeatNumber: 3, DisplayName: "c", Stack: 300},
			},
			want: []Player{
				{SeatNumber: 1, DisplayName: "a", Stack: 100},
				{SeatNumber: 2, DisplayName: "b", Stack: 200},
				{SeatNumber: 3, DisplayName: "c", Stack: 300},
			},
		},
		{
			name: "non-consecutive seats (3,7,10)",
			in: []Player{
				{SeatNumber: 3, DisplayName: "kate", Stack: 500},
				{SeatNumber: 7, DisplayName: "leo", Stack: 600},
				{SeatNumber: 10, DisplayName: "mike", Stack: 700},
			},
			want: []Player{
				{SeatNumber: 1, DisplayName: "kate", Stack: 500},
				{SeatNumber: 2, DisplayName: "leo", Stack: 600},
				{SeatNumber: 3, DisplayName: "mike", Stack: 700},
			},
		},
		{
			name: "reverse order input",
			in: []Player{
				{SeatNumber: 10, DisplayName: "c", Stack: 300},
				{SeatNumber: 5, DisplayName: "b", Stack: 200},
				{SeatNumber: 1, DisplayName: "a", Stack: 100},
			},
			want: []Player{
				{SeatNumber: 1, DisplayName: "a", Stack: 100},
				{SeatNumber: 2, DisplayName: "b", Stack: 200},
				{SeatNumber: 3, DisplayName: "c", Stack: 300},
			},
		},
		{
			name: "single player",
			in: []Player{
				{SeatNumber: 5, DisplayName: "solo", Stack: 1000},
			},
			want: []Player{
				{SeatNumber: 1, DisplayName: "solo", Stack: 1000},
			},
		},
		{
			name: "empty slice",
			in:   []Player{},
			want: []Player{},
		},
		{
			name: "nil slice",
			in:   nil,
			want: nil,
		},
		{
			name: "two players at seats 5 and 9",
			in: []Player{
				{SeatNumber: 5, DisplayName: "x", Stack: 500},
				{SeatNumber: 9, DisplayName: "y", Stack: 900},
			},
			want: []Player{
				{SeatNumber: 1, DisplayName: "x", Stack: 500},
				{SeatNumber: 2, DisplayName: "y", Stack: 900},
			},
		},
		{
			name: "preserves other fields",
			in: []Player{
				{SeatNumber: 8, Name: "full name @ id1", DisplayName: "display", Stack: 1234},
			},
			want: []Player{
				{SeatNumber: 1, Name: "full name @ id1", DisplayName: "display", Stack: 1234},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePlayerSeats(tt.in)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("normalizePlayerSeats() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsJSONFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid JSON object",
			input: `{"key": "value"}`,
			want:  true,
		},
		{
			name:  "valid JSON with whitespace",
			input: `  { "key": "value" }  `,
			want:  true,
		},
		{
			name:  "nested JSON object",
			input: `{"ohh": {"spec_version": "1.0"}}`,
			want:  true,
		},
		{
			name:  "invalid JSON starting with brace",
			input: `{invalid}`,
			want:  false,
		},
		{
			name:  "CSV data",
			input: "entry,at,order\ntest,2025-01-01T00:00:00Z,1",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "only whitespace",
			input: "   \n\t  ",
			want:  false,
		},
		{
			name:  "JSON array (not object)",
			input: `[1, 2, 3]`,
			want:  false,
		},
		{
			name:  "plain text",
			input: "hello world",
			want:  false,
		},
		{
			name:  "JSONL (multiple JSON objects)",
			input: "{\"a\":1}\n{\"b\":2}",
			want:  false, // Not valid as a single JSON object
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJSONFormat([]byte(tt.input))
			if got != tt.want {
				t.Errorf("isJSONFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsJSONLFormat_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid JSONL with two lines",
			input: "{\"id\":\"1\"}\n{\"id\":\"2\"}",
			want:  true,
		},
		{
			name:  "single JSON object (not JSONL)",
			input: `{"id":"1","name":"test"}`,
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "only whitespace",
			input: "   ",
			want:  false,
		},
		{
			name:  "CSV data",
			input: "entry,at,order\ntest,2025-01-01T00:00:00Z,1",
			want:  false,
		},
		{
			name:  "plain text",
			input: "not json at all",
			want:  false,
		},
		{
			name:  "valid JSONL with trailing newline",
			input: "{\"a\":1}\n{\"b\":2}\n",
			want:  true,
		},
		{
			name:  "JSON array (not JSONL)",
			input: "[1, 2, 3]",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJSONLFormat([]byte(tt.input))
			if got != tt.want {
				t.Errorf("isJSONLFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertHandIDToNumeric(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "already numeric", input: "12345"},
		{name: "alphabetic ID", input: "abcdef"},
		{name: "alphanumeric ID", input: "test123"},
		{name: "single char", input: "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertHandIDToNumeric(tt.input)
			if got == "" {
				t.Errorf("convertHandIDToNumeric(%q) returned empty string", tt.input)
			}
			// For numeric input, should return same value
			if tt.input == "12345" && got != "12345" {
				t.Errorf("convertHandIDToNumeric(%q) = %q, want %q", tt.input, got, "12345")
			}
			// For non-numeric input, should return a consistent numeric string
			if tt.input == "abcdef" {
				got2 := convertHandIDToNumeric(tt.input)
				if got != got2 {
					t.Errorf("convertHandIDToNumeric(%q) not deterministic: %q != %q", tt.input, got, got2)
				}
			}
		})
	}
}

func TestStreetToFoldString(t *testing.T) {
	tests := []struct {
		name   string
		street Street
		want   string
	}{
		{name: "preflop", street: StreetPreflop, want: "before Flop"},
		{name: "flop", street: StreetFlop, want: "on the Flop"},
		{name: "turn", street: StreetTurn, want: "on the Turn"},
		{name: "river", street: StreetRiver, want: "on the River"},
		{name: "showdown (default)", street: StreetShowdown, want: "before Flop"},
		{name: "unknown value", street: Street(99), want: "before Flop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := streetToFoldString(tt.street)
			if got != tt.want {
				t.Errorf("streetToFoldString(%d) = %q, want %q", tt.street, got, tt.want)
			}
		})
	}
}

func TestGetDealerSeat(t *testing.T) {
	tests := []struct {
		name string
		hand Hand
		want int
	}{
		{
			name: "dealer found",
			hand: Hand{
				Dealer: "alice",
				Players: []Player{
					{SeatNumber: 1, DisplayName: "bob"},
					{SeatNumber: 2, DisplayName: "alice"},
					{SeatNumber: 3, DisplayName: "charlie"},
				},
			},
			want: 2,
		},
		{
			name: "dealer not found (defaults to 1)",
			hand: Hand{
				Dealer: "unknown",
				Players: []Player{
					{SeatNumber: 1, DisplayName: "bob"},
					{SeatNumber: 2, DisplayName: "alice"},
				},
			},
			want: 1,
		},
		{
			name: "empty dealer (dead button)",
			hand: Hand{
				Dealer: "",
				Players: []Player{
					{SeatNumber: 1, DisplayName: "bob"},
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDealerSeat(tt.hand)
			if got != tt.want {
				t.Errorf("getDealerSeat() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGoldenFile(t *testing.T) {
	inputPath := "../../sample/input/poker_now_log_pglhniqprRDmWFv9sLLZZA-ru.csv"
	outputPath := "../../sample/output/poker_now_log_pglhniqprRDmWFv9sLLZZA-ru.txt"

	// Skip if sample files don't exist
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		t.Skip("Sample input file not found, skipping golden file test")
	}
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Skip("Sample output file not found, skipping golden file test")
	}

	// Read input
	inputFile, err := os.Open(inputPath)
	if err != nil {
		t.Fatalf("Failed to open input file: %v", err)
	}
	defer inputFile.Close()

	// Read expected output
	expectedBytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read expected output file: %v", err)
	}
	expected := string(expectedBytes)

	// Run conversion
	opts := ConvertOptions{
		HeroName:     "whywaita",
		SiteName:     "PokerStars",
		TimeLocation: time.UTC,
	}

	result, err := ParseCSV(inputFile, opts)
	if err != nil {
		t.Fatalf("ParseCSV() error: %v", err)
	}

	if result == nil {
		t.Fatal("ParseCSV() returned nil result")
	}

	got := string(result.HH)

	// Compare output
	if diff := cmp.Diff(expected, got); diff != "" {
		// Show a more helpful message with the first few lines of diff
		lines := strings.Split(diff, "\n")
		maxLines := 50
		if len(lines) > maxLines {
			lines = lines[:maxLines]
		}
		t.Errorf("Golden file mismatch (-expected +got):\n%s\n... (showing first %d lines of diff)",
			strings.Join(lines, "\n"), maxLines)
	}
}

// TestConvertOHHActionType_Helpers tests the OHH action type converter with various inputs
func TestConvertOHHActionType_Helpers(t *testing.T) {
	tests := []struct {
		input   string
		want    ActionType
		wantErr bool
	}{
		{"Post SB", ActionPostSB, false},
		{"Post BB", ActionPostBB, false},
		{"Post Ante", ActionPostAnte, false},
		{"Fold", ActionFold, false},
		{"Check", ActionCheck, false},
		{"Call", ActionCall, false},
		{"Bet", ActionBet, false},
		{"Raise", ActionRaise, false},
		{"Show", ActionShow, false},
		{"Collect", ActionCollect, false},
		{"Uncalled", ActionUncalled, false},
		// Case insensitive
		{"fold", ActionFold, false},
		{"FOLD", ActionFold, false},
		// Unknown action
		{"unknown_action", ActionFold, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := convertOHHActionType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertOHHActionType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("convertOHHActionType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   string
	}{
		{
			name:   "integer without decimal",
			amount: 1.0,
			want:   "1",
		},
		{
			name:   "integer value",
			amount: 100,
			want:   "100",
		},
		{
			name:   "half dollar",
			amount: 0.5,
			want:   "0.50",
		},
		{
			name:   "one and half",
			amount: 1.5,
			want:   "1.50",
		},
		{
			name:   "zero",
			amount: 0,
			want:   "0",
		},
		{
			name:   "large integer",
			amount: 10000,
			want:   "10000",
		},
		{
			name:   "small decimal",
			amount: 0.25,
			want:   "0.25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNumber(tt.amount)
			if got != tt.want {
				t.Errorf("formatNumber(%f) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}
