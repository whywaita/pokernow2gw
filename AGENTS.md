# AGENTS.md

This document contains development information for contributors and AI agents.

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

## Local Web Interface

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

4. Paste your PokerNow CSV log or OHH JSON and your hero name, then click "Convert"

## Command Line Interface (CLI)

### Building the CLI

```bash
go build -o pokernow2gw cmd/pokernow2gw/main.go
```

### Input Formats

The tool supports two input formats:
- **PokerNow CSV**: The standard export format from PokerNow
- **OHH JSON**: Open Hand History format (JSON-based)

The format is automatically detected, so you can use the same command for both.

### Running the Converter

```bash
./pokernow2gw -i input.csv -o output.txt --hero-name "YourName"
```

### Options

- `-i, --input`: Input file (CSV or JSON format, or use stdin)
- `-o, --output`: Output file (or use stdout)
- `--hero-name`: Your display name in the game (required)
- `--timezone`: Timezone for output (default: "UTC")
- `--tournament-name`: Tournament name (optional)

### Examples

#### Convert PokerNow CSV
```bash
./pokernow2gw -i input.csv -o output.txt --hero-name "YourName"
```

#### Convert OHH JSON
```bash
./pokernow2gw -i input.json -o output.txt --hero-name "YourName"
```

#### Using stdin/stdout
```bash
cat input.csv | ./pokernow2gw --hero-name "YourName" > output.txt
cat input.json | ./pokernow2gw --hero-name "YourName" > output.txt
```

### OHH Format Documentation

For detailed information about the OHH JSON format, see [docs/OHH_FORMAT.md](./docs/OHH_FORMAT.md).

Sample OHH files are available in:
- `sample/input/sample_ohh.json` - Basic hand example
- `sample/input/sample_ohh_showdown.json` - Complete hand with showdown
```
