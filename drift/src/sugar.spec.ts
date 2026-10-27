// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { MAIN_WINDOW, setWindowTitle, type StoreState } from "@/index";
import { configureMockStore, MockRuntime } from "@/testutil";

const title = (
  store: Awaited<ReturnType<typeof configureMockStore>>,
): string | undefined => store.getState().drift.windows[MAIN_WINDOW].title;

describe("sugar", () => {
  it("should apply an action that another window emits", async () => {
    const runtime = new MockRuntime<StoreState>(true, { key: MAIN_WINDOW });
    const store = await configureMockStore(runtime);
    runtime.subscribeCallback({ action: setWindowTitle({ title: "a" }), emitter: "b" });
    expect(title(store)).toEqual("a");
  });

  it("should not emit an action that another window emitted", async () => {
    const runtime = new MockRuntime<StoreState>(true, { key: MAIN_WINDOW });
    const store = await configureMockStore(runtime);
    runtime.subscribeCallback({ action: setWindowTitle({ title: "a" }), emitter: "b" });
    // Drift emits in dispatch order, so the later local action marks the end.
    store.dispatch(setWindowTitle({ title: "c" }));
    const titles = (): unknown[] =>
      runtime.emissions.map((e) => (e.action?.payload as { title?: string })?.title);
    await expect.poll(titles).toContain("c");
    expect(titles()).not.toContain("a");
  });

  it("should ignore an action that returns to the window that emitted it", async () => {
    const runtime = new MockRuntime<StoreState>(true, { key: MAIN_WINDOW });
    const store = await configureMockStore(runtime);
    runtime.subscribeCallback({
      action: setWindowTitle({ title: "a" }),
      emitter: MAIN_WINDOW,
    });
    expect(title(store)).toBeUndefined();
  });

  it("should emit a local action to other windows without the window prefix", async () => {
    const runtime = new MockRuntime<StoreState>(true, { key: MAIN_WINDOW });
    const store = await configureMockStore(runtime);
    const action = setWindowTitle({ title: "a" });
    store.dispatch(action);
    await expect
      .poll(() => runtime.emissions.map((e) => e.action?.type))
      .toContain(action.type);
  });
});
