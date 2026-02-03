;; 09_unreachable_trap.wat
;; Valid module that traps at runtime with "unreachable" instruction
;; Expected stage: EXECUTE (runtime trap)
;; Tests: Runtime trap handling

(module
  (func $process (export "process") (param $input i32) (result i32)
    ;; Immediately trap - simulates assertion failure
    unreachable
  )
)
