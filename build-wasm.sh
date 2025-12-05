#!/bin/bash

set -e

echo "Building WASM module with go:wasmexport..."

# Build WASM using go:wasmexport (Go 1.24+)
GOOS=js GOARCH=wasm go build -o web/pokernow2gw.wasm cmd/wasm/wasm.go

# Copy wasm_exec.js (still needed for Go runtime support)
echo "Copying wasm_exec.js..."
GOROOT=$(go env GOROOT)
if [ -f "$GOROOT/lib/wasm/wasm_exec.js" ]; then
    cp "$GOROOT/lib/wasm/wasm_exec.js" web/
elif [ -f "$GOROOT/misc/wasm/wasm_exec.js" ]; then
    cp "$GOROOT/misc/wasm/wasm_exec.js" web/
else
    echo "Error: wasm_exec.js not found!"
    exit 1
fi

echo "Build complete!"
echo ""
echo "Note: This build uses Go's go:wasmexport feature (Go 1.24+)"
echo "      wasm_exec.js is still needed for Go runtime support."
echo "      Exported functions can be called directly from JavaScript."
echo ""
echo "To run the web server:"
echo "  cd web && python3 -m http.server 8080"
echo ""
echo "Then open http://localhost:8080 in your browser"
