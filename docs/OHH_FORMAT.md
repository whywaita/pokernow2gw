# OHH (Open Hand History) Format

This document describes the OHH (Open Hand History) JSON format supported by pokernow2gw.

## Overview

OHH is a JSON-based format for representing poker hand histories. It provides a structured, machine-readable way to store and exchange poker hand data.

## Format Specification

### Root Object

```json
{
  "version": "1.0",
  "hands": [...]
}
```

- `version` (string): The OHH format version
- `hands` (array): An array of hand objects

### Hand Object

Each hand object represents a single poker hand:

```json
{
  "handId": "1234567890",
  "handNumber": "1",
  "gameType": "No Limit Texas Hold'em",
  "tableName": "Test Table",
  "startTime": "2025-11-15T05:08:29.500Z",
  "blinds": {
    "smallBlind": 50,
    "bigBlind": 100
  },
  "ante": 10,
  "players": [...],
  "dealer": {"seatNumber": 1},
  "heroCards": ["Ah", "Kh"],
  "board": {...},
  "actions": [...],
  "winners": [...]
}
```

#### Hand Properties

- `handId` (string): Unique identifier for the hand
- `handNumber` (string): Sequential hand number
- `gameType` (string): Type of poker game (e.g., "No Limit Texas Hold'em")
- `tableName` (string): Name of the table
- `startTime` (string): ISO 8601 timestamp of when the hand started
- `blinds` (object): Blind structure
  - `smallBlind` (number): Small blind amount
  - `bigBlind` (number): Big blind amount
- `ante` (number, optional): Ante amount
- `players` (array): List of players at the table
- `dealer` (object): Reference to the dealer position
- `heroCards` (array, optional): The hero's hole cards
- `board` (object, optional): Community cards
- `actions` (array): Sequence of actions in the hand
- `winners` (array, optional): List of pot winners

### Player Object

```json
{
  "seatNumber": 1,
  "name": "Player1",
  "stack": 1000
}
```

- `seatNumber` (number): Seat position at the table
- `name` (string): Player's display name
- `stack` (number): Player's chip stack at the start of the hand

### Dealer Reference

```json
{
  "seatNumber": 1
}
```

- `seatNumber` (number): Seat number of the dealer

### Board Object

```json
{
  "flop": ["Qh", "Jh", "Th"],
  "turn": "2d",
  "river": "3c"
}
```

- `flop` (array, optional): Three flop cards
- `turn` (string, optional): Turn card
- `river` (string, optional): River card

### Action Object

```json
{
  "player": "Player1",
  "actionType": "bet",
  "amount": 400,
  "street": "flop",
  "isAllIn": false
}
```

- `player` (string): Name of the player taking the action
- `actionType` (string): Type of action
  - `fold`: Player folds
  - `check`: Player checks
  - `call`: Player calls
  - `bet`: Player bets
  - `raise`: Player raises
  - `postSB`: Posts small blind
  - `postBB`: Posts big blind
  - `postAnte`: Posts ante
  - `show`: Shows cards
  - `collect`: Collects pot
  - `uncalled`: Uncalled bet returned
- `amount` (number, optional): Amount of chips (for bet, call, raise actions)
- `street` (string): Betting round
  - `preflop`: Before the flop
  - `flop`: Flop betting round
  - `turn`: Turn betting round
  - `river`: River betting round
  - `showdown`: Showdown phase
- `isAllIn` (boolean, optional): Whether the action was all-in

### Winner Object

```json
{
  "player": "Player1",
  "amount": 1420,
  "handCards": ["Ah", "Kh"]
}
```

- `player` (string): Name of the winning player
- `amount` (number): Amount won
- `handCards` (array, optional): Cards shown at showdown

## Card Notation

Cards are represented using standard poker notation:
- Ranks: `A`, `K`, `Q`, `J`, `T` (10), `9`, `8`, `7`, `6`, `5`, `4`, `3`, `2`
- Suits: `h` (hearts), `d` (diamonds), `c` (clubs), `s` (spades)
- Examples: `Ah` (Ace of Hearts), `Td` (Ten of Diamonds), `7s` (Seven of Spades)

## Example

See `sample/input/sample_ohh.json` for a complete example of an OHH format file.

## Usage

The pokernow2gw tool automatically detects the input format (CSV or OHH JSON) and converts it to GTO Wizard format:

```bash
./pokernow2gw -i input.json --hero-name "YourName" -o output.txt
```

Or via stdin:

```bash
cat input.json | ./pokernow2gw --hero-name "YourName" > output.txt
```
