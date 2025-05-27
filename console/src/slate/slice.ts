// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import {
  type Control,
  type Diagram,
  type Theming,
  type Viewport,
} from "@synnaxlabs/pluto";
import { box, color, id, scale, xy } from "@synnaxlabs/x";

import * as latest from "@/slate/types";

export type SliceState = latest.SliceState;
export type NodeProps = latest.NodeProps;
export type State = latest.State;
export type StateWithName = State & { name: string };
export type ToolbarTab = latest.ToolbarTab;
export type ToolbarState = latest.ToolbarState;
export const ZERO_STATE = latest.ZERO_STATE;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;
export const migrateState = latest.migrateState;
export const anyStateZ = latest.anyStateZ;

export const SLICE_NAME = "slate";

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

export interface SetControlStatusPayload {
  key: string;
  control: Control.Status;
}

export interface ToggleControlPayload {
  key: string;
  status: Control.Status;
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
      // for each slate, find the keys of the selected nodes and edges
      // and add them to the copy buffer. Then get the props of each
      // selected node and edge and add them to the copy buffer.
      const { slates } = state;
      const copyBuffer: latest.CopyBuffer = {
        nodes: [],
        edges: [],
        props: {},
        pos: xy.ZERO,
      };
      Object.values(slates).forEach((slate) => {
        const { nodes, edges, props } = slate;
        const selectedNodes = nodes.filter((node) => node.selected);
        const selectedEdges = edges.filter((edge) => edge.selected);
        copyBuffer.nodes = [...copyBuffer.nodes, ...selectedNodes];
        copyBuffer.edges = [...copyBuffer.edges, ...selectedEdges];
        selectedNodes.forEach((node) => {
          copyBuffer.props[node.key] = props[node.key];
        });
        selectedEdges.forEach((edge) => {
          copyBuffer.props[edge.key] = props[edge.key];
        });
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
      const slate = state.slates[layoutKey];
      const keys: Record<string, string> = {};
      const nextNodes = state.copy.nodes.map((node) => {
        const key: string = id.create();
        slate.props[key] = state.copy.props[node.key];
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
      slate.edges = [
        ...slate.edges.map((edge) => ({ ...edge, selected: false })),
        ...nextEdges,
      ];
      slate.nodes = [
        ...slate.nodes.map((node) => ({ ...node, selected: false })),
        ...nextNodes,
      ];
    },
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const { key: layoutKey } = payload;
      state.slates[layoutKey] = latest.migrateState(payload);
      state.toolbar.activeTab = "symbols";
    },
    clearSelection: (state, { payload }: PayloadAction<ClearSelectionPayload>) => {
      const { key: layoutKey } = payload;
      const slate = state.slates[layoutKey];
      slate.nodes.forEach((node) => {
        node.selected = false;
      });
      slate.edges.forEach((edge) => {
        edge.selected = false;
      });
      state.toolbar.activeTab = "symbols";
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      const { keys: layoutKeys } = payload;
      layoutKeys.forEach((layoutKey) => {
        const slate = state.slates[layoutKey];
        if (slate == null) return;
        delete state.slates[layoutKey];
      });
    },
    addElement: (state, { payload }: PayloadAction<AddElementPayload>) => {
      const { key: layoutKey, elKey: key, props, node } = payload;
      const slate = state.slates[layoutKey];
      if (!slate.editable) return;
      slate.nodes.push({
        key,
        selected: false,
        position: xy.ZERO,
        ...node,
      });
      slate.props[key] = props;
    },
    setElementProps: (state, { payload }: PayloadAction<SetElementPropsPayload>) => {
      const { layoutKey, key, props } = payload;
      const slate = state.slates[layoutKey];
      if (key in slate.props) slate.props[key] = { ...slate.props[key], ...props };
      else {
        const edge = slate.edges.findIndex((edge) => edge.key === key);
        if (edge !== -1) slate.edges[edge] = { ...slate.edges[edge], ...props };
      }
    },
    setNodes: (state, { payload }: PayloadAction<SetNodesPayload>) => {
      const { key: layoutKey, nodes, mode = "replace" } = payload;
      const slate = state.slates[layoutKey];
      if (mode === "replace") slate.nodes = nodes;
      else {
        const keys = nodes.map((node) => node.key);
        slate.nodes = [
          ...slate.nodes.filter((node) => !keys.includes(node.key)),
          ...nodes,
        ];
      }
      const anySelected =
        nodes.some((node) => node.selected) ||
        slate.edges.some((edge) => edge.selected);
      if (anySelected) {
        if (state.toolbar.activeTab !== "properties")
          clearOtherSelections(state, layoutKey);
        state.toolbar.activeTab = "properties";
      } else state.toolbar.activeTab = "symbols";
    },
    setNodePositions: (state, { payload }: PayloadAction<SetNodePositionsPayload>) => {
      const { key: layoutKey, positions } = payload;
      const slate = state.slates[layoutKey];
      Object.entries(positions).forEach(([key, position]) => {
        const node = slate.nodes.find((node) => node.key === key);
        if (node == null) return;
        node.position = position;
      });
    },
    setEdges: (state, { payload }: PayloadAction<SetEdgesPayload>) => {
      const { key: layoutKey, edges } = payload;
      const slate = state.slates[layoutKey];
      slate.edges = edges;
      const anySelected =
        edges.some((edge) => edge.selected) ||
        slate.nodes.some((node) => node.selected);
      if (anySelected) {
        if (state.toolbar.activeTab !== "properties")
          clearOtherSelections(state, layoutKey);
        state.toolbar.activeTab = "properties";
      } else state.toolbar.activeTab = "symbols";
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
      const slate = state.slates[layoutKey];
      slate.viewport = viewport;
    },
    setEditable: (state, { payload }: PayloadAction<SetEditablePayload>) => {
      const { key: layoutKey, editable } = payload;
      const slate = state.slates[layoutKey];
      clearSelections(slate);
      slate.editable = editable;
    },
    setFitViewOnResize: (
      state,
      { payload }: PayloadAction<SetFitViewOnResizePayload>,
    ) => {
      const { key: layoutKey, fitViewOnResize } = payload;
      const slate = state.slates[layoutKey];
      slate.fitViewOnResize = fitViewOnResize;
    },
    setViewportMode: (
      state,
      { payload: { mode } }: PayloadAction<SetViewportModePayload>,
    ) => {
      state.mode = mode;
    },
    setRemoteCreated: (state, { payload }: PayloadAction<SetRemoteCreatedPayload>) => {
      const { key: layoutKey } = payload;
      const slate = state.slates[layoutKey];
      slate.remoteCreated = true;
    },
    fixThemeContrast: (state, { payload }: PayloadAction<FixThemeContrastPayload>) => {
      const { theme } = payload;
      const bgColor = color.construct(theme.colors.gray.l0);
      const shouldChange = (crude: color.Crude): boolean => {
        const c = color.construct(crude);
        return color.grayness(c) > 0.85 && color.contrast(c, bgColor) < 1.3;
      };
      Object.values(state.slates).forEach((slate) => {
        const { nodes, edges, props } = slate;
        nodes.forEach((node) => {
          const nodeProps = props[node.key];
          if ("color" in nodeProps)
            if (shouldChange(nodeProps.color as string))
              nodeProps.color = theme.colors.gray.l11;
        });
        edges.forEach((edge) => {
          if (edge.color != null && shouldChange(edge.color as string))
            edge.color = theme.colors.gray.l11;
          else edge.color ??= theme.colors.gray.l11;
        });
      });
    },
    selectAll: (state, { payload }: PayloadAction<SelectAllPayload>) => {
      const { key: layoutKey } = payload;
      const slate = state.slates[layoutKey];
      slate.nodes.forEach((node) => (node.selected = true));
      slate.edges.forEach((edge) => (edge.selected = true));
    },
  },
});

const clearOtherSelections = (state: SliceState, layoutKey: string): void => {
  Object.keys(state.slates).forEach((key) => {
    // If any of the nodes or edges in other Diagram slices are selected, deselect them.
    if (key === layoutKey) return;
    clearSelections(state.slates[key]);
  });
};

const clearSelections = (state: State): void => {
  state.nodes.forEach((node) => {
    node.selected = false;
  });
  state.edges.forEach((edge) => {
    edge.selected = false;
  });
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
  create,
  setElementProps,
  setActiveToolbarTab,
  setViewport,
  setEditable,
  copySelection,
  pasteSelection,
  setViewportMode,
  setRemoteCreated,
  fixThemeContrast,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
