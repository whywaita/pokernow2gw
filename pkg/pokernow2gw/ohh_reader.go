package pokernow2gw

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
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
	if len(hands) > 0 {
		hasAnyHeroCards := false
		for _, hand := range hands {
			if len(hand.HeroCards) > 0 {
				hasAnyHeroCards = true
				break
			}
		}
		if !hasAnyHeroCards {
			return nil, ErrSpectatorLog
		}
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
	if len(hand.HeroCards) == 0 {
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
			Stack:       int(p.StartingStack),
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
		street := convertOHHSpecStreet(round.Street)
		for _, a := range round.Actions {
			player, ok := playerMap[a.PlayerID]
			if !ok {
				continue
			}

			action := Action{
				Player:     player.Name,
				ActionType: convertOHHSpecActionType(a.Action),
				Amount:     int(a.Amount),
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
			if player.Cards != nil && len(player.Cards) > 0 {
				handCards = player.Cards
			}

			winners = append(winners, Winner{
				Player:    player.Name,
				Amount:    int(win.WinAmount),
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
		SmallBlind: int(spec.SmallBlindAmount),
		BigBlind:   int(spec.BigBlindAmount),
		Ante:       int(spec.AnteAmount),
		Winners:    winners,
		HeroCards:  heroCards,
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
		action := Action{
			Player:     a.Player,
			ActionType: convertOHHActionType(a.ActionType),
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

// convertOHHSpecActionType converts OHH spec action type string to ActionType
func convertOHHSpecActionType(actionType string) ActionType {
	// Normalize to lowercase for comparison
	action := strings.ToLower(actionType)
	
	switch action {
	case "fold":
		return ActionFold
	case "check":
		return ActionCheck
	case "call":
		return ActionCall
	case "bet":
		return ActionBet
	case "raise":
		return ActionRaise
	case "post sb", "postsb":
		return ActionPostSB
	case "post bb", "postbb":
		return ActionPostBB
	case "post ante", "postante":
		return ActionPostAnte
	case "show":
		return ActionShow
	case "collect":
		return ActionCollect
	case "uncalled":
		return ActionUncalled
	default:
		return ActionFold // Default to fold for unknown actions
	}
}

// convertOHHSpecStreet converts OHH spec street string to Street
func convertOHHSpecStreet(street string) Street {
	// Normalize to lowercase for comparison
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

// convertOHHActionType converts OHH action type string to ActionType
func convertOHHActionType(actionType string) ActionType {
	switch actionType {
	case "fold":
		return ActionFold
	case "check":
		return ActionCheck
	case "call":
		return ActionCall
	case "bet":
		return ActionBet
	case "raise":
		return ActionRaise
	case "postSB":
		return ActionPostSB
	case "postBB":
		return ActionPostBB
	case "postAnte":
		return ActionPostAnte
	case "show":
		return ActionShow
	case "collect":
		return ActionCollect
	case "uncalled":
		return ActionUncalled
	default:
		return ActionFold // Default to fold for unknown actions
	}
}

// convertOHHStreet converts OHH street string to Street
func convertOHHStreet(street string) Street {
	switch street {
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
