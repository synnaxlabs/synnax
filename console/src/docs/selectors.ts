// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Location,
  SLICE_NAME,
  type SliceState,
  type StoreState,
} from "@/docs/slice";
import { useMemoSelect } from "@/hooks";

/** Selects the documentation slice from a larger store state. */
export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

/** Selects the current documentation URL i.e. page endpoint and heading. */
export const selectLocation = (state: StoreState): Location =>
  selectSliceState(state).location;

/** Selects the current documentation URL i.e. page endpoint and heading. */
export const useSelectLocation = (): Location => useMemoSelect(selectLocation, []);
