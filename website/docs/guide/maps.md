# Maps

A `Map<K, V>` maps keys of type `K` to values of type `V`.

```bronze
con ages = {"Ada": 36, "Grace": 85}
print(ages)
print(ages["Ada"] else 0)
ages["Linus"] = 54
print(ages.length())
```

```text title="Output"
{"Ada": 36, "Grace": 85}
36
3
```

The type of `{"Ada": 36, "Grace": 85}` is `Map<String, Int>`. All keys must have the same type, and so must all values. Keys can be `Int`, `String`, `Bool` or an enum. A map literal may span lines, and may end with a `,`:

```bronze
con ages = {
    "Ada": 36,
    "Grace": 85,
}
print(ages)
```

```text title="Output"
{"Ada": 36, "Grace": 85}
```

Like `[]`, the empty map `{}` needs a type from where it is used: `con counts: Map<String, Int> = {}`. A key written twice in a literal is an error.

`{` where an expression is expected starts a map, so in `if m == {} { ... }` the first `{}` is a map and the second is the block. In a `match` arm, a `{` right after `=>` is always a block, so a map there is written in parentheses: `_ => ({})`.

## Indexing

`m[k]` is the value for the key `k`, or `null` if the key isn't there, so it is a `V?`. `m[k] = v` adds the key, or replaces its value:

```bronze
con stock = {"apples": 3}
stock["pears"] = 5
stock["apples"] += 1
print(stock["apples"] else 0)
print(stock["plums"] == null)
print(stock)
```

```text title="Output"
4
true
{"apples": 4, "pears": 5}
```

`m[k] += v` needs the key to be there: with a missing key it panics with `key "plums" is not in the map`.

## Order

Maps keep their keys in the order they were first added. Looping, `keys()`, `values()` and `print` all go in that order. Replacing a value keeps the key's place; removing a key and adding it again puts it last.

## Methods

| Method | Does |
|---|---|
| `length() -> Int` | the number of keys |
| `contains(key: K) -> Bool` | whether the key is there |
| `remove(key: K) -> V?` | removes the key and returns its value, or `null` if it wasn't there |
| `keys() -> List<K>` | a new list of the keys |
| `values() -> List<V>` | a new list of the values |

## for over a map

`for k, v in m` gives each key and its value. With one variable, `for k in m` gives the keys:

```bronze
func count(words: List<String>) -> Map<String, Int>
{
    con counts: Map<String, Int> = {}
    for w in words
    {
        if counts.contains(w)
        {
            counts[w] += 1
        }
        else
        {
            counts[w] = 1
        }
    }
    return counts
}

func main() -> Int
{
    for word, n in count(["to", "be", "or", "not", "to", "be"])
    {
        print(word + ": " + String(n))
    }
    return 0
}
```

```text title="Output"
to: 2
be: 2
or: 1
not: 1
```

While a `for` loops over a map, adding a key or removing one panics with `map changed while looping over it`. Replacing the value of a key that is already there is fine.

## Maps are shared

Like lists and objects, maps are shared: assigning or passing a map never copies it.

`==` compares maps by value: the same keys, each with an equal value, in any order. `print` and `String(...)` show a map as `{"Ada": 36}`, with `String` keys and values quoted.

## Next

[Panics](panics.md).
