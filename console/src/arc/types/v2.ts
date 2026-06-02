// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Viewport } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/arc/types/v1";

export const VERSION = "2.0.0";
export type Version = typeof VERSION;
export const TYPE = v1.TYPE;
export type Type = typeof TYPE;

export const STATE_MIGRATION_NAME = v1.STATE_MIGRATION_NAME;
export const SLICE_MIGRATION_NAME = v1.SLICE_MIGRATION_NAME;

export type NodeProps = v1.NodeProps;
export const nodePropsZ = v1.nodePropsZ;
export type GraphState = v1.GraphState;
export const graphStateZ = v1.graphStateZ;
export type Mode = v1.Mode;
export type CopyBuffer = v1.CopyBuffer;
export const copyBufferZ = v1.copyBufferZ;
export type ToolbarTab = v1.ToolbarTab;
export const toolbarTabZ = v1.toolbarTabZ;
export type ToolbarState = v1.ToolbarState;
export const toolbarStateZ = v1.toolbarStateZ;

export const stateZ = z.object({
  key: z.string(),
  version: z.literal(VERSION),
  remoteCreated: z.boolean(),
  graph: graphStateZ,
  text: arc.text.textZ.default({ raw: "" }),
  mode: arc.modeZ.default("graph"),
});

export interface State extends z.infer<typeof stateZ> {}

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

export const ZERO_GRAPH_STATE: GraphState = v1.ZERO_GRAPH_STATE;

export const ZERO_STATE: State = {
  ...v1.ZERO_STATE,
  version: VERSION,
};

export const ZERO_SLICE_STATE: SliceState = {
  ...v1.ZERO_SLICE_STATE,
  version: VERSION,
  arcs: {},
};

const legacySetStatusZ = z.looseObject({
  statusKey: z.string().optional(),
  variant: z.string().optional(),
  message: z.string().optional(),
});

// Rewrites the deprecated set_status node to status.set. The bare set_status
// STL symbol was removed in SY-4122; saved graphs must remap the props.
const rewriteSetStatusProp = (p: NodeProps): NodeProps => {
  if (p.key !== "set_status") return p;
  const legacy = legacySetStatusZ.parse(p);
  const next = {
    key: "status.set",
    key_or_name: legacy.statusKey ?? "",
    variant: legacy.variant ?? "success",
    message: legacy.message ?? "",
  };
  return next;
};

const migrateGraphState = (graph: v1.GraphState): GraphState => ({
  ...graph,
  props: Object.fromEntries(
    Object.entries(graph.props).map(([k, v]) => [k, rewriteSetStatusProp(v)]),
  ),
});

export const stateMigration = migrate.createMigration<v1.State, State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...state,
    version: VERSION,
    graph: migrateGraphState(state.graph),
  }),
});

export const sliceMigration = migrate.createMigration<v1.SliceState, SliceState>({
  name: SLICE_MIGRATION_NAME,
  migrate: ({ arcs, ...rest }) => ({
    ...rest,
    version: VERSION,
    arcs: Object.fromEntries(
      Object.entries(arcs).map(([key, arc]) => [key, stateMigration(arc)]),
    ),
  }),
});
