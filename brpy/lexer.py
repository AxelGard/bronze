import re

# Token specification
token_specification = [
    ('NUMBER',   r'\d+'),
    ('ID',       r'[A-Za-z_][A-Za-z0-9_]*'),
    ('ASSIGN',   r'='),
    ('SEMICOLON',r';'),
    ('LPAREN',   r'\('),
    ('RPAREN',   r'\)'),
    ('PLUS',     r'\+'),
    ('MINUS',    r'-'),
    ('MUL',      r'\*'),
    ('DIV',      r'/'),
    ('RETURN',   r'return'),
    ('INT',      r'int'),
    ('SKIP',     r'[ \t]+'),
    ('NEWLINE',  r'\n'),
    ('MISMATCH', r'.'),
]

tok_regex = '|'.join('(?P<%s>%s)' % pair for pair in token_specification)
get_token = re.compile(tok_regex).match

def tokenize(code):
    pos = 0
    mo = get_token(code)
    while mo is not None:
        kind = mo.lastgroup
        value = mo.group()
        if kind not in ('SKIP', 'NEWLINE', 'MISMATCH'):
            yield kind, value
        pos = mo.end()
        mo = get_token(code, pos)

