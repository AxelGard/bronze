export type Node =
  | FunctionDecl
  | Expr

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
