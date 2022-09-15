
# Freighter Python Implementation

This is the Python implementation of the Freighter interface. It currently has 
unary implementations in HTTP and a streaming implementation using WebSockets.

## Developing

Freighter uses Python 3.10 and [Poetry](https://python-poetry.org/) for dependency 
management. To install Poetry and the freighter dependencies, run:

```
pip install poetry && poetry install
```

Then, to spawn a shell into the virtual environment, run:

```
poetry shell
```

## Testing

Unit tests are held in the `tests` directory. To run the test suite, first spawn
a shell into the virtual environment (see above) and then run:

```
pytest
```

To get a coverage report, run:

```
pytest --cov=freighter
```


