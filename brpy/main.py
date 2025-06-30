from mycompiler.lexer import tokenize
from mycompiler.parser import Parser
from mycompiler.codegen import CodeGen

def main():
    code = """
    int x = 10 + 20;
    return x;
    """
    tokens = tokenize(code)
    parser = Parser(tokens)
    ast = parser.parse()
    codegen = CodeGen()
    llvm_ir = codegen.generate(ast)
    print(llvm_ir)

if __name__ == "__main__":
    main()

