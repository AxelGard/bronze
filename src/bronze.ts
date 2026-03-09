import { tokenize } from "./lexer.ts"
import { parse } from "./parser.ts"
import { run } from "./interpreter.ts"

const code = await Deno.readTextFile(Deno.args[0])

const tokens = tokenize(code)
console.log(tokens)
const ast = parse(tokens)
console.log(ast)

run(ast)
