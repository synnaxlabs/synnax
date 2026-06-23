// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, schematic } from "@synnaxlabs/client";
import { Access, control, Flex, Schematic, Value, Viewport } from "@synnaxlabs/pluto";
import {
  color,
  control as xcontrol,
  dimensions,
  direction,
  location,
  migrate,
  record,
  sticky,
  text,
  uuid,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type Import } from "@/import";
import { create, LAYOUT_TYPE } from "@/layered/service/schematic/layout";

const STATE_MIGRATION_NAME = "schematic.state";

const V0_VERSION = "0.0.0";

// These schemas are a frozen snapshot of the diagram node/edge/viewport shapes
// that shipped at state version 0.0.0. Do NOT replace with Diagram.nodeZ /
// Diagram.edgeZ / Diagram.viewportZ — those references drift with Pluto
// refactors (the diagram edge reshape on this branch turned `source` from a
// string into `{node, param}`) and cause migration parsing to reject or drop
// real historical persisted state.
const v0NodeZ = z.looseObject({
  key: z.string(),
  position: xy.xyZ,
  zIndex: z.number().optional(),
  type: z.string().optional(),
  measured: dimensions.dimensionsZ.optional(),
});
interface V0Node extends z.infer<typeof v0NodeZ> {}

const v0EdgeZ = z.looseObject({
  key: z.string(),
  source: z.string(),
  target: z.string(),
  sourceHandle: z.string().nullable().optional(),
  targetHandle: z.string().nullable().optional(),
});
interface V0Edge extends z.infer<typeof v0EdgeZ> {}

const v0ViewportZ = z.object({ position: xy.xyZ, zoom: z.number() });

const v0LabelZ = z.looseObject({
  label: z.string().optional(),
  level: text.levelZ.optional(),
  orientation: location.locationZ.optional(),
  direction: direction.directionZ.optional(),
  maxInlineSize: z.number().optional(),
  align: Flex.alignmentZ.optional(),
});

const v0NodePropsZ = z.looseObject({
  key: Schematic.Node.variantZ,
  color: color.crudeZ.optional(),
  label: v0LabelZ.optional(),
});
interface V0NodeProps extends z.infer<typeof v0NodePropsZ> {}

const v0StateZ = z.object({
  version: z.literal(V0_VERSION),
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  snapshot: z.boolean(),
  remoteCreated: z.boolean(),
  viewport: v0ViewportZ,
  nodes: z
    .array(z.unknown())
    .transform((nodes) => nodes.filter((node) => v0NodeZ.safeParse(node).success))
    .pipe(z.array(v0NodeZ)),
  edges: z
    .array(z.unknown())
    .transform((edges) => edges.filter((edge) => v0EdgeZ.safeParse(edge).success))
    .pipe(z.array(v0EdgeZ)),
  props: z.record(z.string(), v0NodePropsZ).transform((p) => {
    for (const k in p)
      if (p[k].key === "value") {
        p[k].redline = Value.ZERO_READLINE;
        p[k].stalenessTimeout = 5;
        p[k].stalenessColor = color.ZERO;
      }
    return p;
  }),
  control: control.statusZ,
});
interface V0State extends z.infer<typeof v0StateZ> {}

const v0ToolbarTabZ = z.enum(["symbols", "properties"]);

const v0ToolbarStateZ = z.object({
  activeTab: v0ToolbarTabZ,
  selectedSymbolGroup: z.string().default("general"),
});
interface V0ToolbarState extends z.infer<typeof v0ToolbarStateZ> {}
const v0ZeroToolbarState: V0ToolbarState = {
  activeTab: "symbols",
  selectedSymbolGroup: "general",
};

const V1_VERSION = "1.0.0";

const v1LegendStateZ = z.object({
  visible: z.boolean(),
  position: sticky.xyZ,
  colors: z.record(z.string(), z.string()).default({}),
});
interface V1LegendState extends z.infer<typeof v1LegendStateZ> {}
const v1ZeroLegendState: V1LegendState = {
  visible: true,
  position: {
    x: 50,
    y: 50,
    units: { x: "px", y: "px" },
    root: { x: "left", y: "top" },
  },
  colors: {},
};

const v1StateZ = v0StateZ
  .omit({ version: true })
  .extend({ version: z.literal(V1_VERSION), legend: v1LegendStateZ });
interface V1State extends z.infer<typeof v1StateZ> {}

const v1StateMigration = migrate.createMigration<V0State, V1State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({ ...state, legend: v1ZeroLegendState, version: V1_VERSION }),
});

const V2_VERSION = "2.0.0";

const v2StateZ = v1StateZ.omit({ version: true }).extend({
  version: z.literal(V2_VERSION),
  key: z.string(),
  type: z.literal("schematic"),
  viewportMode: Viewport.modeZ.default("select"),
});
interface V2State extends z.infer<typeof v2StateZ> {}

const v2StateMigration = migrate.createMigration<V1State, V2State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...state,
    version: V2_VERSION,
    key: uuid.create(),
    type: "schematic",
    viewportMode: "select",
  }),
});

const V3_VERSION = "3.0.0";

const v3StateZ = v2StateZ
  .omit({ version: true })
  .extend({ version: z.literal(V3_VERSION) });
interface V3State extends z.infer<typeof v3StateZ> {}

const v3StateMigration = migrate.createMigration<V2State, V3State>({
  name: STATE_MIGRATION_NAME,
  migrate: ({ edges, ...rest }) => ({
    ...rest,
    edges: edges.map((edge) => ({ ...edge, segments: [] })),
    version: V3_VERSION,
  }),
});

const V4_VERSION = "4.0.0";

const v4StateZ = v3StateZ
  .omit({ version: true })
  .extend({ version: z.literal(V4_VERSION), authority: z.number() });
interface V4State extends z.infer<typeof v4StateZ> {}

const v4StateMigration = migrate.createMigration<V3State, V4State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({ ...state, version: V4_VERSION, authority: 1 }),
});

const V5_VERSION = "5.0.0";

const v5StateZ = v4StateZ.omit({ version: true, type: true }).extend({
  version: z.literal(V5_VERSION),
  mode: Viewport.modeZ,
  toolbar: v0ToolbarStateZ,
});
interface V5State extends z.infer<typeof v5StateZ> {}

const v5StateMigration = migrate.createMigration<V4State, V5State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...record.omit(state, "type"),
    version: V5_VERSION,
    mode: "select",
    toolbar: { ...v0ZeroToolbarState },
  }),
});

const V6_VERSION = "6.0.0";

const v6ViewportZ = z.object({
  position: xy.xyZ,
  zoom: z.number(),
  mode: Viewport.modeZ.default("select"),
});

type V6Node = schematic.Node;
type V6Edge = schematic.Edge;

const v6ElementConfigZ = Schematic.elementConfigZ;
type V6EdgeConfig = Schematic.Edge.Config;
type V6ElementConfig = Schematic.ElementConfig;

const v6LegendStateZ = z.object({
  visible: z.boolean(),
  position: sticky.xyZ,
  colors: z.record(z.string(), color.colorZ).default({}),
});
interface V6LegendState extends z.infer<typeof v6LegendStateZ> {}

const v6PendingUploadZ = schematic.schematicZ
  .omit({ configs: true, name: true })
  .extend({ configs: z.record(z.string(), v6ElementConfigZ) });
interface V6PendingUpload extends z.infer<typeof v6PendingUploadZ> {}

const v6StateZ = z.object({
  version: z.literal(V6_VERSION),
  selected: z.array(z.string()).default([]),
  controlStatus: control.statusZ,
  authority: xcontrol.authorityZ,
  legend: v6LegendStateZ,
  toolbar: v0ToolbarStateZ,
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  viewport: v6ViewportZ,
  pendingUpload: v6PendingUploadZ.optional(),
});
interface V6State extends z.infer<typeof v6StateZ> {}
const v6ZeroState: V6State = {
  version: V6_VERSION,
  selected: [],
  controlStatus: "released",
  authority: 1,
  legend: {
    visible: true,
    position: {
      x: 50,
      y: 50,
      units: { x: "px", y: "px" },
      root: { x: "left", y: "top" },
    },
    colors: {},
  },
  toolbar: { activeTab: "symbols", selectedSymbolGroup: "general" },
  editable: false,
  fitViewOnResize: false,
  viewport: { position: xy.ZERO, zoom: 1, mode: "select" },
  pendingUpload: undefined,
};

const migrateNode = (node: V0Node): V6Node => ({
  key: node.key,
  position: node.position,
  zIndex: node.zIndex ?? 0,
});

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

const migrateEdge = (edge: V0Edge): [V6Edge, V6EdgeConfig] => {
  const next: V6Edge = {
    key: edge.key,
    source: { node: edge.source, param: edge.sourceHandle ?? "" },
    target: { node: edge.target, param: edge.targetHandle ?? "" },
  };
  const edgeConfig: V6EdgeConfig = {
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
): V6LegendState["colors"] => {
  if (colors == null) return {};
  const out: V6LegendState["colors"] = {};
  for (const [k, v] of Object.entries(colors)) {
    const parsed = color.colorZ.safeParse(v);
    if (parsed.success) out[k] = parsed.data;
  }
  return out;
};

const migratePropsToConfigs = (
  props: Record<string, V0NodeProps>,
): Record<string, V6ElementConfig> =>
  Object.fromEntries(
    Object.entries(props).map(([k, p]) => {
      const { key, ...rest } = p as V0NodeProps & Record<string, unknown>;
      return [k, { ...rest, variant: key } as V6ElementConfig];
    }),
  );

const buildPendingUpload = (state: V5State): V6PendingUpload => {
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

const v6StateMigration = migrate.createMigration<V5State, V6State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    version: V6_VERSION,
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

type AnyState = V0State | V1State | V2State | V3State | V4State | V5State | V6State;

const ZERO_STATE = v6ZeroState;

const STATE_MIGRATIONS: migrate.Migrations = {
  [V0_VERSION]: v1StateMigration,
  [V1_VERSION]: v2StateMigration,
  [V2_VERSION]: v3StateMigration,
  [V3_VERSION]: v4StateMigration,
  [V4_VERSION]: v5StateMigration,
  [V5_VERSION]: v6StateMigration,
};

const migrateState = migrate.migrator<AnyState, V6State>({
  name: STATE_MIGRATION_NAME,
  migrations: STATE_MIGRATIONS,
  def: ZERO_STATE,
});

export const anyStateZ = z
  .union([v6StateZ, v5StateZ, v4StateZ, v3StateZ, v2StateZ, v1StateZ, v0StateZ])
  .transform((state) => migrateState(state));

export const parseImport = (
  data: unknown,
  fallbackName: string | undefined,
): schematic.New => {
  // Legacy console-state exports are tried first because their schemas are strict: they
  // require a version literal plus the console-only editable / control / viewport /
  // props fields, so a current typed export never matches and falls through to the
  // direct branch. The typed schematicZ is the opposite — it strips unknown keys and
  // defaults configs to {}, so trying it first silently accepts a legacy file, drops
  // its props, and yields a schematic with no symbol configs (a blank import).
  if (typeof data === "object" && data != null) {
    const legacy = anyStateZ.safeParse({ ...data, remoteCreated: false });
    if (legacy.success) {
      if (legacy.data.pendingUpload == null)
        throw new Error("Imported schematic has no graph data");
      const { snapshot, nodes, edges, configs } = legacy.data.pendingUpload;
      return { name: fallbackName ?? "Schematic", snapshot, nodes, edges, configs };
    }
  }
  const { key: _key, ...rest } = schematic.schematicZ.parse(data);
  return { ...rest, name: fallbackName ?? rest.name };
};

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, projectKey },
) => {
  if (!Access.updateGranted({ id: schematic.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import schematics");
  if (client == null) throw new DisconnectedError();
  const newPayload = parseImport(data, layout?.name);
  const created = await client.schematics.create(projectKey, newPayload);
  const { key, name } = created;
  store.schematics.set(key, created);
  placeLayout(create({ ...layout, key, name, type: LAYOUT_TYPE }));
  return schematic.ontologyID(key);
};

export const FILE_INGESTERS: Import.FileIngesters = {
  schematic: ingest,
};
