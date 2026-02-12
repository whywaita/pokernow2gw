package pokernow2gw

import (
	"testing"
)

func TestIsChipsCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     bool
	}{
		{
			name:     "Chips (capital C)",
			currency: "Chips",
			want:     true,
		},
		{
			name:     "chips (lowercase)",
			currency: "chips",
			want:     true,
		},
		{
			name:     "CHIPS (uppercase)",
			currency: "CHIPS",
			want:     true,
		},
		{
			name:     "USD",
			currency: "USD",
			want:     false,
		},
		{
			name:     "empty string",
			currency: "",
			want:     false,
		},
		{
			name:     "EUR",
			currency: "EUR",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isChipsCurrency(tt.currency)
			if got != tt.want {
				t.Errorf("isChipsCurrency(%q) = %v, want %v", tt.currency, got, tt.want)
			}
		})
	}
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		opts     ConvertOptions
		currency string
		want     string
	}{
		{
			name:     "Cash game with USD",
			amount:   100.0,
			opts:     ConvertOptions{GameType: GameTypeCash},
			currency: "USD",
			want:     "$100",
		},
		{
			name:     "Cash game with Chips",
			amount:   100.0,
			opts:     ConvertOptions{GameType: GameTypeCash},
			currency: "Chips",
			want:     "100",
		},
		{
			name:     "Cash game with empty currency (default USD)",
			amount:   100.0,
			opts:     ConvertOptions{GameType: GameTypeCash},
			currency: "",
			want:     "$100",
		},
		{
			name:     "Tournament game (currency ignored)",
			amount:   100.0,
			opts:     ConvertOptions{GameType: GameTypeTournament},
			currency: "USD",
			want:     "100",
		},
		{
			name:     "Tournament game with Chips (currency ignored)",
			amount:   100.0,
			opts:     ConvertOptions{GameType: GameTypeTournament},
			currency: "Chips",
			want:     "100",
		},
		{
			name:     "Cash game with decimal amount and USD",
			amount:   0.50,
			opts:     ConvertOptions{GameType: GameTypeCash},
			currency: "USD",
			want:     "$0.50",
		},
		{
			name:     "Cash game with decimal amount and Chips",
			amount:   0.50,
			opts:     ConvertOptions{GameType: GameTypeCash},
			currency: "Chips",
			want:     "0.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAmount(tt.amount, tt.opts, tt.currency)
			if got != tt.want {
				t.Errorf("formatAmount(%v, %+v, %q) = %q, want %q", tt.amount, tt.opts, tt.currency, got, tt.want)
			}
		})
	}
}
