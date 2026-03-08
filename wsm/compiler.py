import sys

def compile(source: str) -> str:
    wasm = []
    wasm.append('(module')
    wasm.append('(import "env" "print_i32_impl" (func $print_i32_impl (param i32)))')
    wasm.append('(import "env" "alloc" (func $alloc (param i32) (result i32)))')
    wasm.append('(import "env" "print_i32" (func $print_i32 (param i32)))')
    wasm.append('(memory (import "env" "memory") 1)')

    wasm.append('(func (export "main")')

    variables = {}

    for line in source.splitlines():
        line = line.strip()
        if line.startswith("let"):
            _, name, _, value = line.split()
            wasm.append('  i32.const 4')
            wasm.append('  call $alloc')
            wasm.append('  local.set $tmp')

            wasm.append(f'  i32.const {value}')
            wasm.append('  local.get $tmp')
            wasm.append('  i32.store')

            variables[name] = '$tmp'

        elif line.startswith("print"):
            expr = line[line.find("(")+1:line.find(")")]
            a, b = expr.split('+')
            a = a.strip()
            b = b.strip()

            wasm.append(f'  local.get {variables[a]}')
            wasm.append('  i32.load')
            wasm.append(f'  local.get {variables[b]}')
            wasm.append('  i32.load')
            wasm.append('  i32.add')
            wasm.append('  call $print_i32')

    wasm.append(')')
    wasm.append(')')

    return "\n".join(wasm)

if __name__ == "__main__":
    with open(sys.argv[1]) as f:
        print(compile(f.read()))
