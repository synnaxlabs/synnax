// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type workspace } from "@synnaxlabs/client";

import { useMemoSelect } from "@/hooks";
import { SLICE_NAME, type StoreState } from "@/workspace/slice";
import { type SliceState, type Workspace } from "@/workspace/types";

const selectState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectActive = (state: StoreState): Workspace | null =>
  selectState(state).active;

export const useSelectActive = (): Workspace | null => useMemoSelect(selectActive, []);

export const selectActiveKey = (state: StoreState): workspace.Key | null =>
  selectState(state).active?.key ?? null;

export const useSelectActiveKey = (): workspace.Key | null =>
  useMemoSelect(selectActiveKey, []);

export const selectActiveName = (state: StoreState): string | null =>
  selectState(state).active?.name ?? null;

export const useSelectActiveName = (): string | null =>
  useMemoSelect(selectActiveName, []);
