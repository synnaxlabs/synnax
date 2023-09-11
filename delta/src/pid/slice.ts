// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction, createSlice } from "@reduxjs/toolkit";
import { type PID, type Control, type Viewport } from "@synnaxlabs/pluto";
import { box, scale, xy, deep } from "@synnaxlabs/x";
import { nanoid } from "nanoid";

import { type Layout } from "@/layout";

export type NodeProps = object & {
  type: string;
};

export interface State {
  editable: boolean;
  viewport: PID.Viewport;
  nodes: PID.Node[];
  edges: PID.Edge[];
  props: Record<string, object>;
  control: Control.Status;
  controlAcquireTrigger: number;
}

interface CopyBuffer {
  pos: xy.Crude;
  nodes: PID.Node[];
  edges: PID.Edge[];
  props: Record<string, object>;
}

const ZERO_COPY_BUFFER: CopyBuffer = {
  pos: xy.ZERO,
  nodes: [],
  edges: [],
  props: {},
};

// ||||| TOOLBAR |||||

const TOOLBAR_TABS = ["elements", "properties"] as const;
export type ToolbarTab = (typeof TOOLBAR_TABS)[number];

export interface ToolbarState {
  activeTab: ToolbarTab;
}

export interface SliceState {
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
  nodes: [],
  edges: [],
  props: {},
  viewport: { position: xy.ZERO, zoom: 1 },
  editable: true,
  control: "released",
  controlAcquireTrigger: 0,
};

export const ZERO_PID_SLICE_STATE: SliceState = {
  mode: "select",
  copy: { ...ZERO_COPY_BUFFER },
  toolbar: { activeTab: "elements" },
  pids: {},
};

export interface SetViewportPayload {
  layoutKey: string;
  viewport: PID.Viewport;
}

export interface AddElementPayload {
  layoutKey: string;
  key: string;
  props: NodeProps;
  node?: Partial<PID.Node>;
}

export interface SetElementPropsPayload {
  layoutKey: string;
  key: string;
  props: NodeProps;
}

export interface SetNodesPayload {
  layoutKey: string;
  nodes: PID.Node[];
}

export interface SetEdgesPayload {
  layoutKey: string;
  edges: PID.Edge[];
}

export interface CreatePayload {
  key: string;
}

export interface DeletePayload {
  layoutKey: string;
}

export interface SetEditablePayload {
  layoutKey: string;
  editable: boolean;
}

export interface SetControlStatusPayload {
  layoutKey: string;
  control: Control.Status;
}

export interface TogggleControlPayload {
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

export interface SetViewportModePayload {
  mode: Viewport.Mode;
}

export const calculatePos = (
  region: box.Box,
  cursor: xy.XY,
  viewport: PID.Viewport
): xy.XY => {
  const zoomXY = xy.construct(viewport.zoom);
  const s = scale.XY.translate(xy.scale(box.topLeft(region), -1))
    .magnify({
      x: 1 / zoomXY.x,
      y: 1 / zoomXY.y,
    })
    .translate(xy.scale(viewport.position, -1));
  return s.pos(cursor);
};

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_PID_SLICE_STATE,
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
          xy.ZERO
        );
        copyBuffer.pos = xy.scale(pos, 1 / nodes.length);
      }
      state.copy = copyBuffer;
    },
    pasteSelection: (state, { payload }: PayloadAction<PasteSelectionPayload>) => {
      const { pos, layoutKey } = payload;
      const delta = xy.translation(state.copy.pos, pos);
      const pid = state.pids[layoutKey];
      const keys: Record<string, string> = {};
      const nextNodes = state.copy.nodes.map((node) => {
        const key: string = nanoid();
        pid.props[key] = state.copy.props[node.key];
        keys[node.key] = key;
        return {
          ...node,
          position: xy.translate(node.position, delta),
          key,
        };
      });
      const nextEdges = state.copy.edges.map((edge) => {
        const key: string = nanoid();
        return {
          ...edge,
          key,
          source: keys[edge.source],
          target: keys[edge.target],
          points: edge.points.map((point) => xy.translate(point, delta)),
        };
      });
      pid.edges = [...pid.edges, ...nextEdges];
      pid.nodes = [...pid.nodes, ...nextNodes];
    },
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const { key: layoutKey } = payload;
      state.pids[layoutKey] = { ...ZERO_STATE };
    },
    delete: (state, { payload }: PayloadAction<DeletePayload>) => {
      const { layoutKey } = payload;
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete state.pids[layoutKey];
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
      const { layoutKey, nodes } = payload;
      const pid = state.pids[layoutKey];
      pid.nodes = nodes;
      const anySelected = nodes.some((node) => node.selected);
      if (anySelected) state.toolbar.activeTab = "properties";
      else state.toolbar.activeTab = "elements";
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
        if (sourceProps.color === targetProps.color) edge.color = sourceProps.color;
      });
      pid.edges = edges;
      const anySelected = edges.some((edge) => edge.selected);
      if (anySelected) state.toolbar.activeTab = "properties";
      else state.toolbar.activeTab = "elements";
    },
    setActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetActiveToolbarTabPayload>
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
      pid.editable = editable;
    },
    toggleControl: (state, { payload }: PayloadAction<TogggleControlPayload>) => {
      let { layoutKey, status } = payload;
      const pid = state.pids[layoutKey];
      if (status == null) status = pid.control === "released" ? "acquired" : "released";
      pid.controlAcquireTrigger += -2 * Number(status === "released") + 1;
    },
    setControlStatus: (state, { payload }: PayloadAction<SetControlStatusPayload>) => {
      const { layoutKey, control } = payload;
      const pid = state.pids[layoutKey];
      pid.control = control;
      if (control === "acquired") pid.editable = false;
    },
    setViewportMode: (
      state,
      { payload: { mode } }: PayloadAction<SetViewportModePayload>
    ) => {
      state.mode = mode;
    },
  },
});

export const {
  toggleControl,
  setControlStatus,
  addElement,
  setEdges,
  setNodes,
  setElementProps,
  setActiveToolbarTab,
  setViewport,
  setEditable,
  copySelection,
  pasteSelection,
  setViewportMode,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

export type LayoutType = "pid";
export const LAYOUT_TYPE = "pid";

export const create =
  (
    initial: Partial<State> & Omit<Partial<Layout.LayoutState>, "type">
  ): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "PID", location = "mosaic", window, tab, ...rest } = initial;
    const key = initial.key ?? nanoid();
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
