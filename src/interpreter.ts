import { Expr, Node } from "./ast.ts";
import { createGlobalEnv } from "./runtime.ts";

type Value = number | FunctionValue;

interface FunctionValue {
    params: string[];
    body: Expr[];
}

class Environment {
    values = new Map<string, Value>();

    get(name: string): Value {
        return this.values.get(name)!;
    }

    set(name: string, val: Value) {
        this.values.set(name, val);
    }
}

function evaluate(node: Node | Expr, env: Environment)
{
  switch (node.type) {

    case "NumberLiteral":
      return node.value

    case "Identifier":
      if (!(node.name in env)) {
        throw new Error(`Undefined variable ${node.name}`)
      }
      return env[node.name]

    case "BinaryExpr": {
      const left = evaluate(node.left, env)
      const right = evaluate(node.right, env)

      switch (node.op) {
        case "+":
          return left + right
      }

      throw new Error("Unknown operator " + node.op)
    }

    case "CallExpr": {
      const fn = env[node.callee]

      if (!fn) {
        throw new Error(`Undefined function ${node.callee}`)
      }

      const args = node.args.map(arg => evaluate(arg, env))

      // builtin function
      if (typeof fn === "function") {
        return fn(...args)
      }

      // user defined function
      const localEnv = { ...env }

      fn.params.forEach((param: string, i: number) => {
        localEnv[param] = args[i]
      })

      try {
        for (const stmt of fn.body) {
          evaluate(stmt, localEnv)
        }
      } catch (err) {
        if (err instanceof ReturnSignal) {
          return err.value
        }
        throw err
      }

      return null
    }

    case "ReturnStmt": {
      const value = evaluate(node.value, env)
      throw new ReturnSignal(value)
    }

    default:
      throw new Error(`Unknown node type ${node.type}`)
  }
}


export function run(program: Node[]) {
    const env = createGlobalEnv();

    for (const node of program) {
        if (node.type === "FunctionDecl") {
            env[node.name] = {
                params: node.params,
                body: node.body,
            };
        } else {
            evaluate(node, env);
        }
    }
}
