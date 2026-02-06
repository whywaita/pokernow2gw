package pokernow2gw

import (
	"bytes"
	"encoding/json"
	"io"
)

// Parse reads input (CSV or JSON) from reader and converts to GTO Wizard HH format
// Automatically detects the input format (CSV, OHH JSON, or JSONL)
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

	// Try to detect JSONL format (JSON Lines)
	if isJSONLFormat(data) {
		return ReadJSONL(bytes.NewReader(data), opts)
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

// isJSONLFormat checks if the data is JSONL (JSON Lines) format
func isJSONLFormat(data []byte) bool {
	// Trim whitespace
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}

	// JSONL has multiple lines, each starting with {
	// Check if it starts with { but is not a single valid JSON object
	if trimmed[0] != '{' {
		return false
	}

	// If it's valid as a single JSON object, it's not JSONL
	var temp map[string]interface{}
	if json.Unmarshal(trimmed, &temp) == nil {
		// Check if there's more content after the first JSON object
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		decoder.Decode(&temp)
		// If there's more to read, it's JSONL
		if decoder.More() {
			return true
		}
		return false
	}

	// If it starts with { but isn't valid JSON, check if it's JSONL
	lines := bytes.Split(trimmed, []byte("\n"))
	if len(lines) > 1 {
		// Check if first line is valid JSON
		firstLine := bytes.TrimSpace(lines[0])
		if len(firstLine) > 0 && firstLine[0] == '{' {
			var temp map[string]interface{}
			return json.Unmarshal(firstLine, &temp) == nil
		}
	}

	return false
}
