package pokernow2gw

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	// 正規表現パターン
	// (id: ...) 部分はオプション
	// dealer部分は (dealer: "...") または (dead button) をサポート
	reStartingHand = regexp.MustCompile(`^-- starting hand #(\d+)\s+(?:\(id: ([a-z0-9]+)\)\s+)?\(No Limit Texas Hold'em\)\s+(?:\(dealer: "([^"]+)"\)|\(dead button\)) --$`)
	reEndingHand   = regexp.MustCompile(`^-- ending hand #(\d+) --$`)
	rePlayerStacks = regexp.MustCompile(`Player stacks: (.+)$`)
	// Allow quotes in player names by using non-greedy match up to " (
	rePlayerStack = regexp.MustCompile(`#(\d+) "(.+?)" \((\d+)\)`)
	reYourHand    = regexp.MustCompile(`^Your hand is (.+)$`)
	reAnte        = regexp.MustCompile(`^"([^"]+)" posts an ante of (\d+)$`)
	reSmallBlind  = regexp.MustCompile(`^"([^"]+)" posts a small blind of (\d+)$`)
	reBigBlind    = regexp.MustCompile(`^"([^"]+)" posts a big blind of (\d+)$`)
	reFolds       = regexp.MustCompile(`^"([^"]+)" folds$`)
	reChecks      = regexp.MustCompile(`^"([^"]+)" checks$`)
	reCalls       = regexp.MustCompile(`^"([^"]+)" calls (\d+)$`)
	reCallsAllIn  = regexp.MustCompile(`^"([^"]+)" calls (\d+) and go all in$`)
	reBets        = regexp.MustCompile(`^"([^"]+)" bets (\d+)$`)
	reBetsAllIn   = regexp.MustCompile(`^"([^"]+)" bets (\d+) and go all in$`)
	reRaises      = regexp.MustCompile(`^"([^"]+)" raises to (\d+)$`)
	reRaisesAllIn = regexp.MustCompile(`^"([^"]+)" raises to (\d+) and go all in$`)
	reFlop        = regexp.MustCompile(`^Flop:\s+\[([^\]]+)\]$`)
	reTurn        = regexp.MustCompile(`^Turn: [^[]+\[([^\]]+)\]$`)
	reRiver       = regexp.MustCompile(`^River: [^[]+\[([^\]]+)\]$`)
	reShows       = regexp.MustCompile(`^"([^"]+)" shows a (.+)\.$`)
	reCollected   = regexp.MustCompile(`^"([^"]+)" collected (\d+) from pot`)
	reUncalled    = regexp.MustCompile(`^Uncalled bet of (\d+) returned to "([^"]+)"$`)
)

// parseContext holds the mutable state passed to each log handler during parsing.
type parseContext struct {
	currentStreet   *Street
	currentHand     **Hand
	handStartIndex  *int
	entries         []LogEntry
	entryIndex      int
	opts            ConvertOptions
	skippedHands    *int
	skippedInfo     *[]SkippedHandInfo
	extractRawInput func(startIdx, endIdx int) []string
}

// logHandler maps a regex pattern to its handler function.
// The handler receives the regex match groups and the current parsing context.
// It returns an error if parsing fails (e.g., invalid numeric values).
type logHandler struct {
	pattern *regexp.Regexp
	handle  func(matches []string, ctx *parseContext) error
}

// logHandlers is the table-driven dispatch table for log entry parsing.
// Order matters: more specific patterns (e.g., all-in variants) must come
// before their general counterparts to avoid incorrect matches.
var logHandlers = []logHandler{
	// Player stacks
	{
		pattern: rePlayerStacks,
		handle: func(matches []string, ctx *parseContext) error {
			hand := *ctx.currentHand
			stacksStr := matches[1]
			players := parsePlayerStacks(stacksStr)
			playerCount := len(players)

			// Check if player count exceeds 10-max limit
			if playerCount > 10 {
				endIdx := findEndingHandIndex(ctx.entries, ctx.entryIndex, hand.HandNumber)
				*ctx.skippedHands++
				*ctx.skippedInfo = append(*ctx.skippedInfo, SkippedHandInfo{
					HandID:      hand.HandID,
					HandNumber:  hand.HandNumber,
					Reason:      SkipReasonTooManyPlayers,
					Detail:      fmt.Sprintf("Hand #%s has %d players, but GTO Wizard only supports up to 10 players", hand.HandNumber, playerCount),
					PlayerCount: playerCount,
					RawInput:    ctx.extractRawInput(*ctx.handStartIndex, endIdx),
				})
				*ctx.currentHand = nil
				*ctx.handStartIndex = -1
				return nil
			}

			// Apply player count filter based on GTO Wizard plan
			if !ctx.opts.PlayerCountFilter.isPlayerCountAllowed(playerCount) {
				endIdx := findEndingHandIndex(ctx.entries, ctx.entryIndex, hand.HandNumber)
				*ctx.skippedHands++
				*ctx.skippedInfo = append(*ctx.skippedInfo, SkippedHandInfo{
					HandID:      hand.HandID,
					HandNumber:  hand.HandNumber,
					Reason:      SkipReasonFilteredOut,
					Detail:      fmt.Sprintf("Hand #%s has %d players, which does not match the selected filter", hand.HandNumber, playerCount),
					PlayerCount: playerCount,
					RawInput:    ctx.extractRawInput(*ctx.handStartIndex, endIdx),
				})
				*ctx.currentHand = nil
				*ctx.handStartIndex = -1
				return nil
			}

			hand.Players = players
			return nil
		},
	},
	// Your hand (Hero cards)
	{
		pattern: reYourHand,
		handle: func(matches []string, ctx *parseContext) error {
			cards := parseCards(matches[1])
			(*ctx.currentHand).HeroCards = cards
			return nil
		},
	},
	// Ante
	{
		pattern: reAnte,
		handle: func(matches []string, ctx *parseContext) error {
			hand := *ctx.currentHand
			player := extractDisplayName(matches[1])
			amount, err := strconv.Atoi(matches[2])
			if err != nil {
				return fmt.Errorf("failed to parse ante amount %q in hand #%s: %w", matches[2], hand.HandNumber, err)
			}
			hand.Ante = amount
			hand.Actions = append(hand.Actions, Action{
				Player:     player,
				ActionType: ActionPostAnte,
				Amount:     amount,
				Street:     *ctx.currentStreet,
			})
			return nil
		},
	},
	// Small blind
	{
		pattern: reSmallBlind,
		handle: func(matches []string, ctx *parseContext) error {
			hand := *ctx.currentHand
			player := extractDisplayName(matches[1])
			amount, err := strconv.Atoi(matches[2])
			if err != nil {
				return fmt.Errorf("failed to parse small blind amount %q in hand #%s: %w", matches[2], hand.HandNumber, err)
			}
			hand.SmallBlind = amount
			hand.Actions = append(hand.Actions, Action{
				Player:     player,
				ActionType: ActionPostSB,
				Amount:     amount,
				Street:     *ctx.currentStreet,
			})
			return nil
		},
	},
	// Big blind
	{
		pattern: reBigBlind,
		handle: func(matches []string, ctx *parseContext) error {
			hand := *ctx.currentHand
			player := extractDisplayName(matches[1])
			amount, err := strconv.Atoi(matches[2])
			if err != nil {
				return fmt.Errorf("failed to parse big blind amount %q in hand #%s: %w", matches[2], hand.HandNumber, err)
			}
			hand.BigBlind = amount
			hand.Actions = append(hand.Actions, Action{
				Player:     player,
				ActionType: ActionPostBB,
				Amount:     amount,
				Street:     *ctx.currentStreet,
			})
			return nil
		},
	},
	// Flop
	{
		pattern: reFlop,
		handle: func(matches []string, ctx *parseContext) error {
			*ctx.currentStreet = StreetFlop
			cards := parseCards(matches[1])
			if len(cards) >= 3 {
				(*ctx.currentHand).Board.Flop = cards[:3]
			}
			return nil
		},
	},
	// Turn
	{
		pattern: reTurn,
		handle: func(matches []string, ctx *parseContext) error {
			*ctx.currentStreet = StreetTurn
			card := convertCard(strings.TrimSpace(matches[1]))
			(*ctx.currentHand).Board.Turn = card
			return nil
		},
	},
	// River
	{
		pattern: reRiver,
		handle: func(matches []string, ctx *parseContext) error {
			*ctx.currentStreet = StreetRiver
			card := convertCard(strings.TrimSpace(matches[1]))
			(*ctx.currentHand).Board.River = card
			return nil
		},
	},
	// Folds
	{
		pattern: reFolds,
		handle: func(matches []string, ctx *parseContext) error {
			player := extractDisplayName(matches[1])
			(*ctx.currentHand).Actions = append((*ctx.currentHand).Actions, Action{
				Player:     player,
				ActionType: ActionFold,
				Street:     *ctx.currentStreet,
			})
			return nil
		},
	},
	// Checks
	{
		pattern: reChecks,
		handle: func(matches []string, ctx *parseContext) error {
			player := extractDisplayName(matches[1])
			(*ctx.currentHand).Actions = append((*ctx.currentHand).Actions, Action{
				Player:     player,
				ActionType: ActionCheck,
				Street:     *ctx.currentStreet,
			})
			return nil
		},
	},
	// Calls (all-in) — must come before regular calls
	{
		pattern: reCallsAllIn,
		handle: func(matches []string, ctx *parseContext) error {
			player := extractDisplayName(matches[1])
			amount, err := strconv.Atoi(matches[2])
			if err != nil {
				return fmt.Errorf("failed to parse call all-in amount %q in hand #%s: %w", matches[2], (*ctx.currentHand).HandNumber, err)
			}
			(*ctx.currentHand).Actions = append((*ctx.currentHand).Actions, Action{
				Player:     player,
				ActionType: ActionCall,
				Amount:     amount,
				Street:     *ctx.currentStreet,
				IsAllIn:    true,
			})
			return nil
		},
	},
	// Calls
	{
		pattern: reCalls,
		handle: func(matches []string, ctx *parseContext) error {
			player := extractDisplayName(matches[1])
			amount, err := strconv.Atoi(matches[2])
			if err != nil {
				return fmt.Errorf("failed to parse call amount %q in hand #%s: %w", matches[2], (*ctx.currentHand).HandNumber, err)
			}
			(*ctx.currentHand).Actions = append((*ctx.currentHand).Actions, Action{
				Player:     player,
				ActionType: ActionCall,
				Amount:     amount,
				Street:     *ctx.currentStreet,
			})
			return nil
		},
	},
	// Bets (all-in) — must come before regular bets
	{
		pattern: reBetsAllIn,
		handle: func(matches []string, ctx *parseContext) error {
			player := extractDisplayName(matches[1])
			amount, err := strconv.Atoi(matches[2])
			if err != nil {
				return fmt.Errorf("failed to parse bet all-in amount %q in hand #%s: %w", matches[2], (*ctx.currentHand).HandNumber, err)
			}
			(*ctx.currentHand).Actions = append((*ctx.currentHand).Actions, Action{
				Player:     player,
				ActionType: ActionBet,
				Amount:     amount,
				Street:     *ctx.currentStreet,
				IsAllIn:    true,
			})
			return nil
		},
	},
	// Bets
	{
		pattern: reBets,
		handle: func(matches []string, ctx *parseContext) error {
			player := extractDisplayName(matches[1])
			amount, err := strconv.Atoi(matches[2])
			if err != nil {
				return fmt.Errorf("failed to parse bet amount %q in hand #%s: %w", matches[2], (*ctx.currentHand).HandNumber, err)
			}
			(*ctx.currentHand).Actions = append((*ctx.currentHand).Actions, Action{
				Player:     player,
				ActionType: ActionBet,
				Amount:     amount,
				Street:     *ctx.currentStreet,
			})
			return nil
		},
	},
	// Raises (all-in) — must come before regular raises
	{
		pattern: reRaisesAllIn,
		handle: func(matches []string, ctx *parseContext) error {
			player := extractDisplayName(matches[1])
			amount, err := strconv.Atoi(matches[2])
			if err != nil {
				return fmt.Errorf("failed to parse raise all-in amount %q in hand #%s: %w", matches[2], (*ctx.currentHand).HandNumber, err)
			}
			(*ctx.currentHand).Actions = append((*ctx.currentHand).Actions, Action{
				Player:     player,
				ActionType: ActionRaise,
				Amount:     amount,
				Street:     *ctx.currentStreet,
				IsAllIn:    true,
			})
			return nil
		},
	},
	// Raises
	{
		pattern: reRaises,
		handle: func(matches []string, ctx *parseContext) error {
			player := extractDisplayName(matches[1])
			amount, err := strconv.Atoi(matches[2])
			if err != nil {
				return fmt.Errorf("failed to parse raise amount %q in hand #%s: %w", matches[2], (*ctx.currentHand).HandNumber, err)
			}
			(*ctx.currentHand).Actions = append((*ctx.currentHand).Actions, Action{
				Player:     player,
				ActionType: ActionRaise,
				Amount:     amount,
				Street:     *ctx.currentStreet,
			})
			return nil
		},
	},
	// Shows
	{
		pattern: reShows,
		handle: func(matches []string, ctx *parseContext) error {
			hand := *ctx.currentHand
			player := extractDisplayName(matches[1])
			cards := parseCards(matches[2])
			hand.Actions = append(hand.Actions, Action{
				Player:     player,
				ActionType: ActionShow,
				Street:     StreetShowdown,
			})
			// Add to winners (amount will be filled later)
			hand.Winners = append(hand.Winners, Winner{
				Player:    player,
				HandCards: cards,
			})
			return nil
		},
	},
	// Collected
	{
		pattern: reCollected,
		handle: func(matches []string, ctx *parseContext) error {
			hand := *ctx.currentHand
			player := extractDisplayName(matches[1])
			amount, err := strconv.Atoi(matches[2])
			if err != nil {
				return fmt.Errorf("failed to parse collected amount %q in hand #%s: %w", matches[2], hand.HandNumber, err)
			}
			hand.Actions = append(hand.Actions, Action{
				Player:     player,
				ActionType: ActionCollect,
				Amount:     amount,
				Street:     StreetShowdown,
			})
			// Update winner amount
			for i := range hand.Winners {
				if hand.Winners[i].Player == player {
					hand.Winners[i].Amount = amount
					break
				}
			}
			// If winner not found in shows, add new winner
			found := false
			for _, w := range hand.Winners {
				if w.Player == player {
					found = true
					break
				}
			}
			if !found {
				hand.Winners = append(hand.Winners, Winner{
					Player: player,
					Amount: amount,
				})
			}
			return nil
		},
	},
	// Uncalled bet
	{
		pattern: reUncalled,
		handle: func(matches []string, ctx *parseContext) error {
			amount, err := strconv.Atoi(matches[1])
			if err != nil {
				return fmt.Errorf("failed to parse uncalled bet amount %q in hand #%s: %w", matches[1], (*ctx.currentHand).HandNumber, err)
			}
			player := extractDisplayName(matches[2])
			(*ctx.currentHand).Actions = append((*ctx.currentHand).Actions, Action{
				Player:     player,
				ActionType: ActionUncalled,
				Amount:     amount,
				Street:     *ctx.currentStreet,
			})
			return nil
		},
	},
}

// ParseHands parses LogEntry slice into Hand slice
// Returns ErrSpectatorLog if no hero cards are found in any hand (spectator log)
func ParseHands(entries []LogEntry, opts ConvertOptions) ([]Hand, int, []SkippedHandInfo, error) {
	var hands []Hand
	var currentHand *Hand
	var currentStreet Street
	skippedHands := 0
	var skippedHandsInfo []SkippedHandInfo
	handStartIndex := -1 // Track the start index of current hand

	// Helper function to extract raw input entries for a hand
	extractRawInput := func(startIdx, endIdx int) []string {
		if startIdx < 0 || startIdx >= len(entries) {
			return nil
		}
		if endIdx > len(entries) {
			endIdx = len(entries)
		}
		rawInput := make([]string, 0, endIdx-startIdx)
		for j := startIdx; j < endIdx; j++ {
			rawInput = append(rawInput, entries[j].Entry)
		}
		return rawInput
	}

	ctx := &parseContext{
		currentStreet:   &currentStreet,
		currentHand:     &currentHand,
		handStartIndex:  &handStartIndex,
		entries:         entries,
		opts:            opts,
		skippedHands:    &skippedHands,
		skippedInfo:     &skippedHandsInfo,
		extractRawInput: extractRawInput,
	}

	for i := 0; i < len(entries); i++ {
		entry := entries[i].Entry

		// Starting hand — handled inline because it creates a new hand
		if matches := reStartingHand.FindStringSubmatch(entry); matches != nil {
			if currentHand != nil {
				// Previous hand was not properly closed
				skippedHands++
				skippedHandsInfo = append(skippedHandsInfo, SkippedHandInfo{
					HandID:     currentHand.HandID,
					HandNumber: currentHand.HandNumber,
					Reason:     SkipReasonIncomplete,
					Detail:     fmt.Sprintf("Hand #%s was not properly closed before hand #%s started", currentHand.HandNumber, matches[1]),
					RawInput:   extractRawInput(handStartIndex, i),
				})
			}
			handNum := matches[1]
			handID := matches[2]
			// If no ID, use hand number
			if handID == "" {
				handID = handNum
			}
			// Convert hand ID to numeric format for compatibility
			handID = convertHandIDToNumeric(handID)
			dealer := extractDisplayName(matches[3])

			currentHand = &Hand{
				HandNumber: handNum,
				HandID:     handID,
				Dealer:     dealer,
				StartTime:  entries[i].At,
			}
			handStartIndex = i // Record the start index
			currentStreet = StreetPreflop
			continue
		}

		// Ending hand — handled inline because it finalizes the hand
		if matches := reEndingHand.FindStringSubmatch(entry); matches != nil {
			if currentHand != nil {
				hands = append(hands, *currentHand)
				currentHand = nil
			}
			handStartIndex = -1 // Reset start index
			continue
		}

		// Skip if no current hand
		if currentHand == nil {
			continue
		}

		// Dispatch via handler table
		ctx.entryIndex = i
		for _, h := range logHandlers {
			if matches := h.pattern.FindStringSubmatch(entry); matches != nil {
				if err := h.handle(matches, ctx); err != nil {
					return nil, 0, nil, err
				}
				break
			}
		}
	}

	// Check if this is a spectator log (no hero cards in any hand)
	if len(hands) > 0 {
		hasAnyHeroCards := false
		for _, hand := range hands {
			if len(hand.HeroCards) > 0 {
				hasAnyHeroCards = true
				break
			}
		}
		if !hasAnyHeroCards {
			return nil, 0, nil, ErrSpectatorLog
		}
	}

	return hands, skippedHands, skippedHandsInfo, nil
}

// findEndingHandIndex finds the index of the ending hand marker for a given hand number
// Returns the index after the ending hand marker, or len(entries) if not found
func findEndingHandIndex(entries []LogEntry, startIdx int, handNumber string) int {
	endPattern := fmt.Sprintf("-- ending hand #%s --", handNumber)
	for j := startIdx; j < len(entries); j++ {
		if entries[j].Entry == endPattern {
			return j + 1 // Include the ending hand marker
		}
	}
	return len(entries) // Not found, return end of entries
}

// parsePlayerStacks parses player stacks string
// Example: "#5 "ramune @ 3rSQmMhWok" (66998) | #9 "whywaita @ DtjzvbAuKs" (383002)"
func parsePlayerStacks(stacksStr string) []Player {
	var players []Player
	parts := strings.Split(stacksStr, "|")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if matches := rePlayerStack.FindStringSubmatch(part); matches != nil {
			seatNum, _ := strconv.Atoi(matches[1])
			fullName := matches[2]
			stack, _ := strconv.Atoi(matches[3])

			players = append(players, Player{
				SeatNumber:  seatNum,
				Name:        fullName,
				DisplayName: extractDisplayName(fullName),
				Stack:       stack,
			})
		}
	}

	return normalizePlayerSeats(players)
}
