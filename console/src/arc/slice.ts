// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type Diagram, type Theming, type Viewport } from "@synnaxlabs/pluto";
import { box, id, scale, xy } from "@synnaxlabs/x";

import * as latest from "@/arc/types";

export type SliceState = latest.SliceState;
export type NodeProps = latest.NodeProps;
export type State = latest.State;
export type ToolbarTab = latest.ToolbarTab;
export type ToolbarState = latest.ToolbarState;
export const ZERO_STATE = latest.ZERO_STATE;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;
export const migrateState = latest.migrateState;
export const anyStateZ = latest.anyStateZ;

export const SLICE_NAME = "arc";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const PERSIST_EXCLUDE = [];

export interface SetViewportPayload {
  key: string;
  viewport: Diagram.Viewport;
}

export interface AddElementPayload {
  key: string;
  elKey: string;
  props: NodeProps;
  node?: Partial<Diagram.Node>;
}

export interface SetElementPropsPayload {
  layoutKey: string;
  key: string;
  props: NodeProps;
}

export interface FixThemeContrastPayload {
  theme: Theming.ThemeSpec;
}

export interface SetNodesPayload {
  key: string;
  mode?: "replace" | "update";
  nodes: Diagram.Node[];
}

export interface SetNodePositionsPayload {
  key: string;
  positions: Record<string, xy.XY>;
}

export interface SetEdgesPayload {
  key: string;
  edges: Diagram.Edge[];
}

export type CreatePayload = latest.AnyState & {
  key: string;
};

export interface RemovePayload {
  keys: string[];
}

export interface SetEditablePayload {
  key: string;
  editable: boolean;
}

export interface SetFitViewOnResizePayload {
  key: string;
  fitViewOnResize: boolean;
}

export interface SetActiveToolbarTabPayload {
  tab: ToolbarTab;
}

export interface CopySelectionPayload {}

export interface PasteSelectionPayload {
  key: string;
  pos: xy.XY;
}

export interface ClearSelectionPayload {
  key: string;
}

export interface SetViewportModePayload {
  mode: Viewport.Mode;
}

export interface SetRemoteCreatedPayload {
  key: string;
}

export interface SelectAllPayload {
  key: string;
}

export interface SetRawTextPayload {
  key: string;
  raw: string;
}

export const calculatePos = (
  region: box.Box,
  cursor: xy.XY,
  viewport: Diagram.Viewport,
): xy.XY => {
  const zoomXY = xy.construct(viewport.zoom);
  const s = scale.XY.translate(xy.scale(box.topLeft(region), -1))
    .translate(xy.scale(viewport.position, -1))
    .magnify({
      x: 1 / zoomXY.x,
      y: 1 / zoomXY.y,
    });
  return s.pos(cursor);
};

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: latest.ZERO_SLICE_STATE,
  reducers: {
    copySelection: (state, _: PayloadAction<CopySelectionPayload>) => {
      // for each arc, find the keys of the selected nodes and edges
      // and add them to the copy buffer. Then get the props of each
      // selected node and edge and add them to the copy buffer.
      const { arcs } = state;
      const copyBuffer: latest.CopyBuffer = {
        nodes: [],
        edges: [],
        props: {},
        pos: xy.ZERO,
      };
      Object.values(arcs).forEach((arc) => {
        const { nodes, edges, props } = arc.graph;
        const selectedNodes = nodes.filter((node) => node.selected);
        const selectedEdges = edges.filter((edge) => edge.selected);
        copyBuffer.nodes = [...copyBuffer.nodes, ...selectedNodes];
        copyBuffer.edges = [...copyBuffer.edges, ...selectedEdges];
        selectedNodes.forEach((node) => (copyBuffer.props[node.key] = props[node.key]));
        selectedEdges.forEach((edge) => (copyBuffer.props[edge.key] = props[edge.key]));
      });
      const { nodes } = copyBuffer;
      if (nodes.length > 0) {
        const pos = nodes.reduce(
          (acc, node) => xy.translate(acc, node.position),
          xy.ZERO,
        );
        copyBuffer.pos = xy.scale(pos, 1 / nodes.length);
      }
      state.copy = copyBuffer;
    },
    pasteSelection: (state, { payload }: PayloadAction<PasteSelectionPayload>) => {
      const { pos, key: layoutKey } = payload;
      const console = xy.translation(state.copy.pos, pos);
      const arc = state.arcs[layoutKey];
      const keys: Record<string, string> = {};
      const nextNodes = state.copy.nodes.map((node) => {
        const key: string = id.create();
        arc.graph.props[key] = state.copy.props[node.key];
        keys[node.key] = key;
        return {
          ...node,
          position: xy.translate(node.position, console),
          key,
          selected: true,
        };
      });
      const nextEdges = state.copy.edges.map((edge) => {
        const key: string = id.create();
        return {
          ...edge,
          key,
          source: keys[edge.source],
          target: keys[edge.target],
          selected: true,
        };
      });
      arc.graph.edges = [
        ...arc.graph.edges.map((edge) => ({ ...edge, selected: false })),
        ...nextEdges,
      ];
      arc.graph.nodes = [
        ...arc.graph.nodes.map((node) => ({ ...node, selected: false })),
        ...nextNodes,
      ];
    },
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const { key: layoutKey } = payload;
      state.arcs[layoutKey] = latest.migrateState(payload);
      state.toolbar.activeTab = "stages";
    },
    clearSelection: (state, { payload }: PayloadAction<ClearSelectionPayload>) => {
      const { key: layoutKey } = payload;
      const arc = state.arcs[layoutKey];
      arc.graph.nodes.forEach((node) => {
        node.selected = false;
      });
      arc.graph.edges.forEach((edge) => {
        edge.selected = false;
      });
      state.toolbar.activeTab = "stages";
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      const { keys: layoutKeys } = payload;
      layoutKeys.forEach((layoutKey) => {
        const arc = state.arcs[layoutKey];
        if (arc == null) return;
        delete state.arcs[layoutKey];
      });
    },
    addElement: (state, { payload }: PayloadAction<AddElementPayload>) => {
      const { key: layoutKey, elKey: key, props, node } = payload;
      const arc = state.arcs[layoutKey];
      if (!arc.graph.editable) return;
      arc.graph.nodes.push({
        key,
        selected: false,
        position: xy.ZERO,
        ...node,
      });
      arc.graph.props[key] = props;
    },
    setElementProps: (state, { payload }: PayloadAction<SetElementPropsPayload>) => {
      const { layoutKey, key, props } = payload;
      const arc = state.arcs[layoutKey];
      if (key in arc.graph.props)
        arc.graph.props[key] = { ...arc.graph.props[key], ...props };
      else {
        const edge = arc.graph.edges.findIndex((edge) => edge.key === key);
        if (edge !== -1) arc.graph.edges[edge] = { ...arc.graph.edges[edge], ...props };
      }
    },
    setNodes: (state, { payload }: PayloadAction<SetNodesPayload>) => {
      const { key: layoutKey, nodes, mode = "replace" } = payload;
      const arc = state.arcs[layoutKey];
      if (mode === "replace") arc.graph.nodes = nodes;
      else {
        const keys = nodes.map((node) => node.key);
        arc.graph.nodes = [
          ...arc.graph.nodes.filter((node) => !keys.includes(node.key)),
          ...nodes,
        ];
      }
      const anySelected =
        nodes.some((node) => node.selected) ||
        arc.graph.edges.some((edge) => edge.selected);
      if (anySelected) {
        if (state.toolbar.activeTab !== "properties")
          clearOtherSelections(state, layoutKey);
        state.toolbar.activeTab = "properties";
      } else state.toolbar.activeTab = "stages";
    },
    setNodePositions: (state, { payload }: PayloadAction<SetNodePositionsPayload>) => {
      const { key: layoutKey, positions } = payload;
      const arc = state.arcs[layoutKey];
      Object.entries(positions).forEach(([key, position]) => {
        const node = arc.graph.nodes.find((node) => node.key === key);
        if (node == null) return;
        node.position = position;
      });
    },
    setEdges: (state, { payload }: PayloadAction<SetEdgesPayload>) => {
      const { key: layoutKey, edges } = payload;
      const arc = state.arcs[layoutKey];
      arc.graph.edges = edges;
      const anySelected =
        edges.some((edge) => edge.selected) ||
        arc.graph.nodes.some((node) => node.selected);
      if (anySelected) {
        if (state.toolbar.activeTab !== "properties")
          clearOtherSelections(state, layoutKey);
        state.toolbar.activeTab = "properties";
      } else state.toolbar.activeTab = "stages";
    },
    setActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetActiveToolbarTabPayload>,
    ) => {
      const { tab } = payload;
      state.toolbar.activeTab = tab;
    },
    setViewport: (state, { payload }: PayloadAction<SetViewportPayload>) => {
      const { key: layoutKey, viewport } = payload;
      const arc = state.arcs[layoutKey];
      arc.graph.viewport = viewport;
    },
    setEditable: (state, { payload }: PayloadAction<SetEditablePayload>) => {
      const { key: layoutKey, editable } = payload;
      const arc = state.arcs[layoutKey];
      clearSelections(arc);
      arc.graph.editable = editable;
    },
    setFitViewOnResize: (
      state,
      { payload }: PayloadAction<SetFitViewOnResizePayload>,
    ) => {
      const { key: layoutKey, fitViewOnResize } = payload;
      const arc = state.arcs[layoutKey];
      arc.graph.fitViewOnResize = fitViewOnResize;
    },
    setViewportMode: (
      state,
      { payload: { mode } }: PayloadAction<SetViewportModePayload>,
    ) => {
      state.mode = mode;
    },
    setRemoteCreated: (state, { payload }: PayloadAction<SetRemoteCreatedPayload>) => {
      const { key: layoutKey } = payload;
      const arc = state.arcs[layoutKey];
      arc.remoteCreated = true;
    },
    selectAll: (state, { payload }: PayloadAction<SelectAllPayload>) => {
      const { key: layoutKey } = payload;
      const arc = state.arcs[layoutKey];
      arc.graph.nodes.forEach((node) => (node.selected = true));
      arc.graph.edges.forEach((edge) => (edge.selected = true));
    },
    setRawText: (state, { payload }: PayloadAction<SetRawTextPayload>) => {
      const { key: layoutKey, raw } = payload;
      const arc = state.arcs[layoutKey];
      arc.text.raw = raw;
    },
  },
});

const clearOtherSelections = (state: SliceState, layoutKey: string): void => {
  Object.keys(state.arcs).forEach((key) => {
    // If any of the nodes or edges in other Diagram slices are selected, deselect them.
    if (key === layoutKey) return;
    clearSelections(state.arcs[key]);
  });
};

const clearSelections = (state: State): void => {
  state.graph.nodes.forEach((node) => (node.selected = false));
  state.graph.edges.forEach((edge) => (edge.selected = false));
};

export const {
  setNodePositions,
  addElement,
  selectAll,
  setEdges,
  setNodes,
  remove,
  clearSelection,
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
  setRawText,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
