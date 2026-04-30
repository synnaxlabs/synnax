// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Schematic } from "@synnaxlabs/pluto";
import { color, migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/schematic/types/v0";
import * as v1 from "@/schematic/types/v1";
import * as v5 from "@/schematic/types/v5";

export const VERSION = "6.0.0";

// Node, Edge, Handle, Segment, EdgeProps, and Legend are aliases for the
// oracle-generated wire types. Keeping them as aliases (not re-declarations)
// guarantees the console state and the server schema can never drift.
export const nodeZ = schematic.nodeZ;
export type Node = schematic.Node;
export const edgeZ = schematic.edgeZ;
export type Edge = schematic.Edge;
export const handleZ = schematic.handleZ;
export type Handle = schematic.Handle;
export const segmentZ = schematic.segmentZ;
export type Segment = schematic.Segment;
export const edgePropsZ = schematic.edgePropsZ;
export type EdgeProps = schematic.EdgeProps;
export const legendZ = schematic.legendZ;
export type Legend = schematic.Legend;

// NodeProps is per-variant opaque on the server (props is map<string, record>)
// because the per-symbol-variant fields vary. This declares the common base
// shape known to console UI: variant, color, label. Variant identifiers come
// from the symbol registry; label uses pluto Text / Flex / location enums
// that aren't in the oracle schema.
export const nodePropsZ = z.looseObject({
  variant: Schematic.Symbol.variantZ,
  color: color.colorZ.optional(),
  label: v0.labelZ.optional(),
});
export interface NodeProps extends z.infer<typeof nodePropsZ> {}

export const propsZ = z.union([nodePropsZ, edgePropsZ]);
export type Props = z.infer<typeof propsZ>;

export const stateZ = v5.stateZ
  .omit({ version: true, nodes: true, edges: true, props: true, legend: true })
  .extend({
    version: z.literal(VERSION),
    nodes: z.array(nodeZ),
    edges: z.array(edgeZ),
    props: schematic.schematicZ.shape.props,
    legend: legendZ,
    selected: z.array(z.string()).default([]),
  });
export interface State extends z.infer<typeof stateZ> {}

const ZERO_LEGEND: Legend = {
  visible: true,
  position: { x: 50, y: 50, units: { x: "px", y: "px" } },
  colors: {},
};

export const ZERO_STATE: State = {
  ...v5.ZERO_STATE,
  version: VERSION,
  nodes: [],
  edges: [],
  props: {},
  legend: ZERO_LEGEND,
  selected: [],
};

export const copyBufferZ = z.object({
  pos: v0.copyBufferZ.shape.pos,
  nodes: z.array(nodeZ),
  edges: z.array(edgeZ),
  props: schematic.schematicZ.shape.props,
});
export interface CopyBuffer extends z.infer<typeof copyBufferZ> {}
const ZERO_COPY_BUFFER: CopyBuffer = {
  pos: { x: 0, y: 0 },
  nodes: [],
  edges: [],
  props: {},
};

export const sliceStateZ = v5.sliceStateZ
  .omit({ version: true, schematics: true, copy: true })
  .extend({
    version: z.literal(VERSION),
    schematics: z.record(z.string(), stateZ),
    copy: copyBufferZ,
  });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  ...v5.ZERO_SLICE_STATE,
  version: VERSION,
  schematics: {},
  copy: ZERO_COPY_BUFFER,
};

const migrateEdge = (edge: v0.Edge): { edge: Edge; edgeProps?: EdgeProps } => {
  const next: Edge = {
    key: edge.key,
    source: { node: edge.source, param: edge.sourceHandle ?? "" },
    target: { node: edge.target, param: edge.targetHandle ?? "" },
  };
  const data = (edge as Record<string, unknown>).data as
    | Record<string, unknown>
    | undefined;
  if (data == null) return { edge: next };
  const edgeProps: Partial<EdgeProps> = {};
  const segments = z.array(segmentZ).safeParse(data.segments);
  if (segments.success) edgeProps.segments = segments.data;
  const parsedColor = color.colorZ.safeParse(data.color);
  if (parsedColor.success) edgeProps.color = parsedColor.data;
  const parsedVariant = Schematic.Edge.edgeTypeZ.safeParse(data.variant);
  if (parsedVariant.success) edgeProps.variant = parsedVariant.data;
  return { edge: next, edgeProps: edgePropsZ.parse(edgeProps) };
};

const migrateNode = (node: v0.Node): Node => ({
  key: node.key,
  position: node.position,
  zIndex: node.zIndex ?? 0,
});

const migrateLegendColors = (
  colors: Record<string, string> | undefined,
): Legend["colors"] => {
  if (colors == null) return {};
  const out: Legend["colors"] = {};
  for (const [k, v] of Object.entries(colors)) {
    const parsed = color.colorZ.safeParse(v);
    if (parsed.success) out[k] = parsed.data;
  }
  return out;
};

const migrateProps = (props: Record<string, v0.NodeProps>): Record<string, Props> =>
  Object.fromEntries(
    Object.entries(props).map(([k, p]) => {
      const { key, ...rest } = p as v0.NodeProps & Record<string, unknown>;
      return [k, { ...rest, variant: key } as NodeProps];
    }),
  );

export const stateMigration = migrate.createMigration<v5.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => {
    const props = migrateProps(state.props);
    const edges: Edge[] = [];
    for (const e of state.edges) {
      const { edge, edgeProps } = migrateEdge(e);
      edges.push(edge);
      if (edgeProps != null) props[edge.key] = edgeProps;
    }
    const nodes = state.nodes.map(migrateNode);
    return {
      ...state,
      version: VERSION,
      nodes,
      edges,
      props,
      legend: { ...state.legend, colors: migrateLegendColors(state.legend?.colors) },
      selected: [],
    };
  },
});

const migrateCopyBuffer = (copy: v5.SliceState["copy"]): CopyBuffer => {
  const props = migrateProps(copy.props);
  const edges: Edge[] = [];
  for (const e of copy.edges) {
    const { edge, edgeProps } = migrateEdge(e);
    edges.push(edge);
    if (edgeProps != null) props[edge.key] = edgeProps;
  }
  return {
    pos: copy.pos,
    nodes: copy.nodes.map(migrateNode),
    edges,
    props,
  };
};

export const sliceMigration = migrate.createMigration<v5.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ schematics, copy, ...rest }) => ({
    ...rest,
    version: VERSION,
    copy: migrateCopyBuffer(copy),
    schematics: Object.fromEntries(
      Object.entries(schematics).map(([key, schematic]) => [
        key,
        stateMigration(schematic),
      ]),
    ),
  }),
});
