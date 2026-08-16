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
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Nav } from "@/session/nav";

const rootReducer = combineReducers({
  [Nav.SLICE_NAME]: Nav.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
});

const createStore = () => configureStore({ reducer: rootReducer });

const createWrapper = (
  store: ReturnType<typeof createStore>,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("nav selectors", () => {
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
