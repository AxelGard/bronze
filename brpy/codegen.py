from llvmlite import ir
from mycompiler.ast import *

class CodeGen:
    def __init__(self):
        self.module = ir.Module(name="module")
        self.builder = None
        self.func = None
        self.symbols = {}

    def generate(self, ast):
        func_type = ir.FunctionType(ir.IntType(32), [])
        self.func = ir.Function(self.module, func_type, name="main")
        block = self.func.append_basic_block(name="entry")
        self.builder = ir.IRBuilder(block)

        for stmt in ast:
            self.compile_stmt(stmt)

        if not self.builder.block.is_terminated:
            self.builder.ret(ir.Constant(ir.IntType(32), 0))

        return str(self.module)

    def compile_stmt(self, stmt):
        if isinstance(stmt, VarDecl):
            val = self.compile_expr(stmt.expr)
            ptr = self.builder.alloca(ir.IntType(32), name=stmt.name)
            self.builder.store(val, ptr)
            self.symbols[stmt.name] = ptr
        elif isinstance(stmt, Return):
            val = self.compile_expr(stmt.expr)
            self.builder.ret(val)

    def compile_expr(self, expr):
        if isinstance(expr, Number):
            return ir.Constant(ir.IntType(32), expr.value)
        elif isinstance(expr, Variable):
            ptr = self.symbols.get(expr.name)
            return self.builder.load(ptr)
        elif isinstance(expr, BinOp):
            left = self.compile_expr(expr.left)
            right = self.compile_expr(expr.right)
            if expr.op == 'PLUS':
                return self.builder.add(left, right)
            elif expr.op == 'MINUS':
                return self.builder.sub(left, right)
            elif expr.op == 'MUL':
                return self.builder.mul(left, right)
            elif expr.op == 'DIV':
                return self.builder.sdiv(left, right)

