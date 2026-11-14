# Spec for 0.2.0

The exact language that version 0.2.0 implements. It is [spec-0.1](spec-0.1.md) plus the changes on this page: everything in 0.1 still holds unless this page says otherwise.

0.2.0 adds `T?` for all types, ranges, `for`, `break` and `continue`, lists, maps, indexing and slicing, string methods, string interpolation, compound assignment (`+=` and friends), and hover in the language server. The choices come from [`worksheet-0.2.brz`](explorations/worksheet-0.2.brz).

Where this page makes a choice the worksheet didn't, it says **(0.2 choice)**. Those can change later.

## Tokens

- `for`, `in`, `break` and `continue` become keywords. They are no longer only reserved.
- New operators: `..` `..=` `+=` `-=` `*=` `/=` `%=`
- New punctuation: `[` `]`
- `<` and `>` also close and open type arguments, in types only: `List<Int>`, `Map<String, List<Int>>`. `>>` is two `>` tokens, since Bronze has no shift operators.
- A `{` in a string literal now starts an interpolation (see String interpolation). `\{` is still a literal brace, and a `}` outside an interpolation is a literal `}`.

The other reserved words of 0.1 stay reserved.

## Newlines

In addition to the 0.1 rules:

- **Inside `[ ]`** there are no `NEWLINE` tokens, like inside `( )`. So a list literal or an index may span lines.
- **Inside a map literal**, the parser skips newlines after `{`, around `,` and before `}`. (A map literal's `{` can't be told apart from a block's `{` by the lexer, so this is done by the parser.)
- **(0.2 choice)** A list or map literal may have a trailing `,` before its `]` or `}`. Argument lists may not.

```
con ages = {
    "Ada": 36,
    "Grace": 85,
}
```

## Grammar changes

Only the changed and new rules. Everything else is as in 0.1.

```
type        = type_name [ "?" ]
type_name   = "Int" | "Float" | "Bool" | "String"
            | "List" "<" type ">"
            | "Map" "<" type "," type ">"
            | IDENT                                 // a class, an enum, or Error

stmt        = var_decl
            | assign
            | compound
            | if_stmt
            | while_stmt
            | for_stmt
            | "break"
            | "continue"
            | return_stmt
            | match_expr
            | expr

assign      = target "=" expr
compound    = target ( "+=" | "-=" | "*=" | "/=" | "%=" ) expr
target      = IDENT | postfix "." IDENT | postfix "[" expr "]"

for_stmt    = "for" loop_var [ "," loop_var ] "in" expr block
loop_var    = IDENT | "_"

or_expr     = and_expr { "or" and_expr }
and_expr    = cmp_expr { "and" cmp_expr }
cmp_expr    = range_expr [ ( "==" | "!=" | "<" | "<=" | ">" | ">=" ) range_expr ]
            | range_expr "is" IDENT
range_expr  = add_expr [ ( ".." | "..=" ) add_expr ]   // no chaining
add_expr    = mul_expr { ( "+" | "-" ) mul_expr }
mul_expr    = unary { ( "*" | "/" | "%" ) unary }
unary       = ( "not" | "-" ) unary | postfix
postfix     = primary { "(" [ args ] ")" | "." IDENT | "[" index "]" }
index       = expr                                  // an index
            | [ add_expr ] ( ".." | "..=" ) [ add_expr ]   // a slice
primary     = INT | FLOAT | STRING | "true" | "false" | "null" | "this"
            | IDENT | "(" expr ")" | match_expr
            | list_lit | map_lit

list_lit    = "[" [ expr { "," expr } [ "," ] ] "]"
map_lit     = "{" [ entry { "," entry } [ "," ] ] "}"
entry       = expr ":" expr
```

Notes:
- `a..b..c` is a syntax error, like chained comparisons.
- In `index`, a `..` or `..=` makes it a slice. `xs[..]` is allowed and copies the whole list (both ends left out).
- `..=` needs an end: `xs[2..=]` is an error.
- `{` in expression position is a map literal: in `if m == {} { ... }`, the first `{}` is a map and the second is the block. The one exception is a `match` arm: a `{` right after `=>` is always a block, so an arm whose value is a map literal writes it in parentheses, `_ => ({})`. **(0.2 choice)**
- `List` and `Map` are not keywords. They are type names, like `Int`. Declaring a class or enum named `List` or `Map` is an error.

### Precedence

From highest to lowest: calls, `.` and `[]`; `not` and `-x`; `* / %`; `+ -`; `.. ..=`; comparisons and `is`; `and`; `or`; `else`. This is the order in [syntax.md](syntax.md).

## T? for all types

`T?` is now allowed for every type: `Int?`, `Float?`, `Bool?`, `String?`, `List<T>?`, `Map<K, V>?`, and classes, enums and `Error` as before. Everything in spec-0.1's Null section applies to all of them: narrowing, `else` for null, and the error that a `T?` "may be null" where a `T` is needed.

```
func first_or_zero(nums: List<Int>) -> Int
{
    return nums[0] else 0
}
```

- `T??` is still not allowed.
- **(0.2 choice)** The element type of a list, and the key and value types of a map, can't be a `T?`: `List<Int?>` is an error. Otherwise `nums[0]` on a `List<Int?>` would be an `Int??`. This may be relaxed later.
- **(0.2 choice)** `==` and `!=` also compare a `T?` with a `T`: `nums[0] == 5` is true when the element exists and is `5`. `null` equals only `null`.
- `print`, `String(...)` and interpolation don't take a `T?`. Handle the null first.

## Ranges

`a..b` is the `Int`s from `a` up to but not including `b`. `a..=b` includes `b`. Both operands must be `Int`s.

- In a `for` loop, a range is looped over directly.
- In `[ ]`, a range is a slice (see Slicing).
- Anywhere else, a range is a **`List<Int>`**: `con r = 0..5` is `[0, 1, 2, 3, 4]`. There is no `Range` type.
- If `b` is not above `a` (for `..`) or is below `a` (for `..=`), the range is empty: `5..2` is `[]`. Ranges only count up. **(0.2 choice)**
- The bounds are evaluated once, before the loop or the list is made.
- Ranges in `match` patterns (`1..=9 =>`) are not in 0.2.

## Lists

```
con nums = [1, 2, 3]                  // List<Int>
con names: List<String> = []
mut grid: List<List<Int>> = [[1, 2], [3, 4]]
```

- A list literal's type is `List<T>`, where `T` is the type of its elements. All elements must have the same type: "list elements have different types: Int and String". With an expected type, such as an annotation, elements need only fit that type, so `con errs: List<Error> = [NotFound(1), Error("x")]` works.
- `[]` needs a type, like `null`. It gets one where the expected type is known: an annotated `con`, `mut` or field, an argument, a `return`, an assignment, or a comparison with a typed value. Otherwise: "`[]` needs a type here".
- **Lists are shared**, like objects: assigning or passing a list never copies it. `con` fixes the variable, not the list, so a `con` list can still be changed.
- **(0.2 choice)** `List<T>` is only the same type as `List<T>`. A `List<NotFound>` can't be passed as a `List<Error>`, since the callee could then append an `Error` to it.

### Indexing

`xs[i]` with an `Int` `i` gives a **`T?`**: the element, or `null` if `i` is out of range.

- Negative indexes count from the end: `xs[-1]` is the last element, `xs[-length]` the first.
- `xs[i] = v` replaces an element. Out of range (after counting negative indexes from the end) it **panics**: "index 5 out of range for a list of length 3".

### Slicing

`xs[a..b]`, `xs[a..=b]`, `xs[..b]`, `xs[a..]` and `xs[..]` give a **new list** with the elements in that range. The new list is a copy: changing it doesn't change `xs` (the elements themselves are shared, if they are objects or lists).

- **Slices never fail.** Negative bounds count from the end, as for indexing, and then both bounds are clamped to `0..length`: `[1, 2, 3][1..100]` is `[2, 3]`, `[1, 2, 3][5..]` is `[]`. If the start is past the end after clamping, the slice is empty. **(0.2 choice:** negative bounds in slices.)

### Methods

| Method | Does |
|---|---|
| `length() -> Int` | the number of elements |
| `append(x: T)` | adds `x` at the end |
| `insert(i: Int, x: T)` | inserts `x` before index `i`, from `0` to `length()`. Anything else panics: "index 5 out of range for insert into a list of length 3". **(0.2 choice:** no negative `i`.) |
| `remove_at(i: Int) -> T?` | removes the element at `i` (negative from the end) and returns it, or returns `null` if out of range |
| `pop() -> T?` | removes the last element and returns it, or `null` if the list is empty |
| `contains(x: T) -> Bool` | whether an element `== x` |
| `index_of(x: T) -> Int?` | the index of the first element `== x`, or `null` |
| `reverse()` | reverses the list in place |
| `sort()` | sorts the list in place, ascending with `<`. Only on `List<Int>`, `List<Float>` and `List<String>`. (`NaN`s go first.) |
| `join(sep: String) -> String` | only on `List<String>`: the elements with `sep` between them |

`reverse` and `sort` change the list and return nothing.

## Maps

```
con ages = {"Ada": 36, "Grace": 85}   // Map<String, Int>
con empty: Map<String, Int> = {}
```

- A map literal's type is `Map<K, V>`. All keys must have the same type, and so must all values.
- **Keys** can be `Int`, `String`, `Bool` or an enum. Other key types are an error: "a map key can be an Int, String, Bool or enum, not Float".
- `{}` needs a type, like `[]`.
- A key written twice in a literal is an error, when both are literals or enum cases. **(0.2 choice)** (If two computed keys turn out equal at runtime, the later one wins.)
- **Maps are shared**, like lists and objects.
- **Order**: maps keep **insertion order**. Looping, `keys()`, `values()` and `print` go in the order keys were first added. Replacing a value keeps the key's place. Removing a key and adding it again puts it last.

### Indexing

- `m[k]` gives a **`V?`**: the value, or `null` if the key isn't there.
- `m[k] = v` adds the key, or replaces its value.

### Methods

| Method | Does |
|---|---|
| `length() -> Int` | the number of keys |
| `contains(key: K) -> Bool` | whether the key is there |
| `remove(key: K) -> V?` | removes the key and returns its value, or `null` if it wasn't there |
| `keys() -> List<K>` | a new list of the keys |
| `values() -> List<V>` | a new list of the values |

## Strings

Strings are UTF-8 and can't be changed. **Lengths and indexes count characters** (Unicode code points), not bytes.

- `s[i]` gives a **`String?`** with the one character at `i`, or `null`. Negative indexes count from the end.
- `s[a..b]` and the other slices work as for lists, clamped, and give a `String`.
- Assigning to `s[i]` is an error: "strings can't be changed".

### Methods

| Method | Does |
|---|---|
| `length() -> Int` | the number of characters |
| `contains(s: String) -> Bool` | whether `s` appears in the string |
| `starts_with(s: String) -> Bool`, `ends_with(s: String) -> Bool` | |
| `index_of(s: String) -> Int?` | the character index of the first `s`, or `null` |
| `split(sep: String) -> List<String>` | the parts between each `sep`. `"a,b,,c".split(",")` is `["a", "b", "", "c"]`. **(0.2 choice)** An empty `sep` splits into characters |
| `trim() -> String` | without whitespace at the start and end |
| `to_upper() -> String`, `to_lower() -> String` | Unicode upper and lower case |
| `replace(old: String, new: String) -> String` | every `old` replaced by `new`. **(0.2 choice)** An empty `old` gives the string unchanged |

## for

```
for n in nums               // each element
for i, n in nums            // index and element
for name, age in ages       // key and value
for name in ages            // keys
for c in "bronze"           // each character, as a String
for i, c in "bronze"        // index and character
for i in 0..10              // 0 to 9
for i in 0..=10             // 0 to 10
for _ in 0..3               // three times
```

- The expression after `in` is a `List<T>`, a `Map<K, V>`, a `String`, or a range. A `T?` must be handled first.
- **(0.2 choice)** One variable over a map gives the keys. Looping over a `String` gives one-character `String`s. Two variables over a range is an error.
- **Loop variables are like `con`**: assigning to one is an error. They are visible only in the loop's block, and follow the usual rule that a name can't be declared again while it is visible.
- `_` as a loop variable ignores that value. (A lone `_` is otherwise still only a `match` pattern.)
- A `for` never counts as returning, like `while` (see Returns in spec-0.1).

### Changing what you loop over

While a `for` loops over a list or a map, **changing its length or order panics**: "list changed while looping over it" (or "map changed ..."). That covers `append`, `insert`, `remove_at`, `pop`, `reverse` and `sort` on the list, and adding a key with `m[k] = v` or `remove` on the map. The panic happens at the call or assignment that makes the change.

Replacing an element (`nums[i] = x`) or the value of a key that is already there (`ages[k] = v`) is allowed. **(0.2 choice)**

This holds for any loop over the list, including one in a calling function. A list that is not being looped over can be changed freely.

## break and continue

- `break` leaves the innermost `for` or `while`. `continue` goes on to its next pass.
- Outside a loop they are errors: "`break` outside a loop".
- In a `match` statement inside a loop, they act on the loop.

## Compound assignment

`target op= expr`, for `+=`, `-=`, `*=`, `/=` and `%=`, wherever `target = expr` works: variables, fields, and list and map elements.

- It means `target = target op expr`, with the same type rules as the operator: `+=` on `String` joins, `%=` only on `Int`, and no mixing of `Int` and `Float`.
- The parts of the target are evaluated once: in `nums[next()] += 1`, `next()` is called once.
- On a list element, an index out of range panics, as for `=`. **(0.2 choice)** On a map element, a missing key panics: "key \"Zed\" is not in the map".
- Compound assignment is a statement, like `=`. It has no value.

## Equality

- `==` on lists compares **by value**: the same length, and elements equal by their own `==` (objects by identity, as always).
- `==` on maps compares by value: the same keys, each with an equal value. The order doesn't matter. **(0.2 choice)**
- `<`, `<=`, `>` and `>=` don't work on lists or maps.

## String interpolation

```
con name = "Ada"
print("{name} is {36 + 1}")       // Ada is 37
print("first: {nums[0] else 0}")
```

- `{expr}` in a string literal is replaced by `String(expr)`. Any expression works, as long as its type can be converted: `Int`, `Float`, `Bool`, `String`, enums, and lists and maps of those.
- An interpolated expression may contain string literals, with their own interpolations: `"{ages["Ada"] else 0}"`.
- An interpolation can't span lines, and `{}` with nothing inside is an error.
- `\{` is a literal `{`.

## print and String()

`print`, `String(...)` and interpolation now also take lists and maps whose elements, keys and values they take.

| Type | Printed as |
|---|---|
| `List<T>` | `[1, 2, 3]`, `[]`. Nested: `[[1, 2], [3]]` |
| `Map<K, V>` | `{"Ada": 36, "Grace": 85}`, `{}` |
| `String` inside a list or map | quoted, with `\n`, `\t`, `\"` and `\\` escaped: `["a, b", "c"]` |

Other types print inside a list or map as they print on their own: `[1.0, 2.5]`, `[Active, Frozen]`. A `String` on its own still prints as it is.

## Panics

New panic messages, in addition to 0.1's:

| Cause | Message |
|---|---|
| `xs[i] = v` or `xs[i] op= v` out of range | `index 5 out of range for a list of length 3` |
| `insert` out of range | `index 5 out of range for insert into a list of length 3` |
| `m[k] op= v` with a missing key | `key "Zed" is not in the map` (the key as `print` would show it in a list) |
| changing a list or map while looping over it | `list changed while looping over it`, `map changed while looping over it` |

## Language server

`bronze lsp` adds **hover**: on a name, it shows what the name is.

| Name | Hover |
|---|---|
| variable, parameter, loop variable | `con total: Int`, `mut names: List<String>`, `n: Int` |
| field | `mut balance: Int` (in `Account`) |
| function, method | its signature: `func withdraw(amount: Int) -> Int ! Error` |
| class, enum | `class Account`, `enum Status` |
| enum case | `Status.Frozen` |

The hover text is a `bronze` code block. Hover works on declarations and on uses. Built-ins (`print`, `List.append`, ...) show their signature too.

## Not in 0.2

Still for later: multi-line strings, `Set`, ranges as `match` patterns, `Int(Float)`, top-level `con`, and everything the [milestones](milestones.md) put in 0.3 and after.
