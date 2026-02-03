;; 06_abi_mismatch_wrong_type.wat
;; WASM module with "process" export but wrong parameter type (i64 instead of i32)
;; Expected stage: EXECUTE (ABI mismatch - wrong parameter type)
;; Tests: Type checking at function invocation

(module
  ;; "process" exists but expects i64 instead of i32
  (func $process (export "process") (param $input i64) (result i64)
    local.get $input
    i64.const 1
    i64.add
  )
)
