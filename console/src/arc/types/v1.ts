// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Diagram, Viewport } from "@synnaxlabs/pluto";
import { migrate, xy } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/arc/types/v0";

export const VERSION = "1.0.0";
export type Version = typeof VERSION;
export const TYPE = v0.TYPE;
export type Type = typeof TYPE;

export const STATE_MIGRATION_NAME = "arc.state";
export const SLICE_MIGRATION_NAME = "arc.slice";

export type NodeProps = v0.NodeProps;
export const nodePropsZ = v0.nodePropsZ;

const graphStateZ = z.object({
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  viewport: Diagram.viewportZ,
  selected: z.array(z.string()).default([]),
  nodes: z.array(Diagram.nodeZ),
  edges: z.array(Diagram.edgeZ),
  props: z.record(z.string(), nodePropsZ),
});

export interface GraphState extends z.infer<typeof graphStateZ> {}

export type Mode = arc.Mode | undefined;

export const stateZ = z.object({
  key: z.string(),
  version: z.literal(VERSION),
  remoteCreated: z.boolean(),
  graph: graphStateZ,
  text: arc.text.textZ.default({ raw: "" }),
  mode: arc.modeZ.default("graph"),
});

export interface State extends z.infer<typeof stateZ> {}

export const copyBufferZ = z.object({
  pos: xy.xyZ,
  nodes: z.array(Diagram.nodeZ),
  edges: z.array(Diagram.edgeZ),
  props: z.record(z.string(), z.unknown()),
});

export interface CopyBuffer {
  pos: xy.Crude;
  nodes: Diagram.Node[];
  edges: Diagram.Edge[];
  props: Record<string, NodeProps>;
}

const ZERO_COPY_BUFFER: CopyBuffer = { pos: xy.ZERO, nodes: [], edges: [], props: {} };

export type ToolbarTab = v0.ToolbarTab;
export const toolbarTabZ = v0.toolbarTabZ;
export type ToolbarState = v0.ToolbarState;
export const toolbarStateZ = v0.toolbarStateZ;

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  mode: Viewport.modeZ,
  copy: copyBufferZ,
  toolbar: toolbarStateZ,
  arcs: z.record(z.string(), stateZ),
});

export interface SliceState extends migrate.Migratable<Version> {
  mode: Viewport.Mode;
  copy: CopyBuffer;
  toolbar: ToolbarState;
  arcs: Record<string, State>;
}

export const ZERO_GRAPH_STATE: GraphState = {
  editable: true,
  fitViewOnResize: false,
  viewport: { position: xy.ZERO, zoom: 1 },
  selected: [],
  nodes: [],
  edges: [],
  props: {},
};

export const ZERO_STATE: State = {
  key: "",
  version: VERSION,
  graph: ZERO_GRAPH_STATE,
  remoteCreated: false,
  text: { raw: "" },
  mode: "graph",
};

export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  mode: "select",
  copy: { ...ZERO_COPY_BUFFER },
  toolbar: { activeTab: "stages" },
  arcs: {},
};

const migrateEdge = (edge: v0.Edge): Diagram.Edge => ({
  key: edge.key,
  source: { node: edge.source, param: edge.sourceHandle ?? "" },
  target: { node: edge.target, param: edge.targetHandle ?? "" },
});

const migrateGraphState = (graph: v0.GraphState): GraphState => {
  const selected = [
    ...graph.nodes.filter((n) => n.selected).map((n) => n.key),
    ...graph.edges.filter((e) => e.selected).map((e) => e.key),
    ...graph.selected,
  ];
  return {
    editable: graph.editable,
    fitViewOnResize: graph.fitViewOnResize,
    viewport: { position: graph.viewport.position, zoom: graph.viewport.zoom },
    selected: [...new Set(selected)],
    nodes: graph.nodes.map((n) => ({
      key: n.key,
      position: n.position,
      zIndex: n.zIndex,
      type: n.type,
      measured: n.measured,
    })),
    edges: graph.edges.map(migrateEdge),
    props: graph.props,
  };
};

export const stateMigration = migrate.createMigration<v0.State, State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...state,
    version: VERSION,
    graph: migrateGraphState(state.graph),
  }),
});

export const sliceMigration = migrate.createMigration<v0.SliceState, SliceState>({
  name: SLICE_MIGRATION_NAME,
  migrate: ({ arcs, ...rest }) => ({
    ...rest,
    version: VERSION,
    copy: ZERO_COPY_BUFFER,
    arcs: Object.fromEntries(
      Object.entries(arcs).map(([key, arc]) => [key, stateMigration(arc)]),
    ),
  }),
});
