// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
