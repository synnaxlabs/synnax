### Setup

Before starting, make sure you've instantiated a `Synnax` client. If you've [logged in
via the CLI](/reference/client-cli/get-started), you can use the `Synnax` client without any arguments.

```python
import synnax as sy

client = sy.Synnax()
```

If not, provide your connection parameters.

```python
client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
)
```


