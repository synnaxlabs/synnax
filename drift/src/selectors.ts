// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type SliceState, type StoreState } from "@/state";
import { MAIN_WINDOW, type WindowState } from "@/window";

export const selectSliceState = (state: StoreState): SliceState => state.drift;

export const selectWindows = (state: StoreState): WindowState[] =>
  Object.values(selectSliceState(state).windows);

export const selectWindow = (
  state: StoreState,
  keyOrLabel?: string,
): WindowState | null => {
  const driftState = selectSliceState(state);
  if (keyOrLabel == null) return driftState.windows[driftState.label];
  let win = driftState.windows[keyOrLabel];
  if (win != null) return win;
  const label = driftState.keyLabels[keyOrLabel];
  win = driftState.windows[label];
  if (win == null && keyOrLabel != null) return null;
  return win ?? driftState.windows[driftState.label];
};

export const selectWindowKey = (state: StoreState, label?: string): string | null => {
  const driftState = selectSliceState(state);
  if (label == null) return driftState.labelKeys[driftState.label];
  const key = driftState.labelKeys[label];
  if (key == null && label == MAIN_WINDOW) return MAIN_WINDOW;
  return key;
};

export const selectWindowAttribute = <K extends keyof WindowState>(
  state: StoreState,
  keyOrLabel: string,
  attr: K,
): WindowState[K] | null => selectWindow(state, keyOrLabel)?.[attr] ?? null;

export const selectWindowLabel = (state: StoreState, key: string): string | null => {
  const driftState = selectSliceState(state);
  return driftState.keyLabels[key];
};
