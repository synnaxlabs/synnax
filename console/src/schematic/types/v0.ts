// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control, Flex, Schematic, Text, Value, Viewport } from "@synnaxlabs/pluto";
import { color, direction, location, xy } from "@synnaxlabs/x";
import { z } from "zod";

export const VERSION = "0.0.0";

// These schemas are a frozen snapshot of the diagram node/edge/viewport shapes
// that shipped at state version 0.0.0. Do NOT replace with Diagram.nodeZ /
// Diagram.edgeZ / Diagram.viewportZ — those references drift with Pluto
// refactors (the diagram edge reshape on this branch turned `source` from a
// string into `{node, param}`) and cause migration parsing to reject or drop
// real historical persisted state.
const nodeZ = z.looseObject({
  key: z.string(),
  position: xy.xyZ,
  zIndex: z.number().optional(),
  type: z.string().optional(),
  measured: z
    .object({ width: z.number().optional(), height: z.number().optional() })
    .optional(),
});

const edgeZ = z.looseObject({
  key: z.string(),
  source: z.string(),
  target: z.string(),
  sourceHandle: z.string().nullable().optional(),
  targetHandle: z.string().nullable().optional(),
});

const viewportZ = z.object({
  position: xy.xyZ,
  zoom: z.number(),
});

export const labelZ = z.looseObject({
  label: z.string().optional(),
  level: Text.levelZ.optional(),
  orientation: location.locationZ.optional(),
  direction: direction.directionZ.optional(),
  maxInlineSize: z.number().optional(),
  align: Flex.alignmentZ.optional(),
});

export const nodePropsZ = z.looseObject({
  variant: Schematic.Symbol.variantZ,
  color: color.crudeZ.optional(),
  label: labelZ.optional(),
});
export interface NodeProps extends z.infer<typeof nodePropsZ> {}

export const edgePropsZ = z.object({
  color: color.crudeZ.optional(),
  variant: Schematic.Edge.edgeTypeZ.optional(),
});
export interface EdgeProps extends z.infer<typeof edgePropsZ> {}

export const stateZ = z.object({
  version: z.literal(VERSION),
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  snapshot: z.boolean(),
  remoteCreated: z.boolean(),
  viewport: viewportZ,
  nodes: z
    .array(z.unknown())
    .transform((nodes) => nodes.filter((node) => nodeZ.safeParse(node).success))
    .pipe(z.array(nodeZ)),
  edges: z
    .array(z.unknown())
    .transform((edges) => edges.filter((edge) => edgeZ.safeParse(edge).success))
    .pipe(z.array(edgeZ)),
  props: z.record(z.string(), nodePropsZ).transform((p) => {
    for (const key in p)
      if (p[key].variant === "value") {
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
  pos: xy.xyZ,
  nodes: z.array(nodeZ),
  edges: z.array(edgeZ),
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
