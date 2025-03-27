# Freighter Python Implementation

This is the Python implementation of the Freighter interface. It currently has unary
implementations in HTTP and a streaming implementation using WebSockets.

## Developing

Freighter uses Python 3.11 and [Poetry](https://python-poetry.org/) for dependency
management. To install Poetry and the freighter dependencies, run:

```sh
pip install poetry && poetry install
```

## Testing

Unit tests are held in the [`tests`](tests) directory. To run the test suite, you need a
running instance of the [Freighter integration server](../integration/README.md).

Once the integration server is running, you can run the test suite with:

```sh
poetry run pytest
```
