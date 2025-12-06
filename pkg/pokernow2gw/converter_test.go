package pokernow2gw

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestParseCSV(t *testing.T) {
	tests := []struct {
		name    string
		csv     string
		wantErr bool
	}{
		{
			name: "valid CSV with basic hand",
			csv: `entry,at,order
"-- starting hand #1 (id: test123) (No Limit Texas Hold'em) (dealer: ""player1 @ id1"") --",2025-11-15T05:09:14.567Z,1
"Player stacks: #1 ""player1 @ id1"" (1000)",2025-11-15T05:09:14.567Z,2
"Your hand is A♥, K♥",2025-11-15T05:09:14.567Z,3
"-- ending hand #1 --",2025-11-15T05:09:14.567Z,4`,
			wantErr: false,
		},
		{
			name: "empty CSV",
			csv: `entry,at,order
`,
			wantErr: false,
		},
		{
			name:    "invalid CSV format",
			csv:     "invalid data",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.csv)
			opts := ConvertOptions{
				HeroName:     "player1",
				SiteName:     "GTO Wizard",
				TimeLocation: time.UTC,
			}

			result, err := ParseCSV(reader, opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCSV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("ParseCSV() returned nil result without error")
			}
		})
	}
}

func TestParseCSV_WithSampleFile(t *testing.T) {
	// サンプルファイルが存在する場合のみテスト実行
	samplePath := "../../sample/input/poker_now_log_pglhniqprRDmWFv9sLLZZA-ru.csv"
	if _, err := os.Stat(samplePath); os.IsNotExist(err) {
		t.Skip("Sample file not found, skipping test")
	}

	file, err := os.Open(samplePath)
	if err != nil {
		t.Fatalf("Failed to open sample file: %v", err)
	}
	defer file.Close()

	opts := ConvertOptions{
		HeroName:     "whywaita",
		SiteName:     "GTO Wizard",
		TimeLocation: time.UTC,
	}

	result, err := ParseCSV(file, opts)
	if err != nil {
		t.Errorf("ParseCSV() with sample file failed: %v", err)
	}

	if result == nil {
		t.Error("ParseCSV() returned nil result")
	}

	if result != nil && len(result.HH) == 0 {
		t.Error("ParseCSV() returned empty HH")
	}
}

func TestParseHands(t *testing.T) {
	baseTime := time.Date(2025, 11, 15, 5, 9, 14, 567000000, time.UTC)

	tests := []struct {
		name      string
		entries   []LogEntry
		wantHands []Hand
		wantSkip  int
	}{
		{
			name: "basic hand with blinds and fold",
			entries: []LogEntry{
				{Entry: `-- starting hand #1 (id: test123) (No Limit Texas Hold'em) (dealer: "player1 @ id1") --`, At: baseTime, Order: 1},
				{Entry: `Player stacks: #1 "player1 @ id1" (1000) | #2 "player2 @ id2" (1000)`, At: baseTime, Order: 2},
				{Entry: `Your hand is A♥, K♥`, At: baseTime, Order: 3},
				{Entry: `"player1 @ id1" posts a small blind of 10`, At: baseTime, Order: 4},
				{Entry: `"player2 @ id2" posts a big blind of 20`, At: baseTime, Order: 5},
				{Entry: `"player1 @ id1" folds`, At: baseTime, Order: 6},
				{Entry: `"player2 @ id2" collected 10 from pot`, At: baseTime, Order: 7},
				{Entry: `-- ending hand #1 --`, At: baseTime, Order: 8},
			},
			wantHands: []Hand{
				{
					HandNumber: "1",
					HandID:     "17066136185775469334",
					Dealer:     "player1",
					StartTime:  baseTime,
					SmallBlind: 10,
					BigBlind:   20,
					Players: []Player{
						{SeatNumber: 1, Name: "player1 @ id1", DisplayName: "player1", Stack: 1000},
						{SeatNumber: 2, Name: "player2 @ id2", DisplayName: "player2", Stack: 1000},
					},
					HeroCards: []string{"Ah", "Kh"},
					Actions: []Action{
						{Player: "player1", ActionType: ActionPostSB, Amount: 10, Street: StreetPreflop},
						{Player: "player2", ActionType: ActionPostBB, Amount: 20, Street: StreetPreflop},
						{Player: "player1", ActionType: ActionFold, Street: StreetPreflop},
						{Player: "player2", ActionType: ActionCollect, Amount: 10, Street: StreetShowdown},
					},
					Winners: []Winner{
						{Player: "player2", Amount: 10},
					},
				},
			},
			wantSkip: 0,
		},
		{
			name: "hand with flop and betting",
			entries: []LogEntry{
				{Entry: `-- starting hand #2 (No Limit Texas Hold'em) (dealer: "alice @ abc") --`, At: baseTime, Order: 1},
				{Entry: `Player stacks: #1 "alice @ abc" (500) | #2 "bob @ def" (500)`, At: baseTime, Order: 2},
				{Entry: `Your hand is 10♦, 10♣`, At: baseTime, Order: 3},
				{Entry: `"alice @ abc" posts a small blind of 5`, At: baseTime, Order: 4},
				{Entry: `"bob @ def" posts a big blind of 10`, At: baseTime, Order: 5},
				{Entry: `"alice @ abc" calls 5`, At: baseTime, Order: 6},
				{Entry: `"bob @ def" checks`, At: baseTime, Order: 7},
				{Entry: `Flop: [A♥, K♦, Q♠]`, At: baseTime, Order: 8},
				{Entry: `"alice @ abc" bets 20`, At: baseTime, Order: 9},
				{Entry: `"bob @ def" folds`, At: baseTime, Order: 10},
				{Entry: `"alice @ abc" collected 20 from pot`, At: baseTime, Order: 11},
				{Entry: `-- ending hand #2 --`, At: baseTime, Order: 12},
			},
			wantHands: []Hand{
				{
					HandNumber: "2",
					HandID:     "2",
					Dealer:     "alice",
					StartTime:  baseTime,
					SmallBlind: 5,
					BigBlind:   10,
					Players: []Player{
						{SeatNumber: 1, Name: "alice @ abc", DisplayName: "alice", Stack: 500},
						{SeatNumber: 2, Name: "bob @ def", DisplayName: "bob", Stack: 500},
					},
					HeroCards: []string{"Td", "Tc"},
					Board: Board{
						Flop: []string{"Ah", "Kd", "Qs"},
					},
					Actions: []Action{
						{Player: "alice", ActionType: ActionPostSB, Amount: 5, Street: StreetPreflop},
						{Player: "bob", ActionType: ActionPostBB, Amount: 10, Street: StreetPreflop},
						{Player: "alice", ActionType: ActionCall, Amount: 5, Street: StreetPreflop},
						{Player: "bob", ActionType: ActionCheck, Street: StreetPreflop},
						{Player: "alice", ActionType: ActionBet, Amount: 20, Street: StreetFlop},
						{Player: "bob", ActionType: ActionFold, Street: StreetFlop},
						{Player: "alice", ActionType: ActionCollect, Amount: 20, Street: StreetShowdown},
					},
					Winners: []Winner{
						{Player: "alice", Amount: 20},
					},
				},
			},
			wantSkip: 0,
		},
		{
			name: "hand with showdown",
			entries: []LogEntry{
				{Entry: `-- starting hand #3 (id: xyz789) (No Limit Texas Hold'em) (dealer: "charlie @ ghi") --`, At: baseTime, Order: 1},
				{Entry: `Player stacks: #1 "charlie @ ghi" (1000) | #2 "dave @ jkl" (1000)`, At: baseTime, Order: 2},
				{Entry: `Your hand is A♠, 7♥`, At: baseTime, Order: 3},
				{Entry: `"charlie @ ghi" posts a small blind of 10`, At: baseTime, Order: 4},
				{Entry: `"dave @ jkl" posts a big blind of 20`, At: baseTime, Order: 5},
				{Entry: `"charlie @ ghi" calls 10`, At: baseTime, Order: 6},
				{Entry: `"dave @ jkl" checks`, At: baseTime, Order: 7},
				{Entry: `Flop: [2♥, 3♦, 4♠]`, At: baseTime, Order: 8},
				{Entry: `"charlie @ ghi" checks`, At: baseTime, Order: 9},
				{Entry: `"dave @ jkl" bets 50`, At: baseTime, Order: 10},
				{Entry: `"charlie @ ghi" calls 50`, At: baseTime, Order: 11},
				{Entry: `Turn: 2♥, 3♦, 4♠ [5♣]`, At: baseTime, Order: 12},
				{Entry: `"charlie @ ghi" checks`, At: baseTime, Order: 13},
				{Entry: `"dave @ jkl" checks`, At: baseTime, Order: 14},
				{Entry: `River: 2♥, 3♦, 4♠, 5♣ [6♥]`, At: baseTime, Order: 15},
				{Entry: `"charlie @ ghi" bets 100`, At: baseTime, Order: 16},
				{Entry: `"dave @ jkl" calls 100`, At: baseTime, Order: 17},
				{Entry: `"charlie @ ghi" shows a A♠, 7♥.`, At: baseTime, Order: 18},
				{Entry: `"dave @ jkl" shows a A♣, 8♦.`, At: baseTime, Order: 19},
				{Entry: `"charlie @ ghi" collected 170 from pot`, At: baseTime, Order: 20},
				{Entry: `"dave @ jkl" collected 170 from pot`, At: baseTime, Order: 21},
				{Entry: `-- ending hand #3 --`, At: baseTime, Order: 22},
			},
			wantHands: []Hand{
				{
					HandNumber: "3",
					HandID:     "6504957911579380203",
					Dealer:     "charlie",
					StartTime:  baseTime,
					SmallBlind: 10,
					BigBlind:   20,
					Players: []Player{
						{SeatNumber: 1, Name: "charlie @ ghi", DisplayName: "charlie", Stack: 1000},
						{SeatNumber: 2, Name: "dave @ jkl", DisplayName: "dave", Stack: 1000},
					},
					HeroCards: []string{"As", "7h"},
					Board: Board{
						Flop:  []string{"2h", "3d", "4s"},
						Turn:  "5c",
						River: "6h",
					},
					Actions: []Action{
						{Player: "charlie", ActionType: ActionPostSB, Amount: 10, Street: StreetPreflop},
						{Player: "dave", ActionType: ActionPostBB, Amount: 20, Street: StreetPreflop},
						{Player: "charlie", ActionType: ActionCall, Amount: 10, Street: StreetPreflop},
						{Player: "dave", ActionType: ActionCheck, Street: StreetPreflop},
						{Player: "charlie", ActionType: ActionCheck, Street: StreetFlop},
						{Player: "dave", ActionType: ActionBet, Amount: 50, Street: StreetFlop},
						{Player: "charlie", ActionType: ActionCall, Amount: 50, Street: StreetFlop},
						{Player: "charlie", ActionType: ActionCheck, Street: StreetTurn},
						{Player: "dave", ActionType: ActionCheck, Street: StreetTurn},
						{Player: "charlie", ActionType: ActionBet, Amount: 100, Street: StreetRiver},
						{Player: "dave", ActionType: ActionCall, Amount: 100, Street: StreetRiver},
						{Player: "charlie", ActionType: ActionShow, Street: StreetShowdown},
						{Player: "dave", ActionType: ActionShow, Street: StreetShowdown},
						{Player: "charlie", ActionType: ActionCollect, Amount: 170, Street: StreetShowdown},
						{Player: "dave", ActionType: ActionCollect, Amount: 170, Street: StreetShowdown},
					},
					Winners: []Winner{
						{Player: "charlie", Amount: 170, HandCards: []string{"As", "7h"}},
						{Player: "dave", Amount: 170, HandCards: []string{"Ac", "8d"}},
					},
				},
			},
			wantSkip: 0,
		},
		{
			name: "hand with all-in",
			entries: []LogEntry{
				{Entry: `-- starting hand #4 (No Limit Texas Hold'em) (dealer: "eve @ mno") --`, At: baseTime, Order: 1},
				{Entry: `Player stacks: #1 "eve @ mno" (100) | #2 "frank @ pqr" (200)`, At: baseTime, Order: 2},
				{Entry: `Your hand is J♣, J♦`, At: baseTime, Order: 3},
				{Entry: `"eve @ mno" posts a small blind of 10`, At: baseTime, Order: 4},
				{Entry: `"frank @ pqr" posts a big blind of 20`, At: baseTime, Order: 5},
				{Entry: `"eve @ mno" raises to 100 and go all in`, At: baseTime, Order: 6},
				{Entry: `"frank @ pqr" calls 80`, At: baseTime, Order: 7},
				{Entry: `Flop: [J♥, Q♦, K♠]`, At: baseTime, Order: 8},
				{Entry: `"eve @ mno" shows a J♣, J♦.`, At: baseTime, Order: 9},
				{Entry: `"frank @ pqr" shows a A♠, 10♠.`, At: baseTime, Order: 10},
				{Entry: `"frank @ pqr" collected 200 from pot`, At: baseTime, Order: 11},
				{Entry: `-- ending hand #4 --`, At: baseTime, Order: 12},
			},
			wantHands: []Hand{
				{
					HandNumber: "4",
					HandID:     "4",
					Dealer:     "eve",
					StartTime:  baseTime,
					SmallBlind: 10,
					BigBlind:   20,
					Players: []Player{
						{SeatNumber: 1, Name: "eve @ mno", DisplayName: "eve", Stack: 100},
						{SeatNumber: 2, Name: "frank @ pqr", DisplayName: "frank", Stack: 200},
					},
					HeroCards: []string{"Jc", "Jd"},
					Board: Board{
						Flop: []string{"Jh", "Qd", "Ks"},
					},
					Actions: []Action{
						{Player: "eve", ActionType: ActionPostSB, Amount: 10, Street: StreetPreflop},
						{Player: "frank", ActionType: ActionPostBB, Amount: 20, Street: StreetPreflop},
						{Player: "eve", ActionType: ActionRaise, Amount: 100, Street: StreetPreflop, IsAllIn: true},
						{Player: "frank", ActionType: ActionCall, Amount: 80, Street: StreetPreflop},
						{Player: "eve", ActionType: ActionShow, Street: StreetShowdown},
						{Player: "frank", ActionType: ActionShow, Street: StreetShowdown},
						{Player: "frank", ActionType: ActionCollect, Amount: 200, Street: StreetShowdown},
					},
					Winners: []Winner{
						{Player: "eve", Amount: 0, HandCards: []string{"Jc", "Jd"}},
						{Player: "frank", Amount: 200, HandCards: []string{"As", "Ts"}},
					},
				},
			},
			wantSkip: 0,
		},
		{
			name: "hand with dead button",
			entries: []LogEntry{
				{Entry: `-- starting hand #5 (id: deadbtn01) (No Limit Texas Hold'em) (dead button) --`, At: baseTime, Order: 1},
				{Entry: `Player stacks: #1 "grace @ stu" (500) | #2 "henry @ vwx" (500)`, At: baseTime, Order: 2},
				{Entry: `Your hand is Q♥, Q♦`, At: baseTime, Order: 3},
				{Entry: `"grace @ stu" posts a small blind of 10`, At: baseTime, Order: 4},
				{Entry: `"henry @ vwx" posts a big blind of 20`, At: baseTime, Order: 5},
				{Entry: `"grace @ stu" raises to 60`, At: baseTime, Order: 6},
				{Entry: `"henry @ vwx" calls 40`, At: baseTime, Order: 7},
				{Entry: `Flop: [2♣, 7♦, 9♠]`, At: baseTime, Order: 8},
				{Entry: `"grace @ stu" bets 100`, At: baseTime, Order: 9},
				{Entry: `"henry @ vwx" folds`, At: baseTime, Order: 10},
				{Entry: `"grace @ stu" collected 120 from pot`, At: baseTime, Order: 11},
				{Entry: `-- ending hand #5 --`, At: baseTime, Order: 12},
			},
			wantHands: []Hand{
				{
					HandNumber: "5",
					HandID:     "8681563949085652710",
					Dealer:     "",
					StartTime:  baseTime,
					SmallBlind: 10,
					BigBlind:   20,
					Players: []Player{
						{SeatNumber: 1, Name: "grace @ stu", DisplayName: "grace", Stack: 500},
						{SeatNumber: 2, Name: "henry @ vwx", DisplayName: "henry", Stack: 500},
					},
					HeroCards: []string{"Qh", "Qd"},
					Board: Board{
						Flop: []string{"2c", "7d", "9s"},
					},
					Actions: []Action{
						{Player: "grace", ActionType: ActionPostSB, Amount: 10, Street: StreetPreflop},
						{Player: "henry", ActionType: ActionPostBB, Amount: 20, Street: StreetPreflop},
						{Player: "grace", ActionType: ActionRaise, Amount: 60, Street: StreetPreflop},
						{Player: "henry", ActionType: ActionCall, Amount: 40, Street: StreetPreflop},
						{Player: "grace", ActionType: ActionBet, Amount: 100, Street: StreetFlop},
						{Player: "henry", ActionType: ActionFold, Street: StreetFlop},
						{Player: "grace", ActionType: ActionCollect, Amount: 120, Street: StreetShowdown},
					},
					Winners: []Winner{
						{Player: "grace", Amount: 120},
					},
				},
			},
			wantSkip: 0,
		},
		{
			name: "hand with call all-in",
			entries: []LogEntry{
				{Entry: `-- starting hand #6 (id: callallin01) (No Limit Texas Hold'em) (dealer: "iris @ yza") --`, At: baseTime, Order: 1},
				{Entry: `Player stacks: #1 "iris @ yza" (1000) | #2 "john @ bcd" (300)`, At: baseTime, Order: 2},
				{Entry: `Your hand is 9♥, 9♦`, At: baseTime, Order: 3},
				{Entry: `"iris @ yza" posts a small blind of 10`, At: baseTime, Order: 4},
				{Entry: `"john @ bcd" posts a big blind of 20`, At: baseTime, Order: 5},
				{Entry: `"iris @ yza" raises to 60`, At: baseTime, Order: 6},
				{Entry: `"john @ bcd" calls 280 and go all in`, At: baseTime, Order: 7},
				{Entry: `Flop: [A♥, K♦, Q♠]`, At: baseTime, Order: 8},
				{Entry: `"iris @ yza" shows a A♣, K♥.`, At: baseTime, Order: 9},
				{Entry: `"john @ bcd" shows a 9♠, 9♣.`, At: baseTime, Order: 10},
				{Entry: `"iris @ yza" collected 600 from pot`, At: baseTime, Order: 11},
				{Entry: `-- ending hand #6 --`, At: baseTime, Order: 12},
			},
			wantHands: []Hand{
				{
					HandNumber: "6",
					HandID:     "10168127831822994649",
					Dealer:     "iris",
					StartTime:  baseTime,
					SmallBlind: 10,
					BigBlind:   20,
					Players: []Player{
						{SeatNumber: 1, Name: "iris @ yza", DisplayName: "iris", Stack: 1000},
						{SeatNumber: 2, Name: "john @ bcd", DisplayName: "john", Stack: 300},
					},
					HeroCards: []string{"9h", "9d"},
					Board: Board{
						Flop: []string{"Ah", "Kd", "Qs"},
					},
					Actions: []Action{
						{Player: "iris", ActionType: ActionPostSB, Amount: 10, Street: StreetPreflop},
						{Player: "john", ActionType: ActionPostBB, Amount: 20, Street: StreetPreflop},
						{Player: "iris", ActionType: ActionRaise, Amount: 60, Street: StreetPreflop},
						{Player: "john", ActionType: ActionCall, Amount: 280, Street: StreetPreflop, IsAllIn: true},
						{Player: "iris", ActionType: ActionShow, Street: StreetShowdown},
						{Player: "john", ActionType: ActionShow, Street: StreetShowdown},
						{Player: "iris", ActionType: ActionCollect, Amount: 600, Street: StreetShowdown},
					},
					Winners: []Winner{
						{Player: "iris", Amount: 600, HandCards: []string{"Ac", "Kh"}},
						{Player: "john", Amount: 0, HandCards: []string{"9s", "9c"}},
					},
				},
			},
			wantSkip: 0,
		},
		{
			name: "hand with non-consecutive seat numbers (should be normalized to 1,2,3)",
			entries: []LogEntry{
				{Entry: `-- starting hand #7 (id: noncons01) (No Limit Texas Hold'em) (dealer: "kate @ efg") --`, At: baseTime, Order: 1},
				{Entry: `Player stacks: #3 "kate @ efg" (500) | #7 "leo @ hij" (600) | #10 "mike @ klm" (700)`, At: baseTime, Order: 2},
				{Entry: `Your hand is 5♥, 6♦`, At: baseTime, Order: 3},
				{Entry: `"kate @ efg" posts a small blind of 10`, At: baseTime, Order: 4},
				{Entry: `"leo @ hij" posts a big blind of 20`, At: baseTime, Order: 5},
				{Entry: `"mike @ klm" folds`, At: baseTime, Order: 6},
				{Entry: `"kate @ efg" calls 10`, At: baseTime, Order: 7},
				{Entry: `"leo @ hij" checks`, At: baseTime, Order: 8},
				{Entry: `Flop: [5♥, 6♦, 7♠]`, At: baseTime, Order: 9},
				{Entry: `"kate @ efg" checks`, At: baseTime, Order: 10},
				{Entry: `"leo @ hij" bets 40`, At: baseTime, Order: 11},
				{Entry: `"kate @ efg" folds`, At: baseTime, Order: 12},
				{Entry: `"leo @ hij" collected 40 from pot`, At: baseTime, Order: 13},
				{Entry: `-- ending hand #7 --`, At: baseTime, Order: 14},
			},
			wantHands: []Hand{
				{
					HandNumber: "7",
					HandID:     "13372294122307939540",
					Dealer:     "kate",
					StartTime:  baseTime,
					SmallBlind: 10,
					BigBlind:   20,
					Players: []Player{
						{SeatNumber: 1, Name: "kate @ efg", DisplayName: "kate", Stack: 500},
						{SeatNumber: 2, Name: "leo @ hij", DisplayName: "leo", Stack: 600},
						{SeatNumber: 3, Name: "mike @ klm", DisplayName: "mike", Stack: 700},
					},
					HeroCards: []string{"5h", "6d"},
					Board: Board{
						Flop: []string{"5h", "6d", "7s"},
					},
					Actions: []Action{
						{Player: "kate", ActionType: ActionPostSB, Amount: 10, Street: StreetPreflop},
						{Player: "leo", ActionType: ActionPostBB, Amount: 20, Street: StreetPreflop},
						{Player: "mike", ActionType: ActionFold, Street: StreetPreflop},
						{Player: "kate", ActionType: ActionCall, Amount: 10, Street: StreetPreflop},
						{Player: "leo", ActionType: ActionCheck, Street: StreetPreflop},
						{Player: "kate", ActionType: ActionCheck, Street: StreetFlop},
						{Player: "leo", ActionType: ActionBet, Amount: 40, Street: StreetFlop},
						{Player: "kate", ActionType: ActionFold, Street: StreetFlop},
						{Player: "leo", ActionType: ActionCollect, Amount: 40, Street: StreetShowdown},
					},
					Winners: []Winner{
						{Player: "leo", Amount: 40},
					},
				},
			},
			wantSkip: 0,
		},
		{
			name: "hand with more than 10 players (should be skipped)",
			entries: []LogEntry{
				{Entry: `-- starting hand #8 (id: toomany01) (No Limit Texas Hold'em) (dealer: "player1 @ p1") --`, At: baseTime, Order: 1},
				{Entry: `Player stacks: #1 "player1 @ p1" (1000) | #2 "player2 @ p2" (1000) | #3 "player3 @ p3" (1000) | #4 "player4 @ p4" (1000) | #5 "player5 @ p5" (1000) | #6 "player6 @ p6" (1000) | #7 "player7 @ p7" (1000) | #8 "player8 @ p8" (1000) | #9 "player9 @ p9" (1000) | #10 "player10 @ p10" (1000) | #11 "player11 @ p11" (1000)`, At: baseTime, Order: 2},
				{Entry: `"player1 @ p1" posts a small blind of 10`, At: baseTime, Order: 3},
				{Entry: `"player2 @ p2" posts a big blind of 20`, At: baseTime, Order: 4},
				{Entry: `"player3 @ p3" folds`, At: baseTime, Order: 5},
				{Entry: `-- ending hand #8 --`, At: baseTime, Order: 6},
			},
			wantHands: nil,
			wantSkip:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ConvertOptions{
				PlayerCountFilter: PlayerCountAll,
			}
			gotHands, gotSkip, err := ParseHands(tt.entries, opts)
			if err != nil {
				t.Errorf("ParseHands() unexpected error: %v", err)
				return
			}

			if gotSkip != tt.wantSkip {
				t.Errorf("ParseHands() skipped hands = %d, want %d", gotSkip, tt.wantSkip)
			}

			if diff := cmp.Diff(tt.wantHands, gotHands); diff != "" {
				t.Errorf("ParseHands() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseHands_SpectatorLog(t *testing.T) {
	baseTime := time.Date(2025, 11, 15, 5, 9, 14, 567000000, time.UTC)

	tests := []struct {
		name    string
		entries []LogEntry
		wantErr error
	}{
		{
			name: "spectator log (no Your hand is entries)",
			entries: []LogEntry{
				{Entry: `-- starting hand #1 (id: spec123) (No Limit Texas Hold'em) (dealer: "player1 @ id1") --`, At: baseTime, Order: 1},
				{Entry: `Player stacks: #1 "player1 @ id1" (1000) | #2 "player2 @ id2" (1000)`, At: baseTime, Order: 2},
				{Entry: `"player1 @ id1" posts a small blind of 10`, At: baseTime, Order: 3},
				{Entry: `"player2 @ id2" posts a big blind of 20`, At: baseTime, Order: 4},
				{Entry: `"player1 @ id1" folds`, At: baseTime, Order: 5},
				{Entry: `"player2 @ id2" collected 10 from pot`, At: baseTime, Order: 6},
				{Entry: `-- ending hand #1 --`, At: baseTime, Order: 7},
			},
			wantErr: ErrSpectatorLog,
		},
		{
			name: "spectator log with multiple hands (none have hero cards)",
			entries: []LogEntry{
				{Entry: `-- starting hand #1 (id: spec1) (No Limit Texas Hold'em) (dealer: "player1 @ id1") --`, At: baseTime, Order: 1},
				{Entry: `Player stacks: #1 "player1 @ id1" (1000) | #2 "player2 @ id2" (1000)`, At: baseTime, Order: 2},
				{Entry: `"player1 @ id1" posts a small blind of 10`, At: baseTime, Order: 3},
				{Entry: `"player2 @ id2" posts a big blind of 20`, At: baseTime, Order: 4},
				{Entry: `"player1 @ id1" folds`, At: baseTime, Order: 5},
				{Entry: `"player2 @ id2" collected 10 from pot`, At: baseTime, Order: 6},
				{Entry: `-- ending hand #1 --`, At: baseTime, Order: 7},
				{Entry: `-- starting hand #2 (id: spec2) (No Limit Texas Hold'em) (dealer: "player2 @ id2") --`, At: baseTime, Order: 8},
				{Entry: `Player stacks: #1 "player1 @ id1" (990) | #2 "player2 @ id2" (1010)`, At: baseTime, Order: 9},
				{Entry: `"player2 @ id2" posts a small blind of 10`, At: baseTime, Order: 10},
				{Entry: `"player1 @ id1" posts a big blind of 20`, At: baseTime, Order: 11},
				{Entry: `"player2 @ id2" folds`, At: baseTime, Order: 12},
				{Entry: `"player1 @ id1" collected 10 from pot`, At: baseTime, Order: 13},
				{Entry: `-- ending hand #2 --`, At: baseTime, Order: 14},
			},
			wantErr: ErrSpectatorLog,
		},
		{
			name: "player log (has Your hand is entry)",
			entries: []LogEntry{
				{Entry: `-- starting hand #1 (id: player123) (No Limit Texas Hold'em) (dealer: "player1 @ id1") --`, At: baseTime, Order: 1},
				{Entry: `Player stacks: #1 "player1 @ id1" (1000) | #2 "player2 @ id2" (1000)`, At: baseTime, Order: 2},
				{Entry: `Your hand is A♥, K♥`, At: baseTime, Order: 3},
				{Entry: `"player1 @ id1" posts a small blind of 10`, At: baseTime, Order: 4},
				{Entry: `"player2 @ id2" posts a big blind of 20`, At: baseTime, Order: 5},
				{Entry: `"player1 @ id1" folds`, At: baseTime, Order: 6},
				{Entry: `"player2 @ id2" collected 10 from pot`, At: baseTime, Order: 7},
				{Entry: `-- ending hand #1 --`, At: baseTime, Order: 8},
			},
			wantErr: nil,
		},
		{
			name:    "empty log (no hands)",
			entries: []LogEntry{},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ConvertOptions{
				PlayerCountFilter: PlayerCountAll,
			}
			_, _, err := ParseHands(tt.entries, opts)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ParseHands() expected error %v, got nil", tt.wantErr)
				} else if err != tt.wantErr {
					t.Errorf("ParseHands() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("ParseHands() unexpected error: %v", err)
				}
			}
		})
	}
}
