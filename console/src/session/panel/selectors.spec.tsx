// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, configureStore } from "@reduxjs/toolkit";
import { Drift } from "@synnaxlabs/drift";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Panel } from "@/session/panel";

const rootReducer = combineReducers({
  [Panel.SLICE_NAME]: Panel.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
});

type TestState = ReturnType<typeof rootReducer>;

const createStore = () =>
  configureStore({
    reducer: rootReducer,
    middleware: (getDefault) => getDefault().concat(Panel.MIDDLEWARE),
  });

// createState builds a store state whose active window - resolved through Drift exactly
// as the selectors resolve it - holds the given per-window panel state.
const createState = (win?: Panel.WindowState): TestState => {
  const base = createStore().getState();
  if (win == null) return base;
  const windowKey = Drift.selectWindowKey(base);
  if (windowKey == null) throw new Error("expected an active window key");
  return { ...base, [Panel.SLICE_NAME]: { windows: { [windowKey]: win } } };
};

const window = (overrides: Partial<Panel.WindowState>): Panel.WindowState => ({
  ...Panel.ZERO_WINDOW_STATE,
  ...overrides,
});

const createWrapper = (store: ReturnType<typeof createStore>) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  Wrapper.displayName = "PanelSelectorsWrapper";
  return Wrapper;
};

const PANEL = uuid.create();
const OTHER_PANEL = uuid.create();
const TAB = uuid.create();
const OTHER_TAB = uuid.create();

describe("panel selectors", () => {
  describe("raw selectors", () => {
    it("should return the panel slice state", () => {
      const state = createState();
      expect(Panel.selectSliceState(state)).toBe(state[Panel.SLICE_NAME]);
    });

    it("should return the active window's state", () => {
      const win = window({ selected: PANEL });
      expect(Panel.selectWindowState(createState(win))).toEqual(win);
    });

    it("should fall back to the zero window state when the active window has none", () => {
      expect(Panel.selectWindowState(createState())).toEqual(Panel.ZERO_WINDOW_STATE);
    });

    it("should return the selected panel key", () => {
      const state = createState(window({ selected: PANEL }));
      expect(Panel.selectSelected(state)).toEqual(PANEL);
    });

    it("should return a panel's selected tabs in order", () => {
      const state = createState(
        window({ panels: { [PANEL]: { selectedTabs: [TAB, OTHER_TAB] } } }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([TAB, OTHER_TAB]);
    });

    it("should return an empty tab list for a panel with no state", () => {
      const state = createState(
        window({ panels: { [PANEL]: { selectedTabs: [TAB] } } }),
      );
      expect(Panel.selectSelectedTabs(state, OTHER_PANEL)).toEqual([]);
      expect(Panel.selectSelectedTabs(createState(), PANEL)).toEqual([]);
    });
  });

  describe("getters", () => {
    it("should read the selected panel on demand across dispatches", () => {
      const store = createStore();
      const { result } = renderHook(() => Panel.useGetSelected(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toBeUndefined();
      act(() => {
        store.dispatch(Panel.select({ key: PANEL }));
      });
      expect(get()).toEqual(PANEL);
      act(() => {
        store.dispatch(Panel.select({ key: OTHER_PANEL }));
      });
      expect(get()).toEqual(OTHER_PANEL);
    });

    it("should report whether any panel is selected on demand", () => {
      const store = createStore();
      const { result } = renderHook(() => Panel.useGetIsAnySelected(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toBe(false);
      act(() => {
        store.dispatch(Panel.select({ key: PANEL }));
      });
      expect(get()).toBe(true);
      act(() => {
        store.dispatch(Panel.clearSelected({}));
      });
      expect(get()).toBe(false);
    });
  });
});
