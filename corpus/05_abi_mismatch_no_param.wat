;; 05_abi_mismatch_no_param.wat
;; WASM module with "process" export but wrong signature (no parameters)
;; Expected stage: EXECUTE (ABI mismatch - wrong parameter count)
;; Tests: Function signature validation at call time

(module
  ;; "process" exists but takes no parameters
  (func $process (export "process") (result i32)
    i32.const 42
  )
)
