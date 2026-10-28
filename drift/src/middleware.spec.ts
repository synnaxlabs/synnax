// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Middleware, type Reducer, type UnknownAction } from "@reduxjs/toolkit";
import { describe, expect, it, vi } from "vitest";

import {
  configureStore,
  MAIN_WINDOW,
  reducer,
  setWindowTitle,
  type SliceState,
  type StoreState,
  ZERO_SLICE_STATE,
} from "@/index";
import { configureMockStore, createWindowState, MockRuntime } from "@/testutil";

const main = (): MockRuntime<StoreState> =>
  new MockRuntime<StoreState>(true, { key: MAIN_WINDOW });

const emittedTypes = (runtime: MockRuntime<StoreState>): unknown[] =>
  runtime.emissions.map((e) => e.action?.type);

describe("middleware", () => {
  describe("emitting actions", () => {
    it("should emit a local action to other windows", async () => {
      const runtime = main();
      const store = await configureMockStore(runtime);
      store.dispatch({ type: "test" });
      await expect
        .poll(() => runtime.emissions)
        .toContainEqual({ action: { type: "test" }, emitter: MAIN_WINDOW });
    });

    it("should not emit an action an inner middleware swallowed", async () => {
      const runtime = main();
      const swallow: Middleware = () => (next) => (action) =>
        (action as UnknownAction).type === "dropped" ? undefined : next(action);
      const store = await configureStore<StoreState>({
        runtime,
        reducer: { drift: reducer as Reducer<SliceState> },
        enablePrerender: false,
        middleware: () => [swallow] as never,
      });
      store.dispatch({ type: "dropped" });
      store.dispatch({ type: "kept" });
      await expect.poll(() => emittedTypes(runtime)).toContain("kept");
      expect(emittedTypes(runtime)).not.toContain("dropped");
    });
  });

  describe("syncing windows", () => {
    const restored = createWindowState({ key: "restored", reserved: true });
    const hydrating: Reducer<SliceState> = (s, a) =>
      a.type === "persist/hydrate"
        ? { ...ZERO_SLICE_STATE, ...s, windows: { ...s?.windows, restored } }
        : (reducer as Reducer<SliceState>)(s, a);

    it("should sync when a non-drift action changes the slice", async () => {
      const runtime = main();
      const store = await configureStore<StoreState>({
        runtime,
        reducer: { drift: hydrating },
        enablePrerender: false,
      });
      store.dispatch({ type: "persist/hydrate" });
      await expect.poll(() => runtime.hasCreated).toHaveProperty("restored");
    });

    it("should not sync when the slice is untouched", async () => {
      const runtime = main();
      const store = await configureMockStore(runtime);
      store.dispatch({ type: "persist/hydrate" });
      await expect.poll(() => emittedTypes(runtime)).toContain("persist/hydrate");
      expect(runtime.hasCreated).toEqual({});
    });
  });

  describe("label assignment", () => {
    it("should apply a drift action without a key to this window", async () => {
      const store = await configureMockStore();
      store.dispatch(setWindowTitle({ title: "a" }));
      expect(store.getState().drift.windows[MAIN_WINDOW].title).toEqual("a");
    });
  });

  describe("user middleware", () => {
    it("should call a middleware function with the default middleware", async () => {
      const curry = vi.fn();
      await configureStore<StoreState>({
        runtime: main(),
        reducer: { drift: reducer as Reducer<SliceState> },
        enablePrerender: false,
        middleware: (def) => {
          curry();
          return def();
        },
      });
      expect(curry).toHaveBeenCalled();
    });
  });
});
