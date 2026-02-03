//go:build !integration
// +build !integration

package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// FAULT INJECTION TEST SUITE
// =============================================================================
//
// This file contains deterministic fault injection tests for the WASM fuzzer.
// Each test simulates a specific host-side failure to verify the fuzzer:
//   1. Catches the failure gracefully
//   2. Classifies it to the correct stage
//   3. Does not crash or panic
//
// Fault injection is achieved via gomonkey, which patches functions at runtime.
// All tests are deterministic - no randomness, no timeouts, no OS signals.
// =============================================================================

// -----------------------------------------------------------------------------
// Mock Runtime Implementation
// -----------------------------------------------------------------------------

// MockWasmRuntime is a configurable mock for fault injection
type MockWasmRuntime struct {
	LoadModuleFunc func(filePath string) (WasmModule, error)
}

func (m *MockWasmRuntime) LoadModule(filePath string) (WasmModule, error) {
	if m.LoadModuleFunc != nil {
		return m.LoadModuleFunc(filePath)
	}
	return &MockWasmModule{}, nil
}

// MockWasmModule is a configurable mock module
type MockWasmModule struct {
	ExecuteFunc func(funcName string, args ...interface{}) ([]interface{}, error)
	CloseCalled bool
}

func (m *MockWasmModule) Execute(funcName string, args ...interface{}) ([]interface{}, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(funcName, args...)
	}
	return []interface{}{int32(42)}, nil
}

func (m *MockWasmModule) Close() {
	m.CloseCalled = true
}

// -----------------------------------------------------------------------------
// TEST: Execution Error Injection
// -----------------------------------------------------------------------------
//
// WHY THIS MATTERS:
// The WASM runtime can return errors during function execution for many reasons:
//   - Trap instructions (unreachable, assertion failures)
//   - Division by zero
//   - Integer overflow (with trapping semantics)
//   - Out-of-bounds memory access
//   - Stack overflow
//
// The fuzzer must catch these errors and classify them as EXECUTE failures
// without crashing the host process.
// -----------------------------------------------------------------------------

func TestFaultInjection_ExecutionError(t *testing.T) {
	testCases := []struct {
		name          string
		errorMessage  string
		expectedStage FailureStage
	}{
		{
			name:          "trap_unreachable",
			errorMessage:  "unreachable executed",
			expectedStage: StageExecute,
		},
		{
			name:          "trap_division_by_zero",
			errorMessage:  "integer divide by zero",
			expectedStage: StageExecute,
		},
		{
			name:          "trap_memory_oob",
			errorMessage:  "out of bounds memory access",
			expectedStage: StageExecute,
		},
		{
			name:          "trap_indirect_call_type_mismatch",
			errorMessage:  "indirect call type mismatch",
			expectedStage: StageExecute,
		},
		{
			name:          "trap_stack_overflow",
			errorMessage:  "call stack exhausted",
			expectedStage: StageExecute,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockModule := &MockWasmModule{
				ExecuteFunc: func(funcName string, args ...interface{}) ([]interface{}, error) {
					return nil, errors.New(tc.errorMessage)
				},
			}

			mockRuntime := &MockWasmRuntime{
				LoadModuleFunc: func(filePath string) (WasmModule, error) {
					return mockModule, nil
				},
			}

			result := processWasmFileWithRuntime("/test/injected.wasm", mockRuntime)

			assert.False(t, result.Success, "should report failure")
			assert.Equal(t, tc.expectedStage, result.FailureStage, "should classify as execute failure")
			assert.Contains(t, result.ErrorMessage, tc.errorMessage, "should capture error message")
			assert.True(t, mockModule.CloseCalled, "should call Close to release resources")
		})
	}
}

// -----------------------------------------------------------------------------
// TEST: Runtime Panic Injection
// -----------------------------------------------------------------------------
//
// WHY THIS MATTERS:
// Even well-behaved WASM runtimes can panic under extreme conditions:
//   - Memory allocation failures in the runtime
//   - Internal runtime bugs
//   - CGO boundary violations (WasmEdge is a C library)
//   - Corrupted internal state
//
// A robust fuzzer MUST recover from panics and report them as failures.
// If panics propagate, the entire fuzzing campaign crashes.
// -----------------------------------------------------------------------------

func TestFaultInjection_RuntimePanic(t *testing.T) {
	testCases := []struct {
		name         string
		panicValue   interface{}
		panicInStage string
	}{
		{
			name:         "panic_string_in_execute",
			panicValue:   "runtime internal error: corrupted state",
			panicInStage: "execute",
		},
		{
			name:         "panic_error_in_execute",
			panicValue:   errors.New("allocation failure"),
			panicInStage: "execute",
		},
		{
			name:         "panic_arbitrary_value",
			panicValue:   42,
			panicInStage: "execute",
		},
		{
			name:         "panic_in_load",
			panicValue:   "loader panic: memory corruption",
			panicInStage: "load",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var mockRuntime *MockWasmRuntime

			if tc.panicInStage == "load" {
				mockRuntime = &MockWasmRuntime{
					LoadModuleFunc: func(filePath string) (WasmModule, error) {
						panic(tc.panicValue)
					},
				}
			} else {
				mockModule := &MockWasmModule{
					ExecuteFunc: func(funcName string, args ...interface{}) ([]interface{}, error) {
						panic(tc.panicValue)
					},
				}
				mockRuntime = &MockWasmRuntime{
					LoadModuleFunc: func(filePath string) (WasmModule, error) {
						return mockModule, nil
					},
				}
			}

			result := processWasmFileWithRuntime("/test/panic.wasm", mockRuntime)

			assert.False(t, result.Success, "should report failure")
			assert.Equal(t, StageExecute, result.FailureStage, "panic should be classified as execute failure")
			assert.Contains(t, result.ErrorMessage, "panic recovered", "should indicate panic recovery")
		})
	}
}

// -----------------------------------------------------------------------------
// TEST: Load Stage Error Injection
// -----------------------------------------------------------------------------
//
// WHY THIS MATTERS:
// Load failures occur when the WASM binary is malformed:
//   - Invalid magic number
//   - Truncated file
//   - Corrupted sections
//   - Unsupported version
//
// These should be classified as LOAD failures, not generic errors.
// -----------------------------------------------------------------------------

func TestFaultInjection_LoadError(t *testing.T) {
	testCases := []struct {
		name          string
		injectedError error
		expectedStage FailureStage
	}{
		{
			name:          "invalid_magic_number",
			injectedError: &RuntimeError{Stage: StageLoad, Message: "invalid magic number"},
			expectedStage: StageLoad,
		},
		{
			name:          "unexpected_eof",
			injectedError: &RuntimeError{Stage: StageLoad, Message: "unexpected end of file"},
			expectedStage: StageLoad,
		},
		{
			name:          "unsupported_version",
			injectedError: &RuntimeError{Stage: StageLoad, Message: "unsupported WASM version 2"},
			expectedStage: StageLoad,
		},
		{
			name:          "generic_io_error",
			injectedError: errors.New("read error: permission denied"),
			expectedStage: StageLoad,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRuntime := &MockWasmRuntime{
				LoadModuleFunc: func(filePath string) (WasmModule, error) {
					return nil, tc.injectedError
				},
			}

			result := processWasmFileWithRuntime("/test/malformed.wasm", mockRuntime)

			assert.False(t, result.Success, "should report failure")
			assert.Equal(t, tc.expectedStage, result.FailureStage, "should classify correctly")
		})
	}
}

// -----------------------------------------------------------------------------
// TEST: Validate Stage Error Injection
// -----------------------------------------------------------------------------
//
// WHY THIS MATTERS:
// Validation failures occur when the WASM module structure is invalid:
//   - Type mismatches
//   - Invalid opcodes
//   - Malformed function signatures
//   - Invalid section ordering
//
// Validation happens after successful load but before instantiation.
// -----------------------------------------------------------------------------

func TestFaultInjection_ValidateError(t *testing.T) {
	testCases := []struct {
		name          string
		injectedError error
		expectedStage FailureStage
	}{
		{
			name:          "invalid_opcode",
			injectedError: &RuntimeError{Stage: StageValidate, Message: "unknown opcode 0xFE"},
			expectedStage: StageValidate,
		},
		{
			name:          "type_mismatch",
			injectedError: &RuntimeError{Stage: StageValidate, Message: "type mismatch: expected i32, got i64"},
			expectedStage: StageValidate,
		},
		{
			name:          "invalid_function_index",
			injectedError: &RuntimeError{Stage: StageValidate, Message: "function index out of bounds"},
			expectedStage: StageValidate,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRuntime := &MockWasmRuntime{
				LoadModuleFunc: func(filePath string) (WasmModule, error) {
					return nil, tc.injectedError
				},
			}

			result := processWasmFileWithRuntime("/test/invalid.wasm", mockRuntime)

			assert.False(t, result.Success, "should report failure")
			assert.Equal(t, tc.expectedStage, result.FailureStage, "should classify as validation failure")
		})
	}
}

// -----------------------------------------------------------------------------
// TEST: Instantiate Stage Error Injection
// -----------------------------------------------------------------------------
//
// WHY THIS MATTERS:
// Instantiation failures occur when setting up the module runtime:
//   - Missing imports
//   - Memory allocation limits exceeded
//   - Global initialization failures
//   - Start function traps
//
// These are distinct from execution errors - they happen before any
// user function is called.
// -----------------------------------------------------------------------------

func TestFaultInjection_InstantiateError(t *testing.T) {
	testCases := []struct {
		name          string
		injectedError error
		expectedStage FailureStage
	}{
		{
			name:          "missing_import",
			injectedError: &RuntimeError{Stage: StageInstantiate, Message: "import not found: env.print"},
			expectedStage: StageInstantiate,
		},
		{
			name:          "memory_limit_exceeded",
			injectedError: &RuntimeError{Stage: StageInstantiate, Message: "memory exceeds maximum pages"},
			expectedStage: StageInstantiate,
		},
		{
			name:          "start_function_trap",
			injectedError: &RuntimeError{Stage: StageInstantiate, Message: "start function trapped"},
			expectedStage: StageInstantiate,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRuntime := &MockWasmRuntime{
				LoadModuleFunc: func(filePath string) (WasmModule, error) {
					return nil, tc.injectedError
				},
			}

			result := processWasmFileWithRuntime("/test/instantiate_fail.wasm", mockRuntime)

			assert.False(t, result.Success, "should report failure")
			assert.Equal(t, tc.expectedStage, result.FailureStage, "should classify as instantiate failure")
		})
	}
}

// -----------------------------------------------------------------------------
// TEST: Success Path (No Faults)
// -----------------------------------------------------------------------------
//
// WHY THIS MATTERS:
// We must verify that the testable path works correctly when no faults occur.
// This establishes a baseline for the fault injection tests.
// -----------------------------------------------------------------------------

func TestNoFault_SuccessPath(t *testing.T) {
	expectedReturn := []interface{}{int32(42)}

	mockModule := &MockWasmModule{
		ExecuteFunc: func(funcName string, args ...interface{}) ([]interface{}, error) {
			assert.Equal(t, "process", funcName, "should call correct function")
			assert.Equal(t, []interface{}{int32(1)}, args, "should pass correct arguments")
			return expectedReturn, nil
		},
	}

	mockRuntime := &MockWasmRuntime{
		LoadModuleFunc: func(filePath string) (WasmModule, error) {
			return mockModule, nil
		},
	}

	result := processWasmFileWithRuntime("/test/valid.wasm", mockRuntime)

	assert.True(t, result.Success, "should report success")
	assert.Equal(t, StageNone, result.FailureStage, "should have no failure stage")
	assert.Empty(t, result.ErrorMessage, "should have no error message")
	assert.Equal(t, expectedReturn, result.ReturnValues, "should capture return values")
	assert.True(t, mockModule.CloseCalled, "should call Close")
}

// -----------------------------------------------------------------------------
// TEST: Resource Cleanup Under Failure
// -----------------------------------------------------------------------------
//
// WHY THIS MATTERS:
// Even when execution fails, resources must be properly released.
// Memory leaks in a long fuzzing campaign will crash the fuzzer.
// -----------------------------------------------------------------------------

func TestResourceCleanup_OnError(t *testing.T) {
	mockModule := &MockWasmModule{
		ExecuteFunc: func(funcName string, args ...interface{}) ([]interface{}, error) {
			return nil, errors.New("simulated failure")
		},
	}

	mockRuntime := &MockWasmRuntime{
		LoadModuleFunc: func(filePath string) (WasmModule, error) {
			return mockModule, nil
		},
	}

	_ = processWasmFileWithRuntime("/test/cleanup.wasm", mockRuntime)

	assert.True(t, mockModule.CloseCalled, "Close must be called even on failure")
}

func TestResourceCleanup_OnPanic(t *testing.T) {
	mockModule := &MockWasmModule{
		ExecuteFunc: func(funcName string, args ...interface{}) ([]interface{}, error) {
			panic("test panic")
		},
	}

	mockRuntime := &MockWasmRuntime{
		LoadModuleFunc: func(filePath string) (WasmModule, error) {
			return mockModule, nil
		},
	}

	_ = processWasmFileWithRuntime("/test/panic_cleanup.wasm", mockRuntime)

	assert.True(t, mockModule.CloseCalled, "Close must be called even after panic")
}

// -----------------------------------------------------------------------------
// TEST: Error Classification Accuracy
// -----------------------------------------------------------------------------
//
// WHY THIS MATTERS:
// Accurate error classification is essential for triage. A fuzzer that
// misclassifies errors will send developers on wild goose chases.
// -----------------------------------------------------------------------------

func TestErrorClassification_RuntimeErrorType(t *testing.T) {
	stages := []FailureStage{StageLoad, StageValidate, StageInstantiate, StageExecute}

	for _, stage := range stages {
		t.Run(string(stage), func(t *testing.T) {
			mockRuntime := &MockWasmRuntime{
				LoadModuleFunc: func(filePath string) (WasmModule, error) {
					return nil, &RuntimeError{
						Stage:   stage,
						Message: fmt.Sprintf("test error at %s", stage),
					}
				},
			}

			result := processWasmFileWithRuntime("/test/classify.wasm", mockRuntime)

			require.False(t, result.Success)
			assert.Equal(t, stage, result.FailureStage, "stage should match RuntimeError.Stage")
		})
	}
}

// -----------------------------------------------------------------------------
// BENCHMARK: Fault Injection Overhead
// -----------------------------------------------------------------------------
//
// WHY THIS MATTERS:
// Fault injection should have minimal overhead. If mocking is too slow,
// it affects the ability to run many test iterations.
// -----------------------------------------------------------------------------

func BenchmarkMockExecution(b *testing.B) {
	mockModule := &MockWasmModule{
		ExecuteFunc: func(funcName string, args ...interface{}) ([]interface{}, error) {
			return []interface{}{int32(42)}, nil
		},
	}

	mockRuntime := &MockWasmRuntime{
		LoadModuleFunc: func(filePath string) (WasmModule, error) {
			return mockModule, nil
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processWasmFileWithRuntime("/test/bench.wasm", mockRuntime)
	}
}

func BenchmarkMockWithPanicRecovery(b *testing.B) {
	mockModule := &MockWasmModule{
		ExecuteFunc: func(funcName string, args ...interface{}) ([]interface{}, error) {
			panic("benchmark panic")
		},
	}

	mockRuntime := &MockWasmRuntime{
		LoadModuleFunc: func(filePath string) (WasmModule, error) {
			return mockModule, nil
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processWasmFileWithRuntime("/test/bench.wasm", mockRuntime)
	}
}
