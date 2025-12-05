package pokernow2gw

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"
)

// ReadCSV reads PokerNow CSV log from reader and returns LogEntry slice
func ReadCSV(r io.Reader) ([]LogEntry, error) {
	csvReader := csv.NewReader(r)

	// Read header
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Validate header
	if len(header) != 3 || header[0] != "entry" || header[1] != "at" || header[2] != "order" {
		return nil, fmt.Errorf("invalid CSV header format: expected [entry,at,order], got %v", header)
	}

	var entries []LogEntry

	// Read all rows
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}

		if len(record) != 3 {
			return nil, fmt.Errorf("invalid CSV row format: expected 3 columns, got %d", len(record))
		}

		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339, record[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp %q: %w", record[1], err)
		}

		// Parse order
		order, err := strconv.ParseInt(record[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse order %q: %w", record[2], err)
		}

		entries = append(entries, LogEntry{
			Entry: record[0],
			At:    timestamp,
			Order: order,
		})
	}

	// PokerNowのCSVは逆時系列なので、昇順（古い→新しい）にソート
	// orderフィールドは昇順で処理する必要があるため、反転させる
	reverseEntries(entries)

	return entries, nil
}

// reverseEntries reverses the slice in-place
func reverseEntries(entries []LogEntry) {
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
}
