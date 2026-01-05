# Contributing to the Synnax Python Client

If you haven't already, read the [Synnax Contribution Guide](../../docs/CONTRIBUTING.md)
before continuing.

## Development Environment Setup

To get started contributing to the Synnax Python client, read the development
environment setup guide for [macOS](../../docs/tech/setup-macos.md) or
[windows](../../docs/tech/setup-windows.md).

As an additional step, read the
**[Python Build System Guide](../../docs/tech/python/python.md)** for instructions on
how we work with Python in the Synnax monorepo.

## Essential Commands

These are also covered in the
[Python Build System Guide](../../docs/tech/python/python.md).

### Installing Dependencies

To install the dependencies, run the following command from the `client/py` directory:

```bash
uv sync
```

### Running the Tests

To run the entire test suite, you need to have a development Synnax cluster running and
listening on `localhost:9090`. For instructions on how to do this, read the
[Synnax Engine Contribution Guide](../../core/CONTRIBUTING.md). Then, run the following
command from the `client/py` directory:

```bash
pytest
```

## BEFORE YOU SUBMIT A PULL REQUEST

Make sure you read the
**[BEFORE YOU SUBMIT A PULL REQUEST](../../docs/tech/python/python.md#before-you-submit-a-pull-request)**
section of the Python Build System Guide.
