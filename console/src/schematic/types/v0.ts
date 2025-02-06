// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Color,
  type Control,
  control,
  Diagram,
  Schematic,
  Viewport,
} from "@synnaxlabs/pluto";
import { type migrate, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const VERSION = "0.0.0";
export type Version = typeof VERSION;

export type NodeProps = object & {
  key: Schematic.Variant;
  color?: Color.Crude;
  label?: { label?: string };
};

export const nodePropsZ = z
  .object({})
  .and(
    z.object({ key: Schematic.variantZ, color: Color.crudeZ.optional() }).passthrough(),
  );

export const stateZ = z.object({
  version: z.literal(VERSION),
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  snapshot: z.boolean(),
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
  control: control.statusZ,
  controlAcquireTrigger: z.number(),
});

export interface State extends migrate.Migratable<Version> {
  editable: boolean;
  fitViewOnResize: boolean;
  snapshot: boolean;
  remoteCreated: boolean;
  viewport: Diagram.Viewport;
  nodes: Diagram.Node[];
  edges: Diagram.Edge[];
  props: Record<string, NodeProps>;
  control: Control.Status;
  controlAcquireTrigger: number;
}

export const copyBufferZ = z.object({
  pos: xy.xy,
  nodes: z.array(Diagram.nodeZ),
  edges: z.array(z.unknown()),
  props: z.record(z.unknown()),
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
  schematics: z.record(z.string(), stateZ),
});

export interface SliceState extends migrate.Migratable<Version> {
  mode: Viewport.Mode;
  copy: CopyBuffer;
  toolbar: ToolbarState;
  schematics: Record<string, State>;
}

export const ZERO_STATE: State = {
  version: VERSION,
  snapshot: false,
  nodes: [],
  edges: [],
  props: {},
  remoteCreated: false,
  viewport: { position: xy.ZERO, zoom: 1 },
  editable: true,
  control: "released",
  controlAcquireTrigger: 0,
  fitViewOnResize: false,
};

export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  mode: "select",
  copy: { ...ZERO_COPY_BUFFER },
  toolbar: { activeTab: "symbols" },
  schematics: {},
};
