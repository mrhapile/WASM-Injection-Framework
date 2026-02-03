;; 08_invalid_opcode.wat
;; This WAT is valid, but we'll create a binary with invalid opcode manually
;; For now, this demonstrates a module that would fail validation
;; Expected stage: VALIDATE (if binary has bad opcode)
;; Tests: Opcode validation

(module
  ;; This will be replaced with a binary containing invalid opcodes
  ;; Placeholder valid module
  (func $process (export "process") (param $input i32) (result i32)
    local.get $input
  )
)
