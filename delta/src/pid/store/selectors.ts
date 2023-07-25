// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemoSelect } from "@/hooks";
import {
  PIDNode,
  PIDSliceState,
  PIDState,
  PIDStoreState,
  PIDToolbarState,
} from "@/pid/store/slice";

export const selectPIDState = (state: PIDStoreState): PIDSliceState => state.pid;

export const selectPID = (state: PIDStoreState, key: string): PIDState =>
  selectPIDState(state).pids[key];

export const useSelectPID = (key: string): PIDState =>
  useMemoSelect((state: PIDStoreState) => selectPID(state, key), [key]);

export const selectSelectedPIDElementsProps = (
  state: PIDStoreState,
  layoutKey: string
): PIDElementInfo[] => {
  const pid = selectPID(state, layoutKey);
  const selected = pid.nodes.filter((node) => node.selected);
  return selected.map((node) => ({
    key: node.key,
    node,
    props: pid.props[node.key],
  }));
};

export interface PIDElementInfo {
  key: string;
  node: PIDNode;
  props: object;
}

export const useSelectSelectedPIDElementsProps = (
  layoutKey: string
): PIDElementInfo[] =>
  useMemoSelect(
    (state: PIDStoreState) => selectSelectedPIDElementsProps(state, layoutKey),
    [layoutKey]
  );

export const selectPIDElementProps = (
  state: PIDStoreState,
  layoutKey: string,
  key: string
): PIDElementInfo => {
  const pid = selectPID(state, layoutKey);
  const node = pid.nodes.find((node) => node.key === key);
  return {
    key,
    node: node as PIDNode,
    props: pid.props[key],
  };
};

export const useSelectPIDElementProps = (
  layoutKey: string,
  key: string
): PIDElementInfo =>
  useMemoSelect(
    (state: PIDStoreState) => selectPIDElementProps(state, layoutKey, key),
    [layoutKey, key]
  );

export const selectPIDToolbar = (state: PIDStoreState): PIDToolbarState =>
  selectPIDState(state).toolbar;

export const useSelectPIDToolbar = (): PIDToolbarState =>
  useMemoSelect(selectPIDToolbar, []);

export const selectPIDEditable = (state: PIDStoreState, key: string): boolean =>
  selectPID(state, key).editable;

export const useSelectPIDEditable = (key: string): boolean =>
  useMemoSelect((state: PIDStoreState) => selectPIDEditable(state, key), [key]);
