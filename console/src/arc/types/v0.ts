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
import { type migrate, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const VERSION = "0.0.0";
export type Version = typeof VERSION;
export const TYPE = "arc";
export type Type = typeof TYPE;

export type NodeProps = object & {
  key: string;
};

export const nodePropsZ = z.looseObject({
  key: z.string(),
});

// Frozen old node schema - nodes had a selected field and used flat xy
const nodeZ = z.object({
  key: z.string(),
  position: xy.xyZ,
  selected: z.boolean().optional().default(false),
  zIndex: z.number().optional(),
  type: z.string().optional(),
  measured: z
    .object({ width: z.number().optional(), height: z.number().optional() })
    .optional(),
});
export type Node = z.infer<typeof nodeZ>;

// Frozen old edge schema - edges used flat source/target strings with separate handle fields
const edgeZ = z.object({
  key: z.string(),
  source: z.string(),
  target: z.string(),
  id: z.string().optional(),
  selected: z.boolean().optional().default(false),
  sourceHandle: z.string().nullable().optional(),
  targetHandle: z.string().nullable().optional(),
  segments: z.array(z.unknown()).optional(),
  color: z.unknown().optional(),
});
export type Edge = z.infer<typeof edgeZ>;

const graphStateZ = z.object({
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  viewport: z.object({ position: xy.xyZ, zoom: z.number() }),
  selected: z.array(z.string()).default([]),
  nodes: z
    .array(z.any())
    .transform((nodes) => nodes.filter((node) => nodeZ.safeParse(node).success))
    .pipe(z.array(nodeZ)),
  edges: z
    .array(z.any())
    .transform((edges) => edges.filter((edge) => edgeZ.safeParse(edge).success))
    .pipe(z.array(edgeZ)),
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
  nodes: z.array(nodeZ),
  edges: z.array(z.unknown()),
  props: z.record(z.string(), z.unknown()),
});

export interface CopyBuffer {
  pos: xy.Crude;
  nodes: Node[];
  edges: Edge[];
  props: Record<string, NodeProps>;
}

const ZERO_COPY_BUFFER: CopyBuffer = { pos: xy.ZERO, nodes: [], edges: [], props: {} };

const TOOLBAR_TABS = ["stages", "properties"] as const;
export const toolbarTabZ = z.enum(TOOLBAR_TABS);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

export const toolbarStateZ = z.object({ activeTab: toolbarTabZ });
export type ToolbarState = z.infer<typeof toolbarStateZ>;

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
