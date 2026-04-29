// Copyright 2026 Synnax Labs, Inc.
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
import { color, id, xy } from "@synnaxlabs/x";

import * as latest from "@/schematic/types";
import { type RootState } from "@/store";

export type SliceState = latest.SliceState;
export type NodeProps = latest.NodeProps;
export type EdgeProps = latest.EdgeProps;
export type Props = latest.Props;
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
  props: Partial<Props>;
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

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: latest.ZERO_SLICE_STATE,
  reducers: {
    copySelection: (state, _: PayloadAction<CopySelectionPayload>) => {
      const { schematics } = state;
      const copyBuffer: latest.CopyBuffer = {
        nodes: [],
        edges: [],
        props: {},
        pos: xy.ZERO,
      };
      Object.values(schematics).forEach((schematic) => {
        const { nodes, edges, props, selected } = schematic;
        const selectedSet = new Set(selected);
        const selectedNodes = nodes.filter((node) => selectedSet.has(node.key));
        const selectedEdges = edges.filter((edge) => selectedSet.has(edge.key));
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
      const offset = xy.translation(state.copy.pos, pos);
      const schematic = state.schematics[layoutKey];
      const keys: Record<string, string> = {};
      const nextNodes = state.copy.nodes.map((node) => {
        const key: string = id.create();
        if (state.copy.props[node.key] != null)
          schematic.props[key] = state.copy.props[node.key];
        keys[node.key] = key;
        return {
          ...node,
          position: xy.translate(node.position, offset),
          key,
        };
      });
      const nextEdges = state.copy.edges.map((edge) => {
        const key: string = id.create();
        if (state.copy.props[edge.key] != null)
          schematic.props[key] = state.copy.props[edge.key];
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
      setActiveTabFromSelection(state, layoutKey, schematic.selected.length > 0);
    },
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const { key: layoutKey } = payload;
      const schematic: State = purgeState({
        ...ZERO_STATE,
        ...latest.migrateState(payload),
        key: layoutKey,
      });
      if (schematic.snapshot) schematic.editable = false;
      state.schematics[layoutKey] = schematic;
    },
    clearSelection: (state, { payload }: PayloadAction<ClearSelectionPayload>) => {
      const { key: layoutKey } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.selected = [];
      schematic.toolbar.activeTab = "symbols";
    },
    setSelected: (state, { payload }: PayloadAction<SetSelectedPayload>) => {
      const { key: layoutKey, selected } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.selected = selected;
      setActiveTabFromSelection(state, layoutKey, selected.length > 0);
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      const { keys: layoutKeys } = payload;
      layoutKeys.forEach((layoutKey) => {
        if (state.schematics[layoutKey] == null) return;
        delete state.schematics[layoutKey];
      });
    },
    addElement: (state, { payload }: PayloadAction<AddElementPayload>) => {
      const { key: layoutKey, elKey: key, props, node } = payload;
      const schematic = state.schematics[layoutKey];
      if (!schematic.editable) return;
      schematic.nodes.push({
        key,
        position: xy.ZERO,
        ...node,
      });
      schematic.props[key] = props;
    },
    setElementProps: (state, { payload }: PayloadAction<SetElementPropsPayload>) => {
      const { layoutKey, key, props } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.props[key] = {
        ...(schematic.props[key] ?? {}),
        ...props,
      } as Props;
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
    },
    setNodePositions: (state, { payload }: PayloadAction<SetNodePositionsPayload>) => {
      const { key: layoutKey, positions } = payload;
      const schematic = state.schematics[layoutKey];
      positions.forEach(([key, position]) => {
        const node = schematic.nodes.find((node) => node.key === key);
        if (node == null) return;
        node.position = position;
      });
    },
    setEdges: (state, { payload }: PayloadAction<SetEdgesPayload>) => {
      const { key: layoutKey, edges } = payload;
      const schematic = state.schematics[layoutKey];
      // For new edges, sync color from source/target node when both colors match.
      const prevKeys = new Set(schematic.edges.map((edge) => edge.key));
      const newEdges = edges.filter((edge) => !prevKeys.has(edge.key));
      newEdges.forEach((edge) => {
        const sourceProps = schematic.props[edge.source.node] as NodeProps | undefined;
        const targetProps = schematic.props[edge.target.node] as NodeProps | undefined;
        if (sourceProps?.color != null && sourceProps.color === targetProps?.color) {
          const existing = (schematic.props[edge.key] ?? {}) as EdgeProps;
          schematic.props[edge.key] = {
            ...existing,
            color: sourceProps.color,
          };
        }
      });
      schematic.edges = edges;
    },
    applyNodeChanges: (state, { payload }: PayloadAction<ApplyNodeChangesPayload>) => {
      const { key: layoutKey, changes } = payload;
      const schematic = state.schematics[layoutKey];
      for (const change of changes)
        switch (change.type) {
          case "position": {
            const node = schematic.nodes.find((n) => n.key === change.key);
            if (node != null) node.position = change.position;
            break;
          }
          case "remove": {
            schematic.nodes = schematic.nodes.filter((n) => n.key !== change.key);
            schematic.edges = schematic.edges.filter(
              (e) => e.source.node !== change.key && e.target.node !== change.key,
            );
            delete schematic.props[change.key];
            schematic.selected = schematic.selected.filter((k) => k !== change.key);
            break;
          }
          case "dimensions": {
            const node = schematic.nodes.find((n) => n.key === change.key);
            if (node != null) node.measured = change.dimensions;
            break;
          }
        }
    },
    applyEdgeChanges: (state, { payload }: PayloadAction<ApplyEdgeChangesPayload>) => {
      const { key: layoutKey, changes } = payload;
      const schematic = state.schematics[layoutKey];
      for (const change of changes)
        switch (change.type) {
          case "add":
            schematic.edges.push(change.edge);
            break;
          case "remove":
            schematic.edges = schematic.edges.filter((e) => e.key !== change.key);
            delete schematic.props[change.key];
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
      const { key: layoutKey, fitViewOnResize } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.fitViewOnResize = fitViewOnResize;
    },
    setControlStatus: (state, { payload }: PayloadAction<SetControlStatusPayload>) => {
      const { key: layoutKey, control } = payload;
      const schematic = state.schematics[layoutKey];
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
      const { key: layoutKey } = payload;
      const schematic = state.schematics[layoutKey];
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
        Object.values(schematic.props).forEach((p) => {
          if ("color" in p && p.color != null && shouldChange(p.color))
            p.color = color.construct(theme.colors.gray.l11);
        });
      });
    },
    setLegend: (state, { payload }: PayloadAction<SetLegendPayload>) => {
      const { key: layoutKey, legend } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.legend = { ...schematic.legend, ...legend };
    },
    setLegendVisible: (state, { payload }: PayloadAction<SetLegendVisiblePayload>) => {
      const { key: layoutKey, visible } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.legend.visible = visible;
    },
    selectAll: (state, { payload }: PayloadAction<SelectAllPayload>) => {
      const { key: layoutKey } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.selected = [
        ...schematic.nodes.map((n) => n.key),
        ...schematic.edges.map((e) => e.key),
      ];
      setActiveTabFromSelection(state, layoutKey, schematic.selected.length > 0);
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

const clearOtherSelections = (state: SliceState, layoutKey: string): void => {
  Object.keys(state.schematics).forEach((key) => {
    if (key === layoutKey) return;
    state.schematics[key].selected = [];
  });
};

export const {
  setLegend,
  setLegendVisible,
  setNodePositions,
  setControlStatus,
  addElement,
  selectAll,
  setEdges,
  setNodes,
  remove,
  clearSelection,
  setSelected,
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
  applyNodeChanges,
  applyEdgeChanges,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
