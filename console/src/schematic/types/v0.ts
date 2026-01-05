// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control, Diagram, Schematic, Value, Viewport } from "@synnaxlabs/pluto";
import { color, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const VERSION = "0.0.0";

export const nodePropsZ = z.looseObject({
  key: Schematic.Symbol.variantZ,
  color: color.crudeZ.optional(),
  label: z.looseObject({ label: z.string().optional() }).optional(),
});
export interface NodeProps extends z.infer<typeof nodePropsZ> {}

export const edgePropsZ = z.object({
  color: color.crudeZ.optional(),
  variant: Schematic.edgeTypeZ.optional(),
});
export interface EdgeProps extends z.infer<typeof edgePropsZ> {}

export const stateZ = z.object({
  version: z.literal(VERSION),
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  snapshot: z.boolean(),
  remoteCreated: z.boolean(),
  viewport: Diagram.viewportZ,
  nodes: z
    .array(z.unknown())
    .transform((nodes) => nodes.filter((node) => Diagram.nodeZ.safeParse(node).success))
    .pipe(z.array(Diagram.nodeZ)),
  edges: z
    .array(z.unknown())
    .transform((edges) => edges.filter((edge) => Diagram.edgeZ.safeParse(edge).success))
    .pipe(z.array(Diagram.edgeZ)),
  props: z.record(z.string(), nodePropsZ).transform((p) => {
    for (const key in p)
      if (p[key].key === "value") {
        p[key].redline = Value.ZERO_READLINE;
        p[key].stalenessTimeout = 5;
        p[key].stalenessColor = color.ZERO;
      }
    return p;
  }),
  control: control.statusZ,
});
export interface State extends z.infer<typeof stateZ> {}
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
  fitViewOnResize: false,
};

export const copyBufferZ = z.object({
  pos: xy.xy,
  nodes: z.array(Diagram.nodeZ),
  edges: z.array(Diagram.edgeZ),
  props: z.record(z.string(), nodePropsZ),
});
export interface CopyBuffer extends z.infer<typeof copyBufferZ> {}
const ZERO_COPY_BUFFER: CopyBuffer = { pos: xy.ZERO, nodes: [], edges: [], props: {} };

export const toolbarTabZ = z.enum(["symbols", "properties"]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

export const toolbarStateZ = z.object({
  activeTab: toolbarTabZ,
  selectedSymbolGroup: z.string().default("general"),
});
export interface ToolbarState extends z.infer<typeof toolbarStateZ> {}
export const ZERO_TOOLBAR_STATE: ToolbarState = {
  activeTab: "symbols",
  selectedSymbolGroup: "general",
};

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  mode: Viewport.modeZ,
  copy: copyBufferZ,
  toolbar: toolbarStateZ,
  schematics: z.record(z.string(), stateZ),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  mode: "select",
  copy: ZERO_COPY_BUFFER,
  toolbar: ZERO_TOOLBAR_STATE,
  schematics: {},
};
