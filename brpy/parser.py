from mycompiler.lexer import tokenize
from mycompiler.ast import *

class Parser:
    def __init__(self, tokens):
        self.tokens = list(tokens)
        self.pos = 0

    def current(self):
        return self.tokens[self.pos] if self.pos < len(self.tokens) else (None, None)

    def eat(self, kind):
        if self.current()[0] == kind:
            self.pos += 1
        else:
            raise SyntaxError(f"Expected {kind}, got {self.current()}")

    def parse(self):
        return self.statement_list()

    def statement_list(self):
        stmts = []
        while self.current()[0] is not None:
            stmts.append(self.statement())
        return stmts

    def statement(self):
        if self.current()[0] == 'INT':
            return self.declaration()
        elif self.current()[0] == 'RETURN':
            return self.return_stmt()
        else:
            raise SyntaxError("Unknown statement")

    def declaration(self):
        self.eat('INT')
        name = self.current()[1]
        self.eat('ID')
        self.eat('ASSIGN')
        expr = self.expression()
        self.eat('SEMICOLON')
        return VarDecl(name, expr)

    def return_stmt(self):
        self.eat('RETURN')
        expr = self.expression()
        self.eat('SEMICOLON')
        return Return(expr)

    def expression(self):
        left = self.term()
        while self.current()[0] in ('PLUS', 'MINUS'):
            op = self.current()[0]
            self.eat(op)
            right = self.term()
            left = BinOp(left, op, right)
        return left

    def term(self):
        left = self.factor()
        while self.current()[0] in ('MUL', 'DIV'):
            op = self.current()[0]
            self.eat(op)
            right = self.factor()
            left = BinOp(left, op, right)
        return left

    def factor(self):
        if self.current()[0] == 'NUMBER':
            val = int(self.current()[1])
            self.eat('NUMBER')
            return Number(val)
        elif self.current()[0] == 'ID':
            name = self.current()[1]
            self.eat('ID')
            return Variable(name)
        elif self.current()[0] == 'LPAREN':
            self.eat('LPAREN')
            expr = self.expression()
            self.eat('RPAREN')
            return expr
        else:
            raise SyntaxError("Invalid factor")

