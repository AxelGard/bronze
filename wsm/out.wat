(module
(import "env" "print_i32_impl" (func $print_i32_impl (param i32)))
(import "env" "alloc" (func $alloc (param i32) (result i32)))
(import "env" "print_i32" (func $print_i32 (param i32)))
(memory (import "env" "memory") 1)
(func (export "main")
  i32.const 4
  call $alloc
  local.set $tmp
  i32.const 10
  local.get $tmp
  i32.store
  i32.const 4
  call $alloc
  local.set $tmp
  i32.const 32
  local.get $tmp
  i32.store
  local.get $tmp
  i32.load
  local.get $tmp
  i32.load
  i32.add
  call $print_i32
)
)