# WASM Fuzzing Corpus

This directory contains deterministic test cases for the WebAssembly fuzzing harness.

## Test Case Summary

| File | Stage | Description | Runtime Property Tested |
|------|-------|-------------|------------------------|
| `01_valid_module.wasm` | **NONE** (success) | Valid module with correct ABI | Full pipeline works end-to-end |
| `02_empty_file.wasm` | **LOAD** | Empty file (0 bytes) | Magic number / header parsing |
| `03_truncated_binary.wasm` | **LOAD** | Incomplete WASM header | Binary reader EOF handling |
| `04_missing_export.wasm` | **EXECUTE** | No "process" export | Export table lookup |
| `05_abi_mismatch_no_param.wasm` | **EXECUTE** | process() has no parameters | Function signature validation |
| `06_abi_mismatch_wrong_type.wasm` | **EXECUTE** | process(i64) wrong type | Type checking at call time |
| `07_abi_mismatch_no_return.wasm` | **EXECUTE** | process(i32) returns void | Return type handling |
| `08_invalid_opcode.wasm` | **VALIDATE** | Contains reserved opcode 0xFE | Opcode validation |
| `09_unreachable_trap.wasm` | **EXECUTE** | Hits `unreachable` instruction | Trap handling |
| `10_memory_oob.wasm` | **EXECUTE** | Out-of-bounds memory access | Memory bounds checking |

## Detailed Test Cases

### 01_valid_module.wasm — Success Case
```wat
(func $process (export "process") (param i32) (result i32)
  local.get 0
  i32.const 1
  i32.add)
```
**Expected:** Success with return value `2` (input 1 + 1)

---

### 02_empty_file.wasm — Load Failure
**Binary:** 0 bytes  
**Expected:** Load failure — missing WASM magic number `\0asm`

**Tests:** Loader rejects files without valid magic header

---

### 03_truncated_binary.wasm — Load Failure  
**Binary:** `00 61 73 6d 01 00 00` (7 bytes — missing last version byte)  
**Expected:** Load failure — incomplete header

**Tests:** Loader handles unexpected EOF during header parsing

---

### 04_missing_export.wasm — Execute Failure
```wat
(func $compute (export "compute") (param i32) (result i32) ...)
```
**Expected:** Execute failure — `FindFunction("process")` returns nil

**Tests:** Export resolution correctly reports missing exports

---

### 05_abi_mismatch_no_param.wasm — Execute Failure
```wat
(func $process (export "process") (result i32)
  i32.const 42)
```
**Expected:** Execute failure — arity mismatch when calling with 1 argument

**Tests:** Runtime validates parameter count before invocation

---

### 06_abi_mismatch_wrong_type.wasm — Execute Failure
```wat
(func $process (export "process") (param i64) (result i64) ...)
```
**Expected:** Execute failure — type mismatch (i32 passed, i64 expected)

**Tests:** Runtime validates parameter types

---

### 07_abi_mismatch_no_return.wasm — Execute Failure
```wat
(func $process (export "process") (param i32)
  drop)
```
**Expected:** Execute failure or empty return — function returns void

**Tests:** Return value handling when function has no result

---

### 08_invalid_opcode.wasm — Validate Failure
**Binary:** Contains reserved opcode `0xFE` in function body  
**Expected:** Validation failure — unknown/reserved opcode

**Tests:** Validator rejects invalid instruction encodings

---

### 09_unreachable_trap.wasm — Execute Failure (Trap)
```wat
(func $process (export "process") (param i32) (result i32)
  unreachable)
```
**Expected:** Execute failure — runtime trap (unreachable executed)

**Tests:** Trap propagation and error reporting

---

### 10_memory_oob.wasm — Execute Failure (Trap)
```wat
(memory 1)
(func $process (export "process") (param i32) (result i32)
  i32.const 0xFFFFFF
  i32.load)
```
**Expected:** Execute failure — out-of-bounds memory access trap

**Tests:** Memory bounds validation at runtime

---

## Building the Corpus

```bash
# Install wabt (WebAssembly Binary Toolkit)
brew install wabt  # macOS
# apt-get install wabt  # Ubuntu

# Build all WAT files to WASM
./build_corpus.sh
```

## Failure Stage Coverage

| Stage | Test Cases |
|-------|------------|
| LOAD | 02, 03 |
| VALIDATE | 08 |
| INSTANTIATE | (none in this corpus) |
| EXECUTE | 04, 05, 06, 07, 09, 10 |
| SUCCESS | 01 |

## Determinism Guarantees

- **No randomness:** All inputs are static binary files
- **No timestamps:** File content is fixed
- **No external dependencies:** Self-contained binaries
- **Reproducible:** Same input → same failure stage every time
