// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SliceState, StoreState } from "@/state";
import { WindowState } from "@/window";

export const selectSliceState = (state: StoreState): SliceState => state.drift;

export const selectWindow = (
  state: StoreState,
  keyOrLabel?: string
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
  return driftState.labelKeys[label];
};

export const selectWindowAttribute = <K extends keyof WindowState>(
  state: StoreState,
  keyOrLabel: string,
  attr: K
): WindowState[K] | null => {
  const win = selectWindow(state, keyOrLabel);
  return win != null ? win[attr] : null;
};
