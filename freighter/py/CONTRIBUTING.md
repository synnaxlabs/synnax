# Freighter Python Implementation

This is the Python implementation of the Freighter interface. It currently has unary
implementations in HTTP and a streaming implementation using WebSockets.

## Developing

Freighter uses Python 3.12+ and [uv](https://docs.astral.sh/uv/) for dependency
management. To install the dependencies, run:

```sh
uv sync
```

## Testing

Unit tests are held in the [`tests`](tests) directory. To run the test suite, you need a
running instance of the [Freighter integration server](../integration/README.md).

Once the integration server is running, you can run the test suite with:

```sh
uv run pytest
```
