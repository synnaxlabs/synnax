// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";
import { useStore } from "react-redux";

import {
  type Location,
  SLICE_NAME,
  type SliceState,
  type StoreState,
} from "@/session/docs/slice";
import { Select } from "@/session/select";

const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectLocation = (state: StoreState): Location =>
  selectSliceState(state).location;

export const useSelectLocation = (): Location => Select.useMemo(selectLocation, []);

export const useGetLocation = (): (() => Location) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectLocation(store.getState()), [store]);
};
