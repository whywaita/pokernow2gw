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
