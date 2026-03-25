// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  selectSliceState,
  selectWindow,
  selectWindowAttribute,
  selectWindowKey,
  selectWindowLabel,
  selectWindows,
} from "@/selectors";
import { type StoreState, ZERO_SLICE_STATE } from "@/state";
import { INITIAL_WINDOW_STATE, MAIN_WINDOW, type WindowState } from "@/window";

const WINDOW_A: WindowState = {
  ...INITIAL_WINDOW_STATE,
  key: "window-a",
  title: "Window A",
  reserved: true,
};

const baseState = (): StoreState => ({
  drift: {
    ...ZERO_SLICE_STATE,
    windows: {
      main: { ...INITIAL_WINDOW_STATE, key: MAIN_WINDOW, reserved: true },
      "label-a": WINDOW_A,
    },
    labelKeys: { main: MAIN_WINDOW, "label-a": "window-a" },
    keyLabels: { main: MAIN_WINDOW, "window-a": "label-a" },
  },
});

describe("selectors", () => {
  describe("selectSliceState", () => {
    it("should return the drift slice", () => {
      const state = baseState();
      expect(selectSliceState(state)).toBe(state.drift);
    });
  });

  describe("selectWindows", () => {
    it("should return all windows", () => {
      const state = baseState();
      const windows = selectWindows(state);
      expect(windows).toHaveLength(2);
    });

    it("should return the same reference when called with unchanged state", () => {
      const state = baseState();
      const first = selectWindows(state);
      const second = selectWindows(state);
      expect(first).toBe(second);
    });
  });

  describe("selectWindow", () => {
    it("should return the current window when no key is provided", () => {
      const state = baseState();
      state.drift.label = MAIN_WINDOW;
      const win = selectWindow(state);
      expect(win?.key).toBe(MAIN_WINDOW);
    });

    it("should return null when no key is provided and the current label has no window", () => {
      const state = baseState();
      state.drift.label = "nonexistent";
      expect(selectWindow(state)).toBeNull();
    });

    it("should return a window by its label", () => {
      const state = baseState();
      const win = selectWindow(state, "label-a");
      expect(win?.key).toBe("window-a");
    });

    it("should return a window by its key via keyLabels lookup", () => {
      const state = baseState();
      const win = selectWindow(state, "window-a");
      expect(win?.key).toBe("window-a");
    });

    it("should return null for an unknown key", () => {
      expect(selectWindow(baseState(), "nonexistent")).toBeNull();
    });
  });

  describe("selectWindowKey", () => {
    it("should return the key for the current label when no label is provided", () => {
      const state = baseState();
      state.drift.label = "label-a";
      expect(selectWindowKey(state)).toBe("window-a");
    });

    it("should return the key for a given label", () => {
      expect(selectWindowKey(baseState(), "label-a")).toBe("window-a");
    });

    it("should return MAIN_WINDOW when queried with the main label and no explicit mapping", () => {
      const state = baseState();
      delete state.drift.labelKeys.main;
      expect(selectWindowKey(state, MAIN_WINDOW)).toBe(MAIN_WINDOW);
    });

    it("should return null for an unknown label", () => {
      expect(selectWindowKey(baseState(), "nonexistent")).toBeNull();
    });
  });

  describe("selectWindowAttribute", () => {
    it("should return the attribute value for a window", () => {
      expect(selectWindowAttribute(baseState(), "label-a", "title")).toBe(
        "Window A",
      );
    });

    it("should return null for an unknown window", () => {
      expect(
        selectWindowAttribute(baseState(), "nonexistent", "title"),
      ).toBeNull();
    });
  });

  describe("selectWindowLabel", () => {
    it("should return the label for a key", () => {
      expect(selectWindowLabel(baseState(), "window-a")).toBe("label-a");
    });

    it("should return null for an unknown key", () => {
      expect(selectWindowLabel(baseState(), "nonexistent")).toBeNull();
    });
  });
});
