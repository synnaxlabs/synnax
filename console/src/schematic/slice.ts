// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type Control, type Diagram, type Viewport } from "@synnaxlabs/pluto";
import { Color } from "@synnaxlabs/pluto/color";
import { type Theming } from "@synnaxlabs/pluto/theming";
import { box, id, scale, xy } from "@synnaxlabs/x";

import * as latest from "@/schematic/migrations";

export type SliceState = latest.SliceState;
export type NodeProps = latest.NodeProps;
export type State = latest.State;
export type StateWithName = State & { name: string };
export type LegendState = latest.LegendState;
export type ToolbarTab = latest.ToolbarTab;
export type ToolbarState = latest.ToolbarState;
export const ZERO_STATE = latest.ZERO_STATE;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;
export const migrateState = latest.migrateState;
export const parser = latest.parser;

export const SLICE_NAME = "schematic";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

/** Purges fields in schematic state that should not be persisted. */
export const purgeState = (state: State): State => {
  // Reset control states.
  state.control = "released";
  state.controlAcquireTrigger = 0;
  return state;
};

export const purgeSliceState = (state: StoreState): StoreState => {
  Object.values(state[SLICE_NAME].schematics).forEach(purgeState);
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];

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

export interface SetLegendPayload {
  key: string;
  legend: Partial<LegendState>;
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
      // for each schematic, find the keys of the selected nodes and edges
      // and add them to the copy buffer. Then get the props of each
      // selected node and edge and add them to the copy buffer.
      const { schematics } = state;
      const copyBuffer: latest.CopyBuffer = {
        nodes: [],
        edges: [],
        props: {},
        pos: xy.ZERO,
      };
      Object.values(schematics).forEach((schematic) => {
        const { nodes, edges, props } = schematic;
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
      const schematic = state.schematics[layoutKey];
      const keys: Record<string, string> = {};
      const nextNodes = state.copy.nodes.map((node) => {
        const key: string = id.id();
        schematic.props[key] = state.copy.props[node.key];
        keys[node.key] = key;
        return {
          ...node,
          position: xy.translate(node.position, console),
          key,
          selected: true,
        };
      });
      const nextEdges = state.copy.edges.map((edge) => {
        const key: string = id.id();
        return {
          ...edge,
          key,
          source: keys[edge.source],
          target: keys[edge.target],
          selected: true,
        };
      });
      schematic.edges = [
        ...schematic.edges.map((edge) => ({ ...edge, selected: false })),
        ...nextEdges,
      ];
      schematic.nodes = [
        ...schematic.nodes.map((node) => ({ ...node, selected: false })),
        ...nextNodes,
      ];
    },
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const { key: layoutKey } = payload;
      const schematic: State = purgeState({
        ...ZERO_STATE,
        ...latest.migrateState(payload),
        key: layoutKey,
      }) as State;
      if (schematic.snapshot) {
        schematic.editable = false;
        clearSelections(schematic);
      }
      state.schematics[layoutKey] = schematic;
      state.toolbar.activeTab = "symbols";
    },
    clearSelection: (state, { payload }: PayloadAction<ClearSelectionPayload>) => {
      const { key: layoutKey } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.nodes.forEach((node) => {
        node.selected = false;
      });
      schematic.edges.forEach((edge) => {
        edge.selected = false;
      });
      state.toolbar.activeTab = "symbols";
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      const { keys: layoutKeys } = payload;
      layoutKeys.forEach((layoutKey) => {
        const schematic = state.schematics[layoutKey];
        if (schematic == null) return;
        if (schematic.control === "acquired") schematic.controlAcquireTrigger -= 1;
        delete state.schematics[layoutKey];
      });
    },
    addElement: (state, { payload }: PayloadAction<AddElementPayload>) => {
      const { key: layoutKey, elKey: key, props, node } = payload;
      const schematic = state.schematics[layoutKey];
      if (!schematic.editable) return;
      schematic.nodes.push({
        key,
        selected: false,
        position: xy.ZERO,
        ...node,
      });
      schematic.props[key] = props;
    },
    setElementProps: (state, { payload }: PayloadAction<SetElementPropsPayload>) => {
      const { layoutKey, key, props } = payload;
      const schematic = state.schematics[layoutKey];
      if (key in schematic.props)
        schematic.props[key] = { ...schematic.props[key], ...props };
      else {
        const edge = schematic.edges.findIndex((edge) => edge.key === key);
        if (edge !== -1) schematic.edges[edge] = { ...schematic.edges[edge], ...props };
      }
    },
    setNodes: (state, { payload }: PayloadAction<SetNodesPayload>) => {
      const { key: layoutKey, nodes, mode = "replace" } = payload;
      const schematic = state.schematics[layoutKey];
      if (mode === "replace") schematic.nodes = nodes;
      else {
        const keys = nodes.map((node) => node.key);
        schematic.nodes = [
          ...schematic.nodes.filter((node) => !keys.includes(node.key)),
          ...nodes,
        ];
      }
      const anySelected =
        nodes.some((node) => node.selected) ||
        schematic.edges.some((edge) => edge.selected);
      if (anySelected) {
        if (state.toolbar.activeTab !== "properties")
          clearOtherSelections(state, layoutKey);
        state.toolbar.activeTab = "properties";
      } else state.toolbar.activeTab = "symbols";
    },
    setNodePositions: (state, { payload }: PayloadAction<SetNodePositionsPayload>) => {
      const { key: layoutKey, positions } = payload;
      const schematic = state.schematics[layoutKey];
      Object.entries(positions).forEach(([key, position]) => {
        const node = schematic.nodes.find((node) => node.key === key);
        if (node == null) return;
        node.position = position;
      });
    },
    setEdges: (state, { payload }: PayloadAction<SetEdgesPayload>) => {
      const { key: layoutKey, edges } = payload;
      const schematic = state.schematics[layoutKey];
      // check for new edges
      const prevKeys = schematic.edges.map((edge) => edge.key);
      const newEdges = edges.filter((edge) => !prevKeys.includes(edge.key));
      newEdges.forEach((edge) => {
        const source = schematic.nodes.find((node) => node.key === edge.source);
        const target = schematic.nodes.find((node) => node.key === edge.target);
        if (source == null || target == null) return;
        const sourceProps = schematic.props[source.key];
        const targetProps = schematic.props[target.key];
        if (sourceProps.color === targetProps.color && sourceProps.color != null)
          edge.color = sourceProps.color;
      });
      schematic.edges = edges;
      const anySelected =
        edges.some((edge) => edge.selected) ||
        schematic.nodes.some((node) => node.selected);
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
      const schematic = state.schematics[layoutKey];
      schematic.viewport = viewport;
    },
    setEditable: (state, { payload }: PayloadAction<SetEditablePayload>) => {
      const { key: layoutKey, editable } = payload;
      const schematic = state.schematics[layoutKey];
      clearSelections(schematic);
      if (schematic.control === "acquired") schematic.controlAcquireTrigger -= 1;
      if (schematic.snapshot) return;
      schematic.editable = editable;
    },
    setFitViewOnResize: (
      state,
      { payload }: PayloadAction<SetFitViewOnResizePayload>,
    ) => {
      const { key: layoutKey, fitViewOnResize } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.fitViewOnResize = fitViewOnResize;
    },
    toggleControl: (state, { payload }: PayloadAction<ToggleControlPayload>) => {
      const { key: layoutKey } = payload;
      let { status } = payload;
      const schematic = state.schematics[layoutKey];
      status ??= schematic.control === "released" ? "acquired" : "released";
      if (status === "released") schematic.controlAcquireTrigger -= 1;
      else schematic.controlAcquireTrigger += 1;
    },
    setControlStatus: (state, { payload }: PayloadAction<SetControlStatusPayload>) => {
      const { key: layoutKey, control } = payload;
      const schematic = state.schematics[layoutKey];
      if (schematic == null) return;
      schematic.control = control;
      if (control === "acquired") {
        clearSelections(schematic);
        schematic.editable = false;
      }
    },
    setViewportMode: (
      state,
      { payload: { mode } }: PayloadAction<SetViewportModePayload>,
    ) => {
      state.mode = mode;
    },
    setRemoteCreated: (state, { payload }: PayloadAction<SetRemoteCreatedPayload>) => {
      const { key: layoutKey } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.remoteCreated = true;
    },
    fixThemeContrast: (state, { payload }: PayloadAction<FixThemeContrastPayload>) => {
      const { theme } = payload;
      const bgColor = new Color.Color(theme.colors.gray.l0);
      const shouldChange = (crude: Color.Crude): boolean => {
        const c = new Color.Color(crude);
        return c.grayness > 0.85 && c.contrast(bgColor) < 1.3;
      };
      Object.values(state.schematics).forEach((schematic) => {
        const { nodes, edges, props } = schematic;
        nodes.forEach((node) => {
          const nodeProps = props[node.key];
          if ("color" in nodeProps)
            if (shouldChange(nodeProps.color as string))
              nodeProps.color = theme.colors.gray.l9;
        });
        edges.forEach((edge) => {
          if (edge.color != null && shouldChange(edge.color as string))
            edge.color = theme.colors.gray.l9;
          else edge.color ??= theme.colors.gray.l9;
        });
      });
    },
    setLegend: (state, { payload }: PayloadAction<SetLegendPayload>) => {
      const { key: layoutKey, legend } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.legend = { ...schematic.legend, ...legend };
    },
  },
});

const clearOtherSelections = (state: SliceState, layoutKey: string): void => {
  Object.keys(state.schematics).forEach((key) => {
    // If any of the nodes or edges in other Diagram slices are selected, deselect them.
    if (key === layoutKey) return;
    clearSelections(state.schematics[key]);
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
  setLegend,
  setNodePositions,
  toggleControl,
  setControlStatus,
  addElement,
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
  fixThemeContrast,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
