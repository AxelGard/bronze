# Projects

## Layout

```
myapp/
  bronze.toml          project file
  src/
    main.brz           func main() lives here
    billing/
      invoice.brz      namespace billing (from the folder)
      money.brz        namespace billing
  libs/                dependencies
  tests/               tests (test "..." { } blocks; they can also live next to the code)
```

- **Folder = namespace**: everything in `src/billing/` is the namespace `billing`.

## Imports

```
import billing                      // use as billing.Invoice
import billing.{Invoice, Money}     // only some names, used without a prefix
import shop.catalog as cat          // under a short name
```
