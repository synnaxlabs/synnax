# Contributing to Console

## Development Environment

To get started contributing to Console, read the development environment setup guide for [macos](../docs/tech/setup-macos.md) or [windows](../docs/tech/setup-windows.md). Console requires both a nodejs and a rust development environment.

## Running the Development Server

Before running the development server, you must build Console's upstream dependencies. To do this, run the following commands from the root
of the repository:

```bash
pnpm build:drift
```

and

```bash
pnpm build:pluto
```

Then, to run the development server, execute the following command from
the root of the repository:

```bash
pnpm dev:console
```

If you make a change to an upstream dependency, you will need to rebuild
it in order for the changes to reflect in the development server. For more
information on this process, see the [TypeScript Build Guide](../docs/tech/typescript/build.md).

In Console, it's particularly common to make a change to [pluto](../pluto/README.md), rebuild it, and see that change reflected in Console.

Typically, you'l also want to run Console in conjunction with the synnax [database](../synnax/README.md) database itself, so you can read/write data, create visualizations, etc. For information on how to start up a synnax dev database, see the [contributing guide](../synnax/CONTRIBUTING.md).

## Building the Application

There's typically no need to build the application, as the Synnax CI/CD
servers handle the build and release process. If you do need to build the
application, run the following command from the root of the repository:

```bash
pnpm build:console
```

Note, the build may fail at the very end after outputting a correctly compiled binary. This is because `tauri` is looking for an environment variable to sign the binary with. This is not necessary for local development, so you can ignore this error.

## Data Persistence

Console stores a snapshot of the application state on disk in order to re-open the application to the same state it was in when it was last closed. This snapshot is stored in `~/.synnax/console/data`. In the case that you need to clear this data, you can delete the `~/.synnax/console` directory. On bash, you can do this with the following command:

```bash
sudo rm -rf ~/.synnax/console
```
