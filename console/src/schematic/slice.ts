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

import {
  calculateGroupBoundingBox,
  remapGroupIds,
  selectedGroupKeys,
} from "@/schematic/groups";
import * as latest from "@/schematic/types";
import { type RootState } from "@/store";

export type SliceState = latest.SliceState;
export type NodeProps = latest.NodeProps;
export type EdgeProps = latest.EdgeProps;
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
  // Reset control states.
  state.control = "released";
  state.toolbar = { ...state.toolbar, activeTab: "symbols" };
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
  props: Partial<NodeProps> | Partial<EdgeProps>;
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

export interface CutSelectionPayload {
  key: string;
}

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

export interface SetAuthorityPayload {
  key: string;
  authority: number;
}

export interface SetSelectedSymbolGroupPayload {
  key: string;
  group: string;
}

const copySelectedToBuffer = (state: SliceState): void => {
  const copyBuffer: latest.CopyBuffer = {
    nodes: [],
    edges: [],
    props: {},
    pos: xy.ZERO,
  };
  Object.values(state.schematics).forEach((schematic) => {
    const { nodes, edges, props } = schematic;
    const selectedNodes = nodes.filter((node) => node.selected);
    const groupKeys = selectedGroupKeys(selectedNodes, props);
    const groupMembers = nodes.filter((node) => {
      if (node.selected) return false;
      const gid = props[node.key]?.groupId;
      return gid != null && groupKeys.has(gid);
    });
    const allNodes = [...selectedNodes, ...groupMembers];
    const selectedEdges = edges.filter((edge) => edge.selected);
    copyBuffer.nodes = [...copyBuffer.nodes, ...allNodes];
    copyBuffer.edges = [...copyBuffer.edges, ...selectedEdges];
    allNodes.forEach((node) => {
      copyBuffer.props[node.key] = props[node.key];
    });
    selectedEdges.forEach((edge) => {
      copyBuffer.props[edge.key] = props[edge.key];
    });
  });
  const { nodes } = copyBuffer;
  if (nodes.length > 0) {
    const pos = nodes.reduce((acc, node) => xy.translate(acc, node.position), xy.ZERO);
    copyBuffer.pos = xy.scale(pos, 1 / nodes.length);
  }
  state.copy = copyBuffer;
};

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: latest.ZERO_SLICE_STATE,
  reducers: {
    copySelection: (state, _: PayloadAction<CopySelectionPayload>) => {
      copySelectedToBuffer(state);
    },
    // Copies the selected nodes and edges into the copy buffer, then removes
    // them from the schematic.
    cutSelection: (state, { payload }: PayloadAction<CutSelectionPayload>) => {
      copySelectedToBuffer(state);
      const { key: layoutKey } = payload;
      const schematic = state.schematics[layoutKey];
      const { nodes, edges } = state.copy;
      const nodeKeys = new Set(nodes.map((n) => n.key));
      const edgeKeys = new Set(edges.map((e) => e.key));
      schematic.nodes = schematic.nodes.filter((n) => !nodeKeys.has(n.key));
      schematic.edges = schematic.edges.filter((e) => !edgeKeys.has(e.key));
      nodeKeys.forEach((key) => delete schematic.props[key]);
      edgeKeys.forEach((key) => delete schematic.props[key]);
    },
    pasteSelection: (state, { payload }: PayloadAction<PasteSelectionPayload>) => {
      const { pos, key: layoutKey } = payload;
      const console = xy.translation(state.copy.pos, pos);
      const schematic = state.schematics[layoutKey];
      const keys: Record<string, string> = {};
      const nextNodes = state.copy.nodes.map((node) => {
        const key: string = id.create();
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
        const key: string = id.create();
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
      remapGroupIds(schematic.props, keys);
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
      });
      if (schematic.snapshot) {
        schematic.editable = false;
        clearSelections(schematic);
      }
      state.schematics[layoutKey] = schematic;
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
      state.schematics[layoutKey].toolbar.activeTab = "symbols";
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      const { keys: layoutKeys } = payload;
      layoutKeys.forEach((layoutKey) => {
        const schematic = state.schematics[layoutKey];
        if (schematic == null) return;
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
        schematic.props[key] = {
          ...schematic.props[key],
          ...props,
        } as NodeProps;
      else {
        const edge = schematic.edges.findIndex((edge) => edge.key === key);
        if (edge !== -1)
          schematic.edges[edge].data = { ...schematic.edges[edge].data, ...props };
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
        if (state.schematics[layoutKey].toolbar.activeTab !== "properties")
          clearOtherSelections(state, layoutKey);
        state.schematics[layoutKey].toolbar.activeTab = "properties";
      } else state.schematics[layoutKey].toolbar.activeTab = "symbols";
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
      const anySelected =
        edges.some((edge) => edge.selected) ||
        schematic.nodes.some((node) => node.selected);
      if (anySelected) {
        if (state.schematics[layoutKey].toolbar.activeTab !== "properties")
          clearOtherSelections(state, layoutKey);
        state.schematics[layoutKey].toolbar.activeTab = "properties";
      } else state.schematics[layoutKey].toolbar.activeTab = "symbols";
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
      clearSelections(schematic);
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
        clearSelections(schematic);
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
        const { nodes, edges, props } = schematic;
        nodes.forEach((node) => {
          const nodeProps = props[node.key];
          if ("color" in nodeProps)
            if (shouldChange(nodeProps.color as string))
              nodeProps.color = theme.colors.gray.l11;
        });
        edges.forEach((edge) => {
          if (edge.data?.color != null && shouldChange(edge.data.color as string))
            edge.data.color = theme.colors.gray.l11;
          else if (edge.data != null) edge.data.color ??= theme.colors.gray.l11;
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
      schematic.nodes.forEach((node) => (node.selected = true));
      schematic.edges.forEach((edge) => (edge.selected = true));
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
    groupSelection: (
      state,
      { payload }: PayloadAction<{ key: string; props: NodeProps }>,
    ) => {
      const { key: layoutKey, props: groupProps } = payload;
      const schematic = state.schematics[layoutKey];
      const groupKeys = selectedGroupKeys(schematic.nodes, schematic.props);
      const memberNodes = schematic.nodes.filter((n) => {
        if (n.selected) return schematic.props[n.key]?.key !== "group";
        const gid = schematic.props[n.key]?.groupId;
        return gid != null && groupKeys.has(gid);
      });
      if (memberNodes.length < 2) return;
      const { position, dimensions } = calculateGroupBoundingBox(memberNodes);
      const groupKey = id.create();
      for (const key of groupKeys) delete schematic.props[key];
      schematic.nodes = schematic.nodes.filter((n) => !groupKeys.has(n.key));
      schematic.nodes.push({
        key: groupKey,
        position,
        selected: true,
        zIndex: 0,
      });
      schematic.props[groupKey] = { ...groupProps, dimensions };
      for (const node of memberNodes)
        schematic.props[node.key] = {
          ...schematic.props[node.key],
          groupId: groupKey,
        };
    },
    ungroupSelection: (state, { payload }: PayloadAction<{ key: string }>) => {
      const { key: layoutKey } = payload;
      const schematic = state.schematics[layoutKey];
      const groupKeysToRemove = new Set<string>();
      for (const n of schematic.nodes) {
        if (!n.selected) continue;
        const p = schematic.props[n.key];
        if (p?.key === "group") groupKeysToRemove.add(n.key);
        if (p?.groupId != null) groupKeysToRemove.add(p.groupId);
      }
      if (groupKeysToRemove.size === 0) return;
      for (const key of Object.keys(schematic.props)) {
        const p = schematic.props[key];
        if (p?.groupId != null && groupKeysToRemove.has(p.groupId)) delete p.groupId;
        if (p?.key === "group" && groupKeysToRemove.has(key))
          delete schematic.props[key];
      }
      schematic.nodes = schematic.nodes.filter((n) => !groupKeysToRemove.has(n.key));
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
  setLegendVisible,
  setNodePositions,
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
  cutSelection,
  pasteSelection,
  setViewportMode,
  setRemoteCreated,
  fixThemeContrast,
  setAuthority,
  groupSelection,
  ungroupSelection,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
