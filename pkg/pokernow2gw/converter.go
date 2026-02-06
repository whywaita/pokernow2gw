package pokernow2gw

import (
	"bytes"
	"encoding/json"
	"io"
)

// Parse reads input (CSV or JSON) from reader and converts to GTO Wizard HH format
// Automatically detects the input format (CSV or OHH JSON)
func Parse(r io.Reader, opts ConvertOptions) (*ConvertResult, error) {
	// Read all data to detect format
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Try to detect JSON format
	if isJSONFormat(data) {
		return ReadOHH(bytes.NewReader(data), opts)
	}

	// Default to CSV format
	return parseCSV(bytes.NewReader(data), opts)
}

// ParseCSV reads PokerNow CSV from reader and converts to GTO Wizard HH format
// Deprecated: Use Parse instead for automatic format detection
func ParseCSV(r io.Reader, opts ConvertOptions) (*ConvertResult, error) {
	return Parse(r, opts)
}

// parseCSV is the internal CSV parser
func parseCSV(r io.Reader, opts ConvertOptions) (*ConvertResult, error) {
	// Read CSV
	entries, err := ReadCSV(r)
	if err != nil {
		return nil, err
	}

	// Convert entries
	return ConvertEntries(entries, opts)
}

// isJSONFormat checks if the data is JSON format
func isJSONFormat(data []byte) bool {
	// Trim whitespace
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}

	// Check if it starts with { (JSON object)
	if trimmed[0] == '{' {
		// Try to decode as JSON to verify
		var temp map[string]interface{}
		return json.Unmarshal(trimmed, &temp) == nil
	}

	return false
}
