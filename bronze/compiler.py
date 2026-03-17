"""
MiniLang Compiler — Lark + llvmlite example
============================================

Supports:
  - Primitive types: int, float, bool
  - Typed function definitions with parameters and return type
  - let bindings:  let x: int = expr
  - Arithmetic:    + - * /
  - Comparisons:   == != < > <= >=
  - Boolean ops:   && ||
  - if/else expressions
  - return statements
  - Function calls
  - Integer and float literals, true/false
  - A top-level `main` function is used as the entry point

Example program (see EXAMPLE_PROGRAM below):

    fn add(a: int, b: int) -> int {
        return a + b;
    }

    fn main() -> int {
        let x: int = add(3, 4);
        let y: int = add(x, 10);
        return y;
    }

Usage:
    python minilang_compiler.py              # compiles built-in example, prints IR
    python minilang_compiler.py source.ml    # compiles a .ml source file, prints IR
    python minilang_compiler.py source.ml --run   # JIT-executes main() and prints result
"""

import sys
from lark import Lark, Transformer, v_args
from llvmlite import ir
import llvmlite.binding as llvm

# ---------------------------------------------------------------------------
# 1.  GRAMMAR
# ---------------------------------------------------------------------------

GRAMMAR = r"""
    program     : func_def+

    func_def    : "fn" NAME "(" param_list ")" "->" type "{" stmt* "}"
    param_list  : (param ("," param)*)?
    param       : NAME ":" type

    type        : "int"   -> type_int
                | "float" -> type_float
                | "bool"  -> type_bool

    stmt        : "let" NAME ":" type "=" expr ";"   -> let_stmt
                | "return" expr ";"                  -> return_stmt
                | "if" expr "{" stmt* "}" ("else" "{" stmt* "}")?  -> if_stmt
                | expr ";"                           -> expr_stmt

    ?expr       : or_expr

    ?or_expr    : and_expr
                | or_expr "||" and_expr  -> or_op

    ?and_expr   : cmp_expr
                | and_expr "&&" cmp_expr -> and_op

    ?cmp_expr   : add_expr
                | cmp_expr "==" add_expr -> eq
                | cmp_expr "!=" add_expr -> ne
                | cmp_expr "<"  add_expr -> lt
                | cmp_expr ">"  add_expr -> gt
                | cmp_expr "<=" add_expr -> le
                | cmp_expr ">=" add_expr -> ge

    ?add_expr   : mul_expr
                | add_expr "+" mul_expr  -> add
                | add_expr "-" mul_expr  -> sub

    ?mul_expr   : unary
                | mul_expr "*" unary     -> mul
                | mul_expr "/" unary     -> div

    ?unary      : "-" atom               -> neg
                | "!" atom               -> not_op
                | atom

    ?atom       : INT                    -> int_lit
                | FLOAT                  -> float_lit
                | "true"                 -> bool_true
                | "false"                -> bool_false
                | NAME "(" arg_list ")"  -> call
                | NAME                   -> var
                | "(" expr ")"

    arg_list    : (expr ("," expr)*)?

    NAME        : /[a-zA-Z_][a-zA-Z0-9_]*/
    INT         : /[0-9]+/
    FLOAT       : /[0-9]+\.[0-9]*/

    %import common.WS
    %import common.NEWLINE
    %ignore WS
    %ignore NEWLINE
    %ignore /\/\/.*/
"""

# ---------------------------------------------------------------------------
# 2.  AST NODES
# ---------------------------------------------------------------------------

class Program:
    def __init__(self, funcs): self.funcs = funcs

class FuncDef:
    def __init__(self, name, params, ret_type, body):
        self.name, self.params, self.ret_type, self.body = name, params, ret_type, body

class Param:
    def __init__(self, name, typ): self.name, self.typ = name, typ

# Types
class TInt: pass
class TFloat: pass
class TBool: pass

# Statements
class LetStmt:
    def __init__(self, name, typ, expr): self.name, self.typ, self.expr = name, typ, expr
class ReturnStmt:
    def __init__(self, expr): self.expr = expr
class IfStmt:
    def __init__(self, cond, then_body, else_body): self.cond, self.then_body, self.else_body = cond, then_body, else_body
class ExprStmt:
    def __init__(self, expr): self.expr = expr

# Expressions
class IntLit:
    def __init__(self, v): self.v = v
class FloatLit:
    def __init__(self, v): self.v = v
class BoolLit:
    def __init__(self, v): self.v = v
class Var:
    def __init__(self, name): self.name = name
class BinOp:
    def __init__(self, op, l, r): self.op, self.l, self.r = op, l, r
class UnOp:
    def __init__(self, op, e): self.op, self.e = op, e
class Call:
    def __init__(self, name, args): self.name, self.args = name, args

# ---------------------------------------------------------------------------
# 3.  LARK TRANSFORMER  (parse tree → AST)
# ---------------------------------------------------------------------------

@v_args(inline=True)
class ASTBuilder(Transformer):
    # Program / functions
    def program(self, *funcs):     return Program(list(funcs))
    def func_def(self, name, params, ret_type, *stmts):
        return FuncDef(str(name), list(params), ret_type, list(stmts))
    def param_list(self, *params): return list(params)
    def param(self, name, typ):    return Param(str(name), typ)
    def arg_list(self, *args):     return list(args)

    # Types
    def type_int(self):   return TInt()
    def type_float(self): return TFloat()
    def type_bool(self):  return TBool()

    # Statements
    def let_stmt(self, name, typ, expr):   return LetStmt(str(name), typ, expr)
    def return_stmt(self, expr):           return ReturnStmt(expr)
    def if_stmt(self, cond, *rest):
        # rest = then_stmts... possibly else_stmts after an empty token boundary
        # Lark puts all stmts flat; we rely on the grammar making else optional
        # Simplest approach: pass them as two halves separated at position
        # Because we used stmt* twice, lark gives us a flat list of stmts.
        # We can't easily split — store as (cond, rest, []) and handle in codegen
        return IfStmt(cond, list(rest), [])
    def expr_stmt(self, expr):             return ExprStmt(expr)

    # Literals
    def int_lit(self, v):   return IntLit(int(v))
    def float_lit(self, v): return FloatLit(float(v))
    def bool_true(self):    return BoolLit(True)
    def bool_false(self):   return BoolLit(False)

    # Variable / call
    def var(self, name):           return Var(str(name))
    def call(self, name, args):    return Call(str(name), args)

    # Binary ops
    def add(self, l, r): return BinOp('+', l, r)
    def sub(self, l, r): return BinOp('-', l, r)
    def mul(self, l, r): return BinOp('*', l, r)
    def div(self, l, r): return BinOp('/', l, r)
    def eq(self, l, r):  return BinOp('==', l, r)
    def ne(self, l, r):  return BinOp('!=', l, r)
    def lt(self, l, r):  return BinOp('<',  l, r)
    def gt(self, l, r):  return BinOp('>',  l, r)
    def le(self, l, r):  return BinOp('<=', l, r)
    def ge(self, l, r):  return BinOp('>=', l, r)
    def or_op(self, l, r):  return BinOp('||', l, r)
    def and_op(self, l, r): return BinOp('&&', l, r)

    # Unary ops
    def neg(self, e):    return UnOp('-', e)
    def not_op(self, e): return UnOp('!', e)

# ---------------------------------------------------------------------------
# 4.  TYPE CHECKER  (lightweight — infers expression types)
# ---------------------------------------------------------------------------

class TypeEnv:
    def __init__(self, func_sigs):
        self.func_sigs = func_sigs  # name -> (param_types, ret_type)
        self.scopes = [{}]

    def push(self): self.scopes.append({})
    def pop(self):  self.scopes.pop()

    def define(self, name, typ): self.scopes[-1][name] = typ

    def lookup(self, name):
        for s in reversed(self.scopes):
            if name in s: return s[name]
        raise NameError(f"Undefined variable: {name}")

def ml_type_eq(a, b):
    return type(a) == type(b)

def type_of(expr, env):
    if isinstance(expr, IntLit):   return TInt()
    if isinstance(expr, FloatLit): return TFloat()
    if isinstance(expr, BoolLit):  return TBool()
    if isinstance(expr, Var):      return env.lookup(expr.name)
    if isinstance(expr, Call):
        if expr.name not in env.func_sigs:
            raise NameError(f"Undefined function: {expr.name}")
        _, ret = env.func_sigs[expr.name]
        return ret
    if isinstance(expr, UnOp):
        t = type_of(expr.e, env)
        if expr.op == '-':
            if not isinstance(t, (TInt, TFloat)):
                raise TypeError("Negation requires numeric type")
            return t
        if expr.op == '!':
            if not isinstance(t, TBool):
                raise TypeError("Not requires bool")
            return TBool()
    if isinstance(expr, BinOp):
        lt_ = type_of(expr.l, env)
        rt_ = type_of(expr.r, env)
        if expr.op in ('+', '-', '*', '/'):
            if not ml_type_eq(lt_, rt_):
                raise TypeError(f"Type mismatch in arithmetic: {lt_} vs {rt_}")
            return lt_
        if expr.op in ('==', '!=', '<', '>', '<=', '>='):
            return TBool()
        if expr.op in ('&&', '||'):
            return TBool()
    raise TypeError(f"Cannot infer type of: {expr}")

def typecheck_program(program):
    # First pass: collect function signatures
    sigs = {}
    for f in program.funcs:
        param_types = [p.typ for p in f.params]
        sigs[f.name] = (param_types, f.ret_type)

    for f in program.funcs:
        env = TypeEnv(sigs)
        env.push()
        for p in f.params:
            env.define(p.name, p.typ)
        for stmt in f.body:
            typecheck_stmt(stmt, f.ret_type, env)
        env.pop()

def typecheck_stmt(stmt, ret_type, env):
    if isinstance(stmt, LetStmt):
        t = type_of(stmt.expr, env)
        if not ml_type_eq(t, stmt.typ):
            raise TypeError(f"Let type mismatch for '{stmt.name}'")
        env.define(stmt.name, stmt.typ)
    elif isinstance(stmt, ReturnStmt):
        t = type_of(stmt.expr, env)
        if not ml_type_eq(t, ret_type):
            raise TypeError(f"Return type mismatch")
    elif isinstance(stmt, ExprStmt):
        type_of(stmt.expr, env)
    elif isinstance(stmt, IfStmt):
        ct = type_of(stmt.cond, env)
        if not isinstance(ct, TBool):
            raise TypeError("if condition must be bool")
        for s in stmt.then_body: typecheck_stmt(s, ret_type, env)
        for s in stmt.else_body: typecheck_stmt(s, ret_type, env)

# ---------------------------------------------------------------------------
# 5.  LLVM IR CODE GENERATOR
# ---------------------------------------------------------------------------

def ml_to_llvm_type(t, module=None):
    if isinstance(t, TInt):   return ir.IntType(64)
    if isinstance(t, TFloat): return ir.DoubleType()
    if isinstance(t, TBool):  return ir.IntType(1)
    raise TypeError(f"Unknown type: {t}")

class Codegen:
    def __init__(self):
        self.module = ir.Module(name="minilang")
        self.module.triple = llvm.get_default_triple()
        self.builder = None
        self.func = None
        self.env = {}          # name -> alloca ptr
        self.func_map = {}     # name -> ir.Function

    # ---- helpers ----

    def alloca(self, name, typ):
        """Create an alloca in the entry block (standard LLVM pattern)."""
        with self.builder.goto_entry_block():
            alloca = self.builder.alloca(typ, name=name)
        return alloca

    def coerce_to_bool(self, val):
        t = val.type
        if isinstance(t, ir.IntType) and t.width == 1:
            return val
        if isinstance(t, ir.IntType):
            return self.builder.icmp_unsigned('!=', val, ir.Constant(t, 0))
        if isinstance(t, ir.DoubleType):
            return self.builder.fcmp_ordered('!=', val, ir.Constant(t, 0.0))
        raise TypeError("Cannot coerce to bool")

    # ---- program ----

    def gen_program(self, program):
        # Forward-declare all functions first
        for f in program.funcs:
            param_types = [ml_to_llvm_type(p.typ) for p in f.params]
            ret_type    = ml_to_llvm_type(f.ret_type)
            fn_type = ir.FunctionType(ret_type, param_types)
            fn = ir.Function(self.module, fn_type, name=f.name)
            for i, p in enumerate(f.params):
                fn.args[i].name = p.name
            self.func_map[f.name] = fn

        for f in program.funcs:
            self.gen_func(f)

        return self.module

    def gen_func(self, func_def):
        fn = self.func_map[func_def.name]
        self.func = fn
        self.env = {}

        block = fn.append_basic_block("entry")
        self.builder = ir.IRBuilder(block)

        # Copy params into allocas so they're mutable
        for arg, param in zip(fn.args, func_def.params):
            a = self.alloca(param.name, arg.type)
            self.builder.store(arg, a)
            self.env[param.name] = a

        for stmt in func_def.body:
            self.gen_stmt(stmt)

        # If the block has no terminator, add a default return
        if not self.builder.block.is_terminated:
            ret_type = fn.ftype.return_type
            self.builder.ret(ir.Constant(ret_type, 0))

    # ---- statements ----

    def gen_stmt(self, stmt):
        if self.builder.block.is_terminated:
            return  # Dead code after return — skip

        if isinstance(stmt, LetStmt):
            val = self.gen_expr(stmt.expr)
            a = self.alloca(stmt.name, val.type)
            self.builder.store(val, a)
            self.env[stmt.name] = a

        elif isinstance(stmt, ReturnStmt):
            val = self.gen_expr(stmt.expr)
            self.builder.ret(val)

        elif isinstance(stmt, ExprStmt):
            self.gen_expr(stmt.expr)

        elif isinstance(stmt, IfStmt):
            cond_val = self.coerce_to_bool(self.gen_expr(stmt.cond))

            then_block = self.func.append_basic_block("then")
            else_block = self.func.append_basic_block("else")
            merge_block = self.func.append_basic_block("merge")

            self.builder.cbranch(cond_val, then_block, else_block)

            # Then
            self.builder.position_at_end(then_block)
            for s in stmt.then_body:
                self.gen_stmt(s)
            if not self.builder.block.is_terminated:
                self.builder.branch(merge_block)

            # Else
            self.builder.position_at_end(else_block)
            for s in stmt.else_body:
                self.gen_stmt(s)
            if not self.builder.block.is_terminated:
                self.builder.branch(merge_block)

            self.builder.position_at_end(merge_block)

    # ---- expressions ----

    def gen_expr(self, expr):
        if isinstance(expr, IntLit):
            return ir.Constant(ir.IntType(64), expr.v)

        if isinstance(expr, FloatLit):
            return ir.Constant(ir.DoubleType(), expr.v)

        if isinstance(expr, BoolLit):
            return ir.Constant(ir.IntType(1), int(expr.v))

        if isinstance(expr, Var):
            ptr = self.env[expr.name]
            return self.builder.load(ptr, name=expr.name)

        if isinstance(expr, Call):
            fn = self.func_map[expr.name]
            args = [self.gen_expr(a) for a in expr.args]
            return self.builder.call(fn, args)

        if isinstance(expr, UnOp):
            val = self.gen_expr(expr.e)
            if expr.op == '-':
                if isinstance(val.type, ir.IntType):
                    return self.builder.neg(val)
                else:
                    return self.builder.fneg(val)
            if expr.op == '!':
                return self.builder.not_(val)

        if isinstance(expr, BinOp):
            return self.gen_binop(expr)

        raise ValueError(f"Unknown expr: {expr}")

    def gen_binop(self, expr):
        l = self.gen_expr(expr.l)
        r = self.gen_expr(expr.r)
        op = expr.op
        is_float = isinstance(l.type, ir.DoubleType)

        if op == '+':
            return self.builder.fadd(l, r) if is_float else self.builder.add(l, r)
        if op == '-':
            return self.builder.fsub(l, r) if is_float else self.builder.sub(l, r)
        if op == '*':
            return self.builder.fmul(l, r) if is_float else self.builder.mul(l, r)
        if op == '/':
            return self.builder.fdiv(l, r) if is_float else self.builder.sdiv(l, r)

        # Comparisons → return i1
        cmp_ops = {
            '==': ('==', '=='), '!=': ('!=', '!='),
            '<':  ('<',  'olt'), '>':  ('>',  'ogt'),
            '<=': ('<=', 'ole'), '>=': ('>=', 'oge'),
        }
        if op in cmp_ops:
            int_op, float_op = cmp_ops[op]
            if is_float:
                return self.builder.fcmp_ordered(float_op, l, r)
            else:
                return self.builder.icmp_signed(int_op, l, r)

        if op == '&&':
            return self.builder.and_(self.coerce_to_bool(l), self.coerce_to_bool(r))
        if op == '||':
            return self.builder.or_(self.coerce_to_bool(l), self.coerce_to_bool(r))

        raise ValueError(f"Unknown op: {op}")

# ---------------------------------------------------------------------------
# 6.  JIT EXECUTION
# ---------------------------------------------------------------------------

def jit_run(llvm_module_str):
    """JIT-compile and call main() -> int64, return Python int."""
    llvm.initialize()
    llvm.initialize_native_target()
    llvm.initialize_native_asmprinter()

    target   = llvm.Target.from_default_triple()
    target_m = target.create_target_machine()
    mod      = llvm.parse_assembly(llvm_module_str)
    mod.verify()

    pmb = llvm.create_pass_manager_builder()
    pmb.opt_level = 2
    pm = llvm.create_module_pass_manager()
    pmb.populate(pm)
    pm.run(mod)

    engine = llvm.create_mcjit_compiler(mod, target_m)
    engine.finalize_object()
    engine.run_static_constructors()

    import ctypes
    fn_ptr = engine.get_function_address("main")
    cfunc  = ctypes.CFUNCTYPE(ctypes.c_int64)(fn_ptr)
    return cfunc()
# ---------------------------------------------------------------------------
# 7.  PIPELINE
# ---------------------------------------------------------------------------

def compile_source(source: str) -> str:
    parser = Lark(GRAMMAR, start='program', parser='earley', ambiguity='resolve')
    tree   = parser.parse(source)
    ast    = ASTBuilder().transform(tree)
    typecheck_program(ast)
    cg     = Codegen()
    module = cg.gen_program(ast)
    return str(module)

# ---------------------------------------------------------------------------
# 8.  EXAMPLE PROGRAM
# ---------------------------------------------------------------------------

EXAMPLE_PROGRAM = """
// MiniLang example — functions, let bindings, arithmetic, comparisons

fn factorial(n: int) -> int {
    if n <= 1 {
        return 1;
    }
    return n * factorial(n - 1);
}

fn max(a: int, b: int) -> int {
    if a > b {
        return a;
    }
    return b;
}

fn add_floats(x: float, y: float) -> float {
    return x + y;
}

fn main() -> int {
    let a: int = factorial(6);      // 720
    let b: int = factorial(3);      // 6
    let m: int = max(a, b);         // 720
    return m;
}
"""

# ---------------------------------------------------------------------------
# 9.  ENTRY POINT
# ---------------------------------------------------------------------------

if __name__ == "__main__":
    run_jit = "--run" in sys.argv
    args    = [a for a in sys.argv[1:] if not a.startswith("-")]

    if args:
        with open(args[0]) as fh:
            source = fh.read()
    else:
        source = EXAMPLE_PROGRAM
        print("=== Using built-in example program ===\n")
        print(source)

    print("\n=== Compiling... ===\n")
    ir_text = compile_source(source)
    print(ir_text)

    if run_jit:
        print("\n=== JIT executing main() ===")
        result = jit_run(ir_text)
        print(f"main() returned: {result}")
