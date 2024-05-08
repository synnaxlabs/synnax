# Synnax Client

A client library for interacting with a Synnax cluster. It supports TypeScript and can be used in both node and browser environments.

Detailed documentation is available [here](https://docs.synnaxlabs.com/typescript-client/get-started).

## Installation

```bash
npm install @synnaxlabs/client
```

## Development

Synnax's TypeScript client unit tests are written in [Vitest](https://vitest.dev/). To test the framework, run

```bash
npm test
```

Before running tests, make sure you build upstream dependencies, as specified in [this document](../../docs/tech/typescript/build.md):

```
pnpm build:PACKAGE_NAME
```
