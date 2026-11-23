// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnknownAction } from "@reduxjs/toolkit";
import { describe, expect, it, vi } from "vitest";

import {
  createWindow,
  focusWindow,
  MAIN_WINDOW,
  setWindowAlwaysOnTop,
  setWindowDecorations,
  setWindowFullscreen,
  setWindowMaximized,
  setWindowMaxSize,
  setWindowMinimized,
  setWindowMinSize,
  setWindowPosition,
  setWindowResizable,
  setWindowSize,
  setWindowSkipTaskbar,
  setWindowTitle,
  setWindowVisible,
  type StoreState,
  type WindowProps,
} from "@/index";
import { configureMockStore, MockRuntime } from "@/testutil";

const SIZE = { width: 100, height: 100 };

const TESTS: Array<[string, UnknownAction, keyof WindowProps, unknown]> = [
  ["title", setWindowTitle({ title: "Title To Set" }), "title", "Title To Set"],
  ["visible", setWindowVisible({ value: false }), "visible", false],
  ["skipTaskbar", setWindowSkipTaskbar({ value: true }), "skipTaskbar", true],
  ["maximized", setWindowMaximized({ value: true }), "maximized", true],
  ["fullscreen", setWindowFullscreen({ value: true }), "fullscreen", true],
  ["center", { type: "drift/centerWindow", payload: {} }, "center", true],
  ["minimized", setWindowMinimized({ value: true }), "visible", false],
  ["minSize", setWindowMinSize({ size: SIZE }), "minSize", SIZE],
  ["maxSize", setWindowMaxSize({ size: SIZE }), "maxSize", SIZE],
  ["size", setWindowSize({ size: SIZE }), "size", SIZE],
  [
    "position",
    setWindowPosition({ position: { x: 100, y: 100 } }),
    "position",
    { x: 100, y: 100 },
  ],
  ["focus", focusWindow({}), "focus", true],
  ["resizable", setWindowResizable({ value: true }), "resizable", true],
  ["decorations", setWindowDecorations({ value: true }), "decorations", true],
  ["alwaysOnTop", setWindowAlwaysOnTop({ value: true }), "alwaysOnTop", true],
];

describe("sync", () => {
  TESTS.forEach(([name, action, prop, expected]) => {
    it(`should apply ${name} to the window`, async () => {
      const runtime = new MockRuntime<StoreState>(true, { key: MAIN_WINDOW });
      const store = await configureMockStore(runtime);
      store.dispatch(action);
      await expect.poll(() => runtime.props[prop]).toStrictEqual(expected);
    });
  });

  it("should remove a window that fails to create and create the rest", async () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    const runtime = new MockRuntime<StoreState>(true, { key: MAIN_WINDOW });
    runtime.createErrors.child = new Error("label already exists");
    const store = await configureMockStore(runtime);
    store.dispatch(createWindow({ key: "child", label: "child" }));
    store.dispatch(createWindow({ key: "other", label: "other" }));
    await expect.poll(() => Object.keys(runtime.hasCreated)).toEqual(["other"]);
    await expect.poll(() => store.getState().drift.windows.child).toBeUndefined();
    expect(consoleError).toHaveBeenCalledOnce();
    consoleError.mockRestore();
  });
});
