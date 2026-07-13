// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Node,
  SLICE_NAME,
  type SliceState,
  type StoreState,
} from "@/session/node/slice";
import { Select } from "@/session/select";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const useSelectSliceState = (): SliceState =>
  Select.useMemo((s: StoreState) => selectSliceState(s), []);

export const selectSelectedKey = (state: StoreState): string | undefined =>
  selectSliceState(state).selected;

export const useSelectSelectedKey = (): string | undefined =>
  Select.useMemo((s: StoreState) => selectSelectedKey(s), []);

export const selectState = (state: StoreState, key?: string): Node | undefined =>
  Select.byKey(selectSliceState(state).nodes, key, selectSelectedKey(state));

export const useSelectState = (key?: string): Node | undefined =>
  Select.useMemo((s: StoreState) => selectState(s, key), [key]);

export const selectMany = (state: StoreState, keys?: string[]): Node[] =>
  Select.byKeys(state.node.nodes, keys);

export const useSelectMany = (keys?: string[]): Node[] =>
  Select.useMemo((s: StoreState) => selectMany(s, keys), [keys]);

export const selectAllNames = (state: StoreState): string[] =>
  Object.values(selectSliceState(state).nodes).map((c) => c.name);

export const useSelectAllNames = (): string[] =>
  Select.useMemo((s: StoreState) => selectAllNames(s), []);

export const selectIsAnySelected = (state: StoreState): boolean =>
  selectSelectedKey(state) != null;

export const useSelectIsAnySelected = (): boolean =>
  Select.useMemo(selectIsAnySelected, []);
