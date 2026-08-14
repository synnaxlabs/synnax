// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, configureStore } from "@reduxjs/toolkit";
import { type panel } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { deep, uuid } from "@synnaxlabs/x";
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
// actions without an explicit windowKey resolve to the active window. State is read
// back through the public selectors, exercising the slice, middleware, and selectors
// together.
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

    it("should default a window to no selection, no panels, and nothing mounted", () => {
      expect(Panel.windowStateZ.parse({})).toEqual({
        selected: undefined,
        isOverlaid: false,
        panels: {},
        mounted: [],
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

    // The window shows the empty state, but the panels it already warmed stay
    // mounted so selecting one again is still instant.
    it("should leave the mounted set intact", () => {
      const state = run(Panel.select({ key: PANEL }), Panel.clearSelected({}));
      expect(Panel.selectMounted(state)).toEqual([PANEL]);
    });
  });

  describe("mounted", () => {
    // Six panels, one more than the cap, so the tail eviction is observable.
    const KEYS = Array.from({ length: 6 }, () => uuid.create());
    const selectAll = (...keys: panel.Key[]): TestState =>
      run(...keys.map((key) => Panel.select({ key })));

    it("should record selected panels most recently selected first", () => {
      const state = selectAll(KEYS[0], KEYS[1], KEYS[2]);
      expect(Panel.selectMounted(state)).toEqual([KEYS[2], KEYS[1], KEYS[0]]);
    });

    it("should move a reselected panel back to the front without duplicating it", () => {
      const state = selectAll(KEYS[0], KEYS[1], KEYS[0]);
      expect(Panel.selectMounted(state)).toEqual([KEYS[0], KEYS[1]]);
    });

    it("should evict the least recently selected panel past the cap", () => {
      const mounted = Panel.selectMounted(selectAll(...KEYS));
      expect(mounted).toHaveLength(5);
      expect(mounted[0]).toEqual(KEYS[5]);
      expect(mounted).not.toContain(KEYS[0]);
    });

    // The selected panel is the head, so the cap can never evict what the window
    // is currently showing out from under it.
    it("should keep the selected panel mounted at the cap", () => {
      const state = selectAll(...KEYS);
      const selected = Panel.selectSelected(state);
      expect(selected).toEqual(KEYS[5]);
      expect(Panel.selectMounted(state)[0]).toEqual(selected);
    });

    it("should track each window's mounted panels separately", () => {
      const state = run(
        Panel.select({ key: KEYS[0] }),
        Panel.select({ windowKey: "other", key: KEYS[1] }),
      );
      expect(Panel.selectMounted(state)).toEqual([KEYS[0]]);
      expect(state[Panel.SLICE_NAME].windows.other.mounted).toEqual([KEYS[1]]);
    });

    it("should drop a removed panel from every window's mounted set", () => {
      const state = run(
        Panel.select({ key: KEYS[0] }),
        Panel.select({ windowKey: "other", key: KEYS[0] }),
        Panel.remove({ keys: KEYS[0] }),
      );
      expect(Panel.selectMounted(state)).toEqual([]);
      expect(state[Panel.SLICE_NAME].windows.other.mounted).toEqual([]);
    });

    // Panels enter the mounted set on selection, so the fallback is the most
    // recently used survivor rather than the earliest one the window visited.
    it("should fall back to the most recently used panel when the selected one is removed", () => {
      const state = run(
        Panel.select({ key: KEYS[0] }),
        Panel.select({ key: KEYS[1] }),
        Panel.select({ key: KEYS[2] }),
        Panel.remove({ keys: KEYS[2] }),
      );
      expect(Panel.selectSelected(state)).toEqual(KEYS[1]);
      expect(Panel.selectMounted(state)).toEqual([KEYS[1], KEYS[0]]);
    });
  });

  describe("startOverlaying / stopOverlaying", () => {
    it("should set then clear the active window's overlaid flag", () => {
      expect(overlaid(run(Panel.startOverlaying({})))).toBe(true);
      expect(overlaid(run(Panel.startOverlaying({}), Panel.stopOverlaying({})))).toBe(
        false,
      );
    });

    it("should leave the panel's selection untouched", () => {
      const state = run(
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.startOverlaying({}),
      );
      expect(overlaid(state)).toBe(true);
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

    it("should leave a no-op re-selection referentially unchanged", () => {
      const store = configureStore({
        reducer: rootReducer,
        middleware: (getDefault) => getDefault().concat(Panel.MIDDLEWARE),
      });
      store.dispatch(
        Panel.internalSelectTab({
          key: PANEL,
          tabKey: TAB,
          otherTabKeys: [TAB, OTHER_TAB],
        }),
      );
      const before = store.getState()[Panel.SLICE_NAME];
      store.dispatch(
        Panel.internalSelectTab({
          key: PANEL,
          tabKey: TAB,
          otherTabKeys: [TAB, OTHER_TAB],
        }),
      );
      expect(store.getState()[Panel.SLICE_NAME]).toBe(before);
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

  describe("reconcileSelection", () => {
    const THIRD_TAB = uuid.create();

    it("should keep only the most recent selected tab per leaf", () => {
      const state = run(
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.internalSelectTab({
          key: PANEL,
          tabKey: OTHER_TAB,
          otherTabKeys: [OTHER_TAB],
        }),
        Panel.reconcileSelection({ key: PANEL, leaves: [[TAB, OTHER_TAB]] }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([OTHER_TAB]);
    });

    // With no row to place the dropped key in, the leaf falls back to the tab the
    // mosaic already shows while the selection is unresolved: its first.
    it("should drop keys no longer in the tree and fill empty leaves with their first tab", () => {
      const state = run(
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.reconcileSelection({ key: PANEL, leaves: [[OTHER_TAB, THIRD_TAB]] }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([OTHER_TAB]);
    });

    it("should keep one recency-ordered tab per leaf across a split", () => {
      const state = run(
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.internalSelectTab({
          key: PANEL,
          tabKey: OTHER_TAB,
          otherTabKeys: [OTHER_TAB],
        }),
        Panel.reconcileSelection({ key: PANEL, leaves: [[OTHER_TAB], [TAB]] }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([OTHER_TAB, TAB]);
    });

    it("should populate a fresh panel with one tab per leaf", () => {
      const state = run(
        Panel.reconcileSelection({
          key: PANEL,
          leaves: [[TAB, OTHER_TAB], [THIRD_TAB]],
        }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([TAB, THIRD_TAB]);
    });

    // Closing a tab with the X or the context menu removes it from the panel document
    // and nothing else, so the overlay has to end here rather than at the close site.
    it("should exit the overlay when the focused tab leaves the tree", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.startOverlaying({}),
        Panel.reconcileSelection({ key: PANEL, leaves: [[OTHER_TAB]] }),
      );
      expect(overlaid(state)).toBe(false);
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([OTHER_TAB]);
    });

    // A window left flagged overlaid with no tabs opens the next tab it creates
    // straight into focus mode.
    it("should exit the overlay when the panel's last tab leaves the tree", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.startOverlaying({}),
        Panel.reconcileSelection({ key: PANEL, leaves: [[]] }),
      );
      expect(overlaid(state)).toBe(false);
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([]);
    });

    it("should keep the overlay when the focused tab survives", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.startOverlaying({}),
        Panel.reconcileSelection({ key: PANEL, leaves: [[TAB, OTHER_TAB]] }),
      );
      expect(overlaid(state)).toBe(true);
    });

    it("should keep the overlay when an unselected panel loses its focused tab", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({
          key: OTHER_PANEL,
          tabKey: OTHER_TAB,
          otherTabKeys: [OTHER_TAB],
        }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.startOverlaying({}),
        Panel.reconcileSelection({ key: OTHER_PANEL, leaves: [[]] }),
      );
      expect(overlaid(state)).toBe(true);
    });

    it("should target the window named in the payload", () => {
      const state = run(
        Panel.internalSelectTab({
          windowKey: "other",
          key: PANEL,
          tabKey: TAB,
          otherTabKeys: [TAB],
        }),
        Panel.reconcileSelection({
          windowKey: "other",
          key: PANEL,
          leaves: [[OTHER_TAB]],
        }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([]);
      expect(state[Panel.SLICE_NAME].windows.other.panels[PANEL].selectedTabs).toEqual([
        OTHER_TAB,
      ]);
    });

    // Cross-window echoes of the action re-apply against already-converged state, so
    // an in-invariant selection must not churn its reference.
    it("should leave a consistent selection referentially unchanged", () => {
      const store = configureStore({
        reducer: rootReducer,
        middleware: (getDefault) => getDefault().concat(Panel.MIDDLEWARE),
      });
      store.dispatch(
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
      );
      const before = store.getState()[Panel.SLICE_NAME];
      store.dispatch(Panel.reconcileSelection({ key: PANEL, leaves: [[TAB]] }));
      expect(store.getState()[Panel.SLICE_NAME]).toBe(before);
    });
  });

  // The neighbor rule every tab strip follows: the tab beside the closed one takes
  // its place, so closing several in a row needs no re-aiming.
  describe("reconcileSelection closed tabs", () => {
    const TABS = Array.from({ length: 4 }, () => uuid.create());
    const [A, B, C, D] = TABS;

    // The row is what the reconcile before the close saw, exactly as the
    // synchronizer supplies it.
    const close = (
      selected: panel.TabKey,
      previous: panel.TabKey[][],
      leaves: panel.TabKey[][],
    ): TestState =>
      run(
        Panel.internalSelectTab({
          key: PANEL,
          tabKey: selected,
          otherTabKeys: previous.flat(),
        }),
        Panel.reconcileSelection({ key: PANEL, leaves, previous }),
      );

    it("should select the tab to the right of the closed one", () => {
      const state = close(B, [[A, B, C, D]], [[A, C, D]]);
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([C]);
    });

    it("should select the tab to the left when the closed tab was last", () => {
      const state = close(C, [[A, B, C]], [[A, B]]);
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([B]);
    });

    it("should skip tabs closed in the same change", () => {
      const state = close(B, [[A, B, C, D]], [[A, D]]);
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([D]);
    });

    // Focus follows the replacement: the head of the list is the focused tab, so a
    // closed tab's neighbor inherits its place in the recency order.
    it("should focus the replacement when the focused tab is closed", () => {
      const state = run(
        Panel.internalSelectTab({ key: PANEL, tabKey: C, otherTabKeys: [C, D] }),
        Panel.internalSelectTab({ key: PANEL, tabKey: A, otherTabKeys: [A, B] }),
        Panel.reconcileSelection({
          key: PANEL,
          leaves: [[B], [C, D]],
          previous: [
            [A, B],
            [C, D],
          ],
        }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([B, C]);
    });

    // Its leaf is gone, so there is no neighbor to inherit the place; the surviving
    // leaves keep their own selections.
    it("should drop the entry when the closed tab was alone in its leaf", () => {
      const state = run(
        Panel.internalSelectTab({ key: PANEL, tabKey: C, otherTabKeys: [C, D] }),
        Panel.internalSelectTab({ key: PANEL, tabKey: A, otherTabKeys: [A] }),
        Panel.reconcileSelection({
          key: PANEL,
          leaves: [[C, D]],
          previous: [[A], [C, D]],
        }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([C]);
    });

    it("should fall back to the first tab when the closed tab's row is unknown", () => {
      const state = close(B, [], [[A, C]]);
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([A]);
    });
  });

  // A tab dragged into another leaf leaves the same hole a close does, so the leaf
  // it came from answers the same way.
  describe("reconcileSelection dragged tabs", () => {
    const TABS = Array.from({ length: 4 }, () => uuid.create());
    const [A, B, C, D] = TABS;

    // Leaves [A, B, C] and [D]; the dropped tab is selected by the mosaic's drop
    // handler, then the tree change arrives.
    const drag = (dragged: panel.TabKey, selected: panel.TabKey): TestState =>
      run(
        Panel.internalSelectTab({ key: PANEL, tabKey: D, otherTabKeys: [D] }),
        Panel.internalSelectTab({
          key: PANEL,
          tabKey: selected,
          otherTabKeys: [A, B, C],
        }),
        Panel.internalSelectTab({
          key: PANEL,
          tabKey: dragged,
          otherTabKeys: [D, dragged],
        }),
        Panel.reconcileSelection({
          key: PANEL,
          leaves: [[A, B, C].filter((tab) => tab !== dragged), [D, dragged]],
          previous: [[A, B, C], [D]],
        }),
      );

    it("should show the dragged tab where it landed", () => {
      expect(Panel.selectSelectedTabs(drag(B, B), PANEL)[0]).toEqual(B);
    });

    it("should show the tab beside the one dragged out", () => {
      expect(Panel.selectSelectedTabs(drag(B, B), PANEL)).toEqual([B, C]);
    });

    // Dragging a tab the leaf was not showing takes nothing from it.
    it("should leave the leaf alone when an unshown tab is dragged out", () => {
      expect(Panel.selectSelectedTabs(drag(C, A), PANEL)).toEqual([C, A]);
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
        Panel.remove({ keys: PANEL }),
      );
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([]);
      expect(Panel.selectSelectedTabs(state, OTHER_PANEL)).toEqual([OTHER_TAB]);
    });

    it("should clear the selection when the last panel is removed", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.remove({ keys: PANEL }),
      );
      expect(Panel.selectSelected(state)).toBeUndefined();
    });

    it("should exit the overlay when the selected panel is removed", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.startOverlaying({}),
        Panel.remove({ keys: PANEL }),
      );
      expect(overlaid(state)).toBe(false);
    });

    it("should fall back to another panel when the selected one is removed", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.internalSelectTab({
          key: OTHER_PANEL,
          tabKey: OTHER_TAB,
          otherTabKeys: [OTHER_TAB],
        }),
        Panel.remove({ keys: PANEL }),
      );
      expect(Panel.selectSelected(state)).toEqual(OTHER_PANEL);
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
        Panel.remove({ keys: OTHER_PANEL }),
      );
      expect(Panel.selectSelected(state)).toEqual(PANEL);
    });

    it("should drop the panel's state and selection in every window", () => {
      const state = run(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
        Panel.internalSelectTab({
          windowKey: "other",
          key: PANEL,
          tabKey: TAB,
          otherTabKeys: [TAB],
        }),
        Panel.remove({ keys: PANEL }),
      );
      expect(Panel.selectSelected(state)).toBeUndefined();
      Object.values(state[Panel.SLICE_NAME].windows).forEach((win) => {
        expect(win.panels[PANEL]).toBeUndefined();
      });
    });

    // The strip lays panels out in a row, so a deleted panel hands the window to the
    // panel beside it rather than to the one it happened to visit last.
    describe("order", () => {
      const ROW = Array.from({ length: 4 }, () => uuid.create());
      const [FIRST, SECOND, THIRD, FOURTH] = ROW;

      const deleteFrom = (selected: panel.Key, ...keys: panel.Key[]): TestState =>
        run(Panel.select({ key: selected }), Panel.remove({ keys, order: ROW }));

      it("should select the panel to the right of the deleted one", () => {
        expect(Panel.selectSelected(deleteFrom(SECOND, SECOND))).toEqual(THIRD);
      });

      it("should select the panel to the left when the deleted one was last", () => {
        expect(Panel.selectSelected(deleteFrom(FOURTH, FOURTH))).toEqual(THIRD);
      });

      it("should skip panels deleted in the same change", () => {
        expect(Panel.selectSelected(deleteFrom(SECOND, SECOND, THIRD))).toEqual(FOURTH);
      });

      it("should take the neighbor over the most recently used panel", () => {
        const state = run(
          Panel.select({ key: FOURTH }),
          Panel.select({ key: FIRST }),
          Panel.remove({ keys: FIRST, order: ROW }),
        );
        expect(Panel.selectSelected(state)).toEqual(SECOND);
      });

      it("should leave a window whose selection survives alone", () => {
        expect(Panel.selectSelected(deleteFrom(FIRST, SECOND))).toEqual(FIRST);
      });
    });

    it("should leave other panels and their selections untouched", () => {
      const state = run(
        Panel.select({ key: OTHER_PANEL }),
        Panel.internalSelectTab({
          key: OTHER_PANEL,
          tabKey: OTHER_TAB,
          otherTabKeys: [OTHER_TAB],
        }),
        Panel.internalSelectTab({
          windowKey: "other",
          key: PANEL,
          tabKey: TAB,
          otherTabKeys: [TAB],
        }),
        Panel.remove({ keys: [PANEL] }),
      );
      expect(Panel.selectSelected(state)).toEqual(OTHER_PANEL);
      expect(Panel.selectSelectedTabs(state, OTHER_PANEL)).toEqual([OTHER_TAB]);
      expect(state[Panel.SLICE_NAME].windows.other.panels[PANEL]).toBeUndefined();
    });
  });

  describe("purgeSliceState", () => {
    // The persist driver purges a copy of the store state, never the frozen state
    // itself, so the copy is part of the call contract being exercised here.
    const purge = (...actions: Panel.Action[]): TestState =>
      Panel.purgeSliceState(deep.copy(run(...actions)));

    // A project swap hydrates the stored slice wholesale, so clearing the mounted
    // set on write is what stops one project's panels staying mounted into the next.
    // The stored field is what the assertion reads: selectMounted answers for the
    // hydrated window, where the selection alone puts a panel back on screen.
    it("should clear every window's mounted panels", () => {
      const { windows } = purge(
        Panel.select({ key: PANEL }),
        Panel.select({ windowKey: "other", key: OTHER_PANEL }),
      )[Panel.SLICE_NAME];
      Object.values(windows).forEach((win) => expect(win.mounted).toEqual([]));
      expect(Object.keys(windows)).toHaveLength(2);
    });

    it("should leave the selection and per-panel tab state persisted", () => {
      const state = purge(
        Panel.select({ key: PANEL }),
        Panel.internalSelectTab({ key: PANEL, tabKey: TAB, otherTabKeys: [TAB] }),
      );
      expect(Panel.selectSelected(state)).toEqual(PANEL);
      expect(Panel.selectSelectedTabs(state, PANEL)).toEqual([TAB]);
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
