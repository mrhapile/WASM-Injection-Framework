;; 01_valid_module.wat
;; A valid WASM module with correct ABI: int process(int)
;; Expected stage: NONE (success)
;; Tests: Full pipeline execution succeeds

(module
  ;; Export a function named "process" that takes i32 and returns i32
  (func $process (export "process") (param $input i32) (result i32)
    ;; Simple computation: return input + 1
    local.get $input
    i32.const 1
    i32.add
  )
)
