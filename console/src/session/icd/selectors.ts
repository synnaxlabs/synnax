// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";

import {
  SLICE_NAME,
  type SliceState,
  type StoreState,
  ZERO_STATE,
} from "@/session/icd/slice";
import { Select } from "@/session/select";
import { Window } from "@/session/window";

const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

/** @returns the keys of the messages selected in this window's editor of the ICD. */
export const useSelectSelected = (key: icd.Key): icd.MessageKey[] =>
  Select.useMemo(
    (state: StoreState) =>
      (Window.selectDocument(state, selectSliceState(state), key) ?? ZERO_STATE)
        .selected,
    [key],
  );
