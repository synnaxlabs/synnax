import { Drift } from "@synnaxlabs/drift";
import { Panel } from "@synnaxlabs/pluto";

import { useMemoSelect } from "@/hooks";
import {
  SLICE_NAME,
  type SliceState,
  type StoreState,
  type WindowState,
} from "@/panel/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const useSelectSliceState = (): SliceState =>
  useMemoSelect((state: StoreState) => selectSliceState(state), []);

const selectWindowState = (
  state: StoreState & Drift.StoreState,
): WindowState | undefined => {
  const windowKey = Drift.selectWindowKey(state);
  if (windowKey == null) return undefined;
  return selectSliceState(state).windows[windowKey];
};

export const selectIsOverlaid = (
  state: StoreState & Drift.StoreState,
  tabKey: string,
): boolean => selectWindowState(state)?.overlaidTab == tabKey;

export const useSelectIsOverlaid = (): boolean => {
  const tabKey = Panel.useTabKey("useSelectIsOverlaid");
  return useMemoSelect(
    (state: StoreState & Drift.StoreState) => selectIsOverlaid(state, tabKey),
    [tabKey],
  );
};

export const selectIsFocused = (
  state: StoreState & Drift.StoreState,
  tabKey: string,
): boolean => selectWindowState(state)?.focusedTab == tabKey;

export const useSelectIsFocused = (): boolean => {
  const tabKey = Panel.useTabKey("useSelectIsFocused");
  return useMemoSelect(
    (state: StoreState & Drift.StoreState) => selectIsFocused(state, tabKey),
    [tabKey],
  );
};

export const selectFocused = (
  state: StoreState & Drift.StoreState,
): string | undefined => selectWindowState(state)?.focusedTab;

export const useSelectFocused = (): string | undefined =>
  useMemoSelect((state: StoreState & Drift.StoreState) => selectFocused(state), []);

export const selectSelected = (
  state: StoreState & Drift.StoreState,
): string | undefined => selectWindowState(state)?.selected;

export const useSelectSelected = (): string | undefined =>
  useMemoSelect((state: StoreState & Drift.StoreState) => selectFocused(state), []);

export const selectSelectedTabs = (
  state: StoreState & Drift.StoreState,
  panelKey: string,
): string[] => selectWindowState(state)?.panels[panelKey]?.selectedTabs ?? [];

export const useSelectSelectedTabs = () => {
  const key = Panel.useKey("useSelectSelected");
  return useMemoSelect(
    (state: StoreState & Drift.StoreState) => selectSelectedTabs(state, key),
    [key],
  );
};

export const selectIsAnyFocused = (state: StoreState & Drift.StoreState): boolean =>
  selectWindowState(state)?.focusedTab != null;

export const useSelectIsAnyFocused = (): boolean =>
  useMemoSelect(
    (state: StoreState & Drift.StoreState) => selectIsAnyFocused(state),
    [],
  );
