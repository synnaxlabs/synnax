// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { project, UnexpectedError } from "@synnaxlabs/client";
import z from "zod";

import { useMemoSelect } from "@/hooks";

export const SLICE_NAME = "project";

export const sliceStateZ = z.object({
  selected: project.keyZ.optional(),
});

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type SelectPayload = project.Key | undefined;

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: sliceStateZ.parse({}),
  reducers: {
    select: (state, { payload }: PayloadAction<SelectPayload>) => {
      state.selected = payload;
    },
  },
});

export const { select } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

const selectState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectSelected = (state: StoreState): project.Key => {
  const { selected } = selectState(state);
  if (selected == null)
    throw new UnexpectedError(
      `Project.useSelectSelected must be called in a context where a project is guaranteed to be selected`,
    );
  return selected;
};

export const useSelectSelected = (): project.Key => useMemoSelect(selectSelected, []);

export const selectIsAnySelected = (state: StoreState): boolean =>
  selectState(state).selected != null;

export const useSelectIsAnySelected = (): boolean =>
  useMemoSelect(selectIsAnySelected, []);
