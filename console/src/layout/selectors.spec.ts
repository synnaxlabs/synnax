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
import { beforeEach, describe, expect, it } from "vitest";

import { selectTabRenderState } from "@/layout/selectors";
import {
  place,
  reducer,
  setActivePanel,
  setFocus,
  setFocusedTab,
  SLICE_NAME,
  type State,
  ZERO_SLICE_STATE,
} from "@/layout/slice";

const rootReducer = combineReducers({
  [SLICE_NAME]: reducer,
  drift: Drift.reducer,
});

type TestState = ReturnType<typeof rootReducer>;

const modalLayout = (key: string): State => ({
  key,
  windowKey: MAIN_WINDOW,
  type: "schematic",
  name: key,
  location: "modal",
});

describe("layout selectors", () => {
  let store: ReturnType<typeof configureStore<TestState>>;

  beforeEach(() => {
    store = configureStore({
      reducer: rootReducer,
      preloadedState: { [SLICE_NAME]: ZERO_SLICE_STATE },
    });
  });

  const state = () => store.getState();

  describe("selectTabRenderState", () => {
    it("should fall back to the tab's own visibility when no tab is active", () => {
      expect(selectTabRenderState(state(), "a", true)).toEqual({
        focused: false,
        active: true,
      });
      expect(selectTabRenderState(state(), "a", false)).toEqual({
        focused: false,
        active: false,
      });
    });

    it("should activate the active tab regardless of its own visibility", () => {
      store.dispatch(setFocusedTab({ windowKey: MAIN_WINDOW, key: "a" }));
      expect(selectTabRenderState(state(), "a", false)).toEqual({
        focused: false,
        active: true,
      });
    });

    it("should deactivate tabs that are not the active tab", () => {
      store.dispatch(setFocusedTab({ windowKey: MAIN_WINDOW, key: "a" }));
      expect(selectTabRenderState(state(), "b", true)).toEqual({
        focused: false,
        active: false,
      });
    });

    it("should mark the focused tab focused and active", () => {
      store.dispatch(setFocusedTab({ windowKey: MAIN_WINDOW, key: "a" }));
      store.dispatch(setFocus({ windowKey: MAIN_WINDOW, key: "a" }));
      expect(selectTabRenderState(state(), "a", false)).toEqual({
        focused: true,
        active: true,
      });
    });

    it("should deactivate the active tab when a different tab is focused", () => {
      store.dispatch(setFocusedTab({ windowKey: MAIN_WINDOW, key: "a" }));
      store.dispatch(setFocus({ windowKey: MAIN_WINDOW, key: "b" }));
      expect(selectTabRenderState(state(), "a", true)).toEqual({
        focused: false,
        active: false,
      });
    });

    it("should deactivate the active tab when a modal blurs the window", () => {
      store.dispatch(setFocusedTab({ windowKey: MAIN_WINDOW, key: "a" }));
      store.dispatch(place(modalLayout("confirm")));
      expect(selectTabRenderState(state(), "a", true)).toEqual({
        focused: false,
        active: false,
      });
    });

    it("should resolve render state for a panel's active tab after a panel switch", () => {
      store.dispatch(setActivePanel({ windowKey: MAIN_WINDOW, key: "panel-1" }));
      store.dispatch(setFocusedTab({ windowKey: MAIN_WINDOW, key: "a" }));
      expect(selectTabRenderState(state(), "a", false).active).toBe(true);
    });
  });
});
