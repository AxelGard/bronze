# Strings

A `String` is UTF-8 text, and can't be changed. Lengths and indexes count characters (Unicode code points), not bytes.

## Interpolation

`{expr}` in a string is replaced by `String(expr)`:

```bronze
con name = "Ada"
con nums = [1, 2, 3]
print("{name} is {36 + 1}")
print("first: {nums[0] else 0}, all: {nums}")
print("shouting: {name.to_upper()}")
```

```text title="Output"
Ada is 37
first: 1, all: [1, 2, 3]
shouting: ADA
```

Any expression works, as long as `String(...)` takes its type: `Int`, `Float`, `Bool`, `String`, enums, and lists and maps of those. A `T?` must be handled first, as in `{nums[0] else 0}`. The expression may contain strings, with their own interpolations. An interpolation can't span lines, and `{}` with nothing inside is an error. `\{` is a literal `{`, and a `}` on its own is a literal `}`:

```bronze
print("a set: \{1, 2}")
```

```text title="Output"
a set: {1, 2}
```

## Indexing and slicing

`s[i]` is the character at `i`, as a one-character `String?`: `null` if `i` is out of range. Negative indexes count from the end. A slice, like `s[1..4]`, works as it does for [lists](lists.md#slicing), and gives a `String`:

```bronze
con word = "héllo"
print(word.length())
print(word[1] else "?")
print(word[-1] else "?")
print(word[1..4])
print(word[..2])
```

```text title="Output"
5
é
o
éll
hé
```

Assigning to `s[i]` is an error: strings can't be changed. Build a new one instead, with `+`, slices or `replace`.

## Methods

| Method | Does |
|---|---|
| `length() -> Int` | the number of characters |
| `contains(s: String) -> Bool` | whether `s` appears in the string |
| `starts_with(s: String) -> Bool` | whether the string starts with `s` |
| `ends_with(s: String) -> Bool` | whether the string ends with `s` |
| `index_of(s: String) -> Int?` | the character index of the first `s`, or `null` |
| `split(sep: String) -> List<String>` | the parts between each `sep`; an empty `sep` splits into characters |
| `trim() -> String` | without whitespace at the start and end |
| `to_upper() -> String`, `to_lower() -> String` | in upper or lower case |
| `replace(old: String, new: String) -> String` | every `old` replaced by `new`; an empty `old` changes nothing |

```bronze
con line = "  apples,pears,,plums  "
con parts = line.trim().split(",")
print(parts)
print(parts.length())
print("Bronze".to_upper())
print("banana".replace("an", "AN"))
print("bronze".index_of("nz") else -1)
```

```text title="Output"
["apples", "pears", "", "plums"]
4
BRONZE
bANANa
3
```

## for over a String

`for c in s` goes through the characters, each as a one-character `String`. `for i, c in s` also gives the index:

```bronze
mut reversed = ""
for c in "stressed"
{
    reversed = c + reversed
}
print(reversed)
```

```text title="Output"
desserts
```

## Next

[Panics](panics.md).
