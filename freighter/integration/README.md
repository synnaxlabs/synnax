# Freighter Integration Testing Server

This is a server that can be used to test a Freighter implementation in a specific
language.

## Running the Server

```sh
go run main.go
```

## Integration with CI

Integration tests for Freighter implementations in various languages are run as part of
the [Synnax CI pipeline](../../.github/workflows/test.freighter.yaml). The
[`Dockerfile`](Dockerfile) in this directory builds the integration server into a Docker
image and pushes it to the Synnax GitHub Registry. Language specific CI tests can pull
this image and use it as part of their tests.
