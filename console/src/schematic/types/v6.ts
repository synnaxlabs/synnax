// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Diagram, Schematic } from "@synnaxlabs/pluto";
import { color, migrate, record } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/schematic/types/v0";
import * as v1 from "@/schematic/types/v1";
import * as v5 from "@/schematic/types/v5";

export const VERSION = "6.0.0";

export const nodePropsZ = v0.nodePropsZ.omit({ key: true }).extend({
  type: z.literal("node"),
  variant: Schematic.Symbol.variantZ,
});
export interface NodeProps extends z.infer<typeof nodePropsZ> {}
export const edgePropsZ = z.object({
  type: z.literal("edge"),
  color: color.colorZ,
  segments: z.array(Schematic.Edge.connector.segmentZ),
  variant: Schematic.Edge.variantZ,
});
export interface EdgeProps extends z.infer<typeof edgePropsZ> {}

export const propsZ = z.discriminatedUnion("type", [nodePropsZ, edgePropsZ]);
export type Props = z.infer<typeof propsZ>;

export const legendStateZ = v1.legendStateZ
  .omit({ colors: true })
  .extend({ colors: z.record(z.string(), color.colorZ).default({}) });
export interface LegendState extends z.infer<typeof legendStateZ> {}
const ZERO_LEGEND_STATE: LegendState = {
  visible: true,
  position: { x: 50, y: 50, units: { x: "px", y: "px" } },
  colors: {},
};

export const stateZ = v5.stateZ
  .omit({ version: true, nodes: true, edges: true, props: true, legend: true })
  .extend({
    version: z.literal(VERSION),
    nodes: z.array(Diagram.nodeZ),
    edges: z.array(Diagram.edgeZ),
    props: z.record(z.string(), propsZ),
    legend: legendStateZ,
    selected: z.array(z.string()).default([]),
  });
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...v5.ZERO_STATE,
  version: VERSION,
  nodes: [],
  edges: [],
  props: {},
  legend: ZERO_LEGEND_STATE,
  selected: [],
};

export const copyBufferZ = z.object({
  pos: v0.copyBufferZ.shape.pos,
  nodes: z.array(Diagram.nodeZ),
  edges: z.array(Diagram.edgeZ),
  props: z.record(z.string(), propsZ),
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

const migrateEdge = (edge: v0.Edge): [Diagram.Edge, EdgeProps] => {
  const next: Diagram.Edge = {
    key: edge.key,
    source: { node: edge.source, param: edge.sourceHandle ?? "" },
    target: { node: edge.target, param: edge.targetHandle ?? "" },
  };
  const edgeProps: EdgeProps = {
    type: "edge",
    variant: "pipe",
    segments: [],
    color: color.ZERO,
  };
  const parseDataResult = record.unknownZ().safeParse(edge.data);
  if (!parseDataResult.success) return [next, edgeProps];
  const data = parseDataResult.data;
  const segments = z.array(Schematic.Edge.connector.segmentZ).safeParse(data.segments);
  if (segments.success) edgeProps.segments = segments.data;
  const parsedColor = color.colorZ.safeParse(data.color);
  if (parsedColor.success) edgeProps.color = parsedColor.data;
  const parsedVariant = Schematic.Edge.variantZ.safeParse(data.variant);
  if (parsedVariant.success) edgeProps.variant = parsedVariant.data;
  return [next, edgeProps];
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
    const edges = state.edges.map((e) => {
      const [edge, edgeProps] = migrateEdge(e);
      props[edge.key] = edgeProps;
      return edge;
    });
    return {
      ...state,
      version: VERSION,
      edges,
      props,
      legend: { ...state.legend, colors: migrateLegendColors(state.legend?.colors) },
      selected: [],
    };
  },
});

const migrateCopyBuffer = (copy: v5.SliceState["copy"]): CopyBuffer => {
  const props = migrateProps(copy.props);
  const edges = copy.edges.map((e) => {
    const [edge, edgeProps] = migrateEdge(e);
    props[edge.key] = edgeProps;
    return edge;
  });
  return { ...copy, edges, props };
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
