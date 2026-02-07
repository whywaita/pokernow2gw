package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/whywaita/pokernow2gw/pkg/pokernow2gw"
)

// isStdinPiped checks if stdin is piped (not from terminal)
func isStdinPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

var siteName = "PokerStars"

func main() {
	// Define flags
	input := flag.String("input", "", "Input CSV file (optional, stdin if not specified)")
	inputShort := flag.String("i", "", "Input CSV file (shorthand)")
	output := flag.String("output", "", "Output file (optional, stdout if not specified)")
	outputShort := flag.String("o", "", "Output file (shorthand)")
	heroName := flag.String("hero-name", "", "Hero display name (required)")
	timezone := flag.String("timezone", "UTC", "Timezone for output (e.g., UTC, Asia/Tokyo)")
	tournamentName := flag.String("tournament-name", "", "Tournament name (optional)")
	filterHU := flag.Bool("filter-hu", false, "Include heads-up hands (2 players)")
	filterSpinAndGo := flag.Bool("filter-spinandgo", false, "Include Spin-and-Go hands (3 players)")
	filterMTT := flag.Bool("filter-mtt", false, "Include MTT hands (4-9 players)")
	rakePercent := flag.Float64("rake-percent", 0.0, "Rake percentage for cash games (e.g., 5.0 for 5%)")
	rakeCapBB := flag.Float64("rake-cap-bb", 0.0, "Rake cap in big blinds (e.g., 4.0 for 4BB)")

	flag.Parse()

	// Handle shorthand flags
	if *input == "" && *inputShort != "" {
		*input = *inputShort
	}
	if *output == "" && *outputShort != "" {
		*output = *outputShort
	}

	// Validate required flags
	if *heroName == "" {
		fmt.Fprintf(os.Stderr, "Error: --hero-name is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Parse timezone
	loc, err := time.LoadLocation(*timezone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid timezone %q: %v\n", *timezone, err)
		os.Exit(1)
	}

	// Build player count filter
	var playerCountFilter pokernow2gw.PlayerCountFilter
	if !*filterHU && !*filterSpinAndGo && !*filterMTT {
		// No filters specified, use default (all)
		playerCountFilter = pokernow2gw.PlayerCountAll
	} else {
		// Combine selected filters using bitwise OR
		playerCountFilter = 0
		if *filterHU {
			playerCountFilter |= pokernow2gw.PlayerCountHU
		}
		if *filterSpinAndGo {
			playerCountFilter |= pokernow2gw.PlayerCountSpinAndGo
		}
		if *filterMTT {
			playerCountFilter |= pokernow2gw.PlayerCountMTT
		}
	}

	// Determine input source
	var inputReader io.ReadCloser
	stdinPiped := isStdinPiped()

	if *input != "" {
		// Use file input
		file, err := os.Open(*input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to open input file %q: %v\n", *input, err)
			os.Exit(1)
		}
		inputReader = file
		defer inputReader.Close()
	} else if stdinPiped {
		// Use stdin
		inputReader = io.NopCloser(os.Stdin)
		defer inputReader.Close()
	} else {
		// No input source
		fmt.Fprintf(os.Stderr, "Error: no input specified. Provide --input (-i) or pipe data to stdin\n")
		flag.Usage()
		os.Exit(1)
	}

	// Convert
	opts := pokernow2gw.ConvertOptions{
		HeroName:          *heroName,
		SiteName:          siteName,
		TimeLocation:      loc,
		TournamentName:    *tournamentName,
		PlayerCountFilter: playerCountFilter,
		RakePercent:       *rakePercent,
		RakeCapBB:         *rakeCapBB,
	}

	result, err := pokernow2gw.ParseCSV(inputReader, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: conversion failed: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if *output == "" {
		// Write to stdout
		fmt.Print(string(result.HH))
	} else {
		// Write to file
		err := os.WriteFile(*output, result.HH, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to write output file %q: %v\n", *output, err)
			os.Exit(1)
		}
	}

	// Print skipped hands to stderr
	if result.SkippedHands > 0 {
		fmt.Fprintf(os.Stderr, "%d hands were skipped due to parse errors.\n", result.SkippedHands)
	}
}
