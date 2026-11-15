# Built-in types

Worksheet: [`basics.brz`](explorations/basics.brz).

## Numbers

| Type | Meaning |
|---|---|
| `Int` | the default integer, an alias for **`Int64`** |
| `Int8` `Int16` `Int32` `Int64` | sized signed integers |
| `UInt8` `UInt16` `UInt32` `UInt64` | sized unsigned integers |
| `Float` | the default float, an alias for **`Float64`** |
| `Float32` `Float64` | sized floats |

- **Overflow wraps around** silently (like Go, C and Java).
- `Int / Int` gives an `Int`, rounded toward zero: `7 / 2` is `3`, `-7 / 2` is `-3`.
- **No mixed arithmetic**: `7 / 2.0` is a compile error. Convert explicitly: `Float(7) / 2.0`.
- **Integer division by zero panics.** Float division by zero gives `inf` (or `NaN`).
- Literals: `1_000_000`, `0xFF`, `0b1010`, `0o755`, `1.5e-3`.

## Bool

`true` and `false`.

## Bytes

- **`Byte`**: a value from 0 to 255.
- **`Bytes`**: raw bytes, for files, HTTP bodies, hashing and images.
- Turning `Bytes` into a `String` can fail (not valid UTF-8): `logo.to_string() else ""`.

## Strings

- **UTF-8.** `length()` counts characters, and `byte_length()` counts bytes.
- Joined with `+`: `"hello, " + name`.
- **Interpolation** with `{}`: `"hello {name}, next year you are {age + 1}"`.
- **Escapes**: `\n`, `\t`, `\"`, `\\`, and `\{` for a literal brace.
- **Multi-line strings** with triple quotes. Leading indentation is removed:

```
con query = """
    select id, email
    from users
    where active = true
    """
```

## Collections

```
con nums = [1, 2, 3]                    // List<Int>
con ages = {"Ada": 36, "Grace": 85}     // Map<String, Int>
con tags = Set(["a", "b"])              // Set<String>
con empty: List<String> = []
```

- **Indexing returns `T?`**: `nums[0]` is an `Int?`, and `ages["Zed"]` is an `Int?`. The compiler makes you handle the missing case: `nums[10] else 0`.
- **Negative indexes** count from the end: `word[-1]` is the last character.
- **Slicing** with ranges, for strings and lists: `word[2..5]`, `word[2..=5]`, `word[..3]`, `word[2..]`. Slices are clamped and never fail: `word[2..100]` is everything from index 2. A slice of a list is a new list.
- Indexing a string gives a one-character `String?`. There is no `Char` type.
- **Shared**, like objects: assigning or passing a list or map never copies it. `con` fixes the variable, not the collection.
- **`==` compares by value.**
- **Maps keep insertion order.** Keys can be `Int`, `String`, `Bool` or enums.
- `[]` and `{}` need a known type, like `null`: `con ages: Map<String, Int> = {}`.
- Printed as `[1, 2, 3]`, `["a", "b"]` and `{"Ada": 36}`, with strings quoted.

## Enums

A fixed set of named values. Cases can have a **value of one type**, and enums can have **methods**:

```
enum Status
{
    Pending
    Paid
    Shipped
    Cancelled
}

enum Role: String
{
    Admin = "admin"
    Member = "member"

    func can_delete() -> Bool
    {
        return this == Role.Admin
    }
}

con s = Status.Paid
con r = Role.Admin
print(r.can_delete())     // true
```

- The value type is `Int` or `String`, and every case has a value, written out. No two cases have the same value.
- `r.value` gives the value, and `Role.from("admin")` gives a `Role?`. `print(r)` shows the case name.

Enums whose cases carry their own data (`Circle(radius: Float)`) are not part of Bronze for now.
