# Synnax TypeScript Client Library

The Synnax TypeScript client is a client library for interacting with a Synnax cluster.
The client library can be used in both node and browser environments. The [Synnax documentation
website](https://docs.synnaxlabs.com/typescript-client/get-started) contains more
information about using the client.

## Installation

If you want to install the client library, please install it via a package manager such
as npm or yarn:

```shell
npm install @synnaxlabs/client
```

## Examples

Examples of usage of the TypeScript client can be found in our [examples
folder](./examples/node/)

## Development

First, please read our [contribution guidelines](../../docs/CONTRIBUTING.md) and the
TypeScript [build document](../../docs/tech/typescript/build.md) for information on
developing in the Synnax repository with TypeScript.

To properly build upstream packages for developing the client, run the following
command:

```shell
pnpm build:freighter
```

Synnax's TypeScript client unit tests are written in [Vitest](https://vitest.dev/). To
test the framework, run the following command:

```shell
pnpm test
```

If you create changes to the API for the client, generate a new API document by running
the following command:

```shell
pnpm genApi
```

Finally, if changes to the code warrant changing the documentation website, please edit
the correct pages on our [TypeScript Documentation
pages](../../docs/site/src/pages/reference/typescript-client/)
