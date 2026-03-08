import re

class Interpreter:
    def __init__(self):
        self.functions = {}

    def define_function(self, name, args, body):
        self.functions[name] = (args, body)

    def call_function(self, name, args):
        fn_args, body = self.functions[name]
        local_env = dict(zip(fn_args, args))
        return self.execute_block(body, local_env)

    def execute_block(self, stmts, env):
        for stmt in stmts:
            kind = stmt[0]
            if kind == "return":
                return eval(stmt[1], {}, self.make_eval_env(env))
            elif kind == "assign":
                env[stmt[1]] = eval(stmt[2], {}, self.make_eval_env(env))
            elif kind == "print":
                print(eval(stmt[1], {}, self.make_eval_env(env)))
            elif kind == "while":
                cond, body = stmt[1], stmt[2]
                while eval(cond, {}, self.make_eval_env(env)):
                    self.execute_block(body, env)
            elif kind == "if":
                cond, body = stmt[1], stmt[2]
                if eval(cond, {}, self.make_eval_env(env)):
                    self.execute_block(body, env)
        return None

    def make_eval_env(self, env):
        # Allow calling DSL functions inside eval
        eval_env = dict(env)
        for fname in self.functions:
            eval_env[fname] = lambda *args, f=fname: self.call_function(f, args)
        return eval_env

# --- Simple Parser ---
def parse_dsl(src):
    # Remove newlines and extra spaces
    src = re.sub(r"\s+", " ", src.strip())

    # Split functions
    fun_blocks = re.findall(r"fun\s+(\w+)\s*\((.*?)\)\s*:\s*\w+\s*\{(.*?)\}", src)
    functions = []
    for name, args, body in fun_blocks:
        arg_names = [a.split(":")[0].strip() for a in args.split(",") if a.strip()]
        stmts = parse_body(body)
        functions.append((name, arg_names, stmts))
    return functions

def parse_body(body):
    stmts = []
    # Split by semicolons for simplicity
    parts = [p.strip() for p in body.split(";") if p.strip()]
    for p in parts:
        if p.startswith("return"):
            expr = p[len("return"):].strip()
            stmts.append(("return", expr))
        elif p.startswith("print"):
            expr = p[len("print"):].strip("() ")
            stmts.append(("print", expr))
        elif p.startswith("let"):
            # let i: int32 = 0
            m = re.match(r"let\s+(\w+)\s*:\s*\w+\s*=\s*(.+)", p)
            if m:
                var, val = m.groups()
                stmts.append(("assign", var, val))
        elif p.startswith("while"):
            # while cond { body }
            m = re.match(r"while\s+(.+?)\s*\{(.*?)\}", p)
            if m:
                cond, inner = m.groups()
                stmts.append(("while", cond.strip(), parse_body(inner)))
        elif p.startswith("if"):
            m = re.match(r"if\s+(.+?)\s*\{(.*?)\}", p)
            if m:
                cond, inner = m.groups()
                stmts.append(("if", cond.strip(), parse_body(inner)))
        else:
            # assignment like i = i + 1
            m = re.match(r"(\w+)\s*=\s*(.+)", p)
            if m:
                var, val = m.groups()
                stmts.append(("assign", var, val))
    return stmts

# --- Example DSL ---
dsl_code = """
fun fib(n: int32): int32
    if n == 0 || n == 1 
        return n
    return fib(n - 1) + fib(n - 2)

fun main(): null
    let i: int32 = 0
    while i < 20 
        print(fib(i))
        i = i + 1
"""

# --- Run Interpreter ---
interp = Interpreter()
for name, args, body in parse_dsl(dsl_code):
    interp.define_function(name, args, body)

interp.call_function("main", [])
