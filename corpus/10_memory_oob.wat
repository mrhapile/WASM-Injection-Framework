;; 10_memory_oob.wat
;; Valid module that causes out-of-bounds memory access at runtime
;; Expected stage: EXECUTE (memory access violation)
;; Tests: Memory bounds checking

(module
  ;; Declare 1 page of memory (64KB)
  (memory (export "memory") 1)
  
  (func $process (export "process") (param $input i32) (result i32)
    ;; Attempt to load from way out of bounds (address 0xFFFFFF)
    i32.const 0xFFFFFF
    i32.load
  )
)
