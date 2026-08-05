// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, DisconnectedError } from "@synnaxlabs/client";
import { Access, Diagram } from "@synnaxlabs/pluto";
import { dimensions, migrate, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { type Import } from "@/platform/import";

const STATE_MIGRATION_NAME = "arc.state";

// These schemas are a frozen snapshot of the Arc console-state shapes that shipped at
// each state version. They drive file-import migrations only; the live document body
// lives on the server (arc.arcZ). Do NOT collapse them onto live Pluto/oracle types —
// those drift with refactors and would reject or silently drop real historical state.

const nodePropsZ = z.looseObject({ key: z.string() });
type NodeProps = z.infer<typeof nodePropsZ>;

// --- v0 -------------------------------------------------------------------------------

const V0_VERSION = "0.0.0";

const v0NodeZ = z.object({
  key: z.string(),
  position: xy.xyZ,
  selected: z.boolean().optional().default(false),
  zIndex: z.number().optional(),
  type: z.string().optional(),
  measured: dimensions.dimensionsZ.optional(),
});

const v0EdgeZ = z.object({
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

const v0GraphStateZ = z.object({
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  viewport: z.object({ position: xy.xyZ, zoom: z.number() }),
  selected: z.array(z.string()).default([]),
  nodes: z
    .array(z.any())
    .transform((nodes) => nodes.filter((node) => v0NodeZ.safeParse(node).success))
    .pipe(z.array(v0NodeZ)),
  edges: z
    .array(z.any())
    .transform((edges) => edges.filter((edge) => v0EdgeZ.safeParse(edge).success))
    .pipe(z.array(v0EdgeZ)),
  props: z.record(z.string(), nodePropsZ),
});

const v0StateZ = z.object({
  key: z.string(),
  version: z.literal(V0_VERSION),
  remoteCreated: z.boolean(),
  graph: v0GraphStateZ,
  text: z.object({ raw: z.string() }),
  mode: arc.modeZ.default("graph"),
});
type V0State = z.infer<typeof v0StateZ>;

// --- v1 -------------------------------------------------------------------------------

const V1_VERSION = "1.0.0";

const v1GraphStateZ = z.object({
  editable: z.boolean(),
  fitViewOnResize: z.boolean(),
  viewport: Diagram.viewportZ,
  selected: z.array(z.string()).default([]),
  nodes: z.array(Diagram.nodeZ),
  edges: z.array(Diagram.edgeZ),
  props: z.record(z.string(), nodePropsZ),
});

const v1StateZ = z.object({
  key: z.string(),
  version: z.literal(V1_VERSION),
  remoteCreated: z.boolean(),
  graph: v1GraphStateZ,
  text: z.object({ raw: z.string() }),
  mode: arc.modeZ.default("graph"),
});
type V1State = z.infer<typeof v1StateZ>;

// --- v2 -------------------------------------------------------------------------------

const V2_VERSION = "2.0.0";

const v2StateZ = z.object({
  key: z.string(),
  version: z.literal(V2_VERSION),
  remoteCreated: z.boolean(),
  graph: v1GraphStateZ,
  mode: arc.modeZ.default("graph"),
  text: z.object({ raw: z.string() }),
});
type V2State = z.infer<typeof v2StateZ>;

// --- v3 (latest) ----------------------------------------------------------------------

const V3_VERSION = "3.0.0";

const v3PendingUploadZ = z.object({
  mode: arc.modeZ,
  graph: arc.graph.graphZ,
  text: arc.text.textZ,
});

const v3StateZ = z.object({
  key: z.string(),
  version: z.literal(V3_VERSION),
  graph: z.object({
    editable: z.boolean(),
    fitViewOnResize: z.boolean(),
    viewport: Diagram.viewportZ,
    selected: z.array(z.string()).default([]),
  }),
  pendingUpload: v3PendingUploadZ.optional(),
});
type V3State = z.infer<typeof v3StateZ>;

// --- migrations -----------------------------------------------------------------------

const migrateV0Edge = (edge: z.infer<typeof v0EdgeZ>): Diagram.Edge => ({
  key: edge.key,
  source: { node: edge.source, param: edge.sourceHandle ?? "" },
  target: { node: edge.target, param: edge.targetHandle ?? "" },
});

const v1StateMigration = migrate.createMigration<V0State, V1State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => {
    const selected = [
      ...state.graph.nodes.filter((n) => n.selected).map((n) => n.key),
      ...state.graph.edges.filter((e) => e.selected).map((e) => e.key),
      ...state.graph.selected,
    ];
    return {
      ...state,
      version: V1_VERSION,
      graph: {
        editable: state.graph.editable,
        fitViewOnResize: state.graph.fitViewOnResize,
        viewport: state.graph.viewport,
        selected: [...new Set(selected)],
        nodes: state.graph.nodes.map((n) => ({
          key: n.key,
          position: n.position,
          zIndex: n.zIndex,
          type: n.type,
          measured: n.measured,
        })),
        edges: state.graph.edges.map(migrateV0Edge),
        props: state.graph.props,
      },
    };
  },
});

const legacySetStatusZ = z.looseObject({
  statusKey: z.string().optional(),
  variant: z.string().optional(),
  message: z.string().optional(),
});

// Rewrites the deprecated set_status node to status.set. The bare set_status STL symbol
// was removed in SY-4122; saved graphs must remap the props.
const rewriteSetStatusProp = (p: NodeProps): NodeProps => {
  if (p.key !== "set_status") return p;
  const legacy = legacySetStatusZ.parse(p);
  return {
    key: "status.set",
    key_or_name: legacy.statusKey ?? "",
    variant: legacy.variant ?? "success",
    message: legacy.message ?? "",
  };
};

const v2StateMigration = migrate.createMigration<V1State, V2State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...state,
    version: V2_VERSION,
    graph: {
      ...state.graph,
      props: Object.fromEntries(
        Object.entries(state.graph.props).map(([k, v]) => [k, rewriteSetStatusProp(v)]),
      ),
    },
  }),
});

// buildPendingGraph converts a legacy v2 redux graph into a flux graph document, lifting
// each node's inline props (with the function type under "key") into the inputs map
// (under "type"), keyed by node key.
const buildPendingGraph = (state: V2State): arc.graph.Graph => ({
  nodes: state.graph.nodes.map((n) => ({ key: n.key, position: n.position })),
  edges: state.graph.edges.map((e) => ({
    key: e.key,
    source: e.source,
    target: e.target,
    kind: arc.ir.EdgeKind.continuous,
  })),
  inputs: Object.fromEntries(
    Object.entries(state.graph.props).map(([k, { key, ...rest }]) => [
      k,
      { ...rest, type: key },
    ]),
  ),
  functions: [],
});

const v3StateMigration = migrate.createMigration<V2State, V3State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    key: state.key,
    version: V3_VERSION,
    graph: {
      editable: state.graph.editable,
      fitViewOnResize: state.graph.fitViewOnResize,
      viewport: state.graph.viewport,
      selected: state.graph.selected,
    },
    pendingUpload: state.remoteCreated
      ? undefined
      : {
          graph: buildPendingGraph(state),
          text: arc.text.textZ.parse({ raw: state.text.raw }),
          mode: state.mode,
        },
  }),
});

type AnyState = V0State | V1State | V2State | V3State;

const ZERO_STATE: V3State = {
  key: "",
  version: V3_VERSION,
  graph: {
    editable: true,
    fitViewOnResize: false,
    viewport: { position: xy.ZERO, zoom: 1 },
    selected: [],
  },
  pendingUpload: undefined,
};

const STATE_MIGRATIONS: migrate.Migrations = {
  [V0_VERSION]: v1StateMigration,
  [V1_VERSION]: v2StateMigration,
  [V2_VERSION]: v3StateMigration,
};

const migrateState = migrate.migrator<AnyState, V3State>({
  name: STATE_MIGRATION_NAME,
  migrations: STATE_MIGRATIONS,
  def: ZERO_STATE,
});

export const anyStateZ = z
  .union([v3StateZ, v2StateZ, v1StateZ, v0StateZ])
  .transform((state) => migrateState(state));

// parseImport resolves an imported file into an Arc create payload. Legacy console-state
// exports are tried first: they carry the graph parked under pendingUpload by the state
// migration. A current export is the typed Arc itself and falls through to arcZ.
export const parseImport = (data: unknown, fallbackName?: string): arc.New => {
  if (typeof data === "object" && data != null) {
    const legacy = anyStateZ.safeParse({ ...data, remoteCreated: false });
    if (legacy.success) {
      if (legacy.data.pendingUpload == null)
        throw new Error("Imported Arc has no graph data");
      return {
        name: fallbackName ?? "Arc",
        mode: legacy.data.pendingUpload.mode ?? "graph",
        graph: legacy.data.pendingUpload.graph,
        text: legacy.data.pendingUpload.text,
      };
    }
  }
  const { key: _key, ...rest } = arc.arcZ.parse(data);
  return { ...rest, name: fallbackName ?? rest.name };
};

export const ingest: Import.FileIngester = async (data, { name, openTab, client }) => {
  if (!Access.updateGranted({ id: arc.TYPE_ONTOLOGY_ID, client }))
    throw new Error("You do not have permission to import Arc automations");
  if (client == null) throw new DisconnectedError();
  const newPayload = parseImport(data, name);
  const created = await client.arcs.create(newPayload);
  openTab({ variant: "resource", resource: arc.ontologyID(created.key) });
  return arc.ontologyID(created.key);
};
