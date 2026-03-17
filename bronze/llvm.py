from llvmlite import binding, ir

binding.initialize_native_target()
binding.initialize_native_asmprinter()

module = ir.Module(name="jit_module")

func_type = ir.FunctionType(ir.IntType(32), [])
func = ir.Function(module, func_type, name="main")

block = func.append_basic_block(name="entry")
builder = ir.IRBuilder(block)
builder.ret(ir.Constant(ir.IntType(32), 42))

llvm_ir = str(module)

# Compile
target = binding.Target.from_default_triple()
target_machine = target.create_target_machine()
backing_mod = binding.parse_assembly(llvm_ir)
engine = binding.create_mcjit_compiler(backing_mod, target_machine)

engine.finalize_object()

# Run
func_ptr = engine.get_function_address("main")

import ctypes
cfunc = ctypes.CFUNCTYPE(ctypes.c_int)(func_ptr)

print(cfunc())  # should print 42
