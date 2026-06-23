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
import { renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Nav } from "@/layered/session/nav";

const rootReducer = combineReducers({
  [Nav.SLICE_NAME]: Nav.reducer,
  drift: Drift.reducer,
});

const storeWith = (...actions: Nav.Action[]) => {
  const store = configureStore({ reducer: rootReducer });
  actions.forEach((action) => store.dispatch(action));
  return store;
};

const wrapperFor = (store: ReturnType<typeof storeWith>) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  return Wrapper;
};

describe("nav selectors", () => {
  describe("selectSliceState", () => {
    it("should return the slice state", () => {
      const state = storeWith().getState();
      expect(Nav.selectSliceState(state)).toBe(state[Nav.SLICE_NAME]);
    });
  });

  describe("selectWindowState", () => {
    it("should fall back to the zero window state when the window is absent", () => {
      expect(Nav.selectWindowState(storeWith().getState())).toEqual(
        Nav.ZERO_WINDOW_STATE,
      );
    });

    it("should resolve the active window's state", () => {
      const state = storeWith(
        Nav.selectLeft({ windowKey: MAIN_WINDOW, key: "a" }),
      ).getState();
      expect(Nav.selectWindowState(state).left.selected).toBe("a");
    });
  });

  describe("selectLeftSelected", () => {
    it("should return the selected left item for the active window", () => {
      const state = storeWith(
        Nav.selectLeft({ windowKey: MAIN_WINDOW, key: "a" }),
      ).getState();
      expect(Nav.selectLeftSelected(state)).toBe("a");
    });

    it("should return undefined when nothing is selected", () => {
      expect(Nav.selectLeftSelected(storeWith().getState())).toBeUndefined();
    });
  });

  describe("selectBottomVisible", () => {
    it("should report the bottom nav visibility for the active window", () => {
      const state = storeWith(Nav.showBottom({ windowKey: MAIN_WINDOW })).getState();
      expect(Nav.selectBottomVisible(state)).toBe(true);
    });

    it("should default to hidden", () => {
      expect(Nav.selectBottomVisible(storeWith().getState())).toBe(false);
    });
  });

  describe("hooks", () => {
    it("useSelectLeftSelected should read through the Redux provider", () => {
      const store = storeWith(Nav.selectLeft({ windowKey: MAIN_WINDOW, key: "a" }));
      const { result } = renderHook(() => Nav.useSelectLeftSelected(), {
        wrapper: wrapperFor(store),
      });
      expect(result.current).toBe("a");
    });

    it("useSelectBottomVisible should read through the Redux provider", () => {
      const store = storeWith(Nav.showBottom({ windowKey: MAIN_WINDOW }));
      const { result } = renderHook(() => Nav.useSelectBottomVisible(), {
        wrapper: wrapperFor(store),
      });
      expect(result.current).toBe(true);
    });
  });
});
