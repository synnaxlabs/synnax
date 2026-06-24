// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, lineplot } from "@synnaxlabs/client";
import { Access, axis, telem, Viewport } from "@synnaxlabs/pluto";
import { lineplot as etherLineplot } from "@synnaxlabs/pluto/ether";
import {
  bounds,
  box,
  color,
  dimensions,
  direction,
  migrate,
  sticky,
  text,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type Import } from "@/import";
import { create, LAYOUT_TYPE } from "@/layered/service/lineplot/layout";

const STATE_MIGRATION_NAME = "lineplot.state";

// These schemas are a frozen snapshot of the line plot console-state shapes that shipped
// at each state version. They drive file-import migrations only; the live document body
// lives on the server (lineplot.linePlotZ). Do NOT collapse them onto live Pluto/oracle
// types — those drift with refactors and would reject or silently drop real historical
// persisted state.

const V0_VERSION = "0.0.0";

const titleStateZ = z.object({ level: text.levelZ, visible: z.boolean() });
const legendStateZ = z.object({ visible: z.boolean() });

const viewportStateZ = z.object({
  renderTrigger: z.number(),
  zoom: dimensions.dimensionsZ,
  pan: xy.xyZ,
});

const selectionStateZ = z.object({ box: box.box });

const v0AxisStateZ = z.object({
  key: lineplot.axisKeyZ,
  label: z.string(),
  labelDirection: direction.directionZ,
  bounds: bounds.boundsZ(),
  autoBounds: z.object({ lower: z.boolean(), upper: z.boolean() }),
  tickSpacing: z.number(),
  labelLevel: text.levelZ,
});

const v0AxesStateZ = z.object({
  renderTrigger: z.number(),
  hasHadChannelSet: z.boolean(),
  axes: z.object({
    y1: v0AxisStateZ,
    y2: v0AxisStateZ,
    y3: v0AxisStateZ,
    y4: v0AxisStateZ,
    x1: v0AxisStateZ,
    x2: v0AxisStateZ,
  }),
});

const lineStateZ = z.object({
  key: z.string(),
  label: z.string().optional(),
  color: z.string(),
  strokeWidth: z.number(),
  downsample: z.number(),
  downsampleMode: telem.downsampleModeZ.default("decimate"),
});

const ruleStateZ = z.object({
  selected: z.boolean().optional(),
  key: z.string(),
  label: z.string(),
  color: z.string(),
  axis: lineplot.axisKeyZ,
  lineWidth: z.number(),
  lineDash: z.number(),
  units: z.string(),
  position: z.number(),
});

const channelsStateZ = z.object({
  x1: z.number(),
  x2: z.number(),
  y1: z.array(z.number()),
  y2: z.array(z.number()),
  y3: z.array(z.number()),
  y4: z.array(z.number()),
});

const rangesStateZ = z.object({
  x1: z.array(z.string()),
  x2: z.array(z.string()),
});

const v0StateZ = z.object({
  version: z.literal(V0_VERSION),
  key: z.string(),
  remoteCreated: z.boolean(),
  title: titleStateZ,
  legend: legendStateZ,
  channels: channelsStateZ,
  ranges: rangesStateZ,
  viewport: viewportStateZ,
  axes: v0AxesStateZ,
  lines: z.array(lineStateZ),
  rules: z.array(ruleStateZ),
  selection: selectionStateZ,
});
interface V0State extends z.infer<typeof v0StateZ> {}

const controlStateZ = z.object({
  hold: z.boolean(),
  clickMode: z.enum(["annotate", "measure"]).nullable(),
  enableTooltip: z.boolean(),
});
const ZERO_CONTROL_STATE: z.infer<typeof controlStateZ> = {
  clickMode: null,
  hold: false,
  enableTooltip: true,
};

const toolbarStateZ = z.object({
  activeTab: z.enum(["data", "lines", "axes", "annotations", "properties"]),
});
const ZERO_TOOLBAR_STATE: z.infer<typeof toolbarStateZ> = { activeTab: "data" };

const V1_VERSION = "1.0.0";

const v1LegendStateZ = legendStateZ.extend({ position: sticky.xyZ });
const v1ZeroLegendState: z.infer<typeof v1LegendStateZ> = {
  visible: true,
  position: {
    x: 50,
    y: 50,
    root: { x: "left", y: "top" },
    units: { x: "px", y: "px" },
  },
};

const v1StateZ = v0StateZ
  .omit({ legend: true, version: true })
  .extend({ version: z.literal(V1_VERSION), legend: v1LegendStateZ });
interface V1State extends z.infer<typeof v1StateZ> {}

const v1StateMigration = migrate.createMigration<V0State, V1State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({ ...state, version: V1_VERSION, legend: v1ZeroLegendState }),
});

const V2_VERSION = "2.0.0";

const v2AxisStateZ = v0AxisStateZ.extend({ type: axis.tickType.optional() });
const v2AxesStateZ = v0AxesStateZ.omit({ axes: true }).extend({
  axes: z.object({
    y1: v2AxisStateZ,
    y2: v2AxisStateZ,
    y3: v2AxisStateZ,
    y4: v2AxisStateZ,
    x1: v2AxisStateZ,
    x2: v2AxisStateZ,
  }),
});

const v2StateZ = v1StateZ
  .omit({ axes: true, version: true })
  .extend({ version: z.literal(V2_VERSION), axes: v2AxesStateZ });
interface V2State extends z.infer<typeof v2StateZ> {}

const v2StateMigration = migrate.createMigration<V1State, V2State>({
  name: STATE_MIGRATION_NAME,
  migrate: ({ axes, ...rest }) => ({
    ...rest,
    version: V2_VERSION,
    axes: {
      ...axes,
      axes: Object.fromEntries(
        Object.entries(axes.axes).map(([key, ax]) => {
          if (key.startsWith("x")) return [key, { ...ax, type: "time" }];
          return [key, { ...ax, labelDirection: "y" }];
        }),
      ) as V2State["axes"]["axes"],
    },
  }),
});

const V3_VERSION = "3.0.0";

const v3StateZ = v2StateZ.omit({ version: true }).extend({
  version: z.literal(V3_VERSION),
  mode: Viewport.modeZ,
  control: controlStateZ,
  toolbar: toolbarStateZ,
});
interface V3State extends z.infer<typeof v3StateZ> {}

const v3StateMigration = migrate.createMigration<V2State, V3State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...state,
    version: V3_VERSION,
    mode: "zoom",
    control: ZERO_CONTROL_STATE,
    toolbar: ZERO_TOOLBAR_STATE,
  }),
});

const V4_VERSION = "4.0.0";

const v4MeasureStateZ = z.object({ mode: etherLineplot.measure.modeZ });
const v4AnnotationsStateZ = z.object({ visible: z.boolean().default(true) });
const ZERO_MEASURE_STATE: z.infer<typeof v4MeasureStateZ> = { mode: "one" };
const ZERO_ANNOTATIONS_STATE: z.infer<typeof v4AnnotationsStateZ> = { visible: true };

const v4StateZ = v3StateZ.omit({ version: true }).extend({
  version: z.literal(V4_VERSION),
  measure: v4MeasureStateZ,
  annotations: v4AnnotationsStateZ.default(ZERO_ANNOTATIONS_STATE),
});
interface V4State extends z.infer<typeof v4StateZ> {}

const v4StateMigration = migrate.createMigration<V3State, V4State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...state,
    version: V4_VERSION,
    measure: ZERO_MEASURE_STATE,
    annotations: ZERO_ANNOTATIONS_STATE,
  }),
});

const V5_VERSION = "5.0.0";

// toColor lifts a legacy hex-string color into the optional Color the oracle types use.
// The empty string was the legacy "unset" sentinel and maps to undefined so the Console
// assigns a palette/default at render time.
const toColor = (hex: string): color.Color | undefined =>
  hex === "" ? undefined : color.construct(hex);

// pendingUploadZ stages a plot's body on the client until it has landed on the server.
const pendingUploadZ = lineplot.linePlotZ.omit({ name: true }).partial();
interface PendingUpload extends z.infer<typeof pendingUploadZ> {}

const v5StateZ = v4StateZ
  .omit({
    version: true,
    title: true,
    legend: true,
    channels: true,
    ranges: true,
    axes: true,
    lines: true,
    rules: true,
  })
  .extend({
    version: z.literal(V5_VERSION),
    selectedRules: z.array(z.string()).default([]),
    hiddenLines: z.array(z.string()).default([]),
    pendingUpload: pendingUploadZ.optional(),
  });
interface V5State extends z.infer<typeof v5StateZ> {}

// buildPendingUpload projects v4 body fields into the oracle-typed PendingUpload shape.
// Unwraps the v4 axes render-trigger wrapper and drops the per-rule `selected` flag
// (lifted to selectedRules).
const buildPendingUpload = (state: V4State): PendingUpload => ({
  key: state.key,
  title: state.title,
  legend: { position: state.legend.position, hidden: !state.legend.visible },
  channels: state.channels,
  ranges: state.ranges,
  axes: Object.fromEntries(
    Object.entries(state.axes.axes).map(([k, { autoBounds, ...ax }]) => [
      k,
      { ...ax, manualBounds: { lower: !autoBounds.lower, upper: !autoBounds.upper } },
    ]),
  ) as PendingUpload["axes"],
  lines: state.lines.map((l) => ({ ...l, color: toColor(l.color) })),
  rules: state.rules.map(({ selected: _selected, ...rest }) => ({
    ...rest,
    color: toColor(rest.color),
  })),
});

// v5StateMigration drops the v4 body fields. remoteCreated plots are authoritative on
// the server already, so pendingUpload stays undefined. Non-remoteCreated plots are
// projected into pendingUpload. Selection on rules lifts from a per-rule `selected` flag
// to a per-plot selectedRules array.
const v5StateMigration = migrate.createMigration<V4State, V5State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => {
    const selectedRules = state.rules
      .filter((r) => r.selected === true)
      .map((r) => r.key);
    const {
      title: _title,
      legend: _legend,
      channels: _channels,
      ranges: _ranges,
      axes: _axes,
      lines: _lines,
      rules: _rules,
      ...rest
    } = state;
    return {
      ...rest,
      version: V5_VERSION,
      selectedRules,
      hiddenLines: [],
      pendingUpload: state.remoteCreated ? undefined : buildPendingUpload(state),
    };
  },
});

type AnyState = V0State | V1State | V2State | V3State | V4State | V5State;

const ZERO_STATE: V5State = {
  version: V5_VERSION,
  key: "",
  remoteCreated: false,
  viewport: { renderTrigger: 0, zoom: dimensions.DECIMAL, pan: xy.ZERO },
  selection: { box: box.ZERO },
  mode: "zoom",
  control: ZERO_CONTROL_STATE,
  toolbar: ZERO_TOOLBAR_STATE,
  measure: ZERO_MEASURE_STATE,
  annotations: ZERO_ANNOTATIONS_STATE,
  selectedRules: [],
  hiddenLines: [],
  pendingUpload: undefined,
};

const STATE_MIGRATIONS: migrate.Migrations = {
  [V0_VERSION]: v1StateMigration,
  [V1_VERSION]: v2StateMigration,
  [V2_VERSION]: v3StateMigration,
  [V3_VERSION]: v4StateMigration,
  [V4_VERSION]: v5StateMigration,
};

const migrateState = migrate.migrator<AnyState, V5State>({
  name: STATE_MIGRATION_NAME,
  migrations: STATE_MIGRATIONS,
  def: ZERO_STATE,
});

export const anyStateZ = z
  .union([v5StateZ, v4StateZ, v3StateZ, v2StateZ, v1StateZ, v0StateZ])
  .transform((state) => migrateState(state));

export const parseImport = (
  data: unknown,
  fallbackName: string | undefined,
): lineplot.New => {
  // Legacy console-state exports are tried first because their schemas are strict: they
  // require a version literal plus console-only fields (viewport, selection) and a
  // wrapped axes object, so a current typed export never matches and falls through to
  // the direct branch. The typed linePlotZ is the opposite — it strips unknown keys and
  // defaults lines/rules to empty, so trying it first risks silently accepting a legacy
  // file and dropping its body.
  if (typeof data === "object" && data != null) {
    const legacy = anyStateZ.safeParse({ ...data, remoteCreated: false });
    if (legacy.success) {
      if (legacy.data.pendingUpload == null)
        throw new Error("Imported line plot has no body data");
      const { key: _key, ...body } = legacy.data.pendingUpload;
      return { ...body, name: fallbackName ?? "" };
    }
  }
  const { key: _key, ...rest } = lineplot.linePlotZ.parse(data);
  return { ...rest, name: fallbackName ?? rest.name };
};

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, projectKey },
) => {
  if (!Access.updateGranted({ id: lineplot.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import line plots");
  if (client == null) throw new DisconnectedError();
  const newPayload = parseImport(data, layout?.name);
  const created = await client.lineplots.create(projectKey, newPayload);
  store.lineplots.set(created.key, created);
  placeLayout(create({ ...layout, key: created.key, name: created.name }));
};

export const FILE_INGESTERS: Import.FileIngesters = { [LAYOUT_TYPE]: ingest };
