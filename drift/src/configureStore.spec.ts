// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Reducer } from "@reduxjs/toolkit";
import { deep } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  configureStore,
  MAIN_WINDOW,
  reducer,
  setWindowProps,
  type SliceState,
  type StoreState,
  type WindowState,
  ZERO_SLICE_STATE,
} from "@/index";
import { createWindowState, MockRuntime } from "@/testutil";

const stored = (props: Partial<WindowState> = {}): StoreState => ({
  drift: {
    ...deep.copy(ZERO_SLICE_STATE),
    windows: {
      [MAIN_WINDOW]: createWindowState({
        key: MAIN_WINDOW,
        reserved: true,
        ordinal: 1,
        ...props,
      }),
    },
  },
});

// The slice a close of the main window leaves: main gone, its pre-render kept.
const storedWithoutMain = (): StoreState => ({
  drift: {
    ...deep.copy(ZERO_SLICE_STATE),
    windows: { prerender: createWindowState({ key: "prerender", visible: false }) },
    labelKeys: {},
    keyLabels: {},
  },
});

const configure = async (
  state: StoreState,
  runtime = new MockRuntime<StoreState>(true, { key: MAIN_WINDOW }),
  defaultWindowProps?: Partial<WindowState>,
) =>
  await configureStore<StoreState>({
    runtime,
    reducer: { drift: reducer as Reducer<SliceState> },
    preloadedState: state,
    enablePrerender: false,
    defaultWindowProps,
  });

const restored = async (props: Partial<WindowState> = {}): Promise<WindowState> =>
  (await configure(stored(props))).getState().drift.windows[MAIN_WINDOW];

describe("configureStore", () => {
  // Immer freezes the state on its own once a reducer runs, so this pins the contract
  // rather than the construction-time freeze that closes the proxy-compare window.
  it("should hand back state frozen through to the leaves", async () => {
    const store = await configure(stored());
    const state = store.getState();
    expect(Object.isFrozen(state.drift)).toBe(true);
    expect(Object.isFrozen(state.drift.windows[MAIN_WINDOW])).toBe(true);
  });

  // resetInitialState mutates the Drift slice in place, so the freeze has to follow it.
  // Freezing first makes construction throw on the read-only assignment.
  it("should reset transient state before freezing", async () => {
    const store = await configure(stored({ minimized: true }));
    const win = store.getState().drift.windows[MAIN_WINDOW];
    expect(win.minimized).toBeUndefined();
    expect(Object.isFrozen(win)).toBe(true);
  });

  it("should show a main window missing from the preloaded state", async () => {
    const runtime = new MockRuntime<StoreState>(true, {
      key: MAIN_WINDOW,
      visible: false,
    });
    const store = await configure(storedWithoutMain(), runtime);
    store.dispatch(setWindowProps({ visible: true }));
    await expect.poll(() => runtime.props.visible).toBe(true);
  });
});

describe("preloaded state", () => {
  // A window comes back the way the user would open it fresh, not the way they left
  // it: launching is a request to see the app.
  it("should show a window stored minimized", async () => {
    expect((await restored({ minimized: true })).minimized).toBeUndefined();
  });

  it("should show a window stored fullscreen", async () => {
    expect((await restored({ fullscreen: true })).fullscreen).toBeUndefined();
  });

  it("should drop a stored blur", async () => {
    expect((await restored({ focus: false })).focus).toBeUndefined();
  });

  it("should drop a stored error", async () => {
    expect((await restored({ error: "stale" })).error).toBeUndefined();
  });

  it("should zero the counters", async () => {
    expect(await restored({ focusCount: 3, centerCount: 2 })).toMatchObject({
      focusCount: 0,
      centerCount: 0,
    });
  });

  it("should keep the window's geometry", async () => {
    const geometry: Partial<WindowState> = {
      position: { x: 10, y: 20 },
      size: { width: 800, height: 600 },
      maximized: true,
    };
    expect(await restored({ ...geometry, minimized: true })).toMatchObject(geometry);
  });

  it("should take visibility from the default window props", async () => {
    const store = await configure(stored({ visible: false }), undefined, {
      visible: true,
    });
    expect(store.getState().drift.windows[MAIN_WINDOW].visible).toEqual(true);
  });

  it("should drop windows that were never reserved", async () => {
    const state = (await configure(storedWithoutMain())).getState();
    expect(state.drift.windows.prerender).toBeUndefined();
  });

  it("should restore a main window missing from the stored state", async () => {
    const state = (await configure(storedWithoutMain())).getState();
    expect(state.drift.windows[MAIN_WINDOW]).toMatchObject({
      key: MAIN_WINDOW,
      reserved: true,
    });
    expect(state.drift.labelKeys[MAIN_WINDOW]).toEqual(MAIN_WINDOW);
    expect(state.drift.keyLabels[MAIN_WINDOW]).toEqual(MAIN_WINDOW);
  });
});
