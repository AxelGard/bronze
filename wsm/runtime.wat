(module
  (memory (export "memory") 1)

  ;; heap pointer
  (global $heap_ptr (mut i32) (i32.const 1024))

  ;; allocate N bytes
  (func (export "alloc") (param $size i32) (result i32)
    (local $addr i32)
    global.get $heap_ptr
    local.set $addr

    global.get $heap_ptr
    local.get $size
    i32.add
    global.set $heap_ptr

    local.get $addr
  )

  ;; dummy GC (placeholder)
  (func (export "gc")
    ;; later: walk roots, mark, sweep
  )

  ;; print integer
  (func (export "print_i32") (param i32)
    local.get 0
    call $print_i32_impl
  )

  ;; imported print
  (import "env" "print_i32_impl"
    (func $print_i32_impl (param i32)))
)
