import { createSlice } from "@reduxjs/toolkit";
import { panel } from "@synnaxlabs/client";
import { Drift as BaseDrift } from "@synnaxlabs/drift";
import { Panel } from "@synnaxlabs/pluto";
import z from "zod";

import { Drift } from "@/drift";
import { useMemoSelect } from "@/hooks";

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
  key: string;
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
      (state, { payload: { windowKey, key } }) => {
        selectOrCreateWindowState(state, windowKey).selected = key;
      },
    ),
    clear: Drift.withWindowKey<PanelKeyPayload, SliceState>(
      (state, { payload: { windowKey, key } }) => {
        delete selectOrCreateWindowState(state, windowKey).panels[key];
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
      (state, { payload: { windowKey, key: panelKey, selectedTabs } }) => {
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

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const useSelectSliceState = (): SliceState =>
  useMemoSelect((state: StoreState) => selectSliceState(state), []);

const selectWindowState = (
  state: StoreState & BaseDrift.StoreState,
): WindowState | undefined => {
  const windowKey = BaseDrift.selectWindowKey(state);
  if (windowKey == null) return undefined;
  return selectSliceState(state).windows[windowKey];
};

export const selectIsOverlaid = (
  state: StoreState & BaseDrift.StoreState,
  tabKey: string,
): boolean => selectWindowState(state)?.overlaidTab == tabKey;

export const useSelectIsOverlaid = (): boolean => {
  const tabKey = Panel.useTabKey("useSelectIsOverlaid");
  return useMemoSelect(
    (state: StoreState & BaseDrift.StoreState) => selectIsOverlaid(state, tabKey),
    [tabKey],
  );
};

export const selectIsFocused = (
  state: StoreState & BaseDrift.StoreState,
  tabKey: string,
): boolean => selectWindowState(state)?.focusedTab == tabKey;

export const useSelectIsFocused = (): boolean => {
  const tabKey = Panel.useTabKey("useSelectIsFocused");
  return useMemoSelect(
    (state: StoreState & BaseDrift.StoreState) => selectIsFocused(state, tabKey),
    [tabKey],
  );
};

export const selectFocused = (
  state: StoreState & BaseDrift.StoreState,
): string | undefined => selectWindowState(state)?.focusedTab;

export const useSelectFocused = (): string | undefined =>
  useMemoSelect((state: StoreState & BaseDrift.StoreState) => selectFocused(state), []);

export const selectSelected = (
  state: StoreState & BaseDrift.StoreState,
): string | undefined => selectWindowState(state)?.selected;

export const useSelectSelected = (): string | undefined =>
  useMemoSelect((state: StoreState & BaseDrift.StoreState) => selectFocused(state), []);

export const selectSelectedTabs = (
  state: StoreState & BaseDrift.StoreState,
  panelKey: string,
): string[] => selectWindowState(state)?.panels[panelKey]?.selectedTabs ?? [];

export const useSelectSelectedTabs = () => {
  const key = Panel.useKey("useSelectSelected");
  return useMemoSelect(
    (state: StoreState & BaseDrift.StoreState) => selectSelectedTabs(state, key),
    [key],
  );
};

export const selectIsAnyFocused = (state: StoreState & BaseDrift.StoreState): boolean =>
  selectWindowState(state)?.focusedTab != null;

export const useSelectIsAnyFocused = (): boolean =>
  useMemoSelect(
    (state: StoreState & BaseDrift.StoreState) => selectIsAnyFocused(state),
    [],
  );
