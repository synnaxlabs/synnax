// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, configureStore } from "@reduxjs/toolkit";
import { panel, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Drift } from "@synnaxlabs/drift";
import { Panel as Pluto } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
} from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Panel } from "@/session/panel";
import { Synchronizer } from "@/session/synchronizer";
import {
  assertDefined,
  createConsoleWrapper,
  pickSynchronizer,
  type TestStore,
} from "@/testutil";

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
  assertDefined(windowKey, "expected an active window key");
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

    it("should return the active window's key alongside its state", () => {
      const base = createStore().getState();
      const windowKey = Drift.selectWindowKey(base);
      const win = window({ selected: PANEL });
      expect(Panel.selectActiveWindow(createState(win))).toEqual({
        key: windowKey,
        state: win,
      });
    });

    it("should pair the active window's key with the zero state when it has none", () => {
      const base = createStore().getState();
      expect(Panel.selectActiveWindow(base)).toEqual({
        key: Drift.selectWindowKey(base),
        state: Panel.ZERO_WINDOW_STATE,
      });
    });

    it("should return null when no window is active", () => {
      const base = createStore().getState();
      const state = {
        ...base,
        [Drift.SLICE_NAME]: {
          ...base[Drift.SLICE_NAME],
          label: "unlabeled",
          labelKeys: {},
        },
      };
      expect(Panel.selectActiveWindow(state)).toBeNull();
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

  interface SetupOptions {
    scope?: string;
    tabScope?: string;
    client?: Synnax | null;
  }

  // setup mounts the reactive hooks in the production-shaped console stack: the full
  // session store wired through drift's middleware, nested inside the pluto flux
  // provider. The wiring under test is the scope, window-default, and
  // overlaid/focused/visible resolution the console selectors layer on top.
  const setup = async ({ scope, tabScope, client = null }: SetupOptions = {}) => {
    const { wrapper: Console, store } = await createConsoleWrapper({ client });
    const Wrapper = ({ children }: PropsWithChildren): ReactElement => {
      let node: ReactNode = children;
      if (tabScope != null)
        node = (
          <Pluto.TabScope.Provider value={tabScope}>{node}</Pluto.TabScope.Provider>
        );
      if (scope != null)
        node = <Pluto.Scope.Provider value={scope}>{node}</Pluto.Scope.Provider>;
      return <Console>{node}</Console>;
    };
    return { Wrapper, store };
  };

  const selectTab = (store: TestStore, key: string, tabKey: string): void =>
    void store.dispatch(
      Panel.internalSelectTab({ key, tabKey, otherTabKeys: [tabKey] }),
    );

  // renderCounted mounts a hook alongside a render tally. A predicate hook must
  // subscribe to its answer, not to its inputs: state churn that leaves the answer
  // unchanged must not re-render subscribers.
  const renderCounted = <R,>(hook: () => R, Wrapper: FC<PropsWithChildren>) => {
    let renders = 0;
    const { result } = renderHook(
      () => {
        renders += 1;
        return hook();
      },
      { wrapper: Wrapper },
    );
    return { result, renders: () => renders };
  };

  describe("useSelectSelectedTabs", () => {
    it("should return a panel's stored tabs in recency order for an explicit key", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(() => Panel.useSelectSelectedTabs(PANEL), {
        wrapper: Wrapper,
      });
      expect(result.current).toEqual([]);
      act(() => {
        selectTab(store, PANEL, TAB);
        selectTab(store, PANEL, OTHER_TAB);
      });
      expect(result.current).toEqual([OTHER_TAB, TAB]);
    });

    it("should default to the active window's selected panel when no key is given", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(() => Panel.useSelectSelectedTabs(), {
        wrapper: Wrapper,
      });
      act(() => {
        selectTab(store, PANEL, TAB);
        store.dispatch(Panel.select({ key: PANEL }));
      });
      expect(result.current).toEqual([TAB]);
    });

    it("should resolve the key from the surrounding Panel scope", async () => {
      const { Wrapper, store } = await setup({ scope: PANEL });
      const { result } = renderHook(() => Panel.useSelectSelectedTabs(), {
        wrapper: Wrapper,
      });
      act(() => {
        selectTab(store, PANEL, TAB);
      });
      expect(result.current).toEqual([TAB]);
    });
  });

  describe("useSelectSelected", () => {
    it("should reactively track the active window's selected panel", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(() => Panel.useSelectSelected(), {
        wrapper: Wrapper,
      });
      expect(result.current).toBeUndefined();
      act(() => {
        store.dispatch(Panel.select({ key: PANEL }));
      });
      expect(result.current).toEqual(PANEL);
      act(() => {
        store.dispatch(Panel.clearSelected({}));
      });
      expect(result.current).toBeUndefined();
    });
  });

  describe("useSelectFocusedTab", () => {
    it("should return the most recently selected tab of a panel", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(() => Panel.useSelectFocusedTab(PANEL), {
        wrapper: Wrapper,
      });
      expect(result.current).toBeUndefined();
      act(() => {
        selectTab(store, PANEL, TAB);
        selectTab(store, PANEL, OTHER_TAB);
      });
      expect(result.current).toEqual(OTHER_TAB);
    });

    it("should default to the active window's selected panel", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(() => Panel.useSelectFocusedTab(), {
        wrapper: Wrapper,
      });
      act(() => {
        selectTab(store, PANEL, TAB);
        store.dispatch(Panel.select({ key: PANEL }));
      });
      expect(result.current).toEqual(TAB);
    });
  });

  describe("useGetFocusedTab", () => {
    it("should read a panel's focused tab on demand across dispatches", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(() => Panel.useGetFocusedTab(), {
        wrapper: Wrapper,
      });
      const get = result.current;
      expect(get(PANEL)).toBeUndefined();
      act(() => {
        selectTab(store, PANEL, TAB);
      });
      expect(get(PANEL)).toEqual(TAB);
    });

    it("should default the key to the surrounding Panel scope", async () => {
      const { Wrapper, store } = await setup({ scope: PANEL });
      const { result } = renderHook(() => Panel.useGetFocusedTab(), {
        wrapper: Wrapper,
      });
      act(() => {
        selectTab(store, PANEL, TAB);
      });
      expect(result.current()).toEqual(TAB);
    });
  });

  describe("useSelectIsTabFocused", () => {
    it("should be true only for the panel's focused tab", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(
        () => ({
          focused: Panel.useSelectIsTabFocused(PANEL, TAB),
          other: Panel.useSelectIsTabFocused(PANEL, OTHER_TAB),
        }),
        { wrapper: Wrapper },
      );
      act(() => {
        selectTab(store, PANEL, OTHER_TAB);
        selectTab(store, PANEL, TAB);
      });
      expect(result.current.focused).toBe(true);
      expect(result.current.other).toBe(false);
    });

    it("should be false when no tab resolves", async () => {
      const { Wrapper } = await setup();
      const { result } = renderHook(() => Panel.useSelectIsTabFocused(PANEL), {
        wrapper: Wrapper,
      });
      expect(result.current).toBe(false);
    });

    it("should not re-render when the selection grows behind the focused tab", async () => {
      const { Wrapper, store } = await setup();
      act(() => {
        selectTab(store, PANEL, TAB);
      });
      const { result, renders } = renderCounted(
        () => Panel.useSelectIsTabFocused(PANEL, TAB),
        Wrapper,
      );
      expect(result.current).toBe(true);
      const before = renders();
      act(() => {
        store.dispatch(
          Panel.reconcileSelection({
            key: PANEL,
            leaves: [[TAB], [OTHER_TAB]],
          }),
        );
      });
      expect(result.current).toBe(true);
      expect(renders()).toBe(before);
    });
  });

  describe("useGetTabIsFocused", () => {
    it("should report whether the scoped tab is focused on demand", async () => {
      const { Wrapper, store } = await setup({ scope: PANEL, tabScope: TAB });
      const { result } = renderHook(() => Panel.useGetTabIsFocused(), {
        wrapper: Wrapper,
      });
      const get = result.current;
      expect(get()).toBe(false);
      act(() => {
        selectTab(store, PANEL, TAB);
      });
      expect(get()).toBe(true);
      expect(get(PANEL, OTHER_TAB)).toBe(false);
    });
  });

  describe("useSelectOverlaid / useGetIsOverlaid", () => {
    it("should reactively track the active window's overlaid flag", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(() => Panel.useSelectOverlaid(), {
        wrapper: Wrapper,
      });
      expect(result.current).toBe(false);
      act(() => {
        store.dispatch(Panel.startOverlaying({}));
      });
      expect(result.current).toBe(true);
      act(() => {
        store.dispatch(Panel.stopOverlaying({}));
      });
      expect(result.current).toBe(false);
    });

    it("should read the overlaid flag on demand across dispatches", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(() => Panel.useGetIsOverlaid(), {
        wrapper: Wrapper,
      });
      const get = result.current;
      expect(get()).toBe(false);
      act(() => {
        store.dispatch(Panel.startOverlaying({}));
      });
      expect(get()).toBe(true);
    });
  });

  describe("useSelectIsTabOverlaid", () => {
    it("should be true only when the window is overlaid and the tab is focused", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(
        () => ({
          focused: Panel.useSelectIsTabOverlaid(PANEL, TAB),
          other: Panel.useSelectIsTabOverlaid(PANEL, OTHER_TAB),
        }),
        { wrapper: Wrapper },
      );
      act(() => {
        selectTab(store, PANEL, OTHER_TAB);
        selectTab(store, PANEL, TAB);
      });
      expect(result.current.focused).toBe(false);
      act(() => {
        store.dispatch(Panel.startOverlaying({}));
      });
      expect(result.current.focused).toBe(true);
      expect(result.current.other).toBe(false);
    });

    it("should be false when no tab resolves", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(() => Panel.useSelectIsTabOverlaid(PANEL), {
        wrapper: Wrapper,
      });
      act(() => {
        store.dispatch(Panel.startOverlaying({}));
      });
      expect(result.current).toBe(false);
    });

    it("should not re-render on focus changes while the window is not overlaid", async () => {
      const { Wrapper, store } = await setup();
      act(() => {
        selectTab(store, PANEL, TAB);
      });
      const { result, renders } = renderCounted(
        () => Panel.useSelectIsTabOverlaid(PANEL, TAB),
        Wrapper,
      );
      expect(result.current).toBe(false);
      const before = renders();
      act(() => {
        selectTab(store, PANEL, OTHER_TAB);
      });
      expect(result.current).toBe(false);
      expect(renders()).toBe(before);
    });
  });

  describe("useSelectIsTabVisible", () => {
    it("should be visible when the tab is in the panel's selection and not overlaid", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(
        () => ({
          selected: Panel.useSelectIsTabVisible(PANEL, TAB),
          unselected: Panel.useSelectIsTabVisible(PANEL, OTHER_TAB),
        }),
        { wrapper: Wrapper },
      );
      act(() => {
        selectTab(store, PANEL, TAB);
      });
      expect(result.current.selected).toBe(true);
      expect(result.current.unselected).toBe(false);
    });

    it("should show only the focused tab when the window is overlaid", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(
        () => ({
          focused: Panel.useSelectIsTabVisible(PANEL, TAB),
          background: Panel.useSelectIsTabVisible(PANEL, OTHER_TAB),
        }),
        { wrapper: Wrapper },
      );
      act(() => {
        selectTab(store, PANEL, OTHER_TAB);
        selectTab(store, PANEL, TAB);
        store.dispatch(Panel.startOverlaying({}));
      });
      expect(result.current.focused).toBe(true);
      expect(result.current.background).toBe(false);
    });

    it("should be false when no tab resolves", async () => {
      const { Wrapper } = await setup();
      const { result } = renderHook(() => Panel.useSelectIsTabVisible(PANEL), {
        wrapper: Wrapper,
      });
      expect(result.current).toBe(false);
    });

    it("should not re-render a visible tab when a sibling joins the selection", async () => {
      const { Wrapper, store } = await setup();
      act(() => {
        selectTab(store, PANEL, TAB);
      });
      const { result, renders } = renderCounted(
        () => Panel.useSelectIsTabVisible(PANEL, TAB),
        Wrapper,
      );
      expect(result.current).toBe(true);
      const before = renders();
      act(() => {
        store.dispatch(
          Panel.reconcileSelection({
            key: PANEL,
            leaves: [[TAB], [OTHER_TAB]],
          }),
        );
      });
      expect(result.current).toBe(true);
      expect(renders()).toBe(before);
    });

    // Regression: the visibility check previously read the raw key parameter, so an
    // omitted key inside a panel scope fell back to the window's selected panel
    // instead of the scope's panel.
    it("should read the scope panel's selection, not the window's selected panel", async () => {
      const { Wrapper, store } = await setup({ scope: PANEL });
      const { result } = renderHook(
        () => ({
          scoped: Panel.useSelectIsTabVisible(undefined, TAB),
          foreign: Panel.useSelectIsTabVisible(undefined, OTHER_TAB),
        }),
        { wrapper: Wrapper },
      );
      act(() => {
        selectTab(store, PANEL, TAB);
        selectTab(store, OTHER_PANEL, OTHER_TAB);
        store.dispatch(Panel.select({ key: OTHER_PANEL }));
      });
      expect(result.current.scoped).toBe(true);
      expect(result.current.foreign).toBe(false);
    });

    // Regression: a leaf whose stored selection named a vanished tab rendered
    // nothing. The reconcile synchronizer repairs the selection to the live tree.
    it("should show a leaf's tab when the stored selection names a vanished tab", async () => {
      const client = createTestClient();
      const { Wrapper, store } = await setup({ client });
      const { result } = renderHook(
        () => {
          Synchronizer.use(
            pickSynchronizer(Panel.WINDOW_SYNCHRONIZERS, "reconcile tab selections"),
          );
          return Panel.useSelectIsTabVisible(PANEL, TAB);
        },
        { wrapper: Wrapper },
      );
      act(() => {
        selectTab(store, PANEL, OTHER_TAB);
      });
      await act(async () => {
        await client.panels.create(
          panel.panelZ.parse({
            key: PANEL,
            name: "panel",
            root: { variant: "leaf", tabs: [{ variant: "view", key: TAB, type: "t" }] },
          }),
        );
      });
      expect(result.current).toBe(true);
    });
  });

  describe("useStartOverlaying", () => {
    // Overlaying renders the panel's focused tab, so the hook must route the focus
    // through the same selection path any other tab click takes: the tab's leaf
    // siblings lose their selection, and only then does the window overlay.
    it("should focus the tab against its live leaf and overlay the window", async () => {
      const client = createTestClient();
      const { Wrapper, store } = await setup({ client });
      const doc = panel.panelZ.parse({
        key: PANEL,
        name: "panel",
        root: {
          variant: "leaf",
          tabs: [
            { variant: "view", key: TAB, type: "t" },
            { variant: "view", key: OTHER_TAB, type: "t" },
          ],
        },
      });
      await client.panels.create(doc);
      await client.panels.retrieve(PANEL);
      const { result } = renderHook(() => Panel.useStartOverlaying(PANEL), {
        wrapper: Wrapper,
      });
      act(() => {
        selectTab(store, PANEL, OTHER_TAB);
      });
      act(() => {
        result.current(TAB);
      });
      expect(Panel.selectSelectedTabs(store.getState(), PANEL)).toEqual([TAB]);
      expect(Panel.selectWindowState(store.getState()).isOverlaid).toBe(true);
    });

    it("should do nothing when no panel resolves", async () => {
      const { Wrapper, store } = await setup();
      const { result } = renderHook(() => Panel.useStartOverlaying(), {
        wrapper: Wrapper,
      });
      act(() => {
        result.current(TAB);
      });
      expect(Panel.selectWindowState(store.getState()).isOverlaid).toBe(false);
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
