import subprocess
import tempfile
import sys
from wasmtime import Store, Module, Instance, Func, FuncType, ValType, Memory

from compiler import compile

def print_i32(x):
    print(x)

source = open(sys.argv[1]).read()
wasm_text = compile(source)

with open("out.wat", "w") as f:
    f.write(wasm_text)

subprocess.run(["wat2wasm", "out.wat", "-o", "out.wasm"], check=True)

store = Store()

memory = Memory(store, minimum=1)

env = {
    "print_i32_impl": Func(store, FuncType([ValType.i32()], []), print_i32),
    "memory": memory,
}

module = Module.from_file(store.engine, "out.wasm")
instance = Instance(store, module, env)

instance.exports(store)["main"](store)
