import { Token } from "./lexer.ts";
import {
    Node,
    BinaryExpr,
    CallExpr,
    Expr,
    Identifier,
    NumberLiteral,
    FunctionDecl
} from "./ast.ts";

export function parse(tokens: Token[]): Node[] {
  const parser = new Parser(tokens)
  return parser.parseProgram()
}

class Parser {
  private pos = 0

  constructor(private tokens: Token[]) {}

  private peek(): Token {
    return this.tokens[this.pos]
  }

  private consume(): Token {
    return this.tokens[this.pos++]
  }

  parseProgram(): Node[] {
    const nodes: Node[] = []

    while (this.peek().type !== "EOF") {
      if (this.peek().type === "FUNC") {
        nodes.push(this.parseFunction())
      } else {
        nodes.push(this.parseExpression())

        if (this.peek().type === "SEMICOLON") {
          this.consume()
        }
      }
    }

    return nodes
  }

  parseFunction(): FunctionDecl {
    this.consume() // fn

    const name = this.consume() // identifier

    this.consume() // (

    const params: string[] = []

    while (this.peek().type !== "RPAREN") {
      const param = this.consume()
      params.push(param.value!)

      if (this.peek().type === "COMMA") {
        this.consume()
      }
    }

    this.consume() // )

    this.consume() // {

    const body: Expr[] = []

    while (this.peek().type !== "RBRACE") {
      body.push(this.parseExpression())

      if (this.peek().type === "SEMICOLON") {
        this.consume()
      }
    }

    this.consume() // }

    return {
      type: "FunctionDecl",
      name: name.value!,
      params,
      body
    }
  }

  parseExpression(): Expr {
    return this.parseAddition()
  }

  parseAddition(): Expr {
    let left = this.parsePrimary()

    while (this.peek().type === "PLUS") {
      this.consume()

      const right = this.parsePrimary()

      left = {
        type: "BinaryExpr",
        left,
        right,
        op: "+"
      }
    }

    return left
  }

  parsePrimary(): Expr {
    const token = this.consume()

    if (token.type === "NUMBER") {
      return {
        type: "NumberLiteral",
        value: Number(token.value)
      }
    }

    if (token.type === "IDENTIFIER") {
      if (this.peek().type === "LPAREN") {
        return this.parseCall(token.value!)
      }

      return {
        type: "Identifier",
        name: token.value!
      }
    }

    throw new Error("Unexpected token " + token.type)
  }

  parseCall(name: string): CallExpr {
    this.consume() // (

    const args: Expr[] = []

    while (this.peek().type !== "RPAREN") {
      args.push(this.parseExpression())

      if (this.peek().type === "COMMA") {
        this.consume()
      }
    }

    this.consume() // )

    return {
      type: "CallExpr",
      callee: name,
      args
    }
  }
}
