// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemoSelect } from "@/hooks";
import { type Permissions } from "@/permissions/permissions";
import { SLICE_NAME, type SliceState, type StoreState } from "@/permissions/slice";

const selectState = (state: StoreState): SliceState => state[SLICE_NAME];

const selectAdmin = (state: StoreState): boolean =>
  selectState(state).permissions.admin;

export const useSelectAdmin = (): boolean => useMemoSelect(selectAdmin, []);

export const selectSchematic = (state: StoreState): boolean =>
  selectState(state).permissions.schematic;

export const useSelectSchematic = (): boolean => useMemoSelect(selectSchematic, []);

const selectAll = (state: StoreState): Permissions => selectState(state).permissions;

export const useSelectAll = (): Permissions => useMemoSelect(selectAll, []);
