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

// run dispatches the actions through a store wired with the inject-key middleware, so
// actions without an explicit windowKey resolve to the active window. State is read back
// through the public selectors, exercising the slice, middleware, and selectors together.
const run = (...actions: Panel.Action[]): TestState => {
  const store = createStore();
  actions.forEach((action) => store.dispatch(action));
  return store.getState();
};

const overlaid = (state: TestState): boolean =>
  Panel.selectWindowState(state).isOverlaid;

const PANEL = uuid.create();
const OTHER_PANEL = uuid.create();
const TAB = uuid.create();
const OTHER_TAB = uuid.create();

describe("Panel Slice", () => {
  describe("schemas", () => {
    it("should default the slice state to no windows", () => {
      expect(Panel.sliceStateZ.parse({})).toEqual({ windows: {} });
      expect(Panel.ZERO_SLICE_STATE).toEqual({ windows: {} });
    });

    it("should default a window to no selection, no panels, and not overlaid", () => {
      expect(Panel.windowStateZ.parse({})).toEqual({
        selected: undefined,
        isOverlaid: false,
        panels: {},
      });
    });

    it("should default a panel to no selected tabs", () => {
      expect(Panel.stateZ.parse({})).toEqual({ selectedTabs: [] });
    });
  });

  describe("select", () => {
    it("should set the selected panel for the active window", () => {
      const state = run(Panel.select({ key: PANEL }));
      expect(Panel.selectSelected(state)).toEqual(PANEL);
    });
  });

  describe("clearSelected", () => {
    it("should clear the selected panel", () => {
      const state = run(Panel.select({ key: PANEL }), Panel.clearSelected({}));
      expect(Panel.selectSelected(state)).toBeUndefined();
    });
  });

  describe("startOverlaying / stopOverlaying", () => {
    it("should set then clear the active window's overlaid flag", () => {
      expect(overlaid(run(Panel.startOverlaying({})))).toBe(true);
      expect(overlaid(run(Panel.startOverlaying({}), Panel.stopOverlaying({})))).toBe(
        false,
      );
    });

    it("should promote a named tab to the front of the panel's selection", () => {
      const state = run(
        Panel.internalSelectTab({
          key: PANEL,
          tabKey: OTHER_TAB,
          otherTabKeys: [OTHER_TAB],
        }),
        Panel.startOverlaying({ key: PANEL, tabKey: TAB }),
      );
      expect(overlaid(state)).toBe(true);
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([TAB, OTHER_TAB]);
    });

    it("should not duplicate a tab that is already selected", () => {
      const state = run(
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.startOverlaying({ key: PANEL, tabKey: TAB }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([TAB]);
    });

    it("should default the panel to the window's selected one", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.startOverlaying({ tabKey: TAB }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([TAB]);
    });
  });

  describe("selectTab", () => {
    it("should prepend the tab to the panel's selected tabs", () => {
      const state = run(
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([TAB]);
    });

    it("should move an already-selected tab to the front in MRU order", () => {
      const state = run(
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.internalSelectTab({
          key: PANEL,
          tabKey: OTHER_TAB,
          otherTabKeys: [OTHER_TAB],
        }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([TAB, OTHER_TAB]);
    });

    it("should evict sibling tabs of the same leaf when one is selected", () => {
      const state = run(
        Panel.internalSelectTab({
          key: PANEL,
          tabKey: TAB,
          otherTabKeys: [TAB, OTHER_TAB],
        }),
        Panel.internalSelectTab({
          key: PANEL,
          tabKey: OTHER_TAB,
          otherTabKeys: [TAB, OTHER_TAB],
        }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([OTHER_TAB]);
    });

    it("should store a panel's selected tabs without touching other panels", () => {
      const state = run(
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([TAB]);
      expect(Panel.selectSelectedTabs(state, OTHER_PANEL)).toEqual([]);
    });

    it("should scope panel state to the target window", () => {
      const state = run(
        Panel.internalSelectTab({
          windowKey: "other",
          key: PANEL,
          tabKey: TAB,
          otherTabKeys: [TAB],
        }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([]);
      expect(state[Panel.SLICE_NAME].windows.other.panels[PANEL].selectedTabs).toEqual([
        TAB,
      ]);
    });

    // withSelectedState must create the panel's state on first touch rather than no-op
    // against a missing panel.
    it("should lazily create panel state for a not-yet-seen panel", () => {
      const state = run(
        Panel.internalSelectTab({ key: OTHER_PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
      );
      expect(Panel.selectSelectedTabs(state, OTHER_PANEL)).toEqual([TAB]);
    });

    // Regression: reselecting the front tab rebuilt selectedTabs, so the mosaic's
    // focus-follows-click dispatch changed state identity on every click inside a
    // focused tab; the resulting mid-click re-render reverted controlled checkbox
    // toggles before React could read them (SY-4532).
    it("should keep state identity when the tab is already at the front", () => {
      const store = createStore();
      const reselect = () =>
        store.dispatch(
          Panel.internalSelectTab({
            key: PANEL,
            tabKey: TAB,
            otherTabKeys: [TAB, OTHER_TAB],
          }),
        );
      reselect();
      const before = store.getState()[Panel.SLICE_NAME];
      reselect();
      expect(store.getState()[Panel.SLICE_NAME]).toBe(before);
    });
  });

  describe("remove", () => {
    it("should drop a panel's per-panel state", () => {
      const state = run(
        Panel.internalSelectTab({
          key: OTHER_PANEL,
          tabKey: OTHER_TAB,
          otherTabKeys: [OTHER_TAB],
        }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.remove({ key: PANEL }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([]);
      expect(Panel.selectSelectedTabs(state, OTHER_PANEL)).toEqual([OTHER_TAB]);
    });

    it("should clear the selection when the last panel is removed", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.remove({ key: PANEL }),
      );
      expect(Panel.selectSelected(state)).toBeUndefined();
    });

    // Regression: remove previously cleared the selection only when the panel list
    // became empty, leaving a dangling selected key when other panels remained.
    it("should clear the selection when the selected panel is removed while others remain", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.internalSelectTab({
          key: OTHER_PANEL,
          tabKey: OTHER_TAB,
          otherTabKeys: [OTHER_TAB],
        }),
        Panel.remove({ key: PANEL }),
      );
      expect(Panel.selectSelected(state)).toBeUndefined();
      expect(Panel.selectSelectedTabs(state, OTHER_PANEL)).toEqual([OTHER_TAB]);
    });

    // A selected panel has no per-panel entry until a tab is selected in it, so
    // removing the last stored entry must not clear an unrelated selection.
    it("should retain the selection when a different panel is removed", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({
          key: OTHER_PANEL,
          tabKey: OTHER_TAB,
          otherTabKeys: [OTHER_TAB],
        }),
        Panel.remove({ key: OTHER_PANEL }),
      );
      expect(Panel.selectSelected(state)).toEqual(PANEL);
    });
  });

  describe("reset", () => {
    it("should drop every window's panel session state", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.startOverlaying({}),
        Panel.reset(),
      );
      expect(Panel.selectSliceState(state)).toEqual(Panel.ZERO_SLICE_STATE);
    });
  });

  describe("selectSelected", () => {
    // Regression: selectSelected previously returned the focused tab. Selecting a panel
    // and selecting a tab within it must keep the two independent.
    it("should return the selected panel, not the panel's tabs", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
      );
      expect(Panel.selectSelected(state)).toEqual(PANEL);
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([TAB]);
    });
  });
});
