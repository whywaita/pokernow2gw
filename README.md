# pokernow2gw

Convert PokerNow CSV logs to GTO Wizard Hand History format (MTT).

## Notice

This software is not intended to promote or direct users to any online casino sites or applications. It utilizes Poker Now, a completely free platform, solely for the purpose of studying and improving gameplay.

## Motivation

[PokerNow](https://www.pokernow.club/) is an excellent free platform for playing poker with friends, including MTT (Multi-Table Tournament) games. However, when you want to analyze your gameplay using professional poker analysis tools like [GTO Wizard](https://www.gtowizard.com/), [PokerTracker](https://www.pokertracker.com/), or [Hold'em Manager](https://www.holdemmanager.com/), you'll encounter a compatibility issue.

## Features

- Command-line interface for batch conversion
- Web interface (WASM) for browser-based conversion using Go 1.24+ `go:wasmexport`
- Supports standard PokerNow CSV log format
- Outputs GTO Wizard-compatible Hand History format

## Usage

### Web Interface (Recommended for quick conversions)

#### Online Version (GitHub Pages)

The easiest way to use pokernow2gw is through our online version:

**https://whywaita.github.io/pokernow2gw/**

No installation required - just open the link and start converting!

#### Local Version

**Requirements:** Go 1.24+ (uses `go:wasmexport` feature)

1. Build the WASM module:
   ```bash
   ./build-wasm.sh
   ```

2. Start a web server:
   ```bash
   cd web && python3 -m http.server 8080
   ```

3. Open http://localhost:8080 in your browser

4. Paste your PokerNow CSV log and your hero name, then click "Convert"

### Command Line

1. Build the CLI:
   ```bash
   go build -o pokernow2gw cmd/pokernow2gw/main.go
   ```

2. Run the converter:
   ```bash
   ./pokernow2gw -i input.csv -o output.txt --hero-name "YourName"
   ```

   Options:
   - `-i, --input`: Input CSV file (or use stdin)
   - `-o, --output`: Output file (or use stdout)
   - `--hero-name`: Your display name in the game (required)
   - `--timezone`: Timezone for output (default: "UTC")
   - `--tournament-name`: Tournament name (optional)

   Example with stdin/stdout:
   ```bash
   cat input.csv | ./pokernow2gw --hero-name "YourName" > output.txt
   ```

## Important Notes

### Ante Handling

When converting hands where all players post antes (all-in ante format), this tool automatically converts them to Small Blind (SB) and Big Blind (BB) format. This is because GTO Wizard and other analysis tools do not support the all-ante format in their hand history parsers.

**Why this conversion happens:**
- PokerNow supports all-ante tournament structures
- GTO Wizard and similar tools only support SB/BB tournament structures
- To maintain compatibility with analysis tools, antes are automatically converted to blinds

**Example:**
- PokerNow log: All players post 100 ante
- Converted output: Player posts 50 SB, Player posts 100 BB

This conversion allows you to analyze hands in GTO Wizard and other tools, even though the exact blind structure differs from the original game.

## Project Structure

```
pokernow2gw/
├── cmd/
│   ├── pokernow2gw/    # CLI application
│   └── wasm/           # WASM application
├── pkg/
│   └── pokernow2gw/    # Core library
├── web/                # Web interface
│   ├── index.html
│   ├── wasm_exec.js (copied from Go)
│   └── pokernow2gw.wasm (generated)
├── sample/             # Sample data
│   ├── input/
│   └── output/
└── build-wasm.sh       # WASM build script
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

#### CLI Binary
```bash
go build -o pokernow2gw cmd/pokernow2gw/main.go
```

#### WASM Module
```bash
./build-wasm.sh
```
