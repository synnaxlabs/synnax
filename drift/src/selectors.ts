import { DriftState, StoreState } from "@/state";
import { WindowState } from "@/window";

export const selectDriftState = (state: StoreState): DriftState => state.drift;

export const selectWindow = (
  state: StoreState,
  keyOrLabel?: string
): WindowState | null => {
  const driftState = selectDriftState(state);
  if (keyOrLabel == null) return driftState.windows[driftState.label];
  let win = driftState.windows[keyOrLabel];
  if (win != null) return win;
  const label = driftState.keyLabels[keyOrLabel];
  win = driftState.windows[label];
  return win != null ? win : null;
};

export const selectWindowKey = (state: StoreState, label: string): string | null =>
  selectDriftState(state).labelKeys[label];
