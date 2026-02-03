;; 07_abi_mismatch_no_return.wat
;; WASM module with "process" export but no return value
;; Expected stage: EXECUTE (ABI mismatch - missing return)
;; Tests: Return type handling

(module
  ;; 1 page of memory for side effects
  (memory 1)
  
  ;; "process" exists but returns nothing (void)
  (func $process (export "process") (param $input i32)
    ;; Store input to memory as side effect, no return value
    i32.const 0
    local.get $input
    i32.store
  )
)
