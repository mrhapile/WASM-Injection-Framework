#!/bin/bash
# build_corpus.sh
# Compiles all .wat files to .wasm binaries using wat2wasm (from wabt)
#
# Prerequisites:
#   brew install wabt   # or apt-get install wabt

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CORPUS_DIR="$SCRIPT_DIR/corpus"

echo "Building WASM corpus from WAT files..."

# Check for wat2wasm
if ! command -v wat2wasm &> /dev/null; then
    echo "Error: wat2wasm not found. Install wabt:"
    echo "  macOS:  brew install wabt"
    echo "  Ubuntu: apt-get install wabt"
    exit 1
fi

# Compile each .wat file
for wat_file in "$CORPUS_DIR"/*.wat; do
    if [ -f "$wat_file" ]; then
        base_name=$(basename "$wat_file" .wat)
        wasm_file="$CORPUS_DIR/$base_name.wasm"
        
        # Skip if we already have a hand-crafted binary
        if [ -f "$wasm_file" ]; then
            echo "  [SKIP] $base_name.wasm (already exists)"
            continue
        fi
        
        echo "  [BUILD] $base_name.wat -> $base_name.wasm"
        wat2wasm "$wat_file" -o "$wasm_file"
    fi
done

echo ""
echo "Corpus files:"
ls -la "$CORPUS_DIR"/*.wasm 2>/dev/null || echo "No .wasm files found"

echo ""
echo "Build complete."
