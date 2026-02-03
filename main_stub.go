//go:build !integration
// +build !integration

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

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

// runFuzzerWithRuntime processes all WASM files using the provided runtime
func runFuzzerWithRuntime(dirPath string, runtime WasmRuntime) (FuzzingReport, error) {
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
		result := processWasmFileWithRuntime(filePath, runtime)
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
	// Stub main for non-integration builds
	// When running tests, we use processWasmFileWithRuntime with mocks
	fmt.Println("Build with -tags=integration to run the full WasmEdge fuzzer")
	fmt.Println("Run 'go test' to run the fault injection tests")
}
