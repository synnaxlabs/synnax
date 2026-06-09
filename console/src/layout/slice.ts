// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createSlice,
  current,
  type Dispatch,
  type PayloadAction,
  type UnknownAction,
} from "@reduxjs/toolkit";
import { UnexpectedError } from "@synnaxlabs/client";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { type Color, type Haul, type Icon } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type ComponentType } from "react";

import * as latest from "@/layout/types";
import { type RootState } from "@/store";

export type State<A = unknown> = latest.State<A>;
export type SliceState = latest.SliceState;
export type NavDrawerLocation = latest.NavDrawerLocation;
export type NavDrawerEntryState = latest.NavDrawerEntryState;
export type WindowProps = latest.WindowProps;
export type WindowPanelsState = latest.WindowPanelsState;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const MAIN_LAYOUT = latest.MAIN_LAYOUT;
export const migrateSlice = latest.migrateSlice;
export const anySliceStateZ = latest.anySliceStateZ;

/**
 * The name of the layout slice in a larger store.
 * NOTE: This must be the name of the slice in the store, or else all selectors will fail.
 */
export const SLICE_NAME = "layout";

/**
 * Represents a partial view of a larger store that contains the layout slice. This is
 * typically used for hooks that accept the entire store state as a parameter but only
 * need access to the layout slice.
 */
export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const PERSIST_EXCLUDE = ["hauling", "themes", "tabUnsavedChanges"].map(
  (key) => `${SLICE_NAME}.${key}`,
) as Array<deep.Key<RootState>>;

/** Signature for the placeLayout action. */
export type PlacePayload = State;

/** Signature for the removeLayout action. */
export interface RemovePayload {
  keys: string[];
}

/** Signature for the setTheme action. */
export type SetActiveThemePayload = string | undefined;

interface RenamePayload {
  key: string;
  name: string;
}

interface ResizeNavDrawerPayload {
  windowKey: string;
  location: NavDrawerLocation;
  size: number;
}

interface SetFocusPayload {
  key: string | null;
  windowKey: string;
}

interface SetAltKeyPayload {
  key: string;
  altKey: string;
}

interface SetUnsavedChangesPayload {
  key: string;
  unsavedChanges: boolean;
}

interface SetHaulingPayload extends Haul.DraggingState {}

export interface SetNavDrawerPayload extends NavDrawerEntryState {
  location: NavDrawerLocation;
  windowKey: string;
}

export interface SetProjectPayload {
  keepNav?: boolean;
  slice: SliceState;
}

export interface SetNavDrawerVisiblePayload {
  windowKey?: string;
  key?: string;
  location?: NavDrawerLocation;
  value?: boolean;
}

interface StartNavHoverPayload {
  windowKey: string;
  location: NavDrawerLocation;
  key: string;
}

interface ToggleNavHoverPayload {
  windowKey: string;
  key: string;
}

interface StopNavHoverPayload {
  windowKey: string;
  location: NavDrawerLocation;
}

interface SetArgsPayload<T = unknown> {
  key: string;
  args: T;
}

export interface SetColorContextPayload {
  state: Color.ContextState;
}

export interface SetActivePanelPayload {
  windowKey: string;
  key: string;
}

const select = (state: SliceState, key: string): State | null => {
  const layout = state.layouts[key];
  if (layout == null) {
    const altKey = state.altKeyToKey[key];
    if (altKey == null) return null;
    const altLayout = state.layouts[altKey];
    return altLayout ?? null;
  }
  return layout;
};

const layoutsToPreserve = (layouts: Record<string, State>): Record<string, State> =>
  Object.fromEntries(
    Object.entries(layouts).filter(([, layout]) => layout.location === "window"),
  );

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    // place stores a window or modal layout. Document content (formerly
    // location "mosaic") is routed into the active panel by usePlacer through
    // panel actions and never reaches this reducer.
    place: (state, { payload: layout }: PayloadAction<PlacePayload>) => {
      let key = layout.key;
      const prev = select(state, key);
      if (prev != null) {
        key = prev.key;
        layout.key = prev.key;
      }
      state.layouts[key] = layout;
    },
    setHauled: (state, { payload }: PayloadAction<SetHaulingPayload>) => {
      state.hauling = payload;
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      keys.forEach((contentKey) => {
        const layout = select(state, contentKey);
        if (layout == null || layout.key == MAIN_WINDOW) return;
        delete state.layouts[layout.key];
      });
    },
    setAltKey: (
      state,
      { payload: { key, altKey } }: PayloadAction<SetAltKeyPayload>,
    ) => {
      state.keyToAltKey[key] = altKey;
      state.altKeyToKey[altKey] = key;
    },
    rename: (
      state,
      { payload: { key: tabKey, name } }: PayloadAction<RenamePayload>,
    ) => {
      const layout = select(state, tabKey);
      if (layout == null) return;
      layout.name = name;
    },
    setActiveTheme: (state, { payload: key }: PayloadAction<SetActiveThemePayload>) => {
      if (key != null) state.activeTheme = key;
      else {
        const keys = Object.keys(state.themes).sort();
        const index = keys.indexOf(state.activeTheme);
        const next = keys[(index + 1) % keys.length];
        state.activeTheme = next;
      }
    },
    toggleActiveTheme: (state) => {
      const keys = Object.keys(state.themes);
      const index = keys.indexOf(state.activeTheme);
      const next = keys[(index + 1) % keys.length];
      state.activeTheme = next;
    },
    setNavDrawer: (state, { payload }: PayloadAction<SetNavDrawerPayload>) => {
      const { windowKey, location, ...rest } = payload;
      if (!(windowKey in state.nav)) state.nav[windowKey] = { drawers: {} };
      state.nav[windowKey].drawers[location] = rest;
    },
    resizeNavDrawer: (
      state,
      { payload: { windowKey, location, size } }: PayloadAction<ResizeNavDrawerPayload>,
    ) => {
      const navState = state.nav[windowKey];
      if (navState?.drawers[location] == null) return;
      (navState.drawers[location] as NavDrawerEntryState).size = size;
    },
    setNavDrawerVisible: (
      state,
      {
        payload: { windowKey, key, location, value },
      }: PayloadAction<SetNavDrawerVisiblePayload>,
    ) => {
      if (windowKey == null)
        throw new UnexpectedError(
          "setNavDrawerVisible requires a windowKey; the layout middleware should " +
            "have injected one from drift state",
        );
      let navState = state.nav[windowKey];
      if (navState == null) {
        navState = { drawers: {} };
        state.nav[windowKey] = navState;
      }
      if (key != null)
        Object.values(navState.drawers).forEach((drawer) => {
          if (drawer.menuItems.includes(key)) {
            const activeItem = (value ?? drawer.activeItem !== key) ? key : null;
            if (drawer.hover) {
              drawer.activeItem = key;
              drawer.hover = false;
            } else drawer.activeItem = activeItem;
          }
        });
      else if (location != null) {
        let drawer = navState.drawers[location];
        if (drawer == null) {
          drawer = { activeItem: null, menuItems: [] };
          navState.drawers[location] = drawer;
        }
        if (value === true && drawer.activeItem == null)
          drawer.activeItem = drawer.menuItems[0];
        else if (value === false) drawer.activeItem = null;
        else if (drawer.activeItem == null) drawer.activeItem = drawer.menuItems[0];
        else drawer.activeItem = null;
      } else throw new Error("setNavDrawerVisible requires either a key or location");
    },
    startNavHover: (
      state,
      { payload: { windowKey, location, key } }: PayloadAction<StartNavHoverPayload>,
    ) => {
      const navState = state.nav[windowKey];
      if (navState == null) return;
      const drawerState = navState.drawers[location];
      if (
        drawerState == null ||
        (drawerState.activeItem != null && drawerState.hover !== true)
      )
        return;
      drawerState.hover = true;
      drawerState.activeItem = key;
    },
    toggleNavHover: (
      state,
      { payload: { windowKey, key } }: PayloadAction<ToggleNavHoverPayload>,
    ) => {
      const navState = state.nav[windowKey];
      if (navState == null) return;
      const drawer = Object.values(navState.drawers).find((drawer) =>
        drawer.menuItems.includes(key),
      );
      if (drawer == null) return;

      if (drawer.activeItem != null && drawer.hover === false) {
        if (key === drawer.activeItem) drawer.activeItem = null;
        else drawer.activeItem = key;
        return;
      }

      if (drawer.hover === true && key !== drawer.activeItem) {
        drawer.activeItem = key;
        return;
      }

      drawer.hover = !(drawer.hover ?? false);
      if (!drawer.hover && key == drawer.activeItem) drawer.activeItem = null;
      else drawer.activeItem = key;
    },
    stopNavHover: (
      state,
      { payload: { windowKey, location } }: PayloadAction<StopNavHoverPayload>,
    ) => {
      const navState = state.nav[windowKey];
      if (navState == null) return;
      const drawerState = navState.drawers[location];
      if (drawerState == null || !drawerState.hover) return;
      drawerState.hover = false;
      drawerState.activeItem = null;
    },
    setProject: (
      state,
      { payload: { slice, keepNav = true } }: PayloadAction<SetProjectPayload>,
    ) => {
      // Snapshot the draft with current() first, since structuredClone cannot
      // clone Immer's draft Proxies.
      const s = current(state);
      return deep.copy(
        migrateSlice({
          ...slice,
          layouts: {
            ...layoutsToPreserve(s.layouts),
            ...slice.layouts,
            main: MAIN_LAYOUT,
          },
          hauling: s.hauling,
          themes: s.themes,
          activeTheme: s.activeTheme,
          nav: keepNav ? s.nav : slice.nav,
        }),
      );
    },
    clearProject: (state) => ({
      ...ZERO_SLICE_STATE,
      layouts: {
        ...layoutsToPreserve(state.layouts),
        main: MAIN_LAYOUT,
      },
      hauling: state.hauling,
      themes: state.themes,
      activeTheme: state.activeTheme,
      nav: state.nav,
    }),
    setArgs: (state, { payload: { key, args } }: PayloadAction<SetArgsPayload>) => {
      const layout = select(state, key);
      if (layout == null) return;
      layout.args = args;
    },
    // setFocus fullscreen-focuses one tab in a window (Ctrl+L), or clears focus
    // when key is null. Focus is per-window session state in state.focused.
    setFocus: (
      state,
      { payload: { key, windowKey } }: PayloadAction<SetFocusPayload>,
    ) => {
      state.focused[windowKey] = key;
    },
    setColorContext: (state, { payload }: PayloadAction<SetColorContextPayload>) => {
      state.colorContext = payload.state;
    },
    setUnsavedChanges: (
      state,
      { payload }: PayloadAction<SetUnsavedChangesPayload>,
    ) => {
      const layout = select(state, payload.key);
      if (layout == null) return;
      layout.unsavedChanges = payload.unsavedChanges;
    },
    setActivePanel: (state, { payload }: PayloadAction<SetActivePanelPayload>) => {
      const wp = (state.windowPanels[payload.windowKey] ??= {
        active: null,
        activeTab: null,
      });
      wp.active = payload.key;
      // Tab focus is scoped to the active panel; reset on switch so the next
      // panel's adapter falls back to its first tab.
      wp.activeTab = null;
    },
    setActiveTab: (
      state,
      { payload }: PayloadAction<{ windowKey: string; key: string | null }>,
    ) => {
      const wp = (state.windowPanels[payload.windowKey] ??= {
        active: null,
        activeTab: null,
      });
      wp.activeTab = payload.key;
    },
    // setTabUnsavedChanges records, per panel tab key, whether a view tab's form has
    // unsaved edits. Session-only (this operator's draft state); absent entries mean
    // saved, so cleared keys are deleted to keep the map a set of dirty tabs.
    setTabUnsavedChanges: (
      state,
      { payload }: PayloadAction<{ key: string; unsavedChanges: boolean }>,
    ) => {
      if (payload.unsavedChanges) state.tabUnsavedChanges[payload.key] = true;
      else delete state.tabUnsavedChanges[payload.key];
    },
    hideAllNavDrawers: (state) => {
      Object.values(state.nav).forEach((navState) => {
        Object.values(navState.drawers).forEach((drawer) => {
          drawer.activeItem = null;
          drawer.hover = false;
        });
      });
    },
  },
});

export const {
  place,
  setFocus,
  remove,
  toggleActiveTheme,
  setActiveTheme,
  setAltKey,
  rename,
  setNavDrawer,
  resizeNavDrawer,
  setNavDrawerVisible,
  setHauled,
  setProject,
  setColorContext,
  clearProject,
  startNavHover,
  toggleNavHover,
  stopNavHover,
  setUnsavedChanges,
  hideAllNavDrawers,
  setActivePanel,
  setActiveTab,
  setTabUnsavedChanges,
} = actions;

export const setArgs = <T>(pld: SetArgsPayload<T>): PayloadAction<SetArgsPayload<T>> =>
  actions.setArgs(pld) as PayloadAction<SetArgsPayload<T>>;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

/**
 * The props passed to a LayoutRenderer. Note that these props are minimal and only focus
 * on providing information that either allows the renderer to perform more data selections
 * from other locations in state OR allows the renderer to perform actions that may have
 * polymorphic behavior depending the layout location (i.e. closing a layout might remove
 * it from the mosaic or close the window, depending on the location).
 *
 * The goal here is to separate the rendering logic for a particular layout from its location
 * allowing us to mix and move layouts around the UI with ease.
 */
export interface RendererProps {
  /** The unique key of the layout. */
  layoutKey: string;
  visible: boolean;
  focused: boolean;
  /**
   * active is true when this layout is the one the user is currently working in —
   * the selected, non-blurred tab of the active panel (or the visible window/modal).
   * Renderers gate keyboard shortcuts on it so input goes to a single layout.
   */
  active: boolean;
  /**
   * onClose should be called when the layout is ready to be closed. This function is
   * polymorphic and may have different behavior depending on the location of the layout.
   * For example, if the layout is in a window, onClose will close the window. If the
   * layout is in the mosaic, onClose will remove the layout from the mosaic.
   */
  onClose: () => void;
}

export interface OnCloseProps {
  dispatch: Dispatch<UnknownAction>;
  layoutKey: string;
}

/** The result returned by a layout's {@link UseName}. */
export interface NameHookResult {
  retrieve: () => void;
  /**
   * Called when the user renames the layout from the UI (e.g., editing the tab
   * in the mosaic). When undefined, the renderer falls back to dispatching
   * {@link rename} against the layout slice.
   */
  onRename: (name: string) => void;
}

/**
 * A hook bound to a layout {@link Renderer} that owns the name read/write path
 * for the layout. The hook is responsible for invoking {@link NameHookProps.onChange}
 * whenever its source-of-truth name updates and for persisting user-initiated
 * renames via {@link NameHookResult.onRename}. Display name is always read from
 * the layout slice; the hook keeps the slice in sync via `onChange`.
 */
export type UseName = (
  layoutKey: string,
  onChange: (name: string) => void,
) => NameHookResult;

/**
 * A React component that renders a layout for a given type. All layouts in state are
 * rendered by a layout renderer of a specific type. Renderers may optionally bind a
 * {@link UseName} via the `useName` property to take over the name read/write path
 * for layouts of their type, and an `icon` to display in tab strips and selectors for
 * layouts of their type.
 */
export type Renderer = ComponentType<RendererProps> & {
  useName?: UseName;
  icon?: Icon.ReactElement;
};

export interface ContextMenuProps {
  layoutKey: string;
}

export type ContextMenuRenderer = ComponentType<ContextMenuProps>;
