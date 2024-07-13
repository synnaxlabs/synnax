// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Control, type Diagram, type Viewport } from "@synnaxlabs/pluto";

import { useMemoSelect } from "@/hooks";
import {
  type NodeProps,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarState,
} from "@/schematic/slice";

export const selectSliceState = (state: StoreState): SliceState => state.schematic;

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).schematics[key];

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
  const schematic = select(state, layoutKey);
  const nodes: ElementInfo[] = schematic.nodes
    .filter((node) => node.selected)
    .map((node) => ({
      key: node.key,
      type: "node",
      node,
      props: schematic.props[node.key],
    }));
  const edges: ElementInfo[] = schematic.edges
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
  node: Diagram.Node;
  props: NodeProps;
}

export interface EdgeElementInfo {
  key: string;
  type: "edge";
  edge: Diagram.Edge;
}

export type ElementInfo = NodeElementInfo | EdgeElementInfo;

export const useSelectSelectedElementsProps = (layoutKey: string): ElementInfo[] =>
  useMemoSelect(
    (state: StoreState) => selectSelectedElementsProps(state, layoutKey),
    [layoutKey],
  );

export const selectNodeProps = (
  state: StoreState,
  layoutKey: string,
  key: string,
): NodeProps => {
  const schematic = select(state, layoutKey);
  console.log(schematic);
  console.log(schematic.props[key]);
  return schematic.props[key];
};

export const useSelectNodeProps = (layoutKey: string, key: string): NodeProps => {
  return useMemoSelect(
    (state: StoreState) => selectNodeProps(state, layoutKey, key),
    [layoutKey, key],
  );
};

export const selectToolbar = (state: StoreState): ToolbarState =>
  selectSliceState(state).toolbar;

export const useSelectToolbar = (): ToolbarState => useMemoSelect(selectToolbar, []);

export const selectEditable = (state: StoreState, key: string): boolean =>
  select(state, key).editable;

export const useSelectEditable = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectEditable(state, key), [key]);

export const selectViewportMode = (state: StoreState): Viewport.Mode =>
  selectSliceState(state).mode;

export const useSelectViewportMode = (): Viewport.Mode =>
  useMemoSelect(selectViewportMode, []);

export const selectViewport = (state: StoreState, key: string): Diagram.Viewport =>
  select(state, key).viewport;

export const useSelectViewport = (key: string): Diagram.Viewport =>
  useMemoSelect((state: StoreState) => selectViewport(state, key), [key]);

export const selectControlStatus = (
  state: StoreState,
  layoutKey: string,
): Control.Status => select(state, layoutKey).control;

export const useSelectControlStatus = (layoutKey: string): Control.Status =>
  useMemoSelect(
    (state: StoreState) => selectControlStatus(state, layoutKey),
    [layoutKey],
  );
