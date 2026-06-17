import { createSlice } from "@reduxjs/toolkit";
import { panel } from "@synnaxlabs/client";
import z from "zod";

import { Drift } from "@/drift";

export const stateZ = z.object({
  selectedTabs: panel.tabKeyZ.array().default([]),
});

export interface State extends z.output<typeof stateZ> {}

export const windowStateZ = z.object({
  selected: panel.keyZ.optional(),
  focusedTab: panel.tabKeyZ.optional(),
  overlaidTab: panel.tabKeyZ.optional(),
  panels: z.record(panel.keyZ, stateZ).default({}),
});

export interface WindowState extends z.output<typeof windowStateZ> {}

export const sliceStateZ = z.object({
  windows: z.record(z.string(), windowStateZ).default({}),
});

export interface SliceState extends z.output<typeof sliceStateZ> {}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const SLICE_NAME = "panels";

const selectOrCreateWindowState = (
  state: SliceState,
  windowKey: string,
): WindowState => {
  let s = state.windows[windowKey];
  if (s == null) {
    s = windowStateZ.parse({});
    state.windows[windowKey] = s;
  }
  return s;
};

const selectOrCreatePanelState = (
  state: SliceState,
  windowKey: string,
  panelKey: string,
): State => {
  const win = selectOrCreateWindowState(state, windowKey);
  let pan = win.panels[panelKey];
  if (pan == null) {
    pan = stateZ.parse({});
    win.panels[panelKey] = pan;
  }
  return pan;
};

export interface PanelKeyPayload extends Drift.OptionalWindowKeyPayload {
  panelKey: string;
}

export interface TabKeyPayload extends Drift.OptionalWindowKeyPayload {
  tabKey: string;
}

export interface SetSelectedTabsPayload extends PanelKeyPayload {
  selectedTabs: string[];
}

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: sliceStateZ.parse({}),
  reducers: {
    select: Drift.withWindowKey<PanelKeyPayload, SliceState>(
      (state, { payload: { windowKey, panelKey } }) => {
        selectOrCreateWindowState(state, windowKey).selected = panelKey;
      },
    ),
    clear: Drift.withWindowKey<PanelKeyPayload, SliceState>(
      (state, { payload: { windowKey, panelKey } }) => {
        delete selectOrCreateWindowState(state, windowKey).panels[panelKey];
      },
    ),
    focusTab: Drift.withWindowKey<TabKeyPayload, SliceState>(
      (state, { payload: { windowKey, tabKey } }) => {
        selectOrCreateWindowState(state, windowKey).focusedTab = tabKey;
      },
    ),
    overlayTab: Drift.withWindowKey<TabKeyPayload, SliceState>(
      (state, { payload: { windowKey, tabKey } }) => {
        selectOrCreateWindowState(state, windowKey).overlaidTab = tabKey;
      },
    ),
    clearOverlaidTab: Drift.withWindowKey<Drift.OptionalWindowKeyPayload, SliceState>(
      (state, { payload: { windowKey } }) => {
        selectOrCreateWindowState(state, windowKey).overlaidTab = undefined;
      },
    ),
    setSelectedTabs: Drift.withWindowKey<SetSelectedTabsPayload, SliceState>(
      (state, { payload: { windowKey, panelKey, selectedTabs } }) => {
        selectOrCreatePanelState(state, windowKey, panelKey).selectedTabs =
          selectedTabs;
      },
    ),
  },
});

export const {
  select,
  clear,
  focusTab,
  overlayTab,
  clearOverlaidTab,
  setSelectedTabs,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

export const MIDDLEWARE = [
  Drift.createInjectWindowKeyMiddleware(select),
  Drift.createInjectWindowKeyMiddleware(clear),
  Drift.createInjectWindowKeyMiddleware(focusTab),
  Drift.createInjectWindowKeyMiddleware(overlayTab),
  Drift.createInjectWindowKeyMiddleware(clearOverlaidTab),
  Drift.createInjectWindowKeyMiddleware(setSelectedTabs),
];
