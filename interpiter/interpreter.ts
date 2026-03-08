export type TokenType =
    | "NUMBER"
    | "IDENTIFIER"
    | "FUNC"
    | "RETURN"
    | "LPAREN"
    | "RPAREN"
    | "LBRACE"
    | "RBRACE"
    | "COMMA"
    | "COLON"
    | "PLUS"
    | "SEMICOLON"
    | "EOF";

export interface Token {
    type: TokenType
    value?: string
}


export function tokenize(input: string): Token[] {
    const tokens: Token[] = []
    let i = 0
    while (i < input.length) {
        const char = input[i]
        if (/\d/.test(char)) {
            let num = char
            i++

            while (/\d/.test(input[i])) {
                num += input[i++]
            }
            tokens.push({ type: "NUMBER", value: num })
        }
        if (/[a-zA-Z]/.test(char)) {
            let ident = char
            i++

            while (/[a-zA-Z]/.test(input[i])) {
                ident += input[i++]
            }

            if (ident === "func") tokens.push({ type: "FUNC" })
            else if (ident === "return") tokens.push({ type: "RETURN" })
            else tokens.push({ type: "IDENTIFIER", value: ident })

            continue
        }

        i++
    }
    
    tokens.push({type: "EOF"})
    return tokens
}


export type Expr =
  | NumberLiteral
  | BinaryExpr
  | CallExpr
  | Identifier

export interface NumberLiteral {
  type: "NumberLiteral"
  value: number
}

export interface Identifier {
  type: "Identifier"
  name: string
}

export interface BinaryExpr {
  type: "BinaryExpr"
  left: Expr
  right: Expr
  op: "+"
}

export interface CallExpr {
  type: "CallExpr"
  callee: string
  args: Expr[]
}


export interface FunctionDecl {
  type: "FunctionDecl"
  name: string
  params: string[]
  body: Expr[]
}


function parseNumber(token: Token): NumberLiteral {
  return {
    type: "NumberLiteral",
    value: Number(token.value)
  }
}

type Value = number | FunctionValue

interface FunctionValue {
  params: string[]
  body: any
}

class Environment {
  values = new Map<string, Value>()

  get(name: string): Value {
    return this.values.get(name)!
  }

  set(name: string, val: Value) {
    this.values.set(name, val)
  }
}


function evalExpr(expr: Expr, env: Environment): Value {
  switch (expr.type) {
    case "NumberLiteral":
      return expr.value

    case "BinaryExpr":
      const left = Number(evalExpr(expr.left, env))
      const right = Number(evalExpr(expr.right, env))
      return left + right

    case "CallExpr":
      const fn = env.get(expr.callee) as FunctionValue

      const args = expr.args.map(a => evalExpr(a, env))

      const local = new Environment()

      fn.params.forEach((p, i) => {
        local.set(p, args[i])
      })

      return evalExpr(fn.body, local)
  }
}


const code = await Deno.readTextFile(Deno.args[0])

const tokens = tokenize(code)
const ast = parse(tokens)

run(ast)
