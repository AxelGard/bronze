# Control flow

## if

```bronze
func sign(n: Int) -> String
{
    if n < 0
    {
        return "negative"
    }
    else if n == 0
    {
        return "zero"
    }
    else
    {
        return "positive"
    }
}

func main() -> Int
{
    print(sign(-3))
    print(sign(0))
    print(sign(4))
    return 0
}
```

```text title="Output"
negative
zero
positive
```

The condition must be a `Bool`, and needs no parentheses. The braces are always required.

## while

`while` repeats its block as long as the condition is true:

```bronze
mut total = 0
mut i = 1
while i <= 10
{
    total = total + i
    i = i + 1
}
print(total)
```

```text title="Output"
55
```

## for

`for` loops over a range of `Int`s. `a..b` counts from `a` up to but not including `b`, and `a..=b` includes `b`:

```bronze
for i in 0..3
{
    print(i)
}
for i in 1..=2
{
    print(i * 10)
}
```

```text title="Output"
0
1
2
10
20
```

Ranges only count up: if the end is not above the start, the range is empty and the loop doesn't run. The bounds are evaluated once, before the loop starts. Write `_` when you don't need the value:

```bronze
mut line = ""
for _ in 0..5
{
    line += "-"
}
print(line)
```

```text title="Output"
-----
```

The loop variable is like a `con`: it can't be assigned to, and it is visible only inside the loop's block. A `for`, like a `while`, never counts as a `return` at the end of a function.

## break and continue

`break` leaves the innermost `for` or `while`. `continue` skips the rest of the block and goes on to the next pass:

```bronze
mut n = 0
while true
{
    n += 1
    if n % 2 == 0
    {
        continue
    }
    if n > 7
    {
        break
    }
    print(n)
}
```

```text title="Output"
1
3
5
7
```

Outside a loop they are errors:

```bronze
func main() -> Int
{
    break
    return 0
}
```

```text title="Output"
error: `break` outside a loop
  --> main.brz:3:5
  |
3 |     break
  |     ^^^^^
```

## FizzBuzz

```bronze
func fizzbuzz(n: Int)
{
    mut i = 1
    while i <= n
    {
        if i % 3 == 0 and i % 5 == 0
        {
            print("FizzBuzz")
        }
        else if i % 3 == 0
        {
            print("Fizz")
        }
        else if i % 5 == 0
        {
            print("Buzz")
        }
        else
        {
            print(i)
        }
        i = i + 1
    }
}

func main() -> Int
{
    fizzbuzz(15)
    return 0
}
```

```text title="Output"
1
2
Fizz
4
Buzz
Fizz
7
8
Fizz
Buzz
11
Fizz
13
14
FizzBuzz
```

`match` is also a way to choose between cases; see [Enums and match](enums-and-match.md).

## Next

[Functions](functions.md).
