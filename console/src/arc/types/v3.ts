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

import * as v2 from "@/arc/types/v2";

export const VERSION = "3.0.0";
export type Version = typeof VERSION;
export const TYPE = v2.TYPE;
export type Type = typeof TYPE;

export const STATE_MIGRATION_NAME = v2.STATE_MIGRATION_NAME;
export const SLICE_MIGRATION_NAME = v2.SLICE_MIGRATION_NAME;

export type Mode = v2.Mode;
export type ToolbarTab = v2.ToolbarTab;
export const toolbarTabZ = v2.toolbarTabZ;
export type ToolbarState = v2.ToolbarState;
export const toolbarStateZ = v2.toolbarStateZ;

// graphStateZ holds only the session UI state for the graph view. The graph document
// itself (nodes, edges, configs) lives in the flux store, server-synced via Arc actions.
export const graphStateZ = z.object({
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  viewport: Diagram.viewportZ,
  selected: z.array(z.string()).default([]),
});
export interface GraphState extends z.infer<typeof graphStateZ> {}

export const pendingUploadZ = z.object({
  mode: arc.modeZ,
  graph: arc.graph.graphZ,
  text: arc.text.textZ,
});
export interface PendingUpload extends z.infer<typeof pendingUploadZ> {}

export const stateZ = z.object({
  key: z.string(),
  version: z.literal(VERSION),
  graph: graphStateZ,
  pendingUpload: pendingUploadZ.optional(),
});
export interface State extends z.infer<typeof stateZ> {}

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  mode: Viewport.modeZ,
  toolbar: toolbarStateZ,
  arcs: z.record(z.string(), stateZ),
});

export interface SliceState extends migrate.Migratable<Version> {
  mode: Viewport.Mode;
  toolbar: ToolbarState;
  arcs: Record<string, State>;
}

export const ZERO_GRAPH_STATE: GraphState = {
  editable: true,
  fitViewOnResize: false,
  viewport: { position: xy.ZERO, zoom: 1 },
  selected: [],
};

export const ZERO_STATE: State = {
  key: "",
  version: VERSION,
  graph: ZERO_GRAPH_STATE,
  pendingUpload: undefined,
};

export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  mode: "select",
  toolbar: { activeTab: "stages" },
  arcs: {},
};

// buildPendingGraph converts a legacy v2 redux graph into a flux graph document,
// lifting each node's inline props (with the function type under "key") into the configs
// map (under "type"), keyed by node key.
const buildPendingGraph = (state: v2.State): arc.graph.Graph => ({
  nodes: state.graph.nodes.map((n) => ({ key: n.key, position: n.position })),
  edges: state.graph.edges.map((e) => ({
    source: e.source,
    target: e.target,
    kind: arc.ir.EdgeKind.continuous,
  })),
  configs: Object.fromEntries(
    Object.entries(state.graph.props).map(([k, { key, ...rest }]) => [
      k,
      { ...rest, type: key },
    ]),
  ),
  functions: [],
});

export const stateMigration = migrate.createMigration<v2.State, State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    key: state.key,
    version: VERSION,
    remoteCreated: state.remoteCreated,
    graph: {
      editable: state.graph.editable,
      fitViewOnResize: state.graph.fitViewOnResize,
      viewport: state.graph.viewport,
      selected: state.graph.selected,
    },
    mode: state.mode,
    pendingUpload: state.remoteCreated
      ? undefined
      : {
          graph: buildPendingGraph(state),
          text: arc.text.textZ.parse({ raw: state.text.raw }),
          mode: state.mode,
        },
  }),
});

export const sliceMigration = migrate.createMigration<v2.SliceState, SliceState>({
  name: SLICE_MIGRATION_NAME,
  migrate: ({ arcs, ...rest }) => ({
    version: VERSION,
    mode: rest.mode,
    toolbar: rest.toolbar,
    arcs: Object.fromEntries(
      Object.entries(arcs).map(([key, arc]) => [key, stateMigration(arc)]),
    ),
  }),
});
