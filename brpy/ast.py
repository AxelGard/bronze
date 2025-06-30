class Number:
    def __init__(self, value):
        self.value = value

class Variable:
    def __init__(self, name):
        self.name = name

class BinOp:
    def __init__(self, left, op, right):
        self.left = left
        self.op = op
        self.right = right

class VarDecl:
    def __init__(self, name, expr):
        self.name = name
        self.expr = expr

class Return:
    def __init__(self, expr):
        self.expr = expr

