// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Viewport } from "@synnaxlabs/pluto";

import { useMemoSelect } from "@/hooks";

import { SLICE_NAME, SliceState, StoreState } from "./slice";

export const selectVisSlice = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectViewportMode = (state: StoreState): Viewport.Mode =>
  selectVisSlice(state).viewportMode;

export const useSelectViewportMode = (): Viewport.Mode =>
  useMemoSelect(selectViewportMode, []);
