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
