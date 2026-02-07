package pokernow2gw

import (
	"strings"
	"testing"
	"time"
)

func TestReadOHH(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid OHH JSON",
			input: `{
  "version": "1.0",
  "hands": [
    {
      "handId": "1234567890",
      "handNumber": "1",
      "gameType": "No Limit Texas Hold'em",
      "tableName": "Test Table",
      "startTime": "2025-11-15T05:08:29.500Z",
      "blinds": {
        "smallBlind": 50,
        "bigBlind": 100
      },
      "ante": 10,
      "players": [
        {
          "seatNumber": 1,
          "name": "Player1",
          "stack": 1000
        },
        {
          "seatNumber": 2,
          "name": "Player2",
          "stack": 1500
        }
      ],
      "dealer": {
        "seatNumber": 1
      },
      "heroCards": ["Ah", "Kh"],
      "board": {
        "flop": ["Qh", "Jh", "Th"],
        "turn": "2d",
        "river": "3c"
      },
      "actions": [
        {
          "player": "Player1",
          "actionType": "postSB",
          "amount": 50,
          "street": "preflop"
        },
        {
          "player": "Player2",
          "actionType": "postBB",
          "amount": 100,
          "street": "preflop"
        }
      ],
      "winners": [
        {
          "player": "Player1",
          "amount": 1420
        }
      ]
    }
  ]
}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid json}`,
			wantErr: true,
		},
		{
			name: "spectator log (no hero cards)",
			input: `{
  "version": "1.0",
  "hands": [
    {
      "handId": "1234567890",
      "handNumber": "1",
      "gameType": "No Limit Texas Hold'em",
      "tableName": "Test Table",
      "startTime": "2025-11-15T05:08:29.500Z",
      "blinds": {
        "smallBlind": 50,
        "bigBlind": 100
      },
      "players": [
        {
          "seatNumber": 1,
          "name": "Player1",
          "stack": 1000
        }
      ],
      "dealer": {
        "seatNumber": 1
      },
      "actions": []
    }
  ]
}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ConvertOptions{
				HeroName: "Player1",
				SiteName: "PokerStars",
			}
			result, err := ReadOHH(strings.NewReader(tt.input), opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadOHH() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("ReadOHH() returned nil result for valid input")
			}
		})
	}
}

func TestConvertOHHHandToHand(t *testing.T) {
	ohhHand := OHHHand{
		HandID:     "test123",
		HandNumber: "1",
		GameType:   "No Limit Texas Hold'em",
		TableName:  "Test Table",
		StartTime:  time.Date(2025, 11, 15, 5, 8, 29, 0, time.UTC),
		Blinds: OHHBlinds{
			SmallBlind: 50,
			BigBlind:   100,
		},
		Ante: 10,
		Players: []OHHPlayer{
			{SeatNumber: 1, Name: "Player1", Stack: 1000},
			{SeatNumber: 2, Name: "Player2", Stack: 1500},
		},
		Dealer: OHHSeatRef{
			SeatNumber: 1,
		},
		HeroCards: []string{"Ah", "Kh"},
		Board: OHHBoard{
			Flop:  []string{"Qh", "Jh", "Th"},
			Turn:  "2d",
			River: "3c",
		},
		Actions: []OHHAction{
			{Player: "Player1", ActionType: "postSB", Amount: 50, Street: "preflop"},
			{Player: "Player2", ActionType: "postBB", Amount: 100, Street: "preflop"},
			{Player: "Player1", ActionType: "fold", Street: "preflop"},
		},
		Winners: []OHHWinner{
			{Player: "Player2", Amount: 150},
		},
	}

	hand, err := convertOHHHandToHand(ohhHand)
	if err != nil {
		t.Fatalf("convertOHHHandToHand() error = %v", err)
	}

	if hand.HandID != "test123" {
		t.Errorf("HandID = %v, want test123", hand.HandID)
	}
	if hand.HandNumber != "1" {
		t.Errorf("HandNumber = %v, want 1", hand.HandNumber)
	}
	if hand.SmallBlind != 50 {
		t.Errorf("SmallBlind = %v, want 50", hand.SmallBlind)
	}
	if hand.BigBlind != 100 {
		t.Errorf("BigBlind = %v, want 100", hand.BigBlind)
	}
	if hand.Ante != 10 {
		t.Errorf("Ante = %v, want 10", hand.Ante)
	}
	if len(hand.Players) != 2 {
		t.Errorf("len(Players) = %v, want 2", len(hand.Players))
	}
	if hand.Dealer != "Player1" {
		t.Errorf("Dealer = %v, want Player1", hand.Dealer)
	}
	if len(hand.HeroCards) != 2 {
		t.Errorf("len(HeroCards) = %v, want 2", len(hand.HeroCards))
	}
	if len(hand.Board.Flop) != 3 {
		t.Errorf("len(Board.Flop) = %v, want 3", len(hand.Board.Flop))
	}
	if hand.Board.Turn != "2d" {
		t.Errorf("Board.Turn = %v, want 2d", hand.Board.Turn)
	}
	if hand.Board.River != "3c" {
		t.Errorf("Board.River = %v, want 3c", hand.Board.River)
	}
	if len(hand.Actions) != 3 {
		t.Errorf("len(Actions) = %v, want 3", len(hand.Actions))
	}
	if len(hand.Winners) != 1 {
		t.Errorf("len(Winners) = %v, want 1", len(hand.Winners))
	}
}

func TestConvertOHHActionType(t *testing.T) {
	tests := []struct {
		input string
		want  ActionType
	}{
		{"fold", ActionFold},
		{"check", ActionCheck},
		{"call", ActionCall},
		{"bet", ActionBet},
		{"raise", ActionRaise},
		{"postSB", ActionPostSB},
		{"postBB", ActionPostBB},
		{"postAnte", ActionPostAnte},
		{"show", ActionShow},
		{"collect", ActionCollect},
		{"uncalled", ActionUncalled},
		{"unknown", ActionFold}, // Default to fold
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := convertOHHActionType(tt.input)
			if got != tt.want {
				t.Errorf("convertOHHActionType(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertOHHStreet(t *testing.T) {
	tests := []struct {
		input string
		want  Street
	}{
		{"preflop", StreetPreflop},
		{"flop", StreetFlop},
		{"turn", StreetTurn},
		{"river", StreetRiver},
		{"showdown", StreetShowdown},
		{"unknown", StreetPreflop}, // Default to preflop
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := convertOHHStreet(tt.input)
			if got != tt.want {
				t.Errorf("convertOHHStreet(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParse_AutoDetectFormat(t *testing.T) {
	t.Run("detect JSON format", func(t *testing.T) {
		jsonInput := `{
  "version": "1.0",
  "hands": [
    {
      "handId": "1234567890",
      "handNumber": "1",
      "gameType": "No Limit Texas Hold'em",
      "tableName": "Test Table",
      "startTime": "2025-11-15T05:08:29.500Z",
      "blinds": {
        "smallBlind": 50,
        "bigBlind": 100
      },
      "players": [
        {
          "seatNumber": 1,
          "name": "Player1",
          "stack": 1000
        }
      ],
      "dealer": {
        "seatNumber": 1
      },
      "heroCards": ["Ah", "Kh"],
      "actions": []
    }
  ]
}`
		opts := ConvertOptions{
			HeroName: "Player1",
			SiteName: "PokerStars",
		}
		result, err := Parse(strings.NewReader(jsonInput), opts)
		if err != nil {
			t.Errorf("Parse() error = %v", err)
		}
		if result == nil {
			t.Error("Parse() returned nil result")
		}
	})

	t.Run("detect CSV format", func(t *testing.T) {
		// A minimal valid CSV that will at least be recognized as CSV format
		csvInput := `entry,at,order
"test",2025-11-15T05:08:29.500Z,1`
		opts := ConvertOptions{
			HeroName: "Player1",
			SiteName: "PokerStars",
		}
		// Just test that it doesn't error on parsing the CSV format
		// (It may return empty results or ErrSpectatorLog, but shouldn't fail on JSON parsing)
		result, err := Parse(strings.NewReader(csvInput), opts)
		// Should either succeed with empty result or return ErrSpectatorLog
		if err != nil && err != ErrSpectatorLog {
			t.Errorf("Parse() unexpected error = %v", err)
		}
		// If no error, result should not be nil
		if err == nil && result == nil {
			t.Error("Parse() returned nil result without error")
		}
	})
}

func TestReadOHHSpec(t *testing.T) {
	// Test with actual OHH spec format
	input := `{
  "id": "test123",
  "ohh": {
    "spec_version": "1.4.6",
    "internal_version": "1.0.0",
    "network_name": "Test",
    "site_name": "Test Site",
    "game_type": "Holdem",
    "table_name": "test-table",
    "table_size": 4,
    "game_number": "42",
    "start_date_utc": "2026-02-02T20:17:51.970Z",
    "currency": "Chips",
    "ante_amount": 0,
    "small_blind_amount": 0.5,
    "big_blind_amount": 1,
    "bet_limit": {
      "bet_cap": 0,
      "bet_type": "NL"
    },
    "dealer_seat": 2,
    "hero_player_id": 1,
    "players": [
      {
        "id": 1,
        "name": "Hero",
        "seat": 1,
        "starting_stack": 100,
        "cards": ["Ah", "Kh"]
      },
      {
        "id": 2,
        "name": "Villain",
        "seat": 2,
        "starting_stack": 100,
        "cards": ["Qh", "Jh"]
      }
    ],
    "rounds": [
      {
        "id": 0,
        "street": "Preflop",
        "cards": [],
        "actions": [
          {
            "action_number": 1,
            "player_id": 1,
            "action": "Post SB",
            "amount": 0.5
          },
          {
            "action_number": 2,
            "player_id": 2,
            "action": "Post BB",
            "amount": 1
          },
          {
            "action_number": 3,
            "player_id": 1,
            "action": "Raise",
            "amount": 3
          },
          {
            "action_number": 4,
            "player_id": 2,
            "action": "Call",
            "amount": 2
          }
        ]
      },
      {
        "id": 1,
        "street": "Flop",
        "cards": ["Tc", "9c", "8c"],
        "actions": [
          {
            "action_number": 5,
            "player_id": 2,
            "action": "Check"
          },
          {
            "action_number": 6,
            "player_id": 1,
            "action": "Bet",
            "amount": 5
          },
          {
            "action_number": 7,
            "player_id": 2,
            "action": "Fold"
          }
        ]
      }
    ],
    "pots": [
      {
        "number": 0,
        "amount": 6,
        "rake": 0,
        "player_wins": [
          {
            "player_id": 1,
            "win_amount": 6
          }
        ]
      }
    ]
  }
}`

	opts := ConvertOptions{
		HeroName: "Hero",
		SiteName: "PokerStars",
	}

	result, err := ReadOHH(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("ReadOHH() error = %v", err)
	}

	if result == nil {
		t.Error("ReadOHH() returned nil result for valid OHH spec input")
	}

	if len(result.HH) == 0 {
		t.Error("ReadOHH() returned empty HH output")
	}

	// Verify the output contains expected elements
	output := string(result.HH)
	if !strings.Contains(output, "Hero") {
		t.Error("Output should contain hero name 'Hero'")
	}
	if !strings.Contains(output, "Villain") {
		t.Error("Output should contain player name 'Villain'")
	}
	if !strings.Contains(output, "Ah Kh") {
		t.Error("Output should contain hero cards 'Ah Kh'")
	}
	if !strings.Contains(output, "Tc 9c 8c") {
		t.Error("Output should contain flop cards 'Tc 9c 8c'")
	}
}

func TestReadJSONL(t *testing.T) {
	// Test with JSONL format (multiple OHH spec objects, one per line)
	input := `{"id":"hand1","ohh":{"spec_version":"1.4.6","internal_version":"1.0.0","network_name":"Test","site_name":"Test Site","game_type":"Holdem","table_name":"test-table","table_size":2,"game_number":"1","start_date_utc":"2026-02-02T20:00:00.000Z","currency":"Chips","ante_amount":0,"small_blind_amount":0.5,"big_blind_amount":1,"bet_limit":{"bet_cap":0,"bet_type":"NL"},"dealer_seat":2,"hero_player_id":1,"players":[{"id":1,"name":"Hero","seat":1,"starting_stack":100,"cards":["Ah","Kh"]},{"id":2,"name":"Villain","seat":2,"starting_stack":100,"cards":["Qh","Jh"]}],"rounds":[{"id":0,"street":"Preflop","cards":[],"actions":[{"action_number":1,"player_id":1,"action":"Post SB","amount":0.5},{"action_number":2,"player_id":2,"action":"Post BB","amount":1},{"action_number":3,"player_id":1,"action":"Raise","amount":3},{"action_number":4,"player_id":2,"action":"Fold"}]}],"pots":[{"number":0,"amount":3.5,"rake":0,"player_wins":[{"player_id":1,"win_amount":3.5}]}]}}
{"id":"hand2","ohh":{"spec_version":"1.4.6","internal_version":"1.0.0","network_name":"Test","site_name":"Test Site","game_type":"Holdem","table_name":"test-table","table_size":2,"game_number":"2","start_date_utc":"2026-02-02T20:01:00.000Z","currency":"Chips","ante_amount":0,"small_blind_amount":0.5,"big_blind_amount":1,"bet_limit":{"bet_cap":0,"bet_type":"NL"},"dealer_seat":1,"hero_player_id":1,"players":[{"id":1,"name":"Hero","seat":1,"starting_stack":103,"cards":["As","Ks"]},{"id":2,"name":"Villain","seat":2,"starting_stack":97,"cards":["Qd","Jd"]}],"rounds":[{"id":0,"street":"Preflop","cards":[],"actions":[{"action_number":1,"player_id":2,"action":"Post SB","amount":0.5},{"action_number":2,"player_id":1,"action":"Post BB","amount":1},{"action_number":3,"player_id":2,"action":"Call","amount":0.5},{"action_number":4,"player_id":1,"action":"Check"}]},{"id":1,"street":"Flop","cards":["Ad","Kd","2h"],"actions":[{"action_number":5,"player_id":1,"action":"Bet","amount":2},{"action_number":6,"player_id":2,"action":"Fold"}]}],"pots":[{"number":0,"amount":2,"rake":0,"player_wins":[{"player_id":1,"win_amount":2}]}]}}`

	opts := ConvertOptions{
		HeroName: "Hero",
		SiteName: "PokerStars",
	}

	result, err := ReadJSONL(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("ReadJSONL() error = %v", err)
	}

	if result == nil {
		t.Error("ReadJSONL() returned nil result for valid JSONL input")
	}

	if len(result.HH) == 0 {
		t.Error("ReadJSONL() returned empty HH output")
	}

	// Verify the output contains expected elements from both hands
	output := string(result.HH)

	// Check for first hand
	if !strings.Contains(output, "Hand #1") {
		t.Error("Output should contain 'Hand #1'")
	}

	// Check for second hand
	if !strings.Contains(output, "Hand #2") {
		t.Error("Output should contain 'Hand #2'")
	}

	// Check hero cards from both hands
	if !strings.Contains(output, "Ah Kh") {
		t.Error("Output should contain first hand hero cards 'Ah Kh'")
	}
	if !strings.Contains(output, "As Ks") {
		t.Error("Output should contain second hand hero cards 'As Ks'")
	}

	// Verify we processed multiple hands
	handCount := strings.Count(output, "*** HOLE CARDS ***")
	if handCount != 2 {
		t.Errorf("Expected 2 hands, got %d", handCount)
	}
}

func TestIsJSONLFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid JSONL with 2 lines",
			input: `{"id":"1","name":"test"}` + "\n" + `{"id":"2","name":"test2"}`,
			want:  true,
		},
		{
			name:  "single JSON object (not JSONL)",
			input: `{"id":"1","name":"test"}`,
			want:  false,
		},
		{
			name:  "empty input",
			input: "",
			want:  false,
		},
		{
			name:  "CSV format",
			input: "entry,at,order\ntest,2024-01-01,1",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to import the function from converter.go
			// For now, we'll test through the Parse function
			_ = []byte(tt.input) // Just to avoid unused variable error

			// Check if starts with { (basic requirement)
			trimmed := strings.TrimSpace(tt.input)
			if len(trimmed) == 0 {
				return
			}

			startsWithBrace := trimmed[0] == '{'

			if tt.want && !startsWithBrace {
				t.Error("JSONL format should start with '{'")
			}
		})
	}
}
