# Drift - Redux State Synchronization and Declarative Window Management

# Summary

Building multi-window applications with Tauri raises the challenge of synchronizing
state between windows. Communicating over IPC is unintuitive when used in combination
with stateful UI frameworks like React.

Drift is a simple Redux extension that tightly synchronizes state between windows. It
also allows you to create, delete, and alter windows by dispatching actions.

What's more, Drift can prerender windows in the background, allowing new windows to be
ready to display in a fraction of the typical time.

Drift was inspired by the now unmaintained
[Electron Redux](https://github.com/klarna/electron-redux), and exposes a much simpler,
more powerful API.

# Supported Runtimes

| Runtime  | Supported                                                                 | Import                                                   |
| -------- | ------------------------------------------------------------------------- | -------------------------------------------------------- |
| Tauri    | Yes                                                                       | `import { TauriRuntime } from "@synnaxlabs/drift/tauri"` |
| Electron | No. We're looking for someone to add Electron support! Please contribute. | TBA                                                      |

# Installation

With NPM:

```bash
npm install @synnaxlabs/drift
```

With Yarn:

```bash
yarn add @synnaxlabs/drift
```

With pnpm:

```bash
pnpm add @synnaxlabs/drift
```

# Usage

## Configuration

The first step is to reconfigure your store to support Drift. Drift exposes a custom
`configureStore` function returns a **promise** that resolves to a Redux store. This
allows Drift to asynchronously fetch the initial state from the main process. In order
to add declarative window management, you also need to add Drift's custom `reducer` to
your store.

```ts
// store configuration file

import {
  reducer as driftReducer,
  configureStore,
  DRIFT_SLICE_NAME,
  TauriRuntime,
} from "@synnaxlabs/drift";

import { combineReducers } from "@reduxjs/toolkit";

const reducer = combineReducers({
  [DRIFT_SLICE_NAME]: driftReducer,
  // ... your other reducers
});

export const storePromise = configureStore({
  runtime: new TauriRuntime(),
  reducer,
  enablePrerender: true,
});
```

Next, we've created a custom `Provider` that automatically resolves the store promise
and works exactly like the standard Redux `Provider`.

```tsx
// in your main application file

import { Provider } from "@synnaxlabs/drift/react";
import { storePromise } from "./store";

return <Provider store={storePromise}>{/* Your stateful application code*/}</Provider>;
```

State should now be synchronized between all of your Windows!

## Managing Windows

Creating a Window is as easy as dispatching a `createWindow` action.

```ts
import { useDispatch } from "react-redux";
import { createWindow } from "@synnaxlabs/drift";
import { useEffect } from "react";

export const MyReactComponent = () => {
  const dispatch = useDispatch();

  useEffect(() => {
    dispatch(
      createWindow({
        key: "exampleWindow",
        title: "Example Window",
        width: 800,
        height: 600,
      }),
    );
  }, [dispatch]);
};
```

The `key` property is used to uniquely identify the window. If a window with the same
key already exists, Drift will focus that window instead of creating a new one.

You can also dispatch a `closeWindow` action to close a window.

```tsx
import { useDispatch } from "react-redux";
import { closeWindow } from "@synnaxlabs/drift";
import { useEffect } from "react";

export const MyReactComponent = () => {
  const dispatch = useDispatch();

  useEffect(() => {
    dispatch(closeWindow({ key: "exampleWindow" }));
  }, [dispatch]);
};
```

## Accessing Window State

Drift also provides selectors for accessing Window state in React.

```tsx
import {useEffect} from "react;
import {useSelectWindow} from "@synnaxlabs/drift";

export const MyReactComponent = () => {
    // Providing a key is optional. If no key is provided, the current window is selected.
    const window = useSelectWindow()

    useEffect(() => {
        console.log(window)
    }, [window])
}
```
