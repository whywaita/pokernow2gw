package pokernow2gw

import (
	"io"
)

// ParseCSV reads PokerNow CSV from reader and converts to GTO Wizard HH format
func ParseCSV(r io.Reader, opts ConvertOptions) (*ConvertResult, error) {
	// Read CSV
	entries, err := ReadCSV(r)
	if err != nil {
		return nil, err
	}

	// Convert entries
	return ConvertEntries(entries, opts)
}
