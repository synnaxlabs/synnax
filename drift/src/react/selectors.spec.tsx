// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { renderHook } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import {
  useSelectWindow,
  useSelectWindowAttribute,
  useSelectWindowKey,
  useSelectWindows,
} from "@/react/selectors";
import { type SliceState, type StoreState, ZERO_SLICE_STATE } from "@/state";
import { INITIAL_WINDOW_STATE, MAIN_WINDOW, type WindowState } from "@/window";

const WINDOW_A: WindowState = {
  ...INITIAL_WINDOW_STATE,
  key: "window-a",
  title: "Window A",
  reserved: true,
};

const buildState = (overrides: Partial<SliceState> = {}): StoreState => ({
  drift: {
    ...ZERO_SLICE_STATE,
    windows: {
      main: { ...INITIAL_WINDOW_STATE, key: MAIN_WINDOW, reserved: true },
      "label-a": WINDOW_A,
    },
    labelKeys: { main: MAIN_WINDOW, "label-a": "window-a" },
    keyLabels: { main: MAIN_WINDOW, "window-a": "label-a" },
    ...overrides,
  },
});

const createStore = (overrides: Partial<SliceState> = {}) => {
  const initial = buildState(overrides);
  return configureStore({
    reducer: { drift: (state: SliceState = initial.drift) => state },
  });
};

const wrapper = (store: ReturnType<typeof createStore>) => {
  const Wrapper = ({ children }: PropsWithChildren) => (
    <Provider store={store}>{children}</Provider>
  );
  return Wrapper;
};

describe("react selectors", () => {
  describe("useSelectWindow", () => {
    it("should select the current window when no key is provided", () => {
      const store = createStore();
      const { result } = renderHook(() => useSelectWindow(), {
        wrapper: wrapper(store),
      });
      expect(result.current?.key).toBe(MAIN_WINDOW);
    });

    it("should select a window by label", () => {
      const store = createStore();
      const { result } = renderHook(() => useSelectWindow("label-a"), {
        wrapper: wrapper(store),
      });
      expect(result.current?.key).toBe("window-a");
    });

    it("should select a window by key", () => {
      const store = createStore();
      const { result } = renderHook(() => useSelectWindow("window-a"), {
        wrapper: wrapper(store),
      });
      expect(result.current?.key).toBe("window-a");
    });

    it("should return null for an unknown key", () => {
      const store = createStore();
      const { result } = renderHook(() => useSelectWindow("nonexistent"), {
        wrapper: wrapper(store),
      });
      expect(result.current).toBeNull();
    });
  });

  describe("useSelectWindows", () => {
    it("should return all windows", () => {
      const store = createStore();
      const { result } = renderHook(() => useSelectWindows(), {
        wrapper: wrapper(store),
      });
      expect(result.current).toHaveLength(2);
    });
  });

  describe("useSelectWindowKey", () => {
    it("should return the key for the current label when no label is provided", () => {
      const store = createStore();
      const { result } = renderHook(() => useSelectWindowKey(), {
        wrapper: wrapper(store),
      });
      expect(result.current).toBe(MAIN_WINDOW);
    });

    it("should return the key for a given label", () => {
      const store = createStore();
      const { result } = renderHook(() => useSelectWindowKey("label-a"), {
        wrapper: wrapper(store),
      });
      expect(result.current).toBe("window-a");
    });

    it("should return null for an unknown label", () => {
      const store = createStore();
      const { result } = renderHook(() => useSelectWindowKey("nonexistent"), {
        wrapper: wrapper(store),
      });
      expect(result.current).toBeNull();
    });
  });

  describe("useSelectWindowAttribute", () => {
    it("should return the attribute value for a window", () => {
      const store = createStore();
      const { result } = renderHook(
        () => useSelectWindowAttribute("label-a", "title"),
        { wrapper: wrapper(store) },
      );
      expect(result.current).toBe("Window A");
    });

    it("should return null for an unknown window", () => {
      const store = createStore();
      const { result } = renderHook(
        () => useSelectWindowAttribute("nonexistent", "title"),
        { wrapper: wrapper(store) },
      );
      expect(result.current).toBeNull();
    });
  });
});
