"""A Pygments lexer for Bronze (.brz), following doc/spec-0.1.md."""

from pygments.lexer import RegexLexer, bygroups, words
from pygments.token import (Comment, Keyword, Name, Number, Operator,
                            Punctuation, String, Text, Whitespace)


class BronzeLexer(RegexLexer):
    name = "Bronze"
    aliases = ["bronze", "brz"]
    filenames = ["*.brz"]

    tokens = {
        "root": [
            (r"\s+", Whitespace),
            (r"//.*?$", Comment.Single),
            (r"/\*", Comment.Multiline, "comment"),
            (r"(func)(\s+)([A-Za-z_]\w*)",
             bygroups(Keyword.Declaration, Whitespace, Name.Function)),
            (r"(class|enum)(\s+)([A-Za-z_]\w*)",
             bygroups(Keyword.Declaration, Whitespace, Name.Class)),
            (words(("con", "mut", "func", "class", "enum", "use"),
                   suffix=r"\b"), Keyword.Declaration),
            (words(("return", "if", "else", "while", "match", "is", "and",
                    "or", "not"), suffix=r"\b"), Keyword),
            (words(("true", "false", "null"), suffix=r"\b"), Keyword.Constant),
            (r"this\b", Name.Builtin.Pseudo),
            (words(("Int", "Float", "Bool", "String", "Error"),
                   suffix=r"\b"), Keyword.Type),
            (words(("print", "panic", "assert"), suffix=r"\b"), Name.Builtin),
            (r"[A-Z]\w*", Name.Class),
            (r"[a-z_]\w*", Name),
            (r"\d[\d_]*\.\d[\d_]*([eE][+-]?\d+)?", Number.Float),
            (r"\d[\d_]*[eE][+-]?\d+", Number.Float),
            (r"\d[\d_]*", Number.Integer),
            (r'"', String, "string"),
            (r"->|=>|==|!=|<=|>=|[-+*/%<>=!?]", Operator),
            (r"[(){}.,:]", Punctuation),
            (r".", Text),
        ],
        "comment": [
            (r"[^*]+", Comment.Multiline),
            (r"\*/", Comment.Multiline, "#pop"),
            (r"\*", Comment.Multiline),
        ],
        "string": [
            (r'\\[nt"\\{]', String.Escape),
            (r'[^"\\\n]+', String),
            (r'"', String, "#pop"),
            (r".", String),
        ],
    }
