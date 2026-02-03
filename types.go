package main

// FailureStage represents the stage at which a WASM module failed
type FailureStage string

const (
	StageNone        FailureStage = "none"
	StageLoad        FailureStage = "load"
	StageValidate    FailureStage = "validate"
	StageInstantiate FailureStage = "instantiate"
	StageExecute     FailureStage = "execute"
)

// ExecutionResult holds the structured result for a single WASM file
type ExecutionResult struct {
	FilePath     string        `json:"file_path"`
	FileName     string        `json:"file_name"`
	Success      bool          `json:"success"`
	FailureStage FailureStage  `json:"failure_stage"`
	ErrorMessage string        `json:"error_message,omitempty"`
	ReturnValues []interface{} `json:"return_values,omitempty"`
}

// FuzzingReport holds the complete report for all processed files
type FuzzingReport struct {
	TotalFiles    int                  `json:"total_files"`
	Passed        int                  `json:"passed"`
	Failed        int                  `json:"failed"`
	Results       []ExecutionResult    `json:"results"`
	FailureCounts map[FailureStage]int `json:"failure_counts"`
}
