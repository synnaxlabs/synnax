# Freighter Integration Testing Server

This is a server that can be used to test a Freighter implementation in a 
specific language.

## Running the Server

```
go run main.go
```

## Integration with CI

Integration tests for specific freighter implementations are run as part of the 
Synnax CI pipeline. The `Earthfile` in this directory automatically builds the
integration server into a Docker image and pushes it to the Synnax Github Registry.
Language specific CI tests can pull this image and use it as part of their tests.
For an example, see the `.github/workflows/freighter.test.yaml` file located in
the root of Synnax repo. 

