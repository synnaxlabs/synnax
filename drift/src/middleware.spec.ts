// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Middleware, Reducer, UnknownAction } from "@reduxjs/toolkit";
import { describe, expect, it, vi } from "vitest";

import {
  createWindow,
  MAIN_WINDOW,
  reducer,
  setWindowTitle,
  type SliceState,
  type StoreState,
  ZERO_SLICE_STATE,
} from "@/index";
import {
  configureMockStore,
  createWindowState,
  MockRuntime,
  STORE_CHECKS,
} from "@/testutil";

const main = (): MockRuntime<StoreState> =>
  new MockRuntime<StoreState>(true, { key: MAIN_WINDOW });

const emittedTypes = (runtime: MockRuntime<StoreState>): unknown[] =>
  runtime.emissions.map((e) => e.action?.type);

// Holds the emission of the given action type until release resolves.
const blockedRuntime = (
  key: string,
  blockedType: string,
  release: Promise<void>,
): MockRuntime<StoreState> => {
  const runtime = new MockRuntime<StoreState>(true, { key });
  const emit = runtime.emit.bind(runtime);
  runtime.emit = async (event) => {
    if (event.action?.type === blockedType) await release;
    await emit(event);
  };
  return runtime;
};

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
      const store = await configureMockStore(runtime, {
        middleware: (getDefault) => getDefault(STORE_CHECKS).concat(swallow),
      });
      store.dispatch({ type: "dropped" });
      store.dispatch({ type: "kept" });
      await expect.poll(() => emittedTypes(runtime)).toContain("kept");
      expect(emittedTypes(runtime)).not.toContain("dropped");
    });

    it("should emit actions from one store in dispatch order", async () => {
      let release = () => {};
      const runtime = blockedRuntime(
        MAIN_WINDOW,
        "first",
        new Promise((r) => (release = r)),
      );
      const store = await configureMockStore(runtime);
      store.dispatch({ type: "first" });
      store.dispatch({ type: "second" });
      await new Promise((r) => setTimeout(r, 20));
      const ordered = (): unknown[] =>
        emittedTypes(runtime).filter((t) => t === "first" || t === "second");
      expect(ordered()).toEqual([]);
      release();
      await expect.poll(ordered).toEqual(["first", "second"]);
    });

    it("should not hold one store's emissions behind another store's", async () => {
      const blocked = blockedRuntime(MAIN_WINDOW, "stuck", new Promise(() => {}));
      (await configureMockStore(blocked)).dispatch({ type: "stuck" });
      const runtime = main();
      (await configureMockStore(runtime)).dispatch({ type: "free" });
      await expect.poll(() => emittedTypes(runtime)).toContain("free");
      expect(emittedTypes(blocked)).not.toContain("stuck");
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
      const store = await configureMockStore(runtime, {
        reducer: { drift: hydrating },
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

    it("should keep the label of a drift action another window emitted", async () => {
      const runtime = main();
      const store = await configureMockStore(runtime);
      store.dispatch(createWindow({ key: "other", label: "other" }));
      runtime.subscribeCallback({
        action: { type: setWindowTitle.type, payload: { title: "a", label: "other" } },
        emitter: "b",
      });
      expect(store.getState().drift.windows.other.title).toEqual("a");
      expect(store.getState().drift.windows[MAIN_WINDOW].title).toBeUndefined();
    });
  });

  describe("user middleware", () => {
    it("should call a middleware function with the default middleware", async () => {
      const curry = vi.fn();
      await configureMockStore(main(), {
        middleware: (getDefault) => {
          curry();
          return getDefault(STORE_CHECKS);
        },
      });
      expect(curry).toHaveBeenCalled();
    });
  });
});
