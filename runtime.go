//go:build !integration
// +build !integration

package main

import (
	"errors"
	"fmt"
	"path/filepath"
)

// RuntimeError represents an error from the WASM runtime
type RuntimeError struct {
	Stage   FailureStage
	Message string
	Cause   error
}

func (e *RuntimeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Stage, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Stage, e.Message)
}

// WasmRuntime defines the interface for WASM runtime operations
// This abstraction enables fault injection via mocking in tests
type WasmRuntime interface {
	// LoadModule loads a WASM module from the given file path
	LoadModule(filePath string) (WasmModule, error)
}

// WasmModule represents a loaded and instantiated WASM module
type WasmModule interface {
	// Execute runs the named function with the given arguments
	Execute(funcName string, args ...interface{}) ([]interface{}, error)
	// Close releases runtime resources
	Close()
}

// WasmEdgeRuntime implements WasmRuntime using the WasmEdge SDK
type WasmEdgeRuntime struct{}

// WasmEdgeModule wraps a WasmEdge module instance
type WasmEdgeModule struct {
	filePath string
	// In real implementation, these would be WasmEdge objects
	// For testing, we use this simplified structure
}

// NewWasmEdgeRuntime creates a new WasmEdge runtime instance
func NewWasmEdgeRuntime() *WasmEdgeRuntime {
	return &WasmEdgeRuntime{}
}

// LoadModule implements WasmRuntime.LoadModule
func (r *WasmEdgeRuntime) LoadModule(filePath string) (WasmModule, error) {
	// This delegates to the actual WasmEdge implementation
	// In tests, this entire method can be mocked
	return loadWasmEdgeModule(filePath)
}

// loadWasmEdgeModule is the actual implementation that can be mocked
var loadWasmEdgeModule = func(filePath string) (WasmModule, error) {
	// Placeholder - actual implementation uses WasmEdge SDK
	return &WasmEdgeModule{filePath: filePath}, nil
}

// Execute implements WasmModule.Execute
func (m *WasmEdgeModule) Execute(funcName string, args ...interface{}) ([]interface{}, error) {
	// This delegates to the actual execution implementation
	return executeWasmFunction(m.filePath, funcName, args...)
}

// executeWasmFunction is the actual implementation that can be mocked
var executeWasmFunction = func(filePath, funcName string, args ...interface{}) ([]interface{}, error) {
	// Placeholder - actual implementation uses WasmEdge SDK
	return nil, errors.New("not implemented - use processWasmFile for real execution")
}

// Close implements WasmModule.Close
func (m *WasmEdgeModule) Close() {
	// Release resources
}

// processWasmFileWithRuntime processes a WASM file using the provided runtime
// This is the testable version that accepts a runtime interface
func processWasmFileWithRuntime(filePath string, runtime WasmRuntime) (result ExecutionResult) {
	result.FilePath = filePath
	result.FileName = filepath.Base(filePath)
	result.FailureStage = StageNone

	// Defer panic recovery to ensure we never crash
	defer func() {
		if r := recover(); r != nil {
			result.Success = false
			result.FailureStage = StageExecute
			result.ErrorMessage = fmt.Sprintf("panic recovered: %v", r)
		}
	}()

	// Load the module (includes load, validate, instantiate)
	module, err := runtime.LoadModule(filePath)
	if err != nil {
		result.Success = false
		// Classify the error based on RuntimeError type
		var runtimeErr *RuntimeError
		if errors.As(err, &runtimeErr) {
			result.FailureStage = runtimeErr.Stage
			result.ErrorMessage = runtimeErr.Message
		} else {
			result.FailureStage = StageLoad
			result.ErrorMessage = fmt.Sprintf("load failed: %v", err)
		}
		return result
	}
	defer module.Close()

	// Execute the "process" function with input 1
	returns, err := module.Execute("process", int32(1))
	if err != nil {
		result.Success = false
		var runtimeErr *RuntimeError
		if errors.As(err, &runtimeErr) {
			result.FailureStage = runtimeErr.Stage
			result.ErrorMessage = runtimeErr.Message
		} else {
			result.FailureStage = StageExecute
			result.ErrorMessage = fmt.Sprintf("execution failed: %v", err)
		}
		return result
	}

	// Success - capture return values
	result.Success = true
	result.ReturnValues = returns
	return result
}
