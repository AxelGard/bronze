# Lists

A `List<T>` holds values of one type `T`, in order.

```bronze
con primes = [2, 3, 5, 7]
print(primes)
print(primes.length())
for p in primes
{
    print(p * 10)
}
```

```text title="Output"
[2, 3, 5, 7]
4
20
30
50
70
```

The type of `[2, 3, 5, 7]` is `List<Int>`. All the elements must have the same type. An empty list `[]` has no type of its own, so it needs one from where it is used:

```bronze
con names: List<String> = []
names.append("Ada")
names.append("Grace")
print(names)
```

```text title="Output"
["Ada", "Grace"]
```

The type can come from a written type on a variable or field, a parameter, a return type, an assignment, or the other side of `==`. Where there is none, `[]` is an error:

```bronze
func main() -> Int
{
    con names = []
    return 0
}
```

```text title="Output"
error: `[]` needs a type here
  --> main.brz:3:17
  |
3 |     con names = []
  |                 ^^
  |
  = help: write the type where it is declared, like `con names: List<String> = []`
```

A list literal may span lines, and may end with a `,`. Lists can hold lists: `List<List<Int>>`. The elements can't be a `T?`: `List<Int?>` is an error.

## Indexing

`xs[i]` is the element at `i`, counting from `0`. Negative indexes count from the end: `xs[-1]` is the last element. The index may be out of range, so `xs[i]` is a `T?`: the element, or `null`. Handle the null like any other, often with `else`:

```bronze
func first_or_zero(nums: List<Int>) -> Int
{
    return nums[0] else 0
}

func main() -> Int
{
    con nums = [4, 5, 6]
    print(first_or_zero(nums))
    print(first_or_zero([]))
    print(nums[-1] else 0)
    print(nums[10] == null)
    print(nums[1] == 5)
    return 0
}
```

```text title="Output"
4
0
6
true
true
```

`xs[i] = v` replaces an element, and `xs[i] += v` changes it. An index out of range there panics, since there is nothing to replace:

```bronze
func main() -> Int
{
    con nums = [1, 2, 3]
    nums[0] += 10
    print(nums)
    nums[3] = 4
    return 0
}
```

```text title="Output"
[11, 2, 3]
panic: index 3 out of range for a list of length 3
  at main (main.brz:6)
```

## Slicing

`xs[a..b]` is a new list with the elements from `a` up to but not including `b`, and `xs[a..=b]` includes `b`. Either end can be left out: `xs[..b]`, `xs[a..]`, and `xs[..]` for a copy of the whole list (but `..=` needs an end).

```bronze
con xs = [10, 20, 30, 40, 50]
print(xs[1..3])
print(xs[..2])
print(xs[3..])
print(xs[-2..])
print(xs[1..100])
```

```text title="Output"
[20, 30]
[10, 20]
[40, 50]
[40, 50]
[20, 30, 40, 50]
```

Slices never fail. Negative bounds count from the end, and then both bounds are clamped to the list, so a slice past the end is just shorter, or empty. Changing the new list doesn't change the old one, but objects and lists inside are still shared.

## Lists are shared

Like objects, lists are shared: assigning or passing a list never copies it. `con` fixes the variable, not the list, so a `con` list can still be changed.

```bronze
func add_one(nums: List<Int>)
{
    nums.append(1)
}

func main() -> Int
{
    con nums = [5]
    con same = nums
    add_one(same)
    print(nums)
    return 0
}
```

```text title="Output"
[5, 1]
```

A `List<T>` is only the same type as a `List<T>`. A list of `NotFound` errors can't be passed as a `List<Error>`, because the function could then add a different kind of error to it.

## Methods

| Method | Does |
|---|---|
| `length() -> Int` | the number of elements |
| `append(x: T)` | adds `x` at the end |
| `insert(i: Int, x: T)` | inserts `x` before index `i`, from `0` to `length()`; anything else panics |
| `remove_at(i: Int) -> T?` | removes the element at `i` (negative from the end) and returns it, or `null` if out of range |
| `pop() -> T?` | removes the last element and returns it, or `null` if the list is empty |
| `contains(x: T) -> Bool` | whether an element `== x` |
| `index_of(x: T) -> Int?` | the index of the first element `== x`, or `null` |
| `reverse()` | reverses the list in place |
| `sort()` | sorts the list in place, smallest first; only for lists of `Int`, `Float` or `String` |
| `join(sep: String) -> String` | only for a `List<String>`: the elements with `sep` between them |

```bronze
con nums = [3, 1, 2]
nums.sort()
print(nums)
nums.reverse()
print(nums)
print(nums.pop() else 0)
print(nums.index_of(3) else -1)
con words = ["a", "b", "c"]
print(words.join(", "))
```

```text title="Output"
[1, 2, 3]
[3, 2, 1]
1
0
a, b, c
```

## for over a list

`for x in xs` goes through the elements, and `for i, x in xs` also gives the index:

```bronze
con names = ["Ada", "Grace"]
for i, name in names
{
    print(String(i) + ": " + name)
}
```

```text title="Output"
0: Ada
1: Grace
```

While a `for` loops over a list, changing its length or order panics: `append`, `insert`, `remove_at`, `pop`, `reverse` and `sort`. Replacing an element with `xs[i] = v` is fine.

```bronze
func main() -> Int
{
    con nums = [1, 2]
    for n in nums
    {
        nums.append(n)
    }
    return 0
}
```

```text title="Output"
panic: list changed while looping over it
  at main (main.brz:6)
```

## Equality and printing

`==` compares lists by value: the same length, with equal elements. Objects in a list are still compared by identity. `<` and the other orderings don't work on lists.

`print` and `String(...)` take lists of the types they take. Strings inside a list are quoted:

```bronze
print([1, 2] == [1, 2])
print([[1.5], []])
print(["a, b", "c"])
```

```text title="Output"
true
[[1.5], []]
["a, b", "c"]
```

## Ranges as lists

Outside a `for`, a range is a `List<Int>`:

```bronze
con digits = 0..5
print(digits)
print(1..=3)
```

```text title="Output"
[0, 1, 2, 3, 4]
[1, 2, 3]
```

## Next

[Maps](maps.md).
