// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project, UnexpectedError } from "@synnaxlabs/client";

import { SLICE_NAME, type SliceState, type StoreState } from "@/session/project/slice";
import { Select } from "@/session/select";

const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectOptionalSelected = (state: StoreState): project.Key | undefined =>
  selectSliceState(state).selected;

export const useSelectedOptionalSelected = (): project.Key | undefined =>
  Select.useMemo(selectOptionalSelected, []);

export const selectSelected = (state: StoreState): project.Key => {
  const selected = selectOptionalSelected(state);
  if (selected == null)
    throw new UnexpectedError(`Project.selectActive must be used within Project.Guard`);
  return selected;
};

export const useSelectSelected = (): project.Key => Select.useMemo(selectSelected, []);
