# Contributing To Pluto

## Development Environment Setup

To get started contributing to Pluto, read the development environment setup guide for
[MacOS](../docs/tech/setup-macos.md) or [Windows](../docs/tech/setup-windows.md).

## Running the Storybook

Before running the storybook, you must build the upstream dependencies. To do this, run
the following command from the root of the repository:

```bash
pnpm build:pluto
```

Then, to run the storybook, run the following command from the root of the repository:

```bash
pnpm dev:pluto
```

If you make a change to an upstream dependency, you will need to rebuild it in order for
the changes to reflect in the development server. For more information on this process,
see the [TypeScript Build Guide](../docs/tech/typescript/build.md).

### Important Note

Vite HMR (Hot Module Reload) does not work consistently with the latest Storybook
version. You may need to (hard) reload the page to see your changes.

## Running the Tests

To run the tests, run the following command from the root of the repository:

```bash
pnpm test:pluto
```

To run a subset of tests, move into the `pluto` directory and run the following command:

```bash
pnpm test MY_FILTER
```

Where `MY_FILTER` is a string that matches the name of the test suite you want to run.
For example, to test the `Use` components, you would run:

```bash
pnpm test Use
```

## Building the Library

To build the library, run the following command from the root of the repository:

```bash
pnpm build:pluto
```

## Codebase Structure

Pluto maintains a flat directory structure that leverages ES modules as isolated units
of functionality.

### Module Naming Conventions and Imports

Naming conventions are scoped to a module, meaning that a component's name is only
unique within the module it is defined in. For example, a `ButtonIcon` component is
within the `button` module in a file called `Icon.tsx` and the component is named
`Icon`. When using the component in another module, we would import it as follows:

```tsx
import { Button } from "@/button"

const MyComponent = () => {
    return (
        <Button.Icon>
          ...
        </Button.Icon>
    );
};
```

### The Worker Component Tree

**TL;DR - Modules with uppercase names i.e. `Button` are only usable in the main thread,
and modules with lowercase names i.e. `color` are usable in both the main and worker
threads.**

You may have noticed that the case of the module name is different from that of it's
parent folder. This is intentional. Pluto heavily leverages web-workers to do data
fetching, heavy computations, and rendering. To implement this functionality, we've
developed an internal framework called [Aether](src/aether/README.md) that mirrors
specific parts of the React component tree in a web-worker.

This means that we often have tighly paired component definitions; one exists in the
main thread and the other exists in the worker thread. For example, we have a `LinePlot`
component that defines the structure of the plot in the main thread and a `LinePlot`
component that performs the line rendering in the worker thread.

As `tsx` files cannot be imported into web-workers, we need to be careful about the
imports we use in different sections of the codebase. To help with this, we've
established a "case-convention" for modules that are safe to be used in web-workers.

Uppercase modules, like the `Button` module above, are **only** usable in the main
thread, and are uppercase to satisfy the `jsx` compiler. Lowercase modules, like the
`color` module are safe to be used in both the main and worker threads.

This means that we **often have modules with the same name, but different casing**. For
example, the `Color` module contains React components like the `Color.Picker` and
`Color.Swatch` which cannot be used in web-workers, and the `color` module contains
primitives like the `color.Color` class and the `color.Crude` type.

This means that uppercase (main thread) modules are typically **extensions** of the
lowercase (worker thread) modules. For example, the `Color` module *re-exports* the
`color.Color` class as `Color.Color` along with the `Color.Picker` and `Color.Swatch`
and other main thread-specific tooling.

### Z-Indexing

| Name                     | Z-Index | Notes                                                                                |
|--------------------------|---------|--------------------------------------------------------------------------------------|
| Lower 2D Canvas          | -3      | Used for items that should render below the upper 2D and GL canvases such as `Axes`. |
| GL Canvas                | -2      | WebGL rendering for lines.                                                           |
| Lower Schematic Elements | 2       | Below the upper 2D canvas.                                                           |
| Upper 2D Canvas          | 3       | Upper 2D canvas for rendering items above other canvases, such as tooltips.          |
| HTML Legend              | 4       | Above the upper and lower 2D canvases.                                               |
| Upper Schematic Elements | 4       | Above the upper P&ID elements.                                                       |
| Dropdown                 | 5       | Above everything except for tooltips.                                                |
| Dropdown Input           | 6       | Above the dropdown to ensure shadows and border are ok.                              |
| Mosaic Mask              | 7       | Above everything but below tooltips and context.                                     |
| Context Menu             | 8       | Above everything but below tooltips.                                                 |
| Default HTML Tooltips    | 9       | Strictly above everything.                                                           |
