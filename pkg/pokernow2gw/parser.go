package pokernow2gw

import (
	"crypto/sha256"
	"encoding/binary"
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

// ParseHands parses LogEntry slice into Hand slice
// Returns ErrSpectatorLog if no hero cards are found in any hand (spectator log)
func ParseHands(entries []LogEntry, opts ConvertOptions) ([]Hand, int, error) {
	var hands []Hand
	var currentHand *Hand
	var currentStreet Street
	skippedHands := 0

	for i := 0; i < len(entries); i++ {
		entry := entries[i].Entry

		// Starting hand
		if matches := reStartingHand.FindStringSubmatch(entry); matches != nil {
			if currentHand != nil {
				// Previous hand was not properly closed
				skippedHands++
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
			currentStreet = StreetPreflop
			continue
		}

		// Ending hand
		if matches := reEndingHand.FindStringSubmatch(entry); matches != nil {
			if currentHand != nil {
				hands = append(hands, *currentHand)
				currentHand = nil
			}
			continue
		}

		// Skip if no current hand
		if currentHand == nil {
			continue
		}

		// Player stacks
		if matches := rePlayerStacks.FindStringSubmatch(entry); matches != nil {
			stacksStr := matches[1]
			players := parsePlayerStacks(stacksStr)
			playerCount := len(players)

			// Check if player count exceeds 10-max limit
			if playerCount > 10 {
				// Skip this hand as it exceeds 10-max limit
				skippedHands++
				currentHand = nil
				continue
			}

			// Apply player count filter based on GTO Wizard plan
			if !opts.PlayerCountFilter.isPlayerCountAllowed(playerCount) {
				skippedHands++
				currentHand = nil
				continue
			}

			currentHand.Players = players
			continue
		}

		// Your hand (Hero cards)
		if matches := reYourHand.FindStringSubmatch(entry); matches != nil {
			cardsStr := matches[1]
			cards := parseCards(cardsStr)
			currentHand.HeroCards = cards
			continue
		}

		// Ante
		if matches := reAnte.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			amount, _ := strconv.Atoi(matches[2])
			currentHand.Ante = amount
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionPostAnte,
				Amount:     amount,
				Street:     currentStreet,
			})
			continue
		}

		// Small blind
		if matches := reSmallBlind.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			amount, _ := strconv.Atoi(matches[2])
			currentHand.SmallBlind = amount
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionPostSB,
				Amount:     amount,
				Street:     currentStreet,
			})
			continue
		}

		// Big blind
		if matches := reBigBlind.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			amount, _ := strconv.Atoi(matches[2])
			currentHand.BigBlind = amount
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionPostBB,
				Amount:     amount,
				Street:     currentStreet,
			})
			continue
		}

		// Flop
		if matches := reFlop.FindStringSubmatch(entry); matches != nil {
			currentStreet = StreetFlop
			cards := parseCards(matches[1])
			if len(cards) >= 3 {
				currentHand.Board.Flop = cards[:3]
			}
			continue
		}

		// Turn
		if matches := reTurn.FindStringSubmatch(entry); matches != nil {
			currentStreet = StreetTurn
			card := convertCard(strings.TrimSpace(matches[1]))
			currentHand.Board.Turn = card
			continue
		}

		// River
		if matches := reRiver.FindStringSubmatch(entry); matches != nil {
			currentStreet = StreetRiver
			card := convertCard(strings.TrimSpace(matches[1]))
			currentHand.Board.River = card
			continue
		}

		// Folds
		if matches := reFolds.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionFold,
				Street:     currentStreet,
			})
			continue
		}

		// Checks
		if matches := reChecks.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionCheck,
				Street:     currentStreet,
			})
			continue
		}

		// Calls (all-in)
		if matches := reCallsAllIn.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			amount, _ := strconv.Atoi(matches[2])
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionCall,
				Amount:     amount,
				Street:     currentStreet,
				IsAllIn:    true,
			})
			continue
		}

		// Calls
		if matches := reCalls.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			amount, _ := strconv.Atoi(matches[2])
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionCall,
				Amount:     amount,
				Street:     currentStreet,
			})
			continue
		}

		// Bets (all-in)
		if matches := reBetsAllIn.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			amount, _ := strconv.Atoi(matches[2])
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionBet,
				Amount:     amount,
				Street:     currentStreet,
				IsAllIn:    true,
			})
			continue
		}

		// Bets
		if matches := reBets.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			amount, _ := strconv.Atoi(matches[2])
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionBet,
				Amount:     amount,
				Street:     currentStreet,
			})
			continue
		}

		// Raises (all-in)
		if matches := reRaisesAllIn.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			amount, _ := strconv.Atoi(matches[2])
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionRaise,
				Amount:     amount,
				Street:     currentStreet,
				IsAllIn:    true,
			})
			continue
		}

		// Raises
		if matches := reRaises.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			amount, _ := strconv.Atoi(matches[2])
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionRaise,
				Amount:     amount,
				Street:     currentStreet,
			})
			continue
		}

		// Shows
		if matches := reShows.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			cardsStr := matches[2]
			cards := parseCards(cardsStr)
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionShow,
				Street:     StreetShowdown,
			})
			// Add to winners (amount will be filled later)
			currentHand.Winners = append(currentHand.Winners, Winner{
				Player:    player,
				HandCards: cards,
			})
			continue
		}

		// Collected
		if matches := reCollected.FindStringSubmatch(entry); matches != nil {
			player := extractDisplayName(matches[1])
			amount, _ := strconv.Atoi(matches[2])
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionCollect,
				Amount:     amount,
				Street:     StreetShowdown,
			})
			// Update winner amount
			for i := range currentHand.Winners {
				if currentHand.Winners[i].Player == player {
					currentHand.Winners[i].Amount = amount
					break
				}
			}
			// If winner not found in shows, add new winner
			found := false
			for _, w := range currentHand.Winners {
				if w.Player == player {
					found = true
					break
				}
			}
			if !found {
				currentHand.Winners = append(currentHand.Winners, Winner{
					Player: player,
					Amount: amount,
				})
			}
			continue
		}

		// Uncalled bet
		if matches := reUncalled.FindStringSubmatch(entry); matches != nil {
			amount, _ := strconv.Atoi(matches[1])
			player := extractDisplayName(matches[2])
			currentHand.Actions = append(currentHand.Actions, Action{
				Player:     player,
				ActionType: ActionUncalled,
				Amount:     amount,
				Street:     currentStreet,
			})
			continue
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
			return nil, 0, ErrSpectatorLog
		}
	}

	return hands, skippedHands, nil
}

// extractDisplayName extracts display name from PokerNow player string
// Example: "whywaita @ DtjzvbAuKs" -> "whywaita"
// Example: "spa @ ces @ ZQfm6ZDMPO" -> "spa@ces"
// Example: "@atsymbols@@@ @ 3mid0aO0hZ" -> "@atsymbols@@@"
// Example: """quotes""”' @ us2R6psQVF" -> "quotes”'"
func extractDisplayName(fullName string) string {
	// Find the last @ and take everything before it
	lastAtIndex := strings.LastIndex(fullName, "@")
	if lastAtIndex == -1 {
		return fullName
	}

	// Get the part before the last @
	displayName := fullName[:lastAtIndex]

	// Remove all spaces
	displayName = strings.ReplaceAll(displayName, " ", "")

	// Remove double quotes (CSV escaping)
	displayName = strings.ReplaceAll(displayName, `""`, "")

	// Remove leading and trailing single quotes
	displayName = strings.Trim(displayName, `"`)

	return displayName
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

// normalizePlayerSeats renumbers player seats from 1 to N
// This ensures compatibility with GTO Wizard which doesn't recognize button at seat 10
func normalizePlayerSeats(players []Player) []Player {
	if len(players) == 0 {
		return players
	}

	// Sort players by original seat number
	sortedPlayers := make([]Player, len(players))
	copy(sortedPlayers, players)

	// Simple bubble sort by seat number
	for i := 0; i < len(sortedPlayers)-1; i++ {
		for j := i + 1; j < len(sortedPlayers); j++ {
			if sortedPlayers[i].SeatNumber > sortedPlayers[j].SeatNumber {
				sortedPlayers[i], sortedPlayers[j] = sortedPlayers[j], sortedPlayers[i]
			}
		}
	}

	// Renumber seats from 1 to N
	for i := range sortedPlayers {
		sortedPlayers[i].SeatNumber = i + 1
	}

	return sortedPlayers
}

// parseCards parses card string and returns slice of GTO Wizard-formatted cards
// Example: "A♥, 4♦" -> ["Ah", "4d"]
func parseCards(cardsStr string) []string {
	parts := strings.Split(cardsStr, ",")
	var cards []string

	for _, part := range parts {
		card := strings.TrimSpace(part)
		cards = append(cards, convertCard(card))
	}

	return cards
}

// convertCard converts PokerNow card to GTO Wizard format
// Example: "A♥" -> "Ah", "10♦" -> "Td"
func convertCard(card string) string {
	card = strings.TrimSpace(card)

	// Replace suits
	card = strings.ReplaceAll(card, "♥", "h")
	card = strings.ReplaceAll(card, "♦", "d")
	card = strings.ReplaceAll(card, "♣", "c")
	card = strings.ReplaceAll(card, "♠", "s")

	// Replace 10 with T
	card = strings.ReplaceAll(card, "10", "T")

	return card
}

// ConvertEntries converts LogEntry slice to GTO Wizard HH format
func ConvertEntries(entries []LogEntry, opts ConvertOptions) (*ConvertResult, error) {
	// Set defaults
	if opts.SiteName == "" {
		opts.SiteName = "PokerStars"
	}
	if opts.TimeLocation == nil {
		opts.TimeLocation = entries[0].At.Location()
	}

	// Parse hands
	hands, skippedHands, err := ParseHands(entries, opts)
	if err != nil {
		return nil, err
	}

	// Convert to HH format
	hh := convertHandsToHH(hands, opts)

	return &ConvertResult{
		HH:           []byte(hh),
		SkippedHands: skippedHands,
	}, nil
}

// convertHandsToHH converts Hand slice to GTO Wizard HH text
func convertHandsToHH(hands []Hand, opts ConvertOptions) string {
	var sb strings.Builder

	// Determine tournament ID: use first hand's ID if not specified
	tournamentID := opts.TournamentID
	if tournamentID == "" && len(hands) > 0 {
		tournamentID = hands[0].HandID
	}

	for i, hand := range hands {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(convertHandToHH(hand, opts, tournamentID))
	}

	return sb.String()
}

// convertHandToHH converts a single Hand to GTO Wizard HH text
func convertHandToHH(hand Hand, opts ConvertOptions, tournamentID string) string {
	var sb strings.Builder

	// Hand header
	timestamp := hand.StartTime.In(opts.TimeLocation).Format("2006/01/02 15:04:05")
	sb.WriteString(fmt.Sprintf("%s Hand #%s:  Tournament #%s, $0+$0 Hold'em No Limit - Level 1 (%d/%d) - %s\n",
		opts.SiteName, hand.HandID, tournamentID, hand.SmallBlind, hand.BigBlind, timestamp))

	// Table info
	numPlayers := len(hand.Players)
	sb.WriteString(fmt.Sprintf("Table 'PokerNow %s' %d-max Seat #%d is the button\n",
		tournamentID, numPlayers, getDealerSeat(hand)))

	// Seat info
	for _, player := range hand.Players {
		sb.WriteString(fmt.Sprintf("Seat %d: %s (%d in chips)\n",
			player.SeatNumber, player.DisplayName, player.Stack))
	}

	// Actions by street
	sb.WriteString("*** HOLE CARDS ***\n")
	if len(hand.HeroCards) > 0 {
		sb.WriteString(fmt.Sprintf("Dealt to %s [%s]\n", opts.HeroName, strings.Join(hand.HeroCards, " ")))
	}
	sb.WriteString(formatActionsForStreet(hand.Actions, StreetPreflop, hand, ""))
	if len(hand.Board.Flop) > 0 {
		flopStr := fmt.Sprintf("*** FLOP *** [%s]", strings.Join(hand.Board.Flop, " "))
		sb.WriteString(formatActionsForStreet(hand.Actions, StreetFlop, hand, flopStr))
	}
	if hand.Board.Turn != "" {
		turnStr := fmt.Sprintf("*** TURN *** [%s] [%s]",
			strings.Join(hand.Board.Flop, " "), hand.Board.Turn)
		sb.WriteString(formatActionsForStreet(hand.Actions, StreetTurn, hand, turnStr))
	}
	if hand.Board.River != "" {
		riverStr := fmt.Sprintf("*** RIVER *** [%s %s] [%s]",
			strings.Join(hand.Board.Flop, " "), hand.Board.Turn, hand.Board.River)
		sb.WriteString(formatActionsForStreet(hand.Actions, StreetRiver, hand, riverStr))
	}

	// Showdown
	hasShowdown := false
	for _, action := range hand.Actions {
		if action.ActionType == ActionShow {
			hasShowdown = true
			break
		}
	}

	if hasShowdown {
		sb.WriteString("*** SHOW DOWN ***\n")
		for _, action := range hand.Actions {
			if action.ActionType == ActionShow {
				// Find winner to get hand cards
				for _, winner := range hand.Winners {
					if winner.Player == action.Player && len(winner.HandCards) > 0 {
						sb.WriteString(fmt.Sprintf("%s: shows [%s]\n",
							action.Player, strings.Join(winner.HandCards, " ")))
						break
					}
				}
			}
		}
	}

	// "doesn't show hand" for winners without showdown
	if !hasShowdown && len(hand.Winners) > 0 {
		for _, winner := range hand.Winners {
			if winner.Amount > 0 {
				sb.WriteString(fmt.Sprintf("%s: doesn't show hand\n", winner.Player))
			}
		}
	}

	// Summary
	sb.WriteString("*** SUMMARY ***\n")
	sb.WriteString(fmt.Sprintf("Total pot %d | Rake 0\n", calculateTotalPot(hand)))
	if len(hand.Board.Flop) > 0 {
		boardCards := append([]string{}, hand.Board.Flop...)
		if hand.Board.Turn != "" {
			boardCards = append(boardCards, hand.Board.Turn)
		}
		if hand.Board.River != "" {
			boardCards = append(boardCards, hand.Board.River)
		}
		sb.WriteString(fmt.Sprintf("Board [%s]\n", strings.Join(boardCards, " ")))
	}

	// Detailed player summary
	for _, player := range hand.Players {
		sb.WriteString(formatPlayerSummary(hand, player))
	}

	return sb.String()
}

// formatActionsForStreet formats actions for a specific street
func formatActionsForStreet(actions []Action, street Street, hand Hand, streetHeader string) string {
	var sb strings.Builder
	var streetActions []Action

	for _, action := range actions {
		if action.Street == street {
			streetActions = append(streetActions, action)
		}
	}

	if len(streetActions) == 0 && street != StreetPreflop {
		return ""
	}

	if streetHeader != "" {
		sb.WriteString(streetHeader + "\n")
	}

	// Track current bet amounts for each player and the highest bet in the street
	playerBets := make(map[string]int)
	currentBet := 0

	for _, action := range streetActions {
		switch action.ActionType {
		case ActionPostSB:
			sb.WriteString(fmt.Sprintf("%s: posts small blind %d\n", action.Player, action.Amount))
			playerBets[action.Player] = action.Amount
			if action.Amount > currentBet {
				currentBet = action.Amount
			}
		case ActionPostBB:
			sb.WriteString(fmt.Sprintf("%s: posts big blind %d\n", action.Player, action.Amount))
			playerBets[action.Player] = action.Amount
			if action.Amount > currentBet {
				currentBet = action.Amount
			}
		case ActionPostAnte:
			sb.WriteString(fmt.Sprintf("%s: posts an ante of %d\n", action.Player, action.Amount))
		case ActionFold:
			sb.WriteString(fmt.Sprintf("%s: folds\n", action.Player))
		case ActionCheck:
			sb.WriteString(fmt.Sprintf("%s: checks\n", action.Player))
		case ActionCall:
			// Calculate the actual call amount (difference from current bet)
			alreadyBet := playerBets[action.Player]
			callAmount := currentBet - alreadyBet
			if action.IsAllIn {
				sb.WriteString(fmt.Sprintf("%s: calls %d and is all-in\n", action.Player, callAmount))
			} else {
				sb.WriteString(fmt.Sprintf("%s: calls %d\n", action.Player, callAmount))
			}
			playerBets[action.Player] = currentBet
		case ActionBet:
			if action.IsAllIn {
				sb.WriteString(fmt.Sprintf("%s: bets %d and is all-in\n", action.Player, action.Amount))
			} else {
				sb.WriteString(fmt.Sprintf("%s: bets %d\n", action.Player, action.Amount))
			}
			playerBets[action.Player] = action.Amount
			currentBet = action.Amount
		case ActionRaise:
			raiseAmount := action.Amount
			// For all-in, add ante back to get the actual stack committed
			if action.IsAllIn {
				raiseAmount = action.Amount + hand.Ante
			}
			if action.IsAllIn {
				sb.WriteString(fmt.Sprintf("%s: raises to %d and is all-in\n", action.Player, raiseAmount))
			} else {
				sb.WriteString(fmt.Sprintf("%s: raises to %d\n", action.Player, raiseAmount))
			}
			playerBets[action.Player] = raiseAmount
			currentBet = raiseAmount
		case ActionUncalled:
			sb.WriteString(fmt.Sprintf("Uncalled bet (%d) returned to %s\n", action.Amount, action.Player))
		}
	}

	return sb.String()
}

// getDealerSeat returns the seat number of the dealer
func getDealerSeat(hand Hand) int {
	for _, player := range hand.Players {
		if player.DisplayName == hand.Dealer {
			return player.SeatNumber
		}
	}
	return 1
}

// calculateTotalPot calculates the total pot
func calculateTotalPot(hand Hand) int {
	total := 0
	for _, winner := range hand.Winners {
		total += winner.Amount
	}
	// Uncalled bets should NOT be added to the pot
	// because they were returned to the player
	return total
}

// formatPlayerSummary formats a player's summary line
func formatPlayerSummary(hand Hand, player Player) string {
	var sb strings.Builder

	// Determine player role
	role := ""
	dealerSeat := getDealerSeat(hand)
	if player.SeatNumber == dealerSeat {
		role = " (button)"
	}

	// Find SB and BB
	sbPlayer := ""
	bbPlayer := ""
	for _, action := range hand.Actions {
		if action.ActionType == ActionPostSB {
			sbPlayer = action.Player
		}
		if action.ActionType == ActionPostBB {
			bbPlayer = action.Player
		}
	}

	if player.DisplayName == sbPlayer {
		role = " (small blind)"
	} else if player.DisplayName == bbPlayer {
		role = " (big blind)"
	}

	// Find player's last meaningful action
	lastAction := ""
	lastStreet := StreetPreflop
	didBet := false

	for _, action := range hand.Actions {
		if action.Player == player.DisplayName {
			switch action.ActionType {
			case ActionFold:
				lastAction = "folded"
				lastStreet = action.Street
			case ActionCall, ActionBet, ActionRaise:
				// Only voluntary actions count as "bet"
				didBet = true
				lastStreet = action.Street
			case ActionPostSB, ActionPostBB, ActionPostAnte:
				// These are forced, not voluntary
			}
		}
	}

	// Check if player won
	wonAmount := 0
	for _, winner := range hand.Winners {
		if winner.Player == player.DisplayName {
			wonAmount = winner.Amount
			break
		}
	}

	// Check if player showed hand
	showedHand := false
	for _, action := range hand.Actions {
		if action.ActionType == ActionShow && action.Player == player.DisplayName {
			showedHand = true
			break
		}
	}

	// Format the line
	sb.WriteString(fmt.Sprintf("Seat %d: %s%s ", player.SeatNumber, player.DisplayName, role))

	if wonAmount > 0 {
		sb.WriteString(fmt.Sprintf("collected (%d)", wonAmount))
	} else if showedHand {
		// Player showed hand but didn't win
		sb.WriteString("showed and lost")
	} else if lastAction == "folded" {
		streetName := streetToFoldString(lastStreet)
		sb.WriteString(fmt.Sprintf("folded %s", streetName))
		if !didBet {
			sb.WriteString(" (didn't bet)")
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

// streetToFoldString converts street to fold description
func streetToFoldString(street Street) string {
	switch street {
	case StreetPreflop:
		return "before Flop"
	case StreetFlop:
		return "on the Flop"
	case StreetTurn:
		return "on the Turn"
	case StreetRiver:
		return "on the River"
	default:
		return "before Flop"
	}
}

// convertHandIDToNumeric converts a string hand ID to a numeric string
// This ensures compatibility with tools like GTO Wizard that expect numeric hand IDs
func convertHandIDToNumeric(handID string) string {
	// If already numeric, return as is
	if _, err := strconv.ParseInt(handID, 10, 64); err == nil {
		return handID
	}

	// Hash the string ID to get a consistent numeric value
	hash := sha256.Sum256([]byte(handID))
	// Use first 8 bytes to create a 64-bit integer
	numericID := binary.BigEndian.Uint64(hash[:8])
	// Convert to string, ensuring it starts with a digit
	return fmt.Sprintf("%d", numericID)
}
