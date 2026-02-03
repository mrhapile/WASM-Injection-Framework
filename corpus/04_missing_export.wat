;; 04_missing_export.wat
;; A valid WASM module but without the required "process" export
;; Expected stage: EXECUTE (function not found)
;; Tests: Export resolution / function lookup

(module
  ;; Internal function not exported
  (func $internal (param $x i32) (result i32)
    local.get $x
    i32.const 2
    i32.mul
  )
  
  ;; Different export name - not "process"
  (func $compute (export "compute") (param $x i32) (result i32)
    local.get $x
    call $internal
  )
)
