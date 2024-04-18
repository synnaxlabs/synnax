// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction, createSlice } from "@reduxjs/toolkit";
import { type Control, type Viewport, type Diagram, type PID } from "@synnaxlabs/pluto";
import { Color } from "@synnaxlabs/pluto/color";
import { type Theming } from "@synnaxlabs/pluto/theming";
import { box, scale, xy, deep, migrate } from "@synnaxlabs/x";
import { nanoid } from "nanoid/non-secure";
import { v4 as uuidV4 } from "uuid";

import { type Layout } from "@/layout";

export type NodeProps = object & {
  key: PID.Variant;
  color?: Color.Crude;
};

export interface State extends migrate.Migratable {
  editable: boolean;
  snapshot: boolean;
  remoteCreated: boolean;
  viewport: Diagram.Viewport;
  nodes: Diagram.Node[];
  edges: Diagram.Edge[];
  props: Record<string, NodeProps>;
  control: Control.Status;
  controlAcquireTrigger: number;
}

interface CopyBuffer {
  pos: xy.Crude;
  nodes: Diagram.Node[];
  edges: Diagram.Edge[];
  props: Record<string, NodeProps>;
}

const ZERO_COPY_BUFFER: CopyBuffer = {
  pos: xy.ZERO,
  nodes: [],
  edges: [],
  props: {},
};

// ||||| TOOLBAR |||||

const TOOLBAR_TABS = ["symbols", "properties"] as const;
export type ToolbarTab = (typeof TOOLBAR_TABS)[number];

export interface ToolbarState {
  activeTab: ToolbarTab;
}

export interface SliceState extends migrate.Migratable {
  mode: Viewport.Mode;
  copy: CopyBuffer;
  toolbar: ToolbarState;
  pids: Record<string, State>;
}

export const SLICE_NAME = "pid";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_STATE: State = {
  version: "0.0.0",
  snapshot: false,
  nodes: [],
  edges: [],
  props: {},
  remoteCreated: false,
  viewport: { position: xy.ZERO, zoom: 1 },
  editable: true,
  control: "released",
  controlAcquireTrigger: 0,
};

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
  mode: "select",
  copy: { ...ZERO_COPY_BUFFER },
  toolbar: { activeTab: "symbols" },
  pids: {},
};

export interface SetViewportPayload {
  layoutKey: string;
  viewport: Diagram.Viewport;
}

export interface AddElementPayload {
  layoutKey: string;
  key: string;
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
  layoutKey: string;
  mode?: "replace" | "update";
  nodes: Diagram.Node[];
}

export interface SetNodePositionsPayload {
  layoutKey: string;
  positions: Record<string, xy.XY>;
}

export interface SetEdgesPayload {
  layoutKey: string;
  edges: Diagram.Edge[];
}

export interface CreatePayload extends State {
  key: string;
}

export interface RemovePayload {
  layoutKeys: string[];
}

export interface SetEditablePayload {
  layoutKey: string;
  editable: boolean;
}

export interface SetControlStatusPayload {
  layoutKey: string;
  control: Control.Status;
}

export interface ToggleControlPayload {
  layoutKey: string;
  status: Control.Status;
}

export interface SetActiveToolbarTabPayload {
  tab: ToolbarTab;
}

export interface CopySelectionPayload {}

export interface PasteSelectionPayload {
  layoutKey: string;
  pos: xy.XY;
}

export interface ClearSelectionPayload {
  layoutKey: string;
}

export interface SetViewportModePayload {
  mode: Viewport.Mode;
}

export interface SetRemoteCreatedPayload {
  layoutKey: string;
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

const MIGRATIONS: migrate.Migrations = {};

export const migrateSlice = migrate.migrator<SliceState, SliceState>(MIGRATIONS);

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    copySelection: (state, _: PayloadAction<CopySelectionPayload>) => {
      // for each pid, find the keys of the selected nodes and edges
      // and add them to the copy buffer. Then get the props of each
      // selected node and edge and add them to the copy buffer.
      const { pids } = state;
      const copyBuffer: CopyBuffer = {
        nodes: [],
        edges: [],
        props: {},
        pos: xy.ZERO,
      };
      Object.values(pids).forEach((pid) => {
        const { nodes, edges, props } = pid;
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
      const { pos, layoutKey } = payload;
      const console = xy.translation(state.copy.pos, pos);
      const pid = state.pids[layoutKey];
      const keys: Record<string, string> = {};
      const nextNodes = state.copy.nodes.map((node) => {
        const key: string = nanoid();
        pid.props[key] = state.copy.props[node.key];
        keys[node.key] = key;
        return {
          ...node,
          position: xy.translate(node.position, console),
          key,
          selected: true,
        };
      });
      const nextEdges = state.copy.edges.map((edge) => {
        const key: string = nanoid();
        return {
          ...edge,
          key,
          source: keys[edge.source],
          target: keys[edge.target],
          selected: true,
        };
      });
      pid.edges = [
        ...pid.edges.map((edge) => ({ ...edge, selected: false })),
        ...nextEdges,
      ];
      pid.nodes = [
        ...pid.nodes.map((node) => ({ ...node, selected: false })),
        ...nextNodes,
      ];
    },
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const { key: layoutKey } = payload;
      const pid = { ...ZERO_STATE, ...payload };
      if (pid.snapshot) {
        pid.editable = false;
        clearSelections(pid);
      }
      state.pids[layoutKey] = pid;
      state.toolbar.activeTab = "symbols";
    },
    clearSelection: (state, { payload }: PayloadAction<ClearSelectionPayload>) => {
      const { layoutKey } = payload;
      const pid = state.pids[layoutKey];
      pid.nodes.forEach((node) => {
        node.selected = false;
      });
      pid.edges.forEach((edge) => {
        edge.selected = false;
      });
      state.toolbar.activeTab = "symbols";
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      const { layoutKeys } = payload;
      layoutKeys.forEach((layoutKey) => {
        const pid = state.pids[layoutKey];
        if (pid.control === "acquired") pid.controlAcquireTrigger -= 1;
        // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
        delete state.pids[layoutKey];
      });
    },
    addElement: (state, { payload }: PayloadAction<AddElementPayload>) => {
      const { layoutKey, key, props, node } = payload;
      const pid = state.pids[layoutKey];
      if (!pid.editable) return;
      pid.nodes.push({
        key,
        selected: false,
        position: xy.ZERO,
        ...node,
      });
      pid.props[key] = props;
    },
    setElementProps: (state, { payload }: PayloadAction<SetElementPropsPayload>) => {
      const { layoutKey, key, props } = payload;
      const pid = state.pids[layoutKey];
      if (!pid.editable) return;
      if (key in pid.props) {
        pid.props[key] = { ...pid.props[key], ...props };
      } else {
        const edge = pid.edges.findIndex((edge) => edge.key === key);
        if (edge !== -1) {
          pid.edges[edge] = { ...pid.edges[edge], ...props };
        }
      }
    },
    setNodes: (state, { payload }: PayloadAction<SetNodesPayload>) => {
      const { layoutKey, nodes, mode = "replace" } = payload;
      const pid = state.pids[layoutKey];
      if (mode === "replace") pid.nodes = nodes;
      else {
        const keys = nodes.map((node) => node.key);
        pid.nodes = [...pid.nodes.filter((node) => !keys.includes(node.key)), ...nodes];
      }
      const anySelected =
        nodes.some((node) => node.selected) || pid.edges.some((edge) => edge.selected);
      if (anySelected) {
        if (state.toolbar.activeTab !== "properties")
          clearOtherSelections(state, layoutKey);
        state.toolbar.activeTab = "properties";
      } else state.toolbar.activeTab = "symbols";
    },
    setNodePositions: (state, { payload }: PayloadAction<SetNodePositionsPayload>) => {
      const { layoutKey, positions } = payload;
      const pid = state.pids[layoutKey];
      Object.entries(positions).forEach(([key, position]) => {
        const node = pid.nodes.find((node) => node.key === key);
        if (node == null) return;
        node.position = position;
      });
    },
    setEdges: (state, { payload }: PayloadAction<SetEdgesPayload>) => {
      const { layoutKey, edges } = payload;
      const pid = state.pids[layoutKey];
      // check for new edges
      const prevKeys = pid.edges.map((edge) => edge.key);
      const newEdges = edges.filter((edge) => !prevKeys.includes(edge.key));
      newEdges.forEach((edge) => {
        const source = pid.nodes.find((node) => node.key === edge.source);
        const target = pid.nodes.find((node) => node.key === edge.target);
        if (source == null || target == null) return;
        const sourceProps = pid.props[source.key];
        const targetProps = pid.props[target.key];
        if (sourceProps.color === targetProps.color && sourceProps.color != null)
          edge.color = sourceProps.color;
      });
      pid.edges = edges;
      const anySelected =
        edges.some((edge) => edge.selected) || pid.nodes.some((node) => node.selected);
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
      const { layoutKey, viewport } = payload;
      const pid = state.pids[layoutKey];
      pid.viewport = viewport;
    },
    setEditable: (state, { payload }: PayloadAction<SetEditablePayload>) => {
      const { layoutKey, editable } = payload;
      const pid = state.pids[layoutKey];
      clearSelections(pid);
      if (pid.control === "acquired") {
        pid.controlAcquireTrigger -= 1;
      }
      if (pid.snapshot) return;
      pid.editable = editable;
    },
    toggleControl: (state, { payload }: PayloadAction<ToggleControlPayload>) => {
      let { layoutKey, status } = payload;
      const pid = state.pids[layoutKey];
      if (status == null) status = pid.control === "released" ? "acquired" : "released";
      if (status === "released") pid.controlAcquireTrigger -= 1;
      else pid.controlAcquireTrigger += 1;
    },
    setControlStatus: (state, { payload }: PayloadAction<SetControlStatusPayload>) => {
      const { layoutKey, control } = payload;
      const pid = state.pids[layoutKey];
      if (pid == null) return;
      pid.control = control;
      if (control === "acquired") pid.editable = false;
    },
    setViewportMode: (
      state,
      { payload: { mode } }: PayloadAction<SetViewportModePayload>,
    ) => {
      state.mode = mode;
    },
    setRemoteCreated: (state, { payload }: PayloadAction<SetRemoteCreatedPayload>) => {
      const { layoutKey } = payload;
      const pid = state.pids[layoutKey];
      pid.remoteCreated = true;
    },
    fixThemeContrast: (state, { payload }: PayloadAction<FixThemeContrastPayload>) => {
      const { theme } = payload;
      const bgColor = new Color.Color(theme.colors.gray.l0);
      Object.values(state.pids).forEach((pid) => {
        const { nodes, edges, props } = pid;
        nodes.forEach((node) => {
          const nodeProps = props[node.key];
          if ("color" in nodeProps) {
            const c = new Color.Color(nodeProps.color as string);
            // check the contrast of the color
            if (c.contrast(bgColor) < 1.1) {
              // if the contrast is too low, change the color to the contrast color
              nodeProps.color = theme.colors.gray.l9;
            }
          }
        });
        edges.forEach((edge) => {
          if (
            edge.color != null &&
            new Color.Color(edge.color as string).contrast(bgColor) < 1.1
          ) {
            edge.color = theme.colors.gray.l9;
          } else if (edge.color == null) {
            edge.color = theme.colors.gray.l9;
          }
        });
      });
    },
  },
});

const clearOtherSelections = (state: SliceState, layoutKey: string): void => {
  Object.keys(state.pids).forEach((key) => {
    // If any of the nodes or edges in other Diagram slices are selected, deselect them.
    if (key === layoutKey) return;
    clearSelections(state.pids[key]);
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
  toggleControl,
  setControlStatus,
  addElement,
  setEdges,
  setNodes,
  remove,
  clearSelection,
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
export type Payload = Action["payload"];

export type LayoutType = "pid";
export const LAYOUT_TYPE = "pid";

export const create =
  (
    initial: Partial<State> & Omit<Partial<Layout.LayoutState>, "type">,
  ): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "PID", location = "mosaic", window, tab, ...rest } = initial;
    const key = initial.key ?? uuidV4();
    dispatch(actions.create({ ...deep.copy(ZERO_STATE), key, ...rest }));
    return {
      key,
      location,
      name,
      type: LAYOUT_TYPE,
      window,
      tab,
    };
  };
