from lark import Lark, Transformer
from llvmlite import ir

grammar = """
?start: expr
?expr: expr "+" term -> add
     | expr "-" term -> sub
     | term
?term: NUMBER -> number
%import common.NUMBER
%import common.WS
%ignore WS
"""

parser = Lark(grammar, start='start')

module = ir.Module(name="calc_module")
func_type = ir.FunctionType(ir.IntType(32), [])
func = ir.Function(module, func_type, name="main")
block = func.append_basic_block(name="entry")
builder = ir.IRBuilder(block)


# Recursive codegen
def codegen(node):
    if node.data == "number":
        return ir.Constant(ir.IntType(32), int(node.children[0]))
    elif node.data == "add":
        left = codegen(node.children[0])
        right = codegen(node.children[1])
        return builder.add(left, right, name="addtmp")
    elif node.data == "sub":
        left = codegen(node.children[0])
        right = codegen(node.children[1])
        return builder.sub(left, right, name="subtmp")
    else:
        raise NotImplementedError(f"Unknown node: {node.data}")



def run(code):
    tree = parser.parse(code)
    result = codegen(tree)
    builder.ret(result)
    print(module)


if __name__ == "__main__":
    code = """
    x = 2 + 3
    y = x - 1
    return y
    """

    run(code)
