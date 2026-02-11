package pokernow2gw

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ConvertEntries converts LogEntry slice to GTO Wizard HH format
func ConvertEntries(entries []LogEntry, opts ConvertOptions) (*ConvertResult, error) {
	if len(entries) == 0 {
		return &ConvertResult{
			HH:           nil,
			SkippedHands: 0,
		}, nil
	}

	// Set defaults
	if opts.SiteName == "" {
		opts.SiteName = "PokerStars"
	}
	if opts.TimeLocation == nil {
		opts.TimeLocation = entries[0].At.Location()
	}

	// Parse hands
	hands, skippedHands, skippedHandsInfo, err := ParseHands(entries, opts)
	if err != nil {
		return nil, err
	}

	// Convert to HH format
	hh := convertHandsToHH(hands, opts)

	return &ConvertResult{
		HH:               []byte(hh),
		SkippedHands:     skippedHands,
		SkippedHandsInfo: skippedHandsInfo,
	}, nil
}

// formatNumber formats a float64 amount as a string
// Integers are formatted without decimal places (1.0 → "1")
// Decimals are formatted with 2 decimal places (0.5 → "0.50")
func formatNumber(amount float64) string {
	if math.Trunc(amount) == amount {
		// It's an integer
		return strconv.FormatFloat(amount, 'f', 0, 64)
	}
	// It has a decimal part
	return strconv.FormatFloat(amount, 'f', 2, 64)
}

// isChipsCurrency checks if the currency is "Chips" (case-insensitive)
func isChipsCurrency(currency string) bool {
	return strings.EqualFold(currency, "Chips")
}

// formatAmount formats an amount based on game type and currency
// (adds $ prefix for cash games unless currency is "Chips")
func formatAmount(amount float64, opts ConvertOptions, currency string) string {
	if opts.GameType == GameTypeCash && !isChipsCurrency(currency) {
		return fmt.Sprintf("$%s", formatNumber(amount))
	}
	return formatNumber(amount)
}

// convertHandsToHH converts Hand slice to GTO Wizard HH text
func convertHandsToHH(hands []Hand, opts ConvertOptions) string {
	var sb strings.Builder

	// Determine tournament ID: use first hand's ID if not specified (only for tournaments)
	tournamentID := ""
	if opts.GameType != GameTypeCash {
		tournamentID = opts.TournamentID
		if tournamentID == "" && len(hands) > 0 {
			tournamentID = hands[0].HandID
		}
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
	if opts.GameType == GameTypeCash {
		// Cash game format: PokerStars Hand #ID: Hold'em No Limit ($SB/$BB USD) - timestamp ET
		// For Chips currency, omit $ and USD
		if isChipsCurrency(hand.Currency) {
			sb.WriteString(fmt.Sprintf("%s Hand #%s: Hold'em No Limit (%s/%s) - %s ET\n",
				opts.SiteName, hand.HandID, formatNumber(hand.SmallBlind), formatNumber(hand.BigBlind), timestamp))
		} else {
			sb.WriteString(fmt.Sprintf("%s Hand #%s: Hold'em No Limit ($%s/$%s USD) - %s ET\n",
				opts.SiteName, hand.HandID, formatNumber(hand.SmallBlind), formatNumber(hand.BigBlind), timestamp))
		}
	} else {
		// Tournament format
		sb.WriteString(fmt.Sprintf("%s Hand #%s:  Tournament #%s, $0+$0 Hold'em No Limit - Level 1 (%s/%s) - %s\n",
			opts.SiteName, hand.HandID, tournamentID, formatNumber(hand.SmallBlind), formatNumber(hand.BigBlind), timestamp))
	}

	// Table info
	numPlayers := len(hand.Players)
	tableName := "Poker Now"
	if hand.SiteName != "" && hand.TableName != "" {
		tableName = fmt.Sprintf("%s - %s", hand.SiteName, hand.TableName)
	} else if hand.TableName != "" {
		tableName = hand.TableName
	}

	if opts.GameType == GameTypeCash {
		sb.WriteString(fmt.Sprintf("Table '%s' %d-max Seat #%d is the button\n",
			tableName, numPlayers, getDealerSeat(hand)))
	} else {
		sb.WriteString(fmt.Sprintf("Table 'PokerNow %s' %d-max Seat #%d is the button\n",
			tournamentID, numPlayers, getDealerSeat(hand)))
	}

	// Seat info
	for _, player := range hand.Players {
		if opts.GameType == GameTypeCash {
			// For Chips currency, omit $
			if isChipsCurrency(hand.Currency) {
				sb.WriteString(fmt.Sprintf("Seat %d: %s (%s in chips)\n",
					player.SeatNumber, player.DisplayName, formatNumber(player.Stack)))
			} else {
				sb.WriteString(fmt.Sprintf("Seat %d: %s ($%s in chips)\n",
					player.SeatNumber, player.DisplayName, formatNumber(player.Stack)))
			}
		} else {
			sb.WriteString(fmt.Sprintf("Seat %d: %s (%s in chips)\n",
				player.SeatNumber, player.DisplayName, formatNumber(player.Stack)))
		}
	}

	// Actions by street
	sb.WriteString("*** HOLE CARDS ***\n")
	if len(hand.HeroCards) > 0 {
		sb.WriteString(fmt.Sprintf("Dealt to %s [%s]\n", opts.HeroName, strings.Join(hand.HeroCards, " ")))
	}
	sb.WriteString(formatActionsForStreet(hand.Actions, StreetPreflop, hand, "", opts))
	if len(hand.Board.Flop) > 0 {
		flopStr := fmt.Sprintf("*** FLOP *** [%s]", strings.Join(hand.Board.Flop, " "))
		sb.WriteString(formatActionsForStreet(hand.Actions, StreetFlop, hand, flopStr, opts))
	}
	if hand.Board.Turn != "" {
		turnStr := fmt.Sprintf("*** TURN *** [%s] [%s]",
			strings.Join(hand.Board.Flop, " "), hand.Board.Turn)
		sb.WriteString(formatActionsForStreet(hand.Actions, StreetTurn, hand, turnStr, opts))
	}
	if hand.Board.River != "" {
		riverStr := fmt.Sprintf("*** RIVER *** [%s %s] [%s]",
			strings.Join(hand.Board.Flop, " "), hand.Board.Turn, hand.Board.River)
		sb.WriteString(formatActionsForStreet(hand.Actions, StreetRiver, hand, riverStr, opts))
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

	// Output "collected from pot" line before SUMMARY
	for _, winner := range hand.Winners {
		if winner.Amount > 0 {
			sb.WriteString(fmt.Sprintf("%s collected %s from pot\n", winner.Player, formatAmount(winner.Amount, opts, hand.Currency)))
		}
	}

	// Summary
	sb.WriteString("*** SUMMARY ***\n")
	totalPot := calculateTotalPot(hand)
	rake := calculateRake(totalPot, hand.BigBlind, opts.RakePercent, opts.RakeCapBB)
	sb.WriteString(fmt.Sprintf("Total pot %s | Rake %s\n", formatAmount(totalPot, opts, hand.Currency), formatAmount(rake, opts, hand.Currency)))
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
		sb.WriteString(formatPlayerSummary(hand, player, opts))
	}

	return sb.String()
}

// formatActionsForStreet formats actions for a specific street
func formatActionsForStreet(actions []Action, street Street, hand Hand, streetHeader string, opts ConvertOptions) string {
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
	playerBets := make(map[string]float64)
	currentBet := 0.0

	for _, action := range streetActions {
		switch action.ActionType {
		case ActionPostSB:
			sb.WriteString(fmt.Sprintf("%s: posts small blind %s\n", action.Player, formatAmount(action.Amount, opts, hand.Currency)))
			playerBets[action.Player] = action.Amount
			if action.Amount > currentBet {
				currentBet = action.Amount
			}
		case ActionPostBB:
			sb.WriteString(fmt.Sprintf("%s: posts big blind %s\n", action.Player, formatAmount(action.Amount, opts, hand.Currency)))
			playerBets[action.Player] = action.Amount
			if action.Amount > currentBet {
				currentBet = action.Amount
			}
		case ActionPostAnte:
			sb.WriteString(fmt.Sprintf("%s: posts an ante of %s\n", action.Player, formatAmount(action.Amount, opts, hand.Currency)))
		case ActionFold:
			sb.WriteString(fmt.Sprintf("%s: folds\n", action.Player))
		case ActionCheck:
			sb.WriteString(fmt.Sprintf("%s: checks\n", action.Player))
		case ActionCall:
			// Calculate the actual call amount (difference from current bet)
			alreadyBet := playerBets[action.Player]
			callAmount := currentBet - alreadyBet
			if action.IsAllIn {
				sb.WriteString(fmt.Sprintf("%s: calls %s and is all-in\n", action.Player, formatAmount(callAmount, opts, hand.Currency)))
			} else {
				sb.WriteString(fmt.Sprintf("%s: calls %s\n", action.Player, formatAmount(callAmount, opts, hand.Currency)))
			}
			playerBets[action.Player] = currentBet
		case ActionBet:
			if action.IsAllIn {
				sb.WriteString(fmt.Sprintf("%s: bets %s and is all-in\n", action.Player, formatAmount(action.Amount, opts, hand.Currency)))
			} else {
				sb.WriteString(fmt.Sprintf("%s: bets %s\n", action.Player, formatAmount(action.Amount, opts, hand.Currency)))
			}
			playerBets[action.Player] = action.Amount
			currentBet = action.Amount
		case ActionRaise:
			raiseAmount := action.Amount
			// For all-in, add ante back to get the actual stack committed
			if action.IsAllIn {
				raiseAmount = action.Amount + hand.Ante
			}
			// For cash games, use "raises X to Y" format
			if opts.GameType == GameTypeCash {
				raiseDiff := raiseAmount - currentBet
				if action.IsAllIn {
					sb.WriteString(fmt.Sprintf("%s: raises %s to %s and is all-in\n", action.Player, formatAmount(raiseDiff, opts, hand.Currency), formatAmount(raiseAmount, opts, hand.Currency)))
				} else {
					sb.WriteString(fmt.Sprintf("%s: raises %s to %s\n", action.Player, formatAmount(raiseDiff, opts, hand.Currency), formatAmount(raiseAmount, opts, hand.Currency)))
				}
			} else {
				if action.IsAllIn {
					sb.WriteString(fmt.Sprintf("%s: raises to %s and is all-in\n", action.Player, formatNumber(raiseAmount)))
				} else {
					sb.WriteString(fmt.Sprintf("%s: raises to %s\n", action.Player, formatNumber(raiseAmount)))
				}
			}
			playerBets[action.Player] = raiseAmount
			currentBet = raiseAmount
		case ActionUncalled:
			sb.WriteString(fmt.Sprintf("Uncalled bet (%s) returned to %s\n", formatAmount(action.Amount, opts, hand.Currency), action.Player))
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

// formatPlayerSummary formats a player's summary line
func formatPlayerSummary(hand Hand, player Player, opts ConvertOptions) string {
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
	wonAmount := 0.0
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
		sb.WriteString(fmt.Sprintf("collected (%s)", formatAmount(wonAmount, opts, hand.Currency)))
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
