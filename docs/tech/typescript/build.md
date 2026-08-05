# The Synnax front end build system - pnpm and Turborepo

## 0 How the build system works

Synnax is organized as a monorepo. Our front end software consists of five different
libraries:

- `@synnaxlabs/x` (`x/ts`): Common utilities and types used by all other packages.
- `@synnaxlabs/media` (`x/media`): Synnax specific media, including logos and icons. The
  smallest library.
- `@synnaxlabs/freighter` (`freighter/ts`): A transport adapter protocol for
  communicating with Synnax server.
- `@synnaxlabs/client` (`client/ts`): The client library for communicating with a Synnax
  cluster.
- `@synnaxlabs/pluto` (`pluto`): The Synnax component library.
- `@synnaxlabs/drift` (`drift`): A cross window state synchronization library for
  [Tauri](https://tauri.studio/).

We have two main applications:

- `@synnaxlabs/console` - path `console` - The exploratory data analysis, cluster
  management, and control application.
- `@synnaxlabs/docs` - path `docs/site` - The Synnax documentation website.

There are also a few packages that are specifically for defining configurations for
various build/developments tools:

- `@synnaxlabs/eslint-config` - path `configs/eslint` - The ESLint configuration for
  Synnax TypeScript software.
- `@synnaxlabs/stylelint-config` - path `configs/stylelint` - The Stylelint
  configuration for Synnax TypeScript software.
- `@synnaxlabs/tsconfig` - path `configs/ts` - The TypeScript configuration for Synnax
  TypeScript software.
- `@synnaxlabs/vite-plugin` - path `configs/vite` - A custom plugin for building
  TypeScript applications using [Vite](https://vitejs.dev/). We'll discuss Vite in more
  detail later.

Each of these packages are developed and built independently. Note that the
configuration packages above are marked private and are not published to npm — they are
only used within this monorepo.

**Understanding the dependency hierarchy between these packages is critical when
developing Synnax front end software.** We'll revisit that hierarchy in a moment.

## 1 pnpm

We use [pnpm](https://pnpm.js.org/) to manage all of our front end dependencies. This
includes both internal dependencies and those from external sources (e.g. npm).
Installing all dependencies is as simple as running

```bash
pnpm install
```

in the root directory of the repository. Unless you know what you're doing, avoid adding
new dependencies or upgrading dependency versions. These decisions should be made as a
team and handled with care.

As we'll see in a moment, we also use `pnpm` to run the commands that build packages,
run tests, and start development servers.

## 2 Turborepo

If we refer back to the dependency hierarchy above, we can see that
`@synnaxlabs/console` depends on `@synnaxlabs/pluto` and `@synnaxlabs/pluto` depends on
`@synnaxlabs/x` and `@synnaxlabs/client`. This has two implications:

1. If we make a change to `x` that we want reflected in `console`, we'd need to rebuild
   `x` and all of it's downstream dependencies (`pluto` and `console`).
2. We need to have built versions of all upstream dependencies before we can build the
   downstream dependency.

Luckily, we don't need to worry about which dependencies need to be built and in what
order. Instead, we use a tool called [Turborepo](https://turbo.fyi/). Turborepo (or just
"Turbo") is a tool designed to build monorepos. When we edit a file in an upstream
dependency, then build a downstream dependency, Turbo will automatically detect that the
upstream dependency has changed and rebuild it before building the downstream
dependency. This is a huge time saver.

Turbo is configured in the [`turbo.json`](../../../turbo.json) file in the root
directory of the repository.

## 3 Building packages

Building a package is as simple as running

```bash
pnpm build:PACKAGE_NAME
```

where `PACKAGE_NAME` is the name of the package you want to build. For example, to build
`@synnaxlabs/pluto`, we'd run

```bash
pnpm build:pluto
```

This will build `pluto` and all of its dependencies. If we want to build all packages,

Generally speaking, you'll be building `pluto` and `x` most often. We almost never build
`console` or `docs` locally, and instead rely on the CI/CD pipeline to build and publish
these applications for us.

## 4 Important caveats - running tests and development servers

While Turbo is great for managing all of our build tooling, it's not designed for
running tests or development servers. In those situations, we need to make sure we
manually build dependencies whose changes we want reflected in our tests or development
servers.

The most common case here is when we're developing `pluto` and want to see our changes
reflected in `console`. To start the console dev server, we run:

```bash
pnpm dev:console
```

If we make a change to the `Input` component in `pluto`, we need to make sure we run

```bash
pnpm build:pluto
```

to see those changes reflected. The automatic reload on our dev servers should ensure
that there's no need to refresh the page.

The same principle applies to running tests. We need to make sure we build any upstream
dependencies whose changes we want reflected in our tests.

## 5 Vite

There's one more very important, yet less seen and/or modified tool we use to build
Synnax front end software: [Vite](https://vitejs.dev/). Vite is the underlying engine
that Turbo uses to build our packages. You'll never need to run Vite directly, but it's
important to know that it's there.

## 6 Generating libraries with multiple entrypoints

Make sure your `tsconfig.json` has 'composite' set to true. This is necessary for
building libraries with multiple entrypoints.
