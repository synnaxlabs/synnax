# Contributing to the TypeScript Client Library

First, please read our [contribution guidelines](../../docs/CONTRIBUTING.md) and the
TypeScript [build document](../../docs/tech/typescript/build.md) for information on
developing in the Synnax repository with TypeScript.

## Setup

To properly build upstream packages for developing the client, run the following
command in the root of the monorepo:

```shell
pnpm build:freighter
```

## Build

To make sure the client builds properly, run the following command in the `client/ts`
directory:

```shell
pnpm build
```

Synnax's TypeScript client unit tests are written with [Vitest](https://vitest.dev/). To
test the framework, run the following command in the `client/ts` directory:

```shell
pnpm test
```

To check code for linting errors, please run the following command in the `client/ts` directory:

```shell
pnpm lint
```

If you create changes to the API for the client, generate a new API document by running
the following command in the `client/ts` directory:

```shell
pnpm genApi
```

Finally, if changes to the code warrant changing the documentation website, please edit
the correct pages on our [TypeScript Documentation
pages](../../docs/site/src/pages/reference/typescript-client/)
