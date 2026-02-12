package pokernow2gw

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

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

// extractDisplayName extracts display name from PokerNow player string
// Example: "whywaita @ DtjzvbAuKs" -> "whywaita"
// Example: "spa @ ces @ ZQfm6ZDMPO" -> "spa@ces"
// Example: "@atsymbols@@@ @ 3mid0aO0hZ" -> "@atsymbols@@@"
// Example: """quotes"""' @ us2R6psQVF" -> "quotes"'"
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

// normalizePlayerSeats renumbers player seats from 1 to N
// This ensures compatibility with GTO Wizard which doesn't recognize button at seat 10
func normalizePlayerSeats(players []Player) []Player {
	if len(players) == 0 {
		return players
	}

	// Sort players by original seat number
	sortedPlayers := make([]Player, len(players))
	copy(sortedPlayers, players)

	// Sort players by seat number
	sort.Slice(sortedPlayers, func(i, j int) bool {
		return sortedPlayers[i].SeatNumber < sortedPlayers[j].SeatNumber
	})

	// Renumber seats from 1 to N
	for i := range sortedPlayers {
		sortedPlayers[i].SeatNumber = i + 1
	}

	return sortedPlayers
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

// calculateTotalPot calculates the total pot
func calculateTotalPot(hand Hand) float64 {
	total := 0.0
	for _, winner := range hand.Winners {
		total += winner.Amount
	}
	// Uncalled bets should NOT be added to the pot
	// because they were returned to the player
	return total
}

// calculateRake calculates the rake for a hand based on rake settings
func calculateRake(totalPot float64, bigBlind float64, rakePercent float64, rakeCapBB float64) float64 {
	if rakePercent <= 0 {
		return 0
	}

	// Calculate rake as percentage of pot
	rakeFromPercent := totalPot * rakePercent / 100.0

	// Calculate rake cap in chips
	rakeCap := bigBlind * rakeCapBB

	// Take minimum of calculated rake and cap
	rake := rakeFromPercent
	if rakeCap > 0 && rake > rakeCap {
		rake = rakeCap
	}

	return rake
}
