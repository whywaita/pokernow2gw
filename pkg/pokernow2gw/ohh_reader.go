package pokernow2gw

import (
	"encoding/json"
	"fmt"
	"io"
)

// ReadOHH reads Open Hand History JSON from reader and converts to internal Hand format
func ReadOHH(r io.Reader, opts ConvertOptions) (*ConvertResult, error) {
	var ohhFormat OHHFormat
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&ohhFormat); err != nil {
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
