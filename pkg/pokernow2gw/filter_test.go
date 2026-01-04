package pokernow2gw

import (
	"testing"
	"time"
)

func TestPlayerCountFilter_isPlayerCountAllowed(t *testing.T) {
	tests := []struct {
		name        string
		filter      PlayerCountFilter
		playerCount int
		want        bool
	}{
		// PlayerCountAll tests
		{
			name:        "PlayerCountAll allows 2 players",
			filter:      PlayerCountAll,
			playerCount: 2,
			want:        true,
		},
		{
			name:        "PlayerCountAll allows 3 players",
			filter:      PlayerCountAll,
			playerCount: 3,
			want:        true,
		},
		{
			name:        "PlayerCountAll allows 5 players",
			filter:      PlayerCountAll,
			playerCount: 5,
			want:        true,
		},
		{
			name:        "PlayerCountAll allows 9 players",
			filter:      PlayerCountAll,
			playerCount: 9,
			want:        true,
		},
		{
			name:        "PlayerCountAll allows 10 players",
			filter:      PlayerCountAll,
			playerCount: 10,
			want:        true,
		},

		// PlayerCountHU tests
		{
			name:        "PlayerCountHU allows 2 players",
			filter:      PlayerCountHU,
			playerCount: 2,
			want:        true,
		},
		{
			name:        "PlayerCountHU rejects 3 players",
			filter:      PlayerCountHU,
			playerCount: 3,
			want:        false,
		},
		{
			name:        "PlayerCountHU rejects 4 players",
			filter:      PlayerCountHU,
			playerCount: 4,
			want:        false,
		},

		// PlayerCountSpinAndGo tests
		{
			name:        "PlayerCountSpinAndGo rejects 2 players",
			filter:      PlayerCountSpinAndGo,
			playerCount: 2,
			want:        false,
		},
		{
			name:        "PlayerCountSpinAndGo allows 3 players",
			filter:      PlayerCountSpinAndGo,
			playerCount: 3,
			want:        true,
		},
		{
			name:        "PlayerCountSpinAndGo rejects 4 players",
			filter:      PlayerCountSpinAndGo,
			playerCount: 4,
			want:        false,
		},

		// PlayerCountMTT tests
		{
			name:        "PlayerCountMTT rejects 3 players",
			filter:      PlayerCountMTT,
			playerCount: 3,
			want:        false,
		},
		{
			name:        "PlayerCountMTT allows 4 players",
			filter:      PlayerCountMTT,
			playerCount: 4,
			want:        true,
		},
		{
			name:        "PlayerCountMTT allows 5 players",
			filter:      PlayerCountMTT,
			playerCount: 5,
			want:        true,
		},
		{
			name:        "PlayerCountMTT allows 9 players",
			filter:      PlayerCountMTT,
			playerCount: 9,
			want:        true,
		},
		{
			name:        "PlayerCountMTT rejects 10 players",
			filter:      PlayerCountMTT,
			playerCount: 10,
			want:        false,
		},

		// Combined filters tests
		{
			name:        "HU+SpinAndGo allows 2 players",
			filter:      PlayerCountHU | PlayerCountSpinAndGo,
			playerCount: 2,
			want:        true,
		},
		{
			name:        "HU+SpinAndGo allows 3 players",
			filter:      PlayerCountHU | PlayerCountSpinAndGo,
			playerCount: 3,
			want:        true,
		},
		{
			name:        "HU+SpinAndGo rejects 4 players",
			filter:      PlayerCountHU | PlayerCountSpinAndGo,
			playerCount: 4,
			want:        false,
		},
		{
			name:        "SpinAndGo+MTT allows 3 players",
			filter:      PlayerCountSpinAndGo | PlayerCountMTT,
			playerCount: 3,
			want:        true,
		},
		{
			name:        "SpinAndGo+MTT allows 5 players",
			filter:      PlayerCountSpinAndGo | PlayerCountMTT,
			playerCount: 5,
			want:        true,
		},
		{
			name:        "SpinAndGo+MTT rejects 2 players",
			filter:      PlayerCountSpinAndGo | PlayerCountMTT,
			playerCount: 2,
			want:        false,
		},
		{
			name:        "HU+SpinAndGo+MTT allows 2 players",
			filter:      PlayerCountHU | PlayerCountSpinAndGo | PlayerCountMTT,
			playerCount: 2,
			want:        true,
		},
		{
			name:        "HU+SpinAndGo+MTT allows 3 players",
			filter:      PlayerCountHU | PlayerCountSpinAndGo | PlayerCountMTT,
			playerCount: 3,
			want:        true,
		},
		{
			name:        "HU+SpinAndGo+MTT allows 9 players",
			filter:      PlayerCountHU | PlayerCountSpinAndGo | PlayerCountMTT,
			playerCount: 9,
			want:        true,
		},
		{
			name:        "HU+SpinAndGo+MTT rejects 10 players",
			filter:      PlayerCountHU | PlayerCountSpinAndGo | PlayerCountMTT,
			playerCount: 10,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.isPlayerCountAllowed(tt.playerCount)
			if got != tt.want {
				t.Errorf("isPlayerCountAllowed(%d) = %v, want %v", tt.playerCount, got, tt.want)
			}
		})
	}
}

func TestParseHands_WithPlayerCountFilter(t *testing.T) {
	baseTime := time.Date(2025, 11, 15, 5, 9, 14, 567000000, time.UTC)

	// Create test hands with different player counts
	entries2Players := []LogEntry{
		{Entry: `-- starting hand #1 (id: h2p) (No Limit Texas Hold'em) (dealer: "p1 @ id1") --`, At: baseTime, Order: 1},
		{Entry: `Player stacks: #1 "p1 @ id1" (1000) | #2 "p2 @ id2" (1000)`, At: baseTime, Order: 2},
		{Entry: `Your hand is A♥, K♥`, At: baseTime, Order: 3},
		{Entry: `"p1 @ id1" posts a small blind of 10`, At: baseTime, Order: 4},
		{Entry: `"p2 @ id2" posts a big blind of 20`, At: baseTime, Order: 5},
		{Entry: `"p1 @ id1" folds`, At: baseTime, Order: 6},
		{Entry: `"p2 @ id2" collected 10 from pot`, At: baseTime, Order: 7},
		{Entry: `-- ending hand #1 --`, At: baseTime, Order: 8},
	}

	entries3Players := []LogEntry{
		{Entry: `-- starting hand #2 (id: h3p) (No Limit Texas Hold'em) (dealer: "p1 @ id1") --`, At: baseTime, Order: 1},
		{Entry: `Player stacks: #1 "p1 @ id1" (1000) | #2 "p2 @ id2" (1000) | #3 "p3 @ id3" (1000)`, At: baseTime, Order: 2},
		{Entry: `Your hand is A♥, K♥`, At: baseTime, Order: 3},
		{Entry: `"p1 @ id1" posts a small blind of 10`, At: baseTime, Order: 4},
		{Entry: `"p2 @ id2" posts a big blind of 20`, At: baseTime, Order: 5},
		{Entry: `"p3 @ id3" folds`, At: baseTime, Order: 6},
		{Entry: `"p1 @ id1" folds`, At: baseTime, Order: 7},
		{Entry: `"p2 @ id2" collected 20 from pot`, At: baseTime, Order: 8},
		{Entry: `-- ending hand #2 --`, At: baseTime, Order: 9},
	}

	entries5Players := []LogEntry{
		{Entry: `-- starting hand #3 (id: h5p) (No Limit Texas Hold'em) (dealer: "p1 @ id1") --`, At: baseTime, Order: 1},
		{Entry: `Player stacks: #1 "p1 @ id1" (1000) | #2 "p2 @ id2" (1000) | #3 "p3 @ id3" (1000) | #4 "p4 @ id4" (1000) | #5 "p5 @ id5" (1000)`, At: baseTime, Order: 2},
		{Entry: `Your hand is A♥, K♥`, At: baseTime, Order: 3},
		{Entry: `"p1 @ id1" posts a small blind of 10`, At: baseTime, Order: 4},
		{Entry: `"p2 @ id2" posts a big blind of 20`, At: baseTime, Order: 5},
		{Entry: `"p3 @ id3" folds`, At: baseTime, Order: 6},
		{Entry: `"p4 @ id4" folds`, At: baseTime, Order: 7},
		{Entry: `"p5 @ id5" folds`, At: baseTime, Order: 8},
		{Entry: `"p1 @ id1" folds`, At: baseTime, Order: 9},
		{Entry: `"p2 @ id2" collected 30 from pot`, At: baseTime, Order: 10},
		{Entry: `-- ending hand #3 --`, At: baseTime, Order: 11},
	}

	entries10Players := []LogEntry{
		{Entry: `-- starting hand #4 (id: h10p) (No Limit Texas Hold'em) (dealer: "p1 @ id1") --`, At: baseTime, Order: 1},
		{Entry: `Player stacks: #1 "p1 @ id1" (1000) | #2 "p2 @ id2" (1000) | #3 "p3 @ id3" (1000) | #4 "p4 @ id4" (1000) | #5 "p5 @ id5" (1000) | #6 "p6 @ id6" (1000) | #7 "p7 @ id7" (1000) | #8 "p8 @ id8" (1000) | #9 "p9 @ id9" (1000) | #10 "p10 @ id10" (1000)`, At: baseTime, Order: 2},
		{Entry: `Your hand is A♥, K♥`, At: baseTime, Order: 3},
		{Entry: `"p1 @ id1" posts a small blind of 10`, At: baseTime, Order: 4},
		{Entry: `"p2 @ id2" posts a big blind of 20`, At: baseTime, Order: 5},
		{Entry: `"p1 @ id1" folds`, At: baseTime, Order: 6},
		{Entry: `"p2 @ id2" collected 10 from pot`, At: baseTime, Order: 7},
		{Entry: `-- ending hand #4 --`, At: baseTime, Order: 8},
	}

	tests := []struct {
		name             string
		entries          []LogEntry
		filter           PlayerCountFilter
		wantHandsCount   int
		wantSkippedHands int
	}{
		// PlayerCountAll tests
		{
			name:             "PlayerCountAll includes 2 players",
			entries:          entries2Players,
			filter:           PlayerCountAll,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "PlayerCountAll includes 3 players",
			entries:          entries3Players,
			filter:           PlayerCountAll,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "PlayerCountAll includes 5 players",
			entries:          entries5Players,
			filter:           PlayerCountAll,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "PlayerCountAll includes 10 players",
			entries:          entries10Players,
			filter:           PlayerCountAll,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},

		// PlayerCountHU tests
		{
			name:             "PlayerCountHU includes 2 players",
			entries:          entries2Players,
			filter:           PlayerCountHU,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "PlayerCountHU skips 3 players",
			entries:          entries3Players,
			filter:           PlayerCountHU,
			wantHandsCount:   0,
			wantSkippedHands: 1,
		},
		{
			name:             "PlayerCountHU skips 5 players",
			entries:          entries5Players,
			filter:           PlayerCountHU,
			wantHandsCount:   0,
			wantSkippedHands: 1,
		},

		// PlayerCountSpinAndGo tests
		{
			name:             "PlayerCountSpinAndGo skips 2 players",
			entries:          entries2Players,
			filter:           PlayerCountSpinAndGo,
			wantHandsCount:   0,
			wantSkippedHands: 1,
		},
		{
			name:             "PlayerCountSpinAndGo includes 3 players",
			entries:          entries3Players,
			filter:           PlayerCountSpinAndGo,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "PlayerCountSpinAndGo skips 5 players",
			entries:          entries5Players,
			filter:           PlayerCountSpinAndGo,
			wantHandsCount:   0,
			wantSkippedHands: 1,
		},

		// PlayerCountMTT tests
		{
			name:             "PlayerCountMTT skips 2 players",
			entries:          entries2Players,
			filter:           PlayerCountMTT,
			wantHandsCount:   0,
			wantSkippedHands: 1,
		},
		{
			name:             "PlayerCountMTT skips 3 players",
			entries:          entries3Players,
			filter:           PlayerCountMTT,
			wantHandsCount:   0,
			wantSkippedHands: 1,
		},
		{
			name:             "PlayerCountMTT includes 5 players",
			entries:          entries5Players,
			filter:           PlayerCountMTT,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "PlayerCountMTT skips 10 players",
			entries:          entries10Players,
			filter:           PlayerCountMTT,
			wantHandsCount:   0,
			wantSkippedHands: 1,
		},

		// Combined filters tests
		{
			name:             "HU+SpinAndGo includes 2 players",
			entries:          entries2Players,
			filter:           PlayerCountHU | PlayerCountSpinAndGo,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "HU+SpinAndGo includes 3 players",
			entries:          entries3Players,
			filter:           PlayerCountHU | PlayerCountSpinAndGo,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "HU+SpinAndGo skips 5 players",
			entries:          entries5Players,
			filter:           PlayerCountHU | PlayerCountSpinAndGo,
			wantHandsCount:   0,
			wantSkippedHands: 1,
		},
		{
			name:             "SpinAndGo+MTT includes 3 players",
			entries:          entries3Players,
			filter:           PlayerCountSpinAndGo | PlayerCountMTT,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "SpinAndGo+MTT includes 5 players",
			entries:          entries5Players,
			filter:           PlayerCountSpinAndGo | PlayerCountMTT,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "SpinAndGo+MTT skips 2 players",
			entries:          entries2Players,
			filter:           PlayerCountSpinAndGo | PlayerCountMTT,
			wantHandsCount:   0,
			wantSkippedHands: 1,
		},
		{
			name:             "HU+SpinAndGo+MTT includes 2 players",
			entries:          entries2Players,
			filter:           PlayerCountHU | PlayerCountSpinAndGo | PlayerCountMTT,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "HU+SpinAndGo+MTT includes 3 players",
			entries:          entries3Players,
			filter:           PlayerCountHU | PlayerCountSpinAndGo | PlayerCountMTT,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "HU+SpinAndGo+MTT includes 5 players",
			entries:          entries5Players,
			filter:           PlayerCountHU | PlayerCountSpinAndGo | PlayerCountMTT,
			wantHandsCount:   1,
			wantSkippedHands: 0,
		},
		{
			name:             "HU+SpinAndGo+MTT skips 10 players",
			entries:          entries10Players,
			filter:           PlayerCountHU | PlayerCountSpinAndGo | PlayerCountMTT,
			wantHandsCount:   0,
			wantSkippedHands: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ConvertOptions{
				PlayerCountFilter: tt.filter,
			}

			gotHands, gotSkip, _, err := ParseHands(tt.entries, opts)
			if err != nil {
				t.Errorf("ParseHands() unexpected error: %v", err)
				return
			}

			if len(gotHands) != tt.wantHandsCount {
				t.Errorf("ParseHands() returned %d hands, want %d", len(gotHands), tt.wantHandsCount)
			}

			if gotSkip != tt.wantSkippedHands {
				t.Errorf("ParseHands() skipped %d hands, want %d", gotSkip, tt.wantSkippedHands)
			}
		})
	}
}
