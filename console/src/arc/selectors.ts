// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { Arc, type Diagram, type Viewport } from "@synnaxlabs/pluto";

import {
  type NodeProps,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarState,
} from "@/arc/slice";
import { type Mode } from "@/arc/types";
import { useMemoSelect } from "@/hooks";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).arcs[key];

export const selectOptional = select as (
  state: StoreState,
  key: string,
) => State | undefined;

export const useSelect = (key: string): State =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const useSelectOptional = useSelect as (key: string) => State | undefined;

export const selectMany = (state: StoreState, keys: string[]): State[] =>
  keys.map((key) => select(state, key));

export const useSelectMany = (keys: string[]): State[] =>
  useMemoSelect((state: StoreState) => selectMany(state, keys), [keys]);

export interface ElementDigest {
  key: string;
  type: "node" | "edge";
}

export const selectSelectedElementDigests = (
  state: StoreState,
  layoutKey: string,
): ElementDigest[] => {
  const arc = select(state, layoutKey);
  return arc.graph.nodes
    .filter((node) => node.selected)
    .map((node) => ({ key: node.key, type: "node" }));
};

export const useSelectSelectedElementDigests = (layoutKey: string): ElementDigest[] =>
  useMemoSelect(
    (state: StoreState) => selectSelectedElementDigests(state, layoutKey),
    [layoutKey],
  );

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

export const selectSelectedElementsProps = (
  state: StoreState,
  layoutKey: string,
): ElementInfo[] => {
  const arc = select(state, layoutKey);
  if (arc == null) return [];
  const nodes: ElementInfo[] = arc.graph.nodes
    .filter((node) => node.selected)
    .map((node) => ({
      key: node.key,
      type: "node",
      node,
      props: arc.graph.props[node.key],
    }));
  const edges: ElementInfo[] = arc.graph.edges
    .filter((edge) => edge.selected)
    .map((edge) => ({ key: edge.key, type: "edge", edge }));
  return [...nodes, ...edges];
};

export const selectEdge = (
  state: StoreState,
  layoutKey: string,
  key: string,
): Diagram.Edge | undefined =>
  select(state, layoutKey).graph.edges.find((edge) => edge.key === key);

export const useSelectEdge = (
  layoutKey: string,
  key: string,
): Diagram.Edge | undefined =>
  useMemoSelect(
    (state: StoreState) => selectEdge(state, layoutKey, key),
    [layoutKey, key],
  );

export const selectRequiredEdge = (
  state: StoreState,
  layoutKey: string,
  key: string,
): Diagram.Edge => {
  const edge = selectEdge(state, layoutKey, key);
  if (edge == null) throw new UnexpectedError(`Edge not found for key: ${key}`);
  return edge;
};

export const useSelectRequiredEdge = (layoutKey: string, key: string): Diagram.Edge =>
  useMemoSelect(
    (state: StoreState) => selectRequiredEdge(state, layoutKey, key),
    [layoutKey, key],
  );

export const useSelectSelectedElementsProps = (layoutKey: string): ElementInfo[] =>
  useMemoSelect(
    (state: StoreState) => selectSelectedElementsProps(state, layoutKey),
    [layoutKey],
  );

export const selectSelectedElementNames = (
  state: StoreState,
  layoutKey: string,
): (string | null)[] => {
  const elements = selectSelectedElementsProps(state, layoutKey);
  return elements.map((element) => {
    if (element.type === "node")
      return Arc.Stage.REGISTRY[element.props.key]?.name ?? null;
    return null;
  });
};

export const useSelectSelectedElementNames = (layoutKey: string): (string | null)[] =>
  useMemoSelect(
    (s: StoreState) => selectSelectedElementNames(s, layoutKey),
    [layoutKey],
  );

export const selectNodeProps = (
  state: StoreState,
  layoutKey: string,
  key: string,
): NodeProps | undefined => select(state, layoutKey).graph.props[key];

export const selectRequiredNodeProps = (
  state: StoreState,
  layoutKey: string,
  key: string,
): NodeProps => {
  const props = selectNodeProps(state, layoutKey, key);
  if (props == null) throw new UnexpectedError(`Node props not found for key: ${key}`);
  return props;
};

export const useSelectNodeProps = (
  layoutKey: string,
  key: string,
): NodeProps | undefined =>
  useMemoSelect(
    (state: StoreState) => selectNodeProps(state, layoutKey, key),
    [layoutKey, key],
  );

export const useSelectRequiredNodeProps = (layoutKey: string, key: string): NodeProps =>
  useMemoSelect(
    (state: StoreState) => selectRequiredNodeProps(state, layoutKey, key),
    [layoutKey, key],
  );

export const selectToolbar = (state: StoreState): ToolbarState =>
  selectSliceState(state).toolbar;

export const useSelectToolbar = (): ToolbarState => useMemoSelect(selectToolbar, []);

export const selectEditable = (state: StoreState, key: string): boolean =>
  select(state, key).graph.editable;

export const useSelectEditable = (key: string): boolean =>
  useMemoSelect((state: StoreState) => selectEditable(state, key), [key]);

export const selectViewportMode = (state: StoreState): Viewport.Mode =>
  selectSliceState(state).mode;

export const useSelectViewportMode = (): Viewport.Mode =>
  useMemoSelect(selectViewportMode, []);

export const selectViewport = (state: StoreState, key: string): Diagram.Viewport =>
  select(state, key).graph.viewport;

export const useSelectViewport = (key: string): Diagram.Viewport =>
  useMemoSelect((state: StoreState) => selectViewport(state, key), [key]);

export const selectVersion = (state: StoreState, key: string): string | undefined =>
  selectOptional(state, key)?.version;

export const useSelectVersion = (key: string): string | undefined =>
  useMemoSelect((state: StoreState) => selectVersion(state, key), [key]);

export const selectMode = (state: StoreState, key: string): Mode =>
  select(state, key).mode;

export const useSelectMode = (key: string): Mode =>
  useMemoSelect((state: StoreState) => selectMode(state, key), [key]);
