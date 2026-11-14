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

`while` is the only loop for now. `for`, `break` and `continue` come in 0.2.0.

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
