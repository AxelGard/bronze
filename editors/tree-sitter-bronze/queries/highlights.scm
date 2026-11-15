; Highlights for Bronze. Later patterns take priority in Neovim, so the
; general ones come first.

(identifier) @variable

; types
(type name: (identifier) @type)
((type name: (identifier) @type.builtin)
  (#any-of? @type.builtin "Int" "Float" "Bool" "String" "Error" "List" "Map"))
(class_declaration name: (identifier) @type)
(class_declaration use: (identifier) @type)
(enum_declaration name: (identifier) @type)
(interface_declaration name: (identifier) @type)
(data_declaration name: (identifier) @type)
(data_declaration use: (identifier) @type)
(interface_declaration use: (identifier) @type)
(method_signature name: (identifier) @function)
(is_expression class: (identifier) @type)

; enum cases: `Status.Active` and in patterns
(enum_case name: (identifier) @constant)
(pattern enum: (identifier) @type case: (identifier) @constant)
((member_expression
  object: (identifier) @type
  member: (identifier) @constant)
  (#match? @type "^[A-Z]"))

; functions
(function_declaration name: (identifier) @function)
((function_declaration name: (identifier) @constructor)
  (#eq? @constructor "init"))
(call_expression function: (identifier) @function.call)
(call_expression function: (member_expression member: (identifier) @function.method.call))
((call_expression function: (identifier) @function.builtin)
  (#any-of? @function.builtin "print" "panic" "assert"))
((call_expression function: (identifier) @constructor)
  (#match? @constructor "^[A-Z]"))

; fields and parameters
(field_declaration name: (identifier) @variable.member)
(member_expression member: (identifier) @variable.member)
(parameter name: (identifier) @variable.parameter)
(for_statement variable: (identifier) @variable)
(else_handler error: (identifier) @variable.parameter)
(this) @variable.builtin
"super" @variable.builtin
(super class: (identifier) @type)

; literals
(integer) @number
(float) @number.float
(string) @string
(escape_sequence) @string.escape
(interpolation ["{" "}"] @punctuation.special)
(true) @boolean
(false) @boolean
(null) @constant.builtin
((pattern (identifier) @character.special)
  (#eq? @character.special "_"))

(comment) @comment

; keywords
["func" "return"] @keyword.function
["con" "mut" "class" "enum" "use" "static" "override" "interface" "data" "copy"] @keyword
(visibility) @keyword.modifier
["if" "else" "match"] @keyword.conditional
["while" "for" "in" (break_statement) (continue_statement)] @keyword.repeat
["and" "or" "not" "is"] @keyword.operator

; operators and punctuation
["+" "-" "*" "/" "%" "==" "!=" "<" "<=" ">" ">=" "=" "+=" "-=" "*=" "/=" "%=" ".." "..=" "->" "=>" "!" "?"] @operator
["(" ")" "{" "}" "[" "]"] @punctuation.bracket
["," "." ":"] @punctuation.delimiter
