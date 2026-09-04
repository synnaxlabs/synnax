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

import { configureStore, resetInitialState } from "@/configureStore";
import { MockRuntime } from "@/mock";
import { reducer, type SliceState, type StoreState, ZERO_SLICE_STATE } from "@/state";
import { INITIAL_WINDOW_STATE, MAIN_WINDOW, type WindowState } from "@/window";

const stored = (props: Partial<WindowState> = {}): StoreState => ({
  drift: {
    ...deep.copy(ZERO_SLICE_STATE),
    windows: {
      [MAIN_WINDOW]: {
        ...INITIAL_WINDOW_STATE,
        key: MAIN_WINDOW,
        reserved: true,
        ordinal: 1,
        ...props,
      },
    },
  },
});

const restored = (props: Partial<WindowState> = {}): WindowState => {
  const state = resetInitialState(undefined, false, stored(props));
  const win = state?.drift.windows[MAIN_WINDOW];
  if (win == null) throw new Error("the main window did not survive the reset");
  return win;
};

const configure = async (state: StoreState) =>
  await configureStore<StoreState>({
    runtime: new MockRuntime(true, { key: MAIN_WINDOW }),
    reducer: { drift: reducer as Reducer<SliceState> },
    preloadedState: state,
    enablePrerender: false,
  });

describe("configureStore", () => {
  // Immer freezes the state on its own once a reducer runs, so this pins the contract
  // rather than the construction-time freeze that closes the proxy-compare window.
  it("should hand back state frozen through to the leaves", async () => {
    const store = await configure(stored());
    const state = store.getState();
    expect(Object.isFrozen(state.drift)).toBe(true);
    expect(Object.isFrozen(state.drift.windows[MAIN_WINDOW])).toBe(true);
  });

  // resetInitialState mutates the drift slice in place, so the freeze has to follow
  // it. Freezing first makes construction throw on the read-only assignment.
  it("should reset transient state before freezing", async () => {
    const store = await configure(stored({ minimized: true }));
    const win = store.getState().drift.windows[MAIN_WINDOW];
    expect(win.minimized).toBeUndefined();
    expect(Object.isFrozen(win)).toBe(true);
  });
});

describe("resetInitialState", () => {
  // A window comes back the way the user would open it fresh, not the way they left
  // it: launching is a request to see the app.
  it("should show a window stored minimized", () => {
    expect(restored({ minimized: true }).minimized).toBeUndefined();
  });

  it("should show a window stored fullscreen", () => {
    expect(restored({ fullscreen: true }).fullscreen).toBeUndefined();
  });

  // Controls grey out the macOS traffic lights on a blurred window, so a stored blur
  // leaves them dead until the runtime reports a focus.
  it("should drop a stored blur", () => {
    expect(restored({ focus: false }).focus).toBeUndefined();
  });

  // completeProcess reloads a window it finds mid-reload, so the stage has to start
  // over or the next process to finish reloads the fresh window.
  it("should start the lifecycle over", () => {
    expect(restored({ stage: "reloading" }).stage).toEqual("creating");
  });

  it("should drop a stored error", () => {
    expect(restored({ error: "stale" }).error).toBeUndefined();
  });

  it("should zero the counters", () => {
    expect(restored({ focusCount: 3, centerCount: 2, processCount: 1 })).toMatchObject({
      focusCount: 0,
      centerCount: 0,
      processCount: 0,
    });
  });

  it("should keep the window's geometry", () => {
    const geometry: Partial<WindowState> = {
      position: { x: 10, y: 20 },
      size: { width: 800, height: 600 },
      maximized: true,
    };
    expect(restored({ ...geometry, minimized: true })).toMatchObject(geometry);
  });

  it("should take visibility from the default window props", () => {
    const state = resetInitialState(
      { visible: true },
      false,
      stored({ visible: false }),
    );
    expect(state?.drift.windows[MAIN_WINDOW].visible).toEqual(true);
  });

  it("should drop windows that were never reserved", () => {
    const state = resetInitialState(undefined, false, stored({ reserved: false }));
    expect(state?.drift.windows[MAIN_WINDOW]).toBeUndefined();
  });
});
