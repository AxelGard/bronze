from lark import Lark, Transformer

grammar = """
?start: statement+

?statement: "let" NAME "=" expr      -> let_stmt
          | "print" expr             -> print_stmt
          | expr

?expr: expr "+" term -> add
     | expr "-" term -> sub
     | term

?term: term "*" factor -> mul
     | term "/" factor -> div
     | factor

?factor: NUMBER       -> number
       | NAME         -> var
       | "(" expr ")"

%import common.NUMBER
%import common.CNAME -> NAME
%import common.WS
%ignore WS
"""

parser = Lark(grammar, parser="lalr")


class Interpreter(Transformer):
    def __init__(self):
        self.env = {}

    def number(self, n):
        return int(n[0])

    def var(self, name):
        name = str(name[0])
        if name not in self.env:
            raise Exception(f"Undefined variable {name}")
        return self.env[name]

    def add(self, args):
        return args[0] + args[1]

    def sub(self, args):
        return args[0] - args[1]

    def mul(self, args):
        return args[0] * args[1]

    def div(self, args):
        return args[0] // args[1]

    def let_stmt(self, args):
        name = str(args[0])
        value = args[1]
        self.env[name] = value

    def print_stmt(self, args):
        print(args[0])


def run(code):
    tree = parser.parse(code)
    Interpreter().transform(tree)


if __name__ == "__main__":
    code = """
    let x = 10
    let y = x + 5 * 2
    print y
    """

    run(code)
