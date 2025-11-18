// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type Diagram, type Theming } from "@synnaxlabs/pluto";
import { xy } from "@synnaxlabs/x";

export const SLICE_NAME = "schematic";

export interface SetViewportPayload {
  key: string;
  viewport: Diagram.Viewport;
}

export interface AddElementPayload {
  elKey: string;
  props: schematic.Node;
  node?: Partial<Diagram.Node>;
}

export interface SetElementPropsPayload {
  key: string;
  props: Partial<schematic.Node> | Partial<schematic.Edge>;
}

export interface FixThemeContrastPayload {
  theme: Theming.ThemeSpec;
}

export interface SetNodesPayload {
  mode?: "replace" | "update";
  nodes: Diagram.Node[];
}

export interface SetNodePositionsPayload {
  positions: Record<string, xy.XY>;
}

export interface SetEdgesPayload {
  edges: Diagram.Edge[];
}

export type CreatePayload = schematic.State & {
  key: string;
};

export interface RemovePayload {
  keys: string[];
}

export interface SetEditablePayload {
  editable: boolean;
}

export interface SetSelectedSymbolGroupPayload {
  key: string;
  group: string;
}

export const actions: Record<
  string,
  (state: schematic.Schematic, payload: any) => void
> = {
  addElement: ({ data: schematic }, payload: AddElementPayload) => {
    const { elKey: key, props, node } = payload;
    schematic.nodes.push({
      key,
      selected: false,
      position: xy.ZERO,
      ...node,
    });
    schematic.props[key] = props;
  },
  setElementProps: ({ data: schematic }, payload: SetElementPropsPayload) => {
    const { key, props } = payload;
    if (key in schematic.props)
      schematic.props[key] = { ...schematic.props[key], ...props };
    else {
      const edge = schematic.edges.findIndex((edge) => edge.key === key);
      if (edge !== -1)
        schematic.edges[edge].data = { ...schematic.edges[edge].data, ...props };
    }
  },
  setNodes: ({ data: schematic }, payload: SetNodesPayload) => {
    const { nodes, mode = "replace" } = payload;
    if (mode === "replace") schematic.nodes = nodes;
    else {
      const keys = nodes.map((node) => node.key);
      schematic.nodes = [
        ...schematic.nodes.filter((node) => !keys.includes(node.key)),
        ...nodes,
      ];
    }
  },
  setNodePositions: ({ data: schematic }, payload: SetNodePositionsPayload) => {
    const { positions } = payload;
    Object.entries(positions).forEach(([key, position]) => {
      const node = schematic.nodes.find((node) => node.key === key);
      if (node == null) return;
      node.position = position;
    });
  },
  setEdges: ({ data: schematic }, payload: SetEdgesPayload) => {
    const { edges } = payload;
    // check for new edges
    const prevKeys = schematic.edges.map((edge) => edge.key);
    const newEdges = edges.filter((edge) => !prevKeys.includes(edge.key));
    newEdges.forEach((edge) => {
      const source = schematic.nodes.find((node) => node.key === edge.source);
      const target = schematic.nodes.find((node) => node.key === edge.target);
      if (source == null || target == null) return;
      const sourceProps = schematic.props[source.key];
      const targetProps = schematic.props[target.key];
      if (
        sourceProps.color === targetProps.color &&
        sourceProps.color != null &&
        edge.data != null
      )
        edge.data.color = sourceProps.color;
    });
    schematic.edges = edges;
  },
};

export const {
  setLegend,
  setNodePositions,
  toggleControl,
  setControlStatus,
  addElement,
  selectAll,
  setEdges,
  setNodes,
  remove,
  clearSelection,
  setSelectedSymbolGroup,
  setFitViewOnResize,
  create: internalCreate,
  setElementProps,
  setActiveToolbarTab,
  setViewport,
  setEditable,
  copySelection,
  pasteSelection,
  setViewportMode,
  setRemoteCreated,
  fixThemeContrast,
  setAuthority,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
