### Setup

Start by instantiating a `Synnax` client. If you've [logged in
via the CLI](/reference/client-cli/get-started), there's no need to provide any
arguments.

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


