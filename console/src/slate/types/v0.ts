// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Diagram, Viewport } from "@synnaxlabs/pluto";
import { type migrate, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const VERSION = "0.0.0";
export type Version = typeof VERSION;
export const TYPE = "slate";
export type Type = typeof TYPE;

export type NodeProps = object & {
  key: string;
};

export const nodePropsZ = z.object({}).and(z.object({ key: z.string() }).loose());

export const stateZ = z.object({
  key: z.string(),
  type: z.literal(TYPE),
  version: z.literal(VERSION),
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  remoteCreated: z.boolean(),
  viewport: Diagram.viewportZ,
  nodes: z
    .array(z.any())
    .transform((nodes) => nodes.filter((node) => Diagram.nodeZ.safeParse(node).success))
    .pipe(z.array(Diagram.nodeZ)),
  edges: z
    .array(z.any())
    .transform((edges) => edges.filter((edge) => Diagram.edgeZ.safeParse(edge).success))
    .pipe(z.array(Diagram.edgeZ)),
  props: z.record(z.string(), nodePropsZ),
});

export interface State extends migrate.Migratable<Version> {
  key: string;
  editable: boolean;
  fitViewOnResize: boolean;
  remoteCreated: boolean;
  viewport: Diagram.Viewport;
  nodes: Diagram.Node[];
  edges: Diagram.Edge[];
  props: Record<string, NodeProps>;
}

export const copyBufferZ = z.object({
  pos: xy.xy,
  nodes: z.array(Diagram.nodeZ),
  edges: z.array(z.unknown()),
  props: z.record(z.string(), z.unknown()),
});

export interface CopyBuffer {
  pos: xy.Crude;
  nodes: Diagram.Node[];
  edges: Diagram.Edge[];
  props: Record<string, NodeProps>;
}

const ZERO_COPY_BUFFER: CopyBuffer = { pos: xy.ZERO, nodes: [], edges: [], props: {} };

// ||||| TOOLBAR |||||

const TOOLBAR_TABS = ["symbols", "properties"] as const;
export const toolbarTabZ = z.enum(TOOLBAR_TABS);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

export const toolbarStateZ = z.object({ activeTab: toolbarTabZ });
export type ToolbarState = z.infer<typeof toolbarStateZ>;

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  mode: Viewport.modeZ,
  copy: copyBufferZ,
  toolbar: toolbarStateZ,
  slates: z.record(z.string(), stateZ),
});

export interface SliceState extends migrate.Migratable<Version> {
  mode: Viewport.Mode;
  copy: CopyBuffer;
  toolbar: ToolbarState;
  slates: Record<string, State>;
}

export const ZERO_STATE: State = {
  key: "",
  version: VERSION,
  nodes: [],
  edges: [],
  props: {},
  remoteCreated: false,
  viewport: { position: xy.ZERO, zoom: 1 },
  editable: true,
  fitViewOnResize: false,
};

export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  mode: "select",
  copy: { ...ZERO_COPY_BUFFER },
  toolbar: { activeTab: "symbols" },
  slates: {},
};
