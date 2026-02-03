# WASM-Injection-Framework

A deterministic WebAssembly runtime fuzzing harness built with Go and WasmEdge.

## Features

- **Deterministic execution** — No concurrency, no randomness
- **Structured error classification** — Failures categorized by stage (load, validate, instantiate, execute)
- **Panic-safe** — All errors are recovered and reported
- **JSON output** — Structured results for easy parsing and analysis

## Requirements

- Go 1.21+
- [WasmEdge](https://wasmedge.org/) runtime installed
- WasmEdge Go SDK

## Installation

### Install WasmEdge

```bash
curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install.sh | bash
source ~/.wasmedge/env
```

### Build the fuzzer

```bash
go mod tidy
go build -o wasm-fuzzer .
```

## Usage

```bash
./wasm-fuzzer <directory-with-wasm-files>
```

### Example

```bash
./wasm-fuzzer ./testcases
```

## Output Format

The fuzzer outputs structured JSON to stdout:

```json
{
  "total_files": 3,
  "passed": 1,
  "failed": 2,
  "results": [
    {
      "file_path": "./testcases/valid.wasm",
      "file_name": "valid.wasm",
      "success": true,
      "failure_stage": "none",
      "return_values": [42]
    },
    {
      "file_path": "./testcases/invalid.wasm",
      "file_name": "invalid.wasm",
      "success": false,
      "failure_stage": "validate",
      "error_message": "validation failed: invalid module"
    }
  ],
  "failure_counts": {
    "load": 0,
    "validate": 1,
    "instantiate": 0,
    "execute": 1
  }
}
```

## Failure Stages

| Stage | Description |
|-------|-------------|
| `load` | Failed to read/parse the WASM binary |
| `validate` | WASM module failed validation |
| `instantiate` | Failed to create module instance |
| `execute` | Function "process" not found or execution failed |

## WASM Module Requirements

Your WASM modules should export a function named `process` that accepts an `i32` parameter:

```wat
(module
  (func $process (export "process") (param i32) (result i32)
    local.get 0
    i32.const 1
    i32.add
  )
)
```

## License

MIT
