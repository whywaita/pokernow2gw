package pokernow2gw

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	// maxJSONLLines is the maximum number of lines to process in a JSONL file
	// to prevent excessive memory usage
	maxJSONLLines = 10000
)

// ReadOHH reads Open Hand History JSON from reader and converts to internal Hand format
// Supports both simplified OHH format and official OHH spec format
func ReadOHH(r io.Reader, opts ConvertOptions) (*ConvertResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read OHH JSON: %w", err)
	}

	// Try to detect which OHH format this is
	var formatCheck map[string]interface{}
	if err := json.Unmarshal(data, &formatCheck); err != nil {
		return nil, fmt.Errorf("failed to decode OHH JSON: %w", err)
	}

	// Check if this is the official OHH spec format (has "ohh" field)
	if _, hasOHH := formatCheck["ohh"]; hasOHH {
		return readOHHSpecFormat(data, opts)
	}

	// Otherwise, use the simplified format
	return readSimplifiedOHHFormat(data, opts)
}

// readSimplifiedOHHFormat reads the simplified OHH format
func readSimplifiedOHHFormat(data []byte, opts ConvertOptions) (*ConvertResult, error) {
	var ohhFormat OHHFormat
	if err := json.Unmarshal(data, &ohhFormat); err != nil {
		return nil, fmt.Errorf("failed to decode OHH JSON: %w", err)
	}

	// Set defaults
	if opts.SiteName == "" {
		opts.SiteName = "PokerStars"
	}
	if opts.TimeLocation == nil && len(ohhFormat.Hands) > 0 {
		opts.TimeLocation = ohhFormat.Hands[0].StartTime.Location()
	}

	// Convert OHH hands to internal Hand format
	hands := make([]Hand, 0, len(ohhFormat.Hands))
	for _, ohhHand := range ohhFormat.Hands {
		hand, err := convertOHHHandToHand(ohhHand)
		if err != nil {
			// Skip invalid hands
			continue
		}
		hands = append(hands, hand)
	}

	// Check if this is a spectator log (no hero cards in any hand)
	if isSpectatorLog(hands) {
		return nil, ErrSpectatorLog
	}

	// Convert to HH format
	hh := convertHandsToHH(hands, opts)

	return &ConvertResult{
		HH:           []byte(hh),
		SkippedHands: 0,
	}, nil
}

// readOHHSpecFormat reads the official OHH specification format
func readOHHSpecFormat(data []byte, opts ConvertOptions) (*ConvertResult, error) {
	var specFormat OHHSpecFormat
	if err := json.Unmarshal(data, &specFormat); err != nil {
		return nil, fmt.Errorf("failed to decode OHH spec JSON: %w", err)
	}

	// Set defaults
	if opts.SiteName == "" {
		opts.SiteName = specFormat.OHH.SiteName
		if opts.SiteName == "" {
			opts.SiteName = "PokerStars"
		}
	}
	if opts.TimeLocation == nil {
		opts.TimeLocation = specFormat.OHH.StartDateUTC.Location()
	}

	// Convert OHH spec to internal Hand format
	hand, err := convertOHHSpecToHand(specFormat.OHH, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to convert OHH spec: %w", err)
	}

	hands := []Hand{hand}

	// Check if this is a spectator log (no hero cards)
	if isSpectatorLog(hands) {
		return nil, ErrSpectatorLog
	}

	// Convert to HH format
	hh := convertHandsToHH(hands, opts)

	return &ConvertResult{
		HH:           []byte(hh),
		SkippedHands: 0,
	}, nil
}

// convertOHHSpecToHand converts an OHH spec to internal Hand format
func convertOHHSpecToHand(spec OHHSpec, opts ConvertOptions) (Hand, error) {
	// Check bet_type - only NL is supported
	if spec.BetLimit.BetType != "NL" {
		return Hand{}, fmt.Errorf("unsupported bet type: %s (only NL is supported)", spec.BetLimit.BetType)
	}

	// Create player map for quick lookup
	playerMap := make(map[int]*OHHSpecPlayer)
	for i := range spec.Players {
		playerMap[spec.Players[i].ID] = &spec.Players[i]
	}

	// Convert players
	players := make([]Player, 0, len(spec.Players))
	for _, p := range spec.Players {
		players = append(players, Player{
			SeatNumber:  p.Seat,
			Name:        p.Name,
			DisplayName: p.Name,
			Stack:       p.StartingStack,
		})
	}

	// Find dealer name
	dealerName := ""
	for _, p := range players {
		if p.SeatNumber == spec.DealerSeat {
			dealerName = p.DisplayName
			break
		}
	}

	// Get hero cards
	var heroCards []string
	if spec.HeroPlayerID > 0 {
		if heroPlayer, ok := playerMap[spec.HeroPlayerID]; ok {
			heroCards = heroPlayer.Cards
		}
	}

	// Build board from rounds
	var board Board
	for _, round := range spec.Rounds {
		street := strings.ToLower(round.Street)
		switch street {
		case "flop":
			if len(round.Cards) >= 3 {
				board.Flop = round.Cards[:3]
			}
		case "turn":
			if len(round.Cards) > 0 {
				board.Turn = round.Cards[0]
			}
		case "river":
			if len(round.Cards) > 0 {
				board.River = round.Cards[0]
			}
		}
	}

	// Convert actions from rounds
	var actions []Action
	for _, round := range spec.Rounds {
		street := convertOHHStreet(round.Street)
		for _, a := range round.Actions {
			player, ok := playerMap[a.PlayerID]
			if !ok {
				continue
			}

			actionType, err := convertOHHActionType(a.Action)
			if err != nil {
				return Hand{}, fmt.Errorf("in round %q action #%d: %w", round.Street, a.ActionNumber, err)
			}

			action := Action{
				Player:     player.Name,
				ActionType: actionType,
				Amount:     a.Amount,
				Street:     street,
				IsAllIn:    a.IsAllIn,
			}
			actions = append(actions, action)
		}
	}

	// Convert pots to winners
	var winners []Winner
	for _, pot := range spec.Pots {
		for _, win := range pot.PlayerWins {
			player, ok := playerMap[win.PlayerID]
			if !ok {
				continue
			}

			// Find player's cards if they won
			var handCards []string
			if len(player.Cards) > 0 {
				handCards = player.Cards
			}

			winners = append(winners, Winner{
				Player:    player.Name,
				Amount:    win.WinAmount,
				HandCards: handCards,
			})
		}
	}

	// Generate hand ID from spec ID or game number
	handID := spec.GameNumber
	if handID == "" {
		handID = "1"
	}

	return Hand{
		HandNumber: handID,
		HandID:     handID,
		Dealer:     dealerName,
		Players:    players,
		Actions:    actions,
		Board:      board,
		StartTime:  spec.StartDateUTC,
		SmallBlind: spec.SmallBlindAmount,
		BigBlind:   spec.BigBlindAmount,
		Ante:       spec.AnteAmount,
		Winners:    winners,
		HeroCards:  heroCards,
		TableName:  spec.TableName,
		SiteName:   spec.SiteName,
	}, nil
}

// convertOHHHandToHand converts an OHH hand to internal Hand format
func convertOHHHandToHand(ohhHand OHHHand) (Hand, error) {
	// Convert players
	players := make([]Player, 0, len(ohhHand.Players))
	for _, p := range ohhHand.Players {
		players = append(players, Player{
			SeatNumber:  p.SeatNumber,
			Name:        p.Name,
			DisplayName: p.Name,
			Stack:       p.Stack,
		})
	}

	// Find dealer name
	dealerName := ""
	for _, p := range players {
		if p.SeatNumber == ohhHand.Dealer.SeatNumber {
			dealerName = p.DisplayName
			break
		}
	}

	// Convert board
	board := Board{
		Flop:  ohhHand.Board.Flop,
		Turn:  ohhHand.Board.Turn,
		River: ohhHand.Board.River,
	}

	// Convert actions
	actions := make([]Action, 0, len(ohhHand.Actions))
	for _, a := range ohhHand.Actions {
		actionType, err := convertOHHActionType(a.ActionType)
		if err != nil {
			return Hand{}, fmt.Errorf("in hand %s action for player %q: %w", ohhHand.HandID, a.Player, err)
		}
		action := Action{
			Player:     a.Player,
			ActionType: actionType,
			Amount:     a.Amount,
			Street:     convertOHHStreet(a.Street),
			IsAllIn:    a.IsAllIn,
		}
		actions = append(actions, action)
	}

	// Convert winners
	winners := make([]Winner, 0, len(ohhHand.Winners))
	for _, w := range ohhHand.Winners {
		winners = append(winners, Winner{
			Player:    w.Player,
			Amount:    w.Amount,
			HandCards: w.HandCards,
		})
	}

	return Hand{
		HandNumber: ohhHand.HandNumber,
		HandID:     ohhHand.HandID,
		Dealer:     dealerName,
		Players:    players,
		Actions:    actions,
		Board:      board,
		StartTime:  ohhHand.StartTime,
		SmallBlind: ohhHand.Blinds.SmallBlind,
		BigBlind:   ohhHand.Blinds.BigBlind,
		Ante:       ohhHand.Ante,
		Winners:    winners,
		HeroCards:  ohhHand.HeroCards,
	}, nil
}

// convertOHHActionType converts OHH action type string to ActionType.
// Handles both simplified format (e.g. "postSB") and spec format (e.g. "Post SB")
// by normalizing to lowercase before matching.
// Returns an error for unknown action types to prevent silent data corruption.
func convertOHHActionType(actionType string) (ActionType, error) {
	action := strings.ToLower(actionType)

	switch action {
	case "fold":
		return ActionFold, nil
	case "check":
		return ActionCheck, nil
	case "call":
		return ActionCall, nil
	case "bet":
		return ActionBet, nil
	case "raise":
		return ActionRaise, nil
	case "post sb", "postsb":
		return ActionPostSB, nil
	case "post bb", "postbb":
		return ActionPostBB, nil
	case "post ante", "postante":
		return ActionPostAnte, nil
	case "show":
		return ActionShow, nil
	case "collect":
		return ActionCollect, nil
	case "uncalled":
		return ActionUncalled, nil
	default:
		return ActionFold, fmt.Errorf("unknown OHH action type: %q", actionType)
	}
}

// convertOHHStreet converts OHH street string to Street.
// Handles both simplified format (e.g. "preflop") and spec format (e.g. "Preflop")
// by normalizing to lowercase before matching.
func convertOHHStreet(street string) Street {
	s := strings.ToLower(street)

	switch s {
	case "preflop":
		return StreetPreflop
	case "flop":
		return StreetFlop
	case "turn":
		return StreetTurn
	case "river":
		return StreetRiver
	case "showdown":
		return StreetShowdown
	default:
		return StreetPreflop // Default to preflop
	}
}

// isSpectatorLog checks if the given hands represent a spectator log
// (i.e., no hand contains hero cards).
func isSpectatorLog(hands []Hand) bool {
	if len(hands) == 0 {
		return false
	}
	for _, hand := range hands {
		if len(hand.HeroCards) > 0 {
			return false
		}
	}
	return true
}

// ReadJSONL reads JSONL (JSON Lines) format with multiple OHH spec hands
// Each line should contain a complete OHH spec format JSON object
func ReadJSONL(r io.Reader, opts ConvertOptions) (*ConvertResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSONL: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var allHands []Hand
	totalSkipped := 0

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to parse each line as OHH spec format
		var formatCheck map[string]interface{}
		if err := json.Unmarshal([]byte(line), &formatCheck); err != nil {
			// Skip invalid JSON lines
			totalSkipped++
			continue
		}

		// Check if this is the official OHH spec format (has "ohh" field)
		if _, hasOHH := formatCheck["ohh"]; hasOHH {
			var specFormat OHHSpecFormat
			if err := json.Unmarshal([]byte(line), &specFormat); err != nil {
				totalSkipped++
				continue
			}

			// Set time location from first hand
			if opts.TimeLocation == nil {
				opts.TimeLocation = specFormat.OHH.StartDateUTC.Location()
			}

			// Set site name from OHH spec if not already specified
			if opts.SiteName == "" {
				if specFormat.OHH.SiteName != "" {
					opts.SiteName = specFormat.OHH.SiteName
				} else {
					opts.SiteName = "PokerStars"
				}
			}

			hand, err := convertOHHSpecToHand(specFormat.OHH, opts)
			if err != nil {
				totalSkipped++
				continue
			}

			// Check if this hand has hero cards
			if len(hand.HeroCards) == 0 {
				// Skip spectator hands in JSONL
				totalSkipped++
				continue
			}

			allHands = append(allHands, hand)
		} else {
			// Try simplified format
			var ohhHand OHHHand
			if err := json.Unmarshal([]byte(line), &ohhHand); err != nil {
				totalSkipped++
				continue
			}

			if opts.TimeLocation == nil {
				opts.TimeLocation = ohhHand.StartTime.Location()
			}

			hand, err := convertOHHHandToHand(ohhHand)
			if err != nil {
				totalSkipped++
				continue
			}

			// Check if this hand has hero cards
			if len(hand.HeroCards) == 0 {
				totalSkipped++
				continue
			}

			allHands = append(allHands, hand)
		}

		// Limit the number of lines processed to avoid excessive memory usage
		if lineNum > maxJSONLLines {
			break
		}
	}

	if len(allHands) == 0 {
		return nil, ErrSpectatorLog
	}

	// Convert to HH format
	hh := convertHandsToHH(allHands, opts)

	return &ConvertResult{
		HH:           []byte(hh),
		SkippedHands: totalSkipped,
	}, nil
}
