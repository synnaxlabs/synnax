// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type workspace } from "@synnaxlabs/client";

import { useMemoSelect } from "@/hooks";
import { SLICE_NAME, type SliceState, type StoreState } from "@/workspace/slice";

const selectState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectActiveKey = (state: StoreState): string | null =>
  selectState(state).active;

export const useSelectActiveKey = (): string | null =>
  useMemoSelect(selectActiveKey, []);

export const selectActive = (state: StoreState): workspace.Workspace | null => {
  const activeKey = selectActiveKey(state);
  if (activeKey == null) return null;
  return selectState(state).workspaces[activeKey];
};

export const useSelectActive = (): workspace.Workspace | null =>
  useMemoSelect(selectActive, []);

export const select = (state: StoreState, key: string): workspace.Workspace | null =>
  selectState(state).workspaces[key];

export const useSelect = (key: string): workspace.Workspace | null =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);
