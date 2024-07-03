// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { PayloadAction } from "@reduxjs/toolkit";
import { createSlice } from "@reduxjs/toolkit";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { Haul, Mosaic, Theming } from "@synnaxlabs/pluto";
import { type deep, type location, migrate } from "@synnaxlabs/x";
import { nanoid } from "nanoid/non-secure";

import { type State, WindowProps } from "@/layout/layout";

interface NavState extends Record<string, PartialNavState> {
  main: MainNavState;
}

/** The state of the layout slice */
export interface SliceState extends migrate.Migratable {
  /** The current theme. */
  activeTheme: string;
  /**
   * A record of theme keys to themes. The active theme is guaranteed to be present
   * in this record. */
  themes: Record<string, Theming.ThemeSpec>;
  /**
   * A record of layout keys to layouts. These represent the properties of all layouts
   * currently rendered in the mosaic or in external windows.
   */
  layouts: Record<string, State>;
  altKeyToKey: Record<string, string>;
  keyToAltKey: Record<string, string>;
  hauling: Haul.DraggingState;
  mosaics: Record<string, MosaicState>;
  nav: NavState;
  alreadyCheckedGetStarted: boolean;
}

export interface MosaicState {
  activeTab: string | null;
  root: Mosaic.Node;
}

export interface MainNavState {
  drawers: NavDrawerState;
}

export interface PartialNavState {
  drawers: Partial<NavDrawerState>;
}

export type NavDrawerLocation = "right" | "left" | "bottom";

export interface NavDrawerState {
  left: NavDrawerEntryState;
  right: NavDrawerEntryState;
  bottom: NavDrawerEntryState;
}

export interface NavDrawerEntryState {
  activeItem: string | null;
  menuItems: string[];
  size?: number;
}

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

export const MAIN_LAYOUT: State = {
  name: "Main",
  key: "main",
  type: "main",
  location: "window",
  windowKey: MAIN_WINDOW,
  window: {
    navTop: false,
  },
};

const ZERO_MOSAIC_STATE: MosaicState = {
  activeTab: null,
  root: {
    key: 1,
    tabs: [],
  },
};

const MIGRATIONS: migrate.Migrations = {
  "0.0.0": (state: SliceState): SliceState => ({
    ...state,
    themes: {
      synnaxDark: Theming.SYNNAX_THEMES.synnaxDark,
      synnaxLight: Theming.SYNNAX_THEMES.synnaxLight,
    },
    version: "0.1.0",
  }),
  "0.1.0": (state: SliceState): SliceState => ({
    ...state,
    themes: {
      synnaxDark: Theming.SYNNAX_THEMES.synnaxDark,
      synnaxLight: Theming.SYNNAX_THEMES.synnaxLight,
    },
    version: "0.2.0",
  }),
  "0.2.0": (state: Omit<SliceState, "altKeyToKey" | "keyToAltKey">): SliceState => ({
    altKeyToKey: {},
    keyToAltKey: {},
    ...state,
    version: "0.3.0",
  }),
};

export const migrateSlice = migrate.migrator<SliceState, SliceState>(MIGRATIONS);

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.3.0",
  activeTheme: "synnaxDark",
  themes: Theming.SYNNAX_THEMES,
  alreadyCheckedGetStarted: false,
  layouts: {
    main: MAIN_LAYOUT,
  },
  mosaics: {
    main: ZERO_MOSAIC_STATE,
  },
  altKeyToKey: {},
  keyToAltKey: {},
  hauling: Haul.ZERO_DRAGGING_STATE,
  nav: {
    main: {
      drawers: {
        left: {
          activeItem: null,
          menuItems: ["resources"],
        },
        right: {
          activeItem: null,
          menuItems: ["range", "task"],
        },
        bottom: {
          activeItem: null,
          menuItems: ["visualization"],
        },
      },
    },
  },
};

export const PERSIST_EXCLUDE = ["alreadyCheckedGetStarted"].map(
  (key) => `${SLICE_NAME}.${key}`,
) as Array<deep.Key<StoreState>>;

/** Signature for the placeLayout action. */
export type PlacePayload = State;
/** Signature for the removeLayout action. */
export interface RemovePayload {
  keys: string[];
}
/** Signature for the setTheme action. */
export type SetActiveThemePayload = string | undefined;

export interface MoveMosaicTabPayload {
  tabKey: string;
  windowKey?: string;
  key: number;
  loc: location.Location;
}
interface ResizeMosaicTabPayload {
  windowKey: string;
  key: number;
  size: number;
}
interface SelectMosaicTabPayload {
  tabKey: string;
}
interface RenamePayload {
  key: string;
  name: string;
}

interface ResizeNavDrawerPayload {
  windowKey: string;
  location: NavDrawerLocation;
  size: number;
}

interface SetAltKeyPayload {
  key: string;
  altKey: string;
}

interface SetHaulingPayload extends Haul.DraggingState {}

export interface SetNavDrawerPayload extends NavDrawerEntryState {
  location: NavDrawerLocation;
  windowKey: string;
}

export interface SetWorkspacePayload {
  keepNav?: boolean;
  slice: SliceState;
}

interface SetNavDrawerVisiblePayload {
  windowKey: string;
  key?: string;
  location?: NavDrawerLocation;
  value?: boolean;
}

interface SetArgsPayload<T = unknown> {
  key: string;
  args: T;
}

export const GET_STARTED_LAYOUT_TYPE = "getStarted";

const purgeEmptyMosaics = (state: SliceState) => {
  Object.entries(state.mosaics).forEach(([key, mosaic]) => {
    if (key === MAIN_WINDOW || !Mosaic.isEmpty(mosaic.root)) return;
    delete state.mosaics[key];
    delete state.layouts[key];
    delete state.nav[key];
  });
};

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
    Object.entries(layouts).filter(
      ([, layout]) =>
        layout.location === "window" && layout.type !== MOSAIC_WINDOW_TYPE,
    ),
  );

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    place: (state, { payload: layout }: PayloadAction<PlacePayload>) => {
      const { location, name, tab } = layout;
      let key = layout.key;

      const prev = select(state, key);
      const mosaic = state.mosaics[layout.windowKey];
      if (prev != null) {
        key = prev.key;
        layout.key = prev.key;
      }

      if (layout.type === MOSAIC_WINDOW_TYPE) state.mosaics[key] = ZERO_MOSAIC_STATE;

      // If we're moving from a mosaic, remove the tab.
      if (prev != null && prev.location === "mosaic" && location !== "mosaic")
        [mosaic.root] = Mosaic.removeTab(mosaic.root, key);

      const mosaicTab = {
        closable: true,
        ...tab,
        name,
        tabKey: key,
      };
      delete mosaicTab.location;
      delete mosaicTab.mosaicKey;

      // If we're moving to a mosaic, insert a tab.
      if (prev?.location !== "mosaic" && location === "mosaic") {
        mosaic.root = Mosaic.insertTab(
          mosaic.root,
          mosaicTab,
          tab?.location,
          tab?.mosaicKey,
        );
        mosaic.activeTab = key;
      }

      // If the tab already exists and its in the mosaic, make it the active tab
      // and select it. Also rename it.
      if (prev?.location === "mosaic" && location === "mosaic") {
        mosaic.activeTab = key;
        mosaic.root = Mosaic.renameTab(Mosaic.selectTab(mosaic.root, key), key, name);
      }

      state.layouts[key] = layout;
      state.mosaics[layout.windowKey] = mosaic;
      if (layout.type !== MOSAIC_WINDOW_TYPE) purgeEmptyMosaics(state);
    },
    setHauled: (state, { payload }: PayloadAction<SetHaulingPayload>) => {
      state.hauling = payload;
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      keys.forEach((contentKey) => {
        const layout = select(state, contentKey);
        if (layout == null) return;
        const mosaic = state.mosaics[layout.windowKey];
        if (layout == null || mosaic == null) return;
        const { location } = layout;
        if (location === "mosaic")
          [mosaic.root, mosaic.activeTab] = Mosaic.removeTab(mosaic.root, layout.key);

        delete state.layouts[layout.key];
        state.mosaics[layout.windowKey] = mosaic;
        purgeEmptyMosaics(state);
      });
    },
    setAltKey: (
      state,
      { payload: { key, altKey } }: PayloadAction<SetAltKeyPayload>,
    ) => {
      const layout = select(state, key);
      if (layout == null) return;
      state.altKeyToKey[altKey] = key;
      state.keyToAltKey[key] = altKey;
    },
    moveMosaicTab: (
      state,
      { payload: { tabKey, windowKey, key, loc } }: PayloadAction<MoveMosaicTabPayload>,
    ) => {
      const layout = select(state, tabKey);
      if (layout == null) return;
      const prevWindowKey = layout.windowKey;
      if (windowKey == null || prevWindowKey === windowKey) {
        const mosaic = state.mosaics[prevWindowKey];
        [mosaic.root] = Mosaic.moveTab(mosaic.root, layout.key, loc, key);
        state.mosaics[prevWindowKey] = mosaic;
        return;
      }
      const prevMosaic = state.mosaics[prevWindowKey];
      [prevMosaic.root] = Mosaic.removeTab(prevMosaic.root, tabKey);
      state.mosaics[prevWindowKey] = prevMosaic;
      const mosaic = state.mosaics[windowKey];
      if (mosaic.activeTab == null) mosaic.activeTab = tabKey;
      state.layouts[layout.key].windowKey = windowKey;

      const mosaicTab = {
        closable: true,
        ...layout.tab,
        name: layout.name,
        tabKey: layout.key,
      };

      mosaic.root = Mosaic.insertTab(mosaic.root, mosaicTab, loc, key);
      state.mosaics[windowKey] = mosaic;
      purgeEmptyMosaics(state);
    },
    selectMosaicTab: (
      state,
      { payload: { tabKey } }: PayloadAction<SelectMosaicTabPayload>,
    ) => {
      const layout = select(state, tabKey);
      if (layout == null) return;
      const { windowKey } = layout;
      const mosaic = state.mosaics[windowKey];
      if (mosaic.activeTab === tabKey) return;
      mosaic.root = Mosaic.selectTab(mosaic.root, layout.key);
      mosaic.activeTab = layout.key;
      state.mosaics[windowKey] = mosaic;
    },
    resizeMosaicTab: (
      state,
      { payload: { key, size, windowKey } }: PayloadAction<ResizeMosaicTabPayload>,
    ) => {
      const mosaic = state.mosaics[windowKey];
      mosaic.root = Mosaic.resizeNode(mosaic.root, key, size);
      state.mosaics[windowKey] = mosaic;
    },
    rename: (
      state,
      { payload: { key: tabKey, name } }: PayloadAction<RenamePayload>,
    ) => {
      const layout = select(state, tabKey);
      if (layout == null) return;
      const mosaic = state.mosaics[layout.windowKey];
      layout.name = name;
      mosaic.root = Mosaic.renameTab(mosaic.root, layout.key, name);
      state.mosaics[layout.windowKey] = mosaic;
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
      let navState = state.nav[windowKey];
      if (navState == null) {
        navState = { drawers: {} };
        state.nav[windowKey] = navState;
      }

      if (key != null) {
        Object.values(navState.drawers).forEach((drawer) => {
          if (drawer.menuItems.includes(key)) {
            drawer.activeItem = value ?? drawer.activeItem !== key ? key : null;
          }
        });
      } else if (location != null) {
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
      } else {
        throw new Error("setNavDrawerVisible requires either a key or location");
      }
    },
    maybeCreateGetStartedTab: (state) => {
      const checkedGetStarted = state.alreadyCheckedGetStarted;
      state.alreadyCheckedGetStarted = true;
      if (
        Object.values(state.layouts).filter(({ location }) => location === "mosaic")
          .length !== 0 ||
        checkedGetStarted
      )
        return;
      state.mosaics[MAIN_WINDOW].root = Mosaic.insertTab(
        state.mosaics[MAIN_WINDOW].root,
        {
          closable: true,
          tabKey: GET_STARTED_LAYOUT_TYPE,
          name: "Get Started",
          editable: false,
        },
      );
      state.layouts.getStarted = {
        name: "Get Started",
        key: GET_STARTED_LAYOUT_TYPE,
        location: "mosaic",
        type: GET_STARTED_LAYOUT_TYPE,
        windowKey: MAIN_WINDOW,
      };
    },
    setWorkspace: (
      state,
      { payload: { slice, keepNav = true } }: PayloadAction<SetWorkspacePayload>,
    ) => {
      return {
        ...slice,
        layouts: {
          ...layoutsToPreserve(state.layouts),
          ...slice.layouts,
          main: MAIN_LAYOUT,
        },
        hauling: state.hauling,
        themes: state.themes,
        activeTheme: state.activeTheme,
        nav: keepNav ? state.nav : slice.nav,
      };
    },
    clearWorkspace: (state) => {
      return {
        ...ZERO_SLICE_STATE,
        layouts: {
          ...layoutsToPreserve(state.layouts),
          main: MAIN_LAYOUT,
        },
        hauling: state.hauling,
        themes: state.themes,
        activeTheme: state.activeTheme,
        nav: state.nav,
      };
    },
    setArgs: (state, { payload: { key, args } }: PayloadAction<SetArgsPayload>) => {
      const layout = select(state, key);
      if (layout == null) return;
      layout.args = args;
    },
  },
});

export const {
  place,
  remove,
  setAltKey,
  toggleActiveTheme,
  setActiveTheme,
  moveMosaicTab,
  selectMosaicTab,
  resizeMosaicTab,
  rename,
  setNavDrawer,
  resizeNavDrawer,
  setNavDrawerVisible,
  maybeCreateGetStartedTab,
  setHauled,
  setWorkspace,
  clearWorkspace,
} = actions;

export const setArgs = <T>(pld: SetArgsPayload<T>): PayloadAction<SetArgsPayload<T>> =>
  actions.setArgs(pld) as PayloadAction<SetArgsPayload<T>>;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

export const MOSAIC_WINDOW_TYPE = "mosaic";

export const createMosaicWindow = (window?: WindowProps): Omit<State, "windowKey"> => ({
  key: `${MOSAIC_WINDOW_TYPE}-${nanoid()}`,
  name: "Mosaic",
  type: MOSAIC_WINDOW_TYPE,
  location: "window",
  window: {
    ...window,
    size: { width: 800, height: 600 },
    navTop: true,
    visible: true,
    showTitle: false,
  },
});
