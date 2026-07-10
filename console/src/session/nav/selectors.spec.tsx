// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, configureStore } from "@reduxjs/toolkit";
import { Drift, MAIN_WINDOW } from "@synnaxlabs/drift";
import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Nav } from "@/session/nav";

const rootReducer = combineReducers({
  [Nav.SLICE_NAME]: Nav.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
});

const createStore = () => configureStore({ reducer: rootReducer });

const createWrapper =
  (store: ReturnType<typeof createStore>) =>
  ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );

describe("nav selectors", () => {
  describe("getters", () => {
    it("should read the active window's left state on demand across dispatches", () => {
      const store = createStore();
      const { result } = renderHook(() => Nav.useGetLeft(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get().selected).toBeUndefined();
      act(() => {
        store.dispatch(Nav.selectLeft({ windowKey: MAIN_WINDOW, key: "a" }));
      });
      expect(get().selected).toBe("a");
      expect(get().hover).toBe(false);
    });

    it("should read the selected left item on demand across dispatches", () => {
      const store = createStore();
      const { result } = renderHook(() => Nav.useGetLeftSelected(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toBeUndefined();
      act(() => {
        store.dispatch(Nav.selectLeft({ windowKey: MAIN_WINDOW, key: "a" }));
      });
      expect(get()).toBe("a");
      act(() => {
        store.dispatch(Nav.selectLeft({ windowKey: MAIN_WINDOW, key: "a" }));
      });
      expect(get()).toBeUndefined();
    });

    it("should read the active window's bottom state on demand across dispatches", () => {
      const store = createStore();
      const { result } = renderHook(() => Nav.useGetBottom(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get().visible).toBe(false);
      act(() => {
        store.dispatch(Nav.showBottom({ windowKey: MAIN_WINDOW }));
      });
      expect(get().visible).toBe(true);
    });

    it("should report the bottom visibility on demand across dispatches", () => {
      const store = createStore();
      const { result } = renderHook(() => Nav.useGetBottomVisible(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toBe(false);
      act(() => {
        store.dispatch(Nav.showBottom({ windowKey: MAIN_WINDOW }));
      });
      expect(get()).toBe(true);
    });

    it("should fall back to the zero window state when the active window is absent", () => {
      const store = createStore();
      const { result } = renderHook(
        () => ({ left: Nav.useGetLeft(), bottom: Nav.useGetBottom() }),
        { wrapper: createWrapper(store) },
      );
      expect(result.current.left()).toEqual(Nav.ZERO_WINDOW_STATE.left);
      expect(result.current.bottom()).toEqual(Nav.ZERO_WINDOW_STATE.bottom);
    });

    it("should read only the active window's state, ignoring other windows", () => {
      const store = createStore();
      const { result } = renderHook(() => Nav.useGetLeftSelected(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      act(() => {
        store.dispatch(Nav.selectLeft({ windowKey: MAIN_WINDOW, key: "a" }));
        store.dispatch(Nav.selectLeft({ windowKey: "window-2", key: "b" }));
      });
      expect(get()).toBe("a");
    });
  });

  describe("reactive hooks", () => {
    it("useSelectLeft should re-render with the active window's left state", () => {
      const store = createStore();
      const { result } = renderHook(() => Nav.useSelectLeft(), {
        wrapper: createWrapper(store),
      });
      expect(result.current.selected).toBeUndefined();
      act(() => {
        store.dispatch(Nav.selectLeft({ windowKey: MAIN_WINDOW, key: "a" }));
      });
      expect(result.current.selected).toBe("a");
      expect(result.current.hover).toBe(false);
    });

    it("useSelectLeftSelected should re-render with the selected left item", () => {
      const store = createStore();
      const { result } = renderHook(() => Nav.useSelectLeftSelected(), {
        wrapper: createWrapper(store),
      });
      expect(result.current).toBeUndefined();
      act(() => {
        store.dispatch(Nav.selectLeft({ windowKey: MAIN_WINDOW, key: "a" }));
      });
      expect(result.current).toBe("a");
    });

    it("useSelectBottom should re-render with the active window's bottom state", () => {
      const store = createStore();
      const { result } = renderHook(() => Nav.useSelectBottom(), {
        wrapper: createWrapper(store),
      });
      expect(result.current.visible).toBe(false);
      act(() => {
        store.dispatch(Nav.showBottom({ windowKey: MAIN_WINDOW }));
      });
      expect(result.current.visible).toBe(true);
    });

    it("useSelectBottomVisible should re-render with the bottom visibility", () => {
      const store = createStore();
      const { result } = renderHook(() => Nav.useSelectBottomVisible(), {
        wrapper: createWrapper(store),
      });
      expect(result.current).toBe(false);
      act(() => {
        store.dispatch(Nav.showBottom({ windowKey: MAIN_WINDOW }));
      });
      expect(result.current).toBe(true);
    });
  });
});
