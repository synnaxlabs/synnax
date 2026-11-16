// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { closeWindow, ZERO_SLICE_STATE } from "@/state";
import { sync } from "@/sync";
import { MockRuntime } from "@/testutil";
import { INITIAL_WINDOW_STATE, type WindowProps, type WindowState } from "@/window";

describe("sync", () => {
  const TESTS: Array<[keyof WindowState, keyof WindowProps, any, any]> = [
    ["title", "title", "Title To Set", "Title To Set"],
    ["visible", "visible", false, false],
    ["skipTaskbar", "skipTaskbar", true, true],
    ["maximized", "maximized", true, true],
    ["fullscreen", "fullscreen", true, true],
    ["centerCount", "center", 5, true],
    ["minimized", "visible", true, false],
    ["minSize", "minSize", { width: 100, height: 100 }, { width: 100, height: 100 }],
    ["maxSize", "maxSize", { width: 100, height: 100 }, { width: 100, height: 100 }],
    ["size", "size", { width: 100, height: 100 }, { width: 100, height: 100 }],
    ["position", "position", { x: 100, y: 100 }, { x: 100, y: 100 }],
    ["focusCount", "focus", 5, true],
    ["resizable", "resizable", true, true],
    ["decorations", "decorations", true, true],
    ["alwaysOnTop", "alwaysOnTop", true, true],
  ];
  TESTS.forEach(([keyToSet, keyToCheck, valueToSet, expectedValue]) => {
    it(`should set ${keyToSet} to ${JSON.stringify(valueToSet)}`, async () => {
      const runtime = new MockRuntime(true, { key: "main" });
      const nextState = deep.copy(ZERO_SLICE_STATE);
      const win = nextState.windows[runtime.label()];
      nextState.windows[runtime.label()] = { ...win, [keyToSet]: valueToSet };
      await sync(ZERO_SLICE_STATE, nextState, (a) => a, runtime, false);
      expect(runtime.props[keyToCheck]).toStrictEqual(expectedValue);
    });
  });

  it("should remove a window that fails to create and create the rest", async () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    const runtime = new MockRuntime(true, { key: "main" });
    runtime.createErrors.child = new Error("label already exists");
    const nextState = deep.copy(ZERO_SLICE_STATE);
    nextState.windows.child = { ...INITIAL_WINDOW_STATE, key: "child" };
    nextState.windows.other = { ...INITIAL_WINDOW_STATE, key: "other" };
    const dispatched: unknown[] = [];
    await sync(
      ZERO_SLICE_STATE,
      nextState,
      (a) => {
        dispatched.push(a);
        return a;
      },
      runtime,
      false,
    );
    expect(dispatched).toEqual([closeWindow({ key: "child" })]);
    expect(Object.keys(runtime.hasCreated)).toEqual(["other"]);
    expect(consoleError).toHaveBeenCalledOnce();
    consoleError.mockRestore();
  });
});
