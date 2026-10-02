// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SLICE_NAME, type SliceState, type StoreState } from "@/session/account/slice";
import { Select } from "@/session/select";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectPending = (state: StoreState): string | undefined =>
  selectSliceState(state).pending;

export const useSelectEmail = (): string | undefined =>
  Select.useMemo((state: StoreState) => selectSliceState(state).link?.email, []);

export const useSelectSecret = (): string | undefined =>
  Select.useMemo((state: StoreState) => selectSliceState(state).link?.secret, []);
