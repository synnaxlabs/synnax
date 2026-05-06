// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type schematic } from "@synnaxlabs/client";
import {
  type Control,
  type Diagram,
  type Theming,
  type Viewport,
} from "@synnaxlabs/pluto";
import { color, deep, id, type require, xy } from "@synnaxlabs/x";

import * as latest from "@/schematic/types";
import { ZERO_COPY_BUFFER } from "@/schematic/types/v6";
import { type RootState } from "@/store";

export type SliceState = latest.SliceState;
export type NodeConfig = latest.NodeConfig;
export type EdgeConfig = latest.EdgeConfig;
export type ElementConfig = latest.ElementConfig;
export type State = latest.State;
export type LegendState = latest.LegendState;
export type ToolbarTab = latest.ToolbarTab;
export type ToolbarState = latest.ToolbarState;
export const ZERO_STATE = latest.ZERO_STATE;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;
export const migrateState = latest.migrateState;
export const anyStateZ = latest.anyStateZ;

export const SLICE_NAME = "schematic";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

/** Purges fields in schematic state that should not be persisted. */
export const purgeState = (state: State): State => {
  state.control = "released";
  state.toolbar = { ...state.toolbar, activeTab: "symbols" };
  state.selected = [];
  return state;
};

export const purgeSliceState = (state: RootState): RootState => {
  Object.values(state[SLICE_NAME].schematics).forEach(purgeState);
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];

export const fromRemote = (s: schematic.Schematic): State => ({
  ...ZERO_STATE,
  ...s,
  // TODO: remove this assertion when schematic element configs are strongly typed on
  // the core.
  configs: s.configs as Record<string, ElementConfig>,
  remoteCreated: true,
});

export interface SetViewportPayload {
  key: string;
  viewport: Diagram.Viewport;
}

export interface AddNodePayload {
  key: string;
  config: NodeConfig;
  node: require.Require<Partial<Diagram.Node>, "key">;
}

export interface SetElementConfigPayload {
  key: string;
  elKey: string;
  config: Partial<ElementConfig>;
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
  positions: Array<[string, xy.XY]>;
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

export interface SetActiveToolbarTabPayload {
  key: string;
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
  key: string;
  mode: Viewport.Mode;
}

export interface SetRemoteCreatedPayload {
  key: string;
}

export interface SetLegendPayload {
  key: string;
  legend: Partial<LegendState>;
}

export interface SetLegendVisiblePayload {
  key: string;
  visible: boolean;
}

export interface SelectAllPayload {
  key: string;
}

export interface SetSelectedPayload {
  key: string;
  selected: string[];
}

export interface SetAuthorityPayload {
  key: string;
  authority: number;
}

export interface SetSelectedSymbolGroupPayload {
  key: string;
  group: string;
}

export interface ApplyNodeChangesPayload {
  key: string;
  changes: Diagram.NodeChange[];
}

export interface ApplyEdgeChangesPayload {
  key: string;
  changes: Diagram.EdgeChange[];
}

const setActiveTabFromSelection = (
  state: SliceState,
  layoutKey: string,
  hasSelection: boolean,
): void => {
  const schematic = state.schematics[layoutKey];
  if (schematic == null) return;
  if (hasSelection) {
    if (schematic.toolbar.activeTab !== "properties")
      clearOtherSelections(state, layoutKey);
    schematic.toolbar.activeTab = "properties";
  } else schematic.toolbar.activeTab = "symbols";
};

const syncEdgeColorFromEndpoints = (schematic: State, edge: Diagram.Edge): void => {
  const source = schematic.configs[edge.source.node];
  if (source.color == null) return;
  schematic.configs[edge.key].color = color.construct(source.color);
};

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: latest.ZERO_SLICE_STATE,
  reducers: {
    copySelection: (state, _: PayloadAction<CopySelectionPayload>) => {
      const { schematics } = state;
      const copyBuffer: latest.CopyBuffer = deep.copy(ZERO_COPY_BUFFER);
      Object.values(schematics).forEach((schematic) => {
        const { nodes, edges, configs, selected } = schematic;
        const selectedSet = new Set(selected);
        const selectedNodes = nodes.filter((node) => selectedSet.has(node.key));
        const selectedEdges = edges.filter((edge) => selectedSet.has(edge.key));
        copyBuffer.nodes = [...copyBuffer.nodes, ...selectedNodes];
        copyBuffer.edges = [...copyBuffer.edges, ...selectedEdges];
        selectedNodes.forEach(
          (node) => (copyBuffer.configs[node.key] = configs[node.key]),
        );
        selectedEdges.forEach(
          (edge) => (copyBuffer.configs[edge.key] = configs[edge.key]),
        );
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
      const { pos, key } = payload;
      const offset = xy.translation(state.copy.pos, pos);
      const schematic = state.schematics[key];
      const keys: Record<string, string> = {};
      const nextNodes = state.copy.nodes.map((node) => {
        const key: string = id.create();
        if (state.copy.configs[node.key] != null)
          schematic.configs[key] = state.copy.configs[node.key];
        keys[node.key] = key;
        return {
          ...node,
          position: xy.translate(node.position, offset),
          key,
        };
      });
      const nextEdges = state.copy.edges.map((edge) => {
        const key: string = id.create();
        if (state.copy.configs[edge.key] != null)
          schematic.configs[key] = state.copy.configs[edge.key];
        return {
          key,
          source: {
            node: keys[edge.source.node] ?? edge.source.node,
            param: edge.source.param,
          },
          target: {
            node: keys[edge.target.node] ?? edge.target.node,
            param: edge.target.param,
          },
        };
      });
      schematic.edges = [...schematic.edges, ...nextEdges];
      schematic.nodes = [...schematic.nodes, ...nextNodes];
      schematic.selected = [
        ...nextNodes.map((n) => n.key),
        ...nextEdges.map((e) => e.key),
      ];
      setActiveTabFromSelection(state, key, schematic.selected.length > 0);
    },
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const { key } = payload;
      const schematic: State = purgeState({
        ...ZERO_STATE,
        ...latest.migrateState(payload),
      });
      if (schematic.snapshot) schematic.editable = false;
      state.schematics[key] = schematic;
    },
    clearSelection: (state, { payload }: PayloadAction<ClearSelectionPayload>) => {
      const { key } = payload;
      const schematic = state.schematics[key];
      schematic.selected = [];
      schematic.toolbar.activeTab = "symbols";
    },
    setSelected: (state, { payload }: PayloadAction<SetSelectedPayload>) => {
      const { key, selected } = payload;
      const schematic = state.schematics[key];
      schematic.selected = selected;
      setActiveTabFromSelection(state, key, selected.length > 0);
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      const { keys } = payload;
      keys.forEach((key) => delete state.schematics[key]);
    },
    addNode: (state, { payload }: PayloadAction<AddNodePayload>) => {
      const { key, config, node } = payload;
      const schematic = state.schematics[key];
      if (!schematic.editable) return;
      schematic.nodes.push({ ...node, position: node.position ?? { ...xy.ZERO } });
      schematic.configs[node.key] = config;
    },
    setElementConfig: (state, { payload }: PayloadAction<SetElementConfigPayload>) => {
      const { key, elKey, config } = payload;
      const schem = state.schematics[key];
      schem.configs[elKey] = { ...schem.configs[elKey], ...config } as ElementConfig;
    },
    setNodes: (state, { payload }: PayloadAction<SetNodesPayload>) => {
      const { key, nodes, mode = "replace" } = payload;
      const schematic = state.schematics[key];
      if (mode === "replace") schematic.nodes = nodes;
      else {
        const keys = nodes.map((node) => node.key);
        schematic.nodes = [
          ...schematic.nodes.filter((node) => !keys.includes(node.key)),
          ...nodes,
        ];
      }
    },
    setNodePositions: (state, { payload }: PayloadAction<SetNodePositionsPayload>) => {
      const { key, positions } = payload;
      const schematic = state.schematics[key];
      positions.forEach(([key, position]) => {
        const node = schematic.nodes.find((node) => node.key === key);
        if (node == null) return;
        node.position = position;
      });
    },
    setEdges: (state, { payload }: PayloadAction<SetEdgesPayload>) => {
      const { key, edges } = payload;
      const schematic = state.schematics[key];
      const prevKeys = new Set(schematic.edges.map((edge) => edge.key));
      edges
        .filter((edge) => !prevKeys.has(edge.key))
        .forEach((edge) => syncEdgeColorFromEndpoints(schematic, edge));
      schematic.edges = edges;
    },
    applyNodeChanges: (state, { payload }: PayloadAction<ApplyNodeChangesPayload>) => {
      const { key, changes } = payload;
      const schematic = state.schematics[key];
      for (const change of changes)
        switch (change.type) {
          case "position": {
            const node = schematic.nodes.find((n) => n.key === change.key);
            if (node != null) node.position = change.position;
            break;
          }
          case "dimensions": {
            const node = schematic.nodes.find((n) => n.key === change.key);
            if (node != null) node.measured = change.dimensions;
            break;
          }
          case "remove": {
            schematic.nodes = schematic.nodes.filter((n) => n.key !== change.key);
            schematic.edges = schematic.edges.filter(
              (e) => e.source.node !== change.key && e.target.node !== change.key,
            );
            delete schematic.configs[change.key];
            schematic.selected = schematic.selected.filter((k) => k !== change.key);
            break;
          }
        }
    },
    applyEdgeChanges: (state, { payload }: PayloadAction<ApplyEdgeChangesPayload>) => {
      const { key, changes } = payload;
      const schematic = state.schematics[key];
      for (const change of changes)
        switch (change.type) {
          case "add":
            schematic.edges.push(change.edge);
            schematic.configs[change.edge.key] = {
              variant: "pipe",
              color: [...color.ZERO],
              segments: [],
            };
            syncEdgeColorFromEndpoints(schematic, change.edge);
            break;
          case "remove":
            schematic.edges = schematic.edges.filter((e) => e.key !== change.key);
            delete schematic.configs[change.key];
            schematic.selected = schematic.selected.filter((k) => k !== change.key);
            break;
        }
    },
    setActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetActiveToolbarTabPayload>,
    ) => {
      const { key, tab } = payload;
      state.schematics[key].toolbar.activeTab = tab;
    },
    setViewport: (state, { payload }: PayloadAction<SetViewportPayload>) => {
      const { key: layoutKey, viewport } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.viewport = viewport;
    },
    setEditable: (state, { payload }: PayloadAction<SetEditablePayload>) => {
      const { key: layoutKey, editable } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.selected = [];
      if (schematic.snapshot) return;
      schematic.editable = editable;
    },
    setFitViewOnResize: (
      state,
      { payload }: PayloadAction<SetFitViewOnResizePayload>,
    ) => {
      const { key, fitViewOnResize } = payload;
      const schematic = state.schematics[key];
      schematic.fitViewOnResize = fitViewOnResize;
    },
    setControlStatus: (state, { payload }: PayloadAction<SetControlStatusPayload>) => {
      const { key, control } = payload;
      const schematic = state.schematics[key];
      if (schematic == null) return;
      schematic.control = control;
      if (control === "acquired") {
        schematic.selected = [];
        schematic.editable = false;
      }
    },
    setViewportMode: (
      state,
      { payload: { key, mode } }: PayloadAction<SetViewportModePayload>,
    ) => {
      state.schematics[key].mode = mode;
    },
    setRemoteCreated: (state, { payload }: PayloadAction<SetRemoteCreatedPayload>) => {
      const { key } = payload;
      const schematic = state.schematics[key];
      schematic.remoteCreated = true;
    },
    fixThemeContrast: (state, { payload }: PayloadAction<FixThemeContrastPayload>) => {
      const { theme } = payload;
      const bgColor = color.construct(theme.colors.gray.l0);
      const shouldChange = (crude: color.Crude): boolean => {
        const c = color.construct(crude);
        return color.grayness(c) > 0.85 && color.contrast(c, bgColor) < 1.3;
      };
      Object.values(state.schematics).forEach((schematic) => {
        Object.values(schematic.configs).forEach((p) => {
          if ("color" in p && p.color != null && shouldChange(p.color))
            p.color = color.construct(theme.colors.gray.l11);
        });
      });
    },
    setLegend: (state, { payload }: PayloadAction<SetLegendPayload>) => {
      const { key, legend } = payload;
      const schematic = state.schematics[key];
      schematic.legend = { ...schematic.legend, ...legend };
    },
    setLegendVisible: (state, { payload }: PayloadAction<SetLegendVisiblePayload>) => {
      const { key, visible } = payload;
      const schematic = state.schematics[key];
      schematic.legend.visible = visible;
    },
    selectAll: (state, { payload }: PayloadAction<SelectAllPayload>) => {
      const { key } = payload;
      const schematic = state.schematics[key];
      schematic.selected = [
        ...schematic.nodes.map((n) => n.key),
        ...schematic.edges.map((e) => e.key),
      ];
      setActiveTabFromSelection(state, key, schematic.selected.length > 0);
    },
    setAuthority: (state, { payload }: PayloadAction<SetAuthorityPayload>) => {
      const { key, authority } = payload;
      const schematic = state.schematics[key];
      schematic.authority = authority;
    },
    setSelectedSymbolGroup: (
      state,
      { payload }: PayloadAction<SetSelectedSymbolGroupPayload>,
    ) => {
      const { key, group } = payload;
      state.schematics[key].toolbar.selectedSymbolGroup = group;
    },
  },
});

const clearOtherSelections = (state: SliceState, target: string): void => {
  Object.keys(state.schematics).forEach((key) => {
    if (key !== target) state.schematics[key].selected = [];
  });
};

export const {
  setLegend,
  setLegendVisible,
  setNodePositions,
  setControlStatus,
  addNode,
  selectAll,
  setEdges,
  setNodes,
  remove,
  clearSelection,
  setSelected,
  setSelectedSymbolGroup,
  setFitViewOnResize,
  create: internalCreate,
  setElementConfig,
  setActiveToolbarTab,
  setViewport,
  setEditable,
  copySelection,
  pasteSelection,
  setViewportMode,
  setRemoteCreated,
  fixThemeContrast,
  setAuthority,
  applyNodeChanges,
  applyEdgeChanges,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
