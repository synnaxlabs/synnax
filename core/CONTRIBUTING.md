# Contributing to Synnax server

## Development environment setup

To get started contributing to Server, read the development environment setup guide for
[macOS](../docs/tech/setup-macos.md) or [windows](../docs/tech/setup-windows.md).

## Starting a development server

The easiest way to develop with synnax is to start an insecure, memory-backed server.
Run the following command in the `synnax` direction of the repository:

```bash
go run main.go start --listen localhost:9090 --verbose --insecure --mem
```

As a shorthand, you can also run

```bash
go run main.go start -vmi
```

## Running the test suite

To run the test suite, run the following command from the `synnax` directory of the
repository:

```bash
go test -v ./...
```

This will run every test suite in the server codebase. You can also run a test suite for
a specific package by running `go test` in the directory of that package.
