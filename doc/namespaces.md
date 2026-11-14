# Namespaces and visibility

Example: [`namespaces.brz`](explorations/namespaces.brz).

## Namespaces

Namespaces group classes, data classes, functions and constants under one name. They are a good fit for DDD, with one namespace per bounded context.

```
namespace billing
{
    con VAT = 0.25

    data LineItem(name: String, price: Money, count: Int)

    class Invoice { ... }

    func total(items: List<LineItem>) -> Money { ... }
}

con invoice = billing.Invoice()
```

## Visibility

Everything is **public by default**.

| Keyword | Visible to |
|---|---|
| *(none)* | everyone |
| **`private`** | only the class, data class or namespace it is declared in |
| **`protected`** | the class **and its children** (class members only) |

This applies to fields, methods and functions.

```
namespace accounts
{
    class Account
    {
        mut balance = 0                  // public
        private mut pin = ""             // only Account
        protected func fee() -> Int      // Account and its children
        {
            return 0
        }
    }

    func open_account() -> Account { ... }      // public: accounts.open_account()
    private func log(msg: String) { ... }       // only inside accounts
}
```
