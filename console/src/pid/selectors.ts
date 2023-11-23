// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type PIDSymbols,
  type Control,
  type PID,
  type Viewport,
} from "@synnaxlabs/pluto";

import { useMemoSelect } from "@/hooks";
import {
  type NodeProps,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarState,
} from "@/pid/slice";

export const selectSliceState = (state: StoreState): SliceState => state.pid;

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).pids[key];

export const useSelect = (key: string): State =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const selectMany = (state: StoreState, keys: string[]): State[] =>
  keys.map((key) => select(state, key));

export const useSelectMany = (keys: string[]): State[] =>
  useMemoSelect((state: StoreState) => selectMany(state, keys), [keys]);

export const selectSelectedElementsProps = (
  state: StoreState,
  layoutKey: string,
): ElementInfo[] => {
  const pid = select(state, layoutKey);
  const nodes: ElementInfo[] = pid.nodes
    .filter((node) => node.selected)
    .map((node) => ({
      key: node.key,
      type: "node",
      node,
      props: pid.props[node.key],
    }));
  const edges: ElementInfo[] = pid.edges
    .filter((edge) => edge.selected)
    .map((edge) => ({
      key: edge.key,
      type: "edge",
      edge,
    }));
  return [...nodes, ...edges];
};

export interface NodeElementInfo {
  key: string;
  type: "node";
  node: PID.Node;
  variant: PIDSymbols.Variant;
  props: NodeProps;
}

export interface EdgeElementInfo {
  key: string;
  type: "edge";
  edge: PID.Edge;
  props: NodeProps;
}

export type ElementInfo = NodeElementInfo | EdgeElementInfo;

export const useSelectSelectedElementsProps = (layoutKey: string): ElementInfo[] =>
  useMemoSelect(
    (state: StoreState) => selectSelectedElementsProps(state, layoutKey),
    [layoutKey],
  );

export const selectElementProps = (
  state: StoreState,
  layoutKey: string,
  key: string,
): ElementInfo => {
  const pid = select(state, layoutKey);
  const node = pid.nodes.find((node) => node.key === key);
  return {
    key,
    node: node as PID.Node,
    props: pid.props[key],
  };
};

export const useSelectElementProps = (layoutKey: string, key: string): ElementInfo =>
  useMemoSelect(
    (state: StoreState) => selectElementProps(state, layoutKey, key),
    [layoutKey, key],
  );

export const selectToolbar = (state: StoreState): ToolbarState =>
  selectSliceState(state).toolbar;

export const useSelectToolbar = (): ToolbarState => useMemoSelect(selectToolbar, []);

export const selectEditable = (state: StoreState, key: string): boolean =>
  select(state, key).editable;

export const useSelectEditable = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectEditable(state, key), [key]);

export const selectViewportMode = (state: StoreState): Viewport.Mode =>
  selectSliceState(state).mode;

export const useSelectViewporMode = (): Viewport.Mode =>
  useMemoSelect(selectViewportMode, []);

export const selecControlStatus = (
  state: StoreState,
  layoutKey: string,
): Control.Status => select(state, layoutKey).control;

export const useSelectControlStatus = (layoutKey: string): Control.Status =>
  useMemoSelect(
    (state: StoreState) => selecControlStatus(state, layoutKey),
    [layoutKey],
  );
