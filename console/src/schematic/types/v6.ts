// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { control, Schematic, Viewport } from "@synnaxlabs/pluto";
import { color, control as xcontrol, migrate, record, sticky, xy } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/schematic/types/v0";
import * as v1 from "@/schematic/types/v1";
import type * as v5 from "@/schematic/types/v5";

export const VERSION = "6.0.0";

export const toolbarTabZ = v0.toolbarTabZ;
export type ToolbarTab = v0.ToolbarTab;

export const viewportZ = z.object({
  position: xy.xyZ,
  zoom: z.number(),
  mode: Viewport.modeZ.default("select"),
});
export interface Viewport extends z.infer<typeof viewportZ> {}

export const nodeZ = schematic.nodeZ;
export type Node = schematic.Node;
export const edgeZ = schematic.edgeZ;
export type Edge = schematic.Edge;
export const handleZ = schematic.handleZ;
export type Handle = schematic.Handle;

export const elementConfigZ = Schematic.elementConfigZ;
export type EdgeConfig = Schematic.Edge.Config;
export type NodeConfig = Schematic.Node.Config;
export type ElementConfig = Schematic.ElementConfig;

export const legendStateZ = z.object({
  visible: z.boolean(),
  position: sticky.xyZ,
  colors: z.record(z.string(), color.colorZ).default({}),
});
export interface LegendState extends z.infer<typeof legendStateZ> {}

export const pendingUploadZ = schematic.schematicZ
  .omit({ configs: true, name: true })
  .extend({ configs: z.record(z.string(), elementConfigZ) });
export interface PendingUpload extends z.infer<typeof pendingUploadZ> {}

export const stateZ = z.object({
  version: z.literal(VERSION),
  selected: z.array(z.string()).default([]),
  controlStatus: control.statusZ,
  authority: xcontrol.authorityZ,
  legend: legendStateZ,
  toolbar: v0.toolbarStateZ,
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  viewport: viewportZ,
  pendingUpload: pendingUploadZ.optional(),
});
export interface State extends z.infer<typeof stateZ> {}

export const ZERO_STATE: State = {
  version: VERSION,
  selected: [],
  controlStatus: "released",
  authority: 1,
  legend: {
    visible: true,
    position: { x: 50, y: 50, units: { x: "px", y: "px" } },
    colors: {},
  },
  toolbar: { activeTab: "symbols", selectedSymbolGroup: "general" },
  editable: false,
  fitViewOnResize: false,
  viewport: { position: xy.ZERO, zoom: 1, mode: "select" },
  pendingUpload: undefined,
};

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  schematics: z.record(z.string(), stateZ),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = { version: VERSION, schematics: {} };

const migrateNode = (node: v0.Node): Node => {
  const next: Node = { key: node.key, position: node.position };
  if (node.zIndex != null) next.zIndex = node.zIndex;
  return next;
};

const migrateEdge = (edge: v0.Edge): [Edge, EdgeConfig] => {
  const next: Edge = {
    key: edge.key,
    source: { node: edge.source, param: edge.sourceHandle ?? "" },
    target: { node: edge.target, param: edge.targetHandle ?? "" },
  };
  const edgeConfig: EdgeConfig = {
    variant: "pipe" as Schematic.Edge.Variant,
    segments: [],
    color: color.ZERO,
  };
  const parseDataResult = record.unknownZ().safeParse(edge.data);
  if (!parseDataResult.success) return [next, edgeConfig];
  const data = parseDataResult.data;
  const segments = z.array(Schematic.Edge.Segmented.segmentZ).safeParse(data.segments);
  if (segments.success) edgeConfig.segments = segments.data;
  const parsedColor = color.colorZ.safeParse(data.color);
  if (parsedColor.success) edgeConfig.color = parsedColor.data;
  const parsedVariant = Schematic.Edge.variantZ.safeParse(data.variant);
  if (parsedVariant.success) edgeConfig.variant = parsedVariant.data;
  return [next, edgeConfig];
};

const migrateLegendColors = (
  colors: Record<string, string> | undefined,
): LegendState["colors"] => {
  if (colors == null) return {};
  const out: LegendState["colors"] = {};
  for (const [k, v] of Object.entries(colors)) {
    const parsed = color.colorZ.safeParse(v);
    if (parsed.success) out[k] = parsed.data;
  }
  return out;
};

const migratePropsToConfigs = (
  props: Record<string, v0.NodeProps>,
): Record<string, ElementConfig> =>
  Object.fromEntries(
    Object.entries(props).map(([k, p]) => {
      const { key, ...rest } = p as v0.NodeProps & Record<string, unknown>;
      return [k, { ...rest, variant: key } as ElementConfig];
    }),
  );

const buildPendingUpload = (state: v5.State): PendingUpload => {
  const configs = migratePropsToConfigs(state.props);
  const edges = state.edges.map((e) => {
    const [edge, edgeConfig] = migrateEdge(e);
    configs[edge.key] = edgeConfig;
    return edge;
  });
  return {
    key: state.key,
    nodes: state.nodes.map(migrateNode),
    edges,
    configs,
    snapshot: state.snapshot,
  };
};

export const stateMigration = migrate.createMigration<v5.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({
    version: VERSION,
    selected: [],
    authority: state.authority,
    controlStatus: state.control,
    legend: {
      visible: state.legend?.visible ?? true,
      position: state.legend?.position ?? {
        x: 50,
        y: 50,
        units: { x: "px", y: "px" },
      },
      colors: migrateLegendColors(state.legend?.colors),
    },
    toolbar: state.toolbar,
    editable: state.editable,
    fitViewOnResize: state.fitViewOnResize,
    viewport: { ...state.viewport, mode: "select" },
    pendingUpload: state.remoteCreated ? undefined : buildPendingUpload(state),
  }),
});

export const sliceMigration = migrate.createMigration<v5.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ schematics }) => ({
    version: VERSION,
    schematics: Object.fromEntries(
      Object.entries(schematics).map(([key, schematic]) => [
        key,
        stateMigration(schematic),
      ]),
    ),
  }),
});
