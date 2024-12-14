// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
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

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).schematics[key];

export const useSelect = (key: string): State =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const selectMany = (state: StoreState, keys: string[]): State[] =>
  keys.map((key) => select(state, key));

export const useSelectMany = (keys: string[]): State[] =>
  useMemoSelect((state: StoreState) => selectMany(state, keys), [keys]);

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
  const schematic = select(state, layoutKey);
  if (schematic == null) return [];
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
    if (element.type === "node") return element.props.label?.label ?? null;
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
): NodeProps | undefined => select(state, layoutKey).props[key];

export const useSelectNodeProps = (
  layoutKey: string,
  key: string,
): NodeProps | undefined =>
  useMemoSelect(
    (state: StoreState) => selectNodeProps(state, layoutKey, key),
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

export const selectHasPermission = (state: Permissions.StoreState): boolean =>
  Permissions.selectCanUseType(state, schematic.ONTOLOGY_TYPE);

export const useSelectHasPermission = (): boolean =>
  useMemoSelect(selectHasPermission, []);

export const selectVersion = (state: StoreState, key: string): string | undefined =>
  select(state, key).version;

export const useSelectVersion = (key: string): string | undefined =>
  useMemoSelect((state: StoreState) => selectVersion(state, key), [key]);
