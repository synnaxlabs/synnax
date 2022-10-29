import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import {
  insertMosaicTab,
  removeMosaicTab,
  moveMosaicTab,
  resizeMosaicLeaf,
  MosaicLeaf,
  selectMosaicTab,
  Location,
  Theming,
  Theme,
} from "@synnaxlabs/pluto";
import { Layout } from "../types";

export type LayoutState = {
  theme: string;
  themes: Record<string, Theme>;
  layouts: Record<string, Layout>;
  mosaic: MosaicLeaf;
};

export interface LayoutStoreState {
  layout: LayoutState;
}

const initialState: LayoutState = {
  theme: "synnaxDark",
  themes: Theming.themes,
  layouts: {
    main: {
      title: "Main",
      key: "main",
      type: "main",
      location: "window",
      window: {
        navTop: false,
      },
    },
  },
  mosaic: {
    key: 0,
    level: 0,
    tabs: [],
  },
};

export type PlaceLayoutAction = PayloadAction<Layout>;
export type RemoveLayoutAction = PayloadAction<string>;

export type SetThemeAction = PayloadAction<string>;
export type ToggleThemeAction = PayloadAction<void>;

type DeleteLayoutMosaicTabAction = PayloadAction<{ tabKey: string }>;
type MoveLayoutMosaicTabAction = PayloadAction<{
  tabKey: string;
  key: number;
  loc: Location;
}>;
type ResizeLayoutMosaicTabAction = PayloadAction<{ key: number; size: number }>;
type SelectLayoutMosaicTabAction = PayloadAction<{ tabKey: string }>;

export const {
  actions: {
    placeLayout,
    removeLayout,
    deleteLayoutMosaicTab,
    moveLayoutMosaicTab,
    selectLayoutMosaicTab,
    resizeLayoutMosaicTab,
    toggleTheme,
    setTheme,
  },
  reducer: layoutReducer,
} = createSlice({
  name: "layout",
  initialState,
  reducers: {
    placeLayout: (state, { payload: layout }: PlaceLayoutAction) => {
      const { key, title, location, window } = layout;

      const prev = state.layouts[key];

      // If we're moving from a mosaic, remove the tab.
      if (prev && prev.location === "mosaic" && location !== "mosaic") {
        state.mosaic = removeMosaicTab(initialState.mosaic, key);
      }

      // If we're move to a mosaic, insert a tab.
      if (location === "mosaic")
        state.mosaic = insertMosaicTab(state.mosaic, { tabKey: key, title });

      state.layouts[key] = layout;
    },
    removeLayout: (state, { payload: contentKey }: RemoveLayoutAction) => {
      const layout = state.layouts[contentKey];
      if (!layout) return;
      const { location } = layout;

      if (location === "mosaic")
        state.mosaic = removeMosaicTab(state.mosaic, contentKey);

      delete state.layouts[contentKey];
    },
    deleteLayoutMosaicTab: (
      state,
      { payload: { tabKey } }: DeleteLayoutMosaicTabAction
    ) => {
      state.mosaic = removeMosaicTab(state.mosaic, tabKey);
      delete state.layouts[tabKey];
    },
    moveLayoutMosaicTab: (
      state,
      { payload: { tabKey, key, loc } }: MoveLayoutMosaicTabAction
    ) => {
      const m = moveMosaicTab(state.mosaic, tabKey, key, loc);
      state.mosaic = m;
    },
    selectLayoutMosaicTab: (
      state,
      { payload: { tabKey } }: SelectLayoutMosaicTabAction
    ) => {
      const mosaic = selectMosaicTab(state.mosaic, tabKey);
      state.mosaic = mosaic;
    },
    resizeLayoutMosaicTab: (
      state,
      { payload: { key, size } }: ResizeLayoutMosaicTabAction
    ) => {
      state.mosaic = resizeMosaicLeaf(state.mosaic, key, size);
    },
    setTheme: (state, { payload: key }: SetThemeAction) => {
      state.theme = key;
    },
    toggleTheme: (state) => {
      const keys = Object.keys(state.themes);
      const index = keys.indexOf(state.theme);
      const next = keys[(index + 1) % keys.length];
      state.theme = next;
    },
  },
});
