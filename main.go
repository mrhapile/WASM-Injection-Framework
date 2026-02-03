//go:build integration
// +build integration

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/second-state/WasmEdge-go/wasmedge"
)

// processWasmFile processes a single WASM file through all stages
// It never panics - all errors are captured and returned in the result
func processWasmFile(filePath string) (result ExecutionResult) {
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

	// Initialize WasmEdge configuration
	conf := wasmedge.NewConfigure()
	defer conf.Release()

	// Stage 1: Load WASM file
	loader := wasmedge.NewLoader()
	defer loader.Release()

	ast, err := loader.LoadFile(filePath)
	if err != nil {
		result.Success = false
		result.FailureStage = StageLoad
		result.ErrorMessage = fmt.Sprintf("load failed: %v", err)
		return result
	}
	defer ast.Release()

	// Stage 2: Validate WASM module
	validator := wasmedge.NewValidator()
	defer validator.Release()

	err = validator.Validate(ast)
	if err != nil {
		result.Success = false
		result.FailureStage = StageValidate
		result.ErrorMessage = fmt.Sprintf("validation failed: %v", err)
		return result
	}

	// Stage 3: Instantiate WASM module
	store := wasmedge.NewStore()
	defer store.Release()

	executor := wasmedge.NewExecutor()
	defer executor.Release()

	module, err := executor.Instantiate(store, ast)
	if err != nil {
		result.Success = false
		result.FailureStage = StageInstantiate
		result.ErrorMessage = fmt.Sprintf("instantiation failed: %v", err)
		return result
	}
	defer module.Release()

	// Stage 4: Execute the "process" function with input 1
	funcInstance := module.FindFunction("process")
	if funcInstance == nil {
		result.Success = false
		result.FailureStage = StageExecute
		result.ErrorMessage = "function 'process' not found in module exports"
		return result
	}

	// Call the function with input value 1
	returns, err := executor.Invoke(funcInstance, int32(1))
	if err != nil {
		result.Success = false
		result.FailureStage = StageExecute
		result.ErrorMessage = fmt.Sprintf("execution failed: %v", err)
		return result
	}

	// Success - capture return values
	result.Success = true
	result.ReturnValues = make([]interface{}, len(returns))
	for i, v := range returns {
		result.ReturnValues[i] = v
	}

	return result
}

// collectWasmFiles returns all .wasm files in the given directory
func collectWasmFiles(dirPath string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".wasm" {
			fullPath := filepath.Join(dirPath, entry.Name())
			files = append(files, fullPath)
		}
	}

	return files, nil
}

// runFuzzer processes all WASM files in the directory and generates a report
func runFuzzer(dirPath string) (FuzzingReport, error) {
	report := FuzzingReport{
		Results:       make([]ExecutionResult, 0),
		FailureCounts: make(map[FailureStage]int),
	}

	// Initialize failure counts
	report.FailureCounts[StageLoad] = 0
	report.FailureCounts[StageValidate] = 0
	report.FailureCounts[StageInstantiate] = 0
	report.FailureCounts[StageExecute] = 0

	// Collect all WASM files
	files, err := collectWasmFiles(dirPath)
	if err != nil {
		return report, err
	}

	report.TotalFiles = len(files)

	// Process each file sequentially (no concurrency)
	for _, filePath := range files {
		result := processWasmFile(filePath)
		report.Results = append(report.Results, result)

		if result.Success {
			report.Passed++
		} else {
			report.Failed++
			report.FailureCounts[result.FailureStage]++
		}
	}

	return report, nil
}

// outputJSON writes the report as formatted JSON to stdout
func outputJSON(report FuzzingReport) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func main() {
	// Ensure we never panic from main
	defer func() {
		if r := recover(); r != nil {
			errorResult := map[string]interface{}{
				"error":   "fatal panic in main",
				"details": fmt.Sprintf("%v", r),
			}
			json.NewEncoder(os.Stderr).Encode(errorResult)
			os.Exit(1)
		}
	}()

	// Validate command line arguments
	if len(os.Args) < 2 {
		errorResult := map[string]string{
			"error": "usage: wasm-fuzzer <directory>",
		}
		json.NewEncoder(os.Stderr).Encode(errorResult)
		os.Exit(1)
	}

	dirPath := os.Args[1]

	// Verify directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		errorResult := map[string]string{
			"error":   "directory access failed",
			"details": err.Error(),
		}
		json.NewEncoder(os.Stderr).Encode(errorResult)
		os.Exit(1)
	}

	if !info.IsDir() {
		errorResult := map[string]string{
			"error": "path is not a directory",
			"path":  dirPath,
		}
		json.NewEncoder(os.Stderr).Encode(errorResult)
		os.Exit(1)
	}

	// Initialize WasmEdge globally (required before any WasmEdge operations)
	wasmedge.SetLogErrorLevel()

	// Run the fuzzer
	report, err := runFuzzer(dirPath)
	if err != nil {
		errorResult := map[string]string{
			"error":   "fuzzer execution failed",
			"details": err.Error(),
		}
		json.NewEncoder(os.Stderr).Encode(errorResult)
		os.Exit(1)
	}

	// Output results as JSON
	if err := outputJSON(report); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode JSON output: %v\n", err)
		os.Exit(1)
	}
}
