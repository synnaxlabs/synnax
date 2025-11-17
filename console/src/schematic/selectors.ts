// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { type Control, type Diagram, type Viewport } from "@synnaxlabs/pluto";

import { useMemoSelect } from "@/hooks";
import { Permissions } from "@/permissions";
import {
  type NodeProps,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
  type ToolbarState,
} from "@/schematic/slice";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectRequired = (state: StoreState, key: string): State => {
  const schematic = selectSliceState(state).schematics[key];
  if (schematic == null)
    throw new UnexpectedError(`Schematic not found for key: ${key}`);
  return schematic;
};

export const useSelectRequired = (key: string): State =>
  useMemoSelect((state: StoreState) => selectRequired(state, key), [key]);

export const selectOptional = (state: StoreState, key: string): State | undefined =>
  selectSliceState(state).schematics[key];

export const useSelectOptional = (key: string): State | undefined =>
  useMemoSelect((state: StoreState) => selectOptional(state, key), [key]);

export const selectRequiredMany = (state: StoreState, keys: string[]): State[] =>
  keys.map((key) => selectRequired(state, key));

export const useSelectRequiredMany = (keys: string[]): State[] =>
  useMemoSelect((state: StoreState) => selectRequiredMany(state, keys), [keys]);

export interface ElementDigest {
  key: string;
  type: "node" | "edge";
}

export const selectSelectedElementDigests = (
  state: StoreState,
  layoutKey: string,
): ElementDigest[] => {
  const schematic = selectOptional(state, layoutKey);
  if (schematic == null) return [];
  return [
    ...schematic.nodes
      .filter((node) => node.selected)
      .map<ElementDigest>((node) => ({ key: node.key, type: "node" })),
    ...schematic.edges
      .filter((edge) => edge.selected)
      .map<ElementDigest>((edge) => ({ key: edge.key, type: "edge" })),
  ];
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
  const schematic = selectOptional(state, layoutKey);
  if (schematic == null) return [];
  const nodes: ElementInfo[] = schematic.nodes
    .filter((node) => node.selected)
    .map((node) => ({
      key: node.key,
      type: "node",
      node,
      props: schematic.props[node.key] ?? {},
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

export const useSelectSelectedElementsProps = (layoutKey: string): ElementInfo[] =>
  useMemoSelect(
    (state: StoreState) => selectSelectedElementsProps(state, layoutKey),
    [layoutKey],
  );

export const selectEdge = (
  state: StoreState,
  layoutKey: string,
  key: string,
): Diagram.Edge | undefined =>
  selectOptional(state, layoutKey)?.edges.find((edge) => edge.key === key);

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

export const selectSelectedElementNames = (
  state: StoreState,
  layoutKey: string,
): (string | null)[] => {
  const elements = selectSelectedElementsProps(state, layoutKey);
  return elements.map((element) => {
    if (element.type === "node" && element.props?.label?.label != null)
      return element.props.label.label;
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
): NodeProps | undefined => selectOptional(state, layoutKey)?.props[key];

export const useSelectNodeProps = (
  layoutKey: string,
  key: string,
): NodeProps | undefined =>
  useMemoSelect(
    (state: StoreState) => selectNodeProps(state, layoutKey, key),
    [layoutKey, key],
  );

export const selectRequiredNodeProps = (
  state: StoreState,
  layoutKey: string,
  key: string,
): NodeProps => {
  const props = selectNodeProps(state, layoutKey, key);
  if (props == null) throw new UnexpectedError(`Node props not found for key: ${key}`);
  return props;
};

export const useSelectRequiredNodeProps = (layoutKey: string, key: string): NodeProps =>
  useMemoSelect(
    (state: StoreState) => selectRequiredNodeProps(state, layoutKey, key),
    [layoutKey, key],
  );

export const selectToolbar = (
  state: StoreState,
  key: string,
): ToolbarState | undefined => selectOptional(state, key)?.toolbar;

export const useSelectToolbar = (key: string): ToolbarState | undefined =>
  useMemoSelect((state: StoreState) => selectToolbar(state, key), [key]);

export const selectEditable = (state: StoreState, key: string): boolean | undefined =>
  selectOptional(state, key)?.editable;

export const useSelectEditable = (key: string): boolean | undefined =>
  useMemoSelect((state: StoreState) => selectEditable(state, key), [key]);

export const selectRequiredViewportMode = (
  state: StoreState,
  key: string,
): Viewport.Mode => selectRequired(state, key).mode;

export const useSelectRequiredViewportMode = (key: string): Viewport.Mode =>
  useMemoSelect((state: StoreState) => selectRequiredViewportMode(state, key), [key]);

export const selectViewport = (
  state: StoreState,
  key: string,
): Diagram.Viewport | undefined => selectOptional(state, key)?.viewport;

export const useSelectViewport = (key: string): Diagram.Viewport | undefined =>
  useMemoSelect((state: StoreState) => selectViewport(state, key), [key]);

export const selectControlStatus = (
  state: StoreState,
  layoutKey: string,
): Control.Status | undefined => selectOptional(state, layoutKey)?.control;

export const useSelectControlStatus = (layoutKey: string): Control.Status | undefined =>
  useMemoSelect(
    (state: StoreState) => selectControlStatus(state, layoutKey),
    [layoutKey],
  );

export const selectHasPermission = (state: Permissions.StoreState): boolean =>
  Permissions.selectCanUseType(state, "schematic");

export const useSelectHasPermission = (): boolean =>
  useMemoSelect(selectHasPermission, []);

export const selectVersion = (state: StoreState, key: string): string | undefined =>
  selectOptional(state, key)?.version;

export const useSelectVersion = (key: string): string | undefined =>
  useMemoSelect((state: StoreState) => selectVersion(state, key), [key]);

export const selectIsSnapshot = (state: StoreState, key: string): boolean | undefined =>
  selectOptional(state, key)?.snapshot;

export const useSelectIsSnapshot = (key: string): boolean | undefined =>
  useMemoSelect((state: StoreState) => selectIsSnapshot(state, key), [key]);

export const selectAuthority = (state: StoreState, key: string): number | undefined =>
  selectOptional(state, key)?.authority;

export const useSelectAuthority = (key: string): number | undefined =>
  useMemoSelect((state: StoreState) => selectAuthority(state, key), [key]);

export const selectSelectedSymbolGroup = (state: StoreState, key: string): string =>
  selectRequired(state, key).toolbar.selectedSymbolGroup;

export const useSelectSelectedSymbolGroup = (key: string): string =>
  useMemoSelect((state: StoreState) => selectSelectedSymbolGroup(state, key), [key]);
