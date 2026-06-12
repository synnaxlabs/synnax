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
import {
  caseconv,
  color,
  control as xcontrol,
  location,
  migrate,
  record,
  sticky,
  xy,
} from "@synnaxlabs/x";
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

type Segment = Schematic.Edge.Segmented.Segment;

const segmentOrientation = (seg: Segment): location.Outer =>
  seg.direction === "x"
    ? seg.length > 0
      ? "right"
      : "left"
    : seg.length > 0
      ? "bottom"
      : "top";

const STUMP_LENGTH = Schematic.Edge.Segmented.STUMP_LENGTH;

// An edge whose stumps overlap has no middle to preserve: a single segment shorter than
// two stumps, or a first/last segment shorter than one stump. Subtracting a full stump
// from those would flip the segment and fold a spur, so they are cleared to auto-route
// instead (which handles short, facing handles cleanly).
const hasNoStrippableMiddle = (segments: Segment[]): boolean =>
  segments.length === 1
    ? Math.abs(segments[0].length) <= 2 * STUMP_LENGTH - 0.5
    : Math.abs(segments[0].length) <= STUMP_LENGTH - 0.5 ||
      Math.abs(segments[segments.length - 1].length) <= STUMP_LENGTH - 0.5;

// Pre-0.56 edges stored the full path including both stumps; the current model stores
// only the middle and re-derives stumps on render, so leaving them doubles the stumps
// and folds a pigtail over the target. The stumps are the first and last segments.
const stripLegacyStumps = (segments: Segment[]): Segment[] => {
  if (segments.length === 0) return segments;
  if (hasNoStrippableMiddle(segments)) return [];
  const sourceOrientation = segmentOrientation(segments[0]);
  const targetOrientation = location.swap(
    segmentOrientation(segments[segments.length - 1]),
  ) as location.Outer;
  return Schematic.Edge.Segmented.extractMiddle(
    segments,
    sourceOrientation,
    targetOrientation,
  );
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
  if (segments.success) edgeConfig.segments = stripLegacyStumps(segments.data);
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

// segProp reads a property from a named segment of a legacy pipeline spec;
// single-segment specs store the segment at the top level. Mirrors convert.go.
const segProp = (spec: unknown, segment: string, prop: string): unknown => {
  if (typeof spec !== "object" || spec == null) return undefined;
  const props = (spec as record.Unknown).props;
  if (typeof props !== "object" || props == null) return undefined;
  let segments = (props as record.Unknown).segments;
  if (typeof segments !== "object" || segments == null) segments = { [segment]: spec };
  const seg = (segments as record.Unknown)[segment];
  if (typeof seg !== "object" || seg == null) return undefined;
  const segProps = (seg as record.Unknown).props;
  if (typeof segProps !== "object" || segProps == null) return undefined;
  const v = (segProps as record.Unknown)[prop];
  if (prop === "channel" && v === 0) return undefined;
  return v;
};

// extractTelemArgs rewrites legacy telem pipeline specs into the schema's
// semantic arguments, in place. Mirrors extractTelemArgs in convert.go.
const extractTelemArgs = (cfg: record.Unknown): void => {
  const setIfPresent = (key: string, v: unknown): void => {
    if (v !== undefined) cfg[key] = v;
  };
  switch (cfg.variant) {
    case "value":
    case "gauge":
      setIfPresent("channel", segProp(cfg.telem, "valueStream", "channel"));
      setIfPresent(
        "rollingAverage",
        segProp(cfg.telem, "rollingAverage", "windowSize"),
      );
      delete cfg.telem;
      delete cfg.backgroundTelem;
      break;
    case "light":
      setIfPresent("channel", segProp(cfg.source, "valueStream", "channel"));
      setIfPresent("threshold", segProp(cfg.source, "threshold", "trueBound"));
      delete cfg.source;
      break;
    case "state_indicator":
      setIfPresent("channel", segProp(cfg.source, "valueStream", "channel"));
      delete cfg.source;
      break;
    case "setpoint":
      setIfPresent("stateChannel", segProp(cfg.source, "valueStream", "channel"));
      setIfPresent("commandChannel", segProp(cfg.sink, "setter", "channel"));
      delete cfg.source;
      delete cfg.sink;
      break;
    case "button":
    case "select":
    case "input":
      setIfPresent("commandChannel", segProp(cfg.sink, "setter", "channel"));
      delete cfg.sink;
      break;
    default:
      if ("source" in cfg) {
        setIfPresent("stateChannel", segProp(cfg.source, "valueStream", "channel"));
        delete cfg.source;
      }
      if ("sink" in cfg) {
        setIfPresent("commandChannel", segProp(cfg.sink, "setter", "channel"));
        delete cfg.sink;
      }
  }
  const control = cfg.control;
  if (typeof control !== "object" || control == null) return;
  const ctl = control as record.Unknown;
  const obj = (v: unknown): record.Unknown | undefined =>
    typeof v === "object" && v != null ? (v as record.Unknown) : undefined;
  const authority = obj(obj(obj(ctl.chip)?.sink)?.props)?.authority;
  if (authority !== undefined) ctl.authority = authority;
  delete ctl.chip;
  delete ctl.indicator;
};

const migratePropsToConfigs = (
  props: Record<string, v0.NodeProps>,
): Record<string, ElementConfig> =>
  Object.fromEntries(
    Object.entries(props).map(([k, p]) => {
      const { key, ...rest } = p as v0.NodeProps & Record<string, unknown>;
      const cfg: record.Unknown = { ...rest, variant: caseconv.camelToSnake(key) };
      extractTelemArgs(cfg);
      return [k, cfg as ElementConfig];
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
