// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot as clientLineplot, type lineplot } from "@synnaxlabs/client";
import { expectLive } from "@synnaxlabs/client/testutil";
import { box, dimensions, xy } from "@synnaxlabs/x";
import { assert, describe, expect, it, vi } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { client, project } from "@/feature/lineplot/testutil";
import { type Panel } from "@/platform/panel";
import { awaitGranted, uniqueName } from "@/testutil";

const zeroAxis = (key: string) => ({
  key,
  label: "",
  labelDirection: key.startsWith("x") ? "x" : "y",
  bounds: { lower: 0, upper: 0 },
  autoBounds: { lower: true, upper: true },
  tickSpacing: 75,
  labelLevel: "small",
});

const AXES = {
  renderTrigger: 0,
  hasHadChannelSet: false,
  axes: {
    y1: zeroAxis("y1"),
    y2: zeroAxis("y2"),
    y3: zeroAxis("y3"),
    y4: zeroAxis("y4"),
    x1: zeroAxis("x1"),
    x2: zeroAxis("x2"),
  },
};

const V0_ZERO = {
  version: "0.0.0",
  key: "88aee41e-53b7-4a76-9df9-aceccc220089",
  remoteCreated: false,
  title: { level: "h4", visible: false },
  legend: { visible: true },
  channels: { x1: 0, x2: 0, y1: [], y2: [], y3: [], y4: [] },
  ranges: { x1: [], x2: [] },
  viewport: { renderTrigger: 0, zoom: { width: 1, height: 1 }, pan: { x: 0, y: 0 } },
  axes: AXES,
  lines: [],
  rules: [],
  selection: {
    box: {
      one: { x: 0, y: 0 },
      two: { x: 0, y: 0 },
      root: { x: "left", y: "top" },
    },
  },
};

const LEGEND_V1 = {
  visible: true,
  position: {
    x: 50,
    y: 50,
    root: { x: "left", y: "top" },
    units: { x: "px", y: "px" },
  },
};

const V1_ZERO = { ...V0_ZERO, version: "1.0.0", legend: LEGEND_V1 };
const V2_ZERO = { ...V1_ZERO, version: "2.0.0" };
const V3_ZERO = {
  ...V2_ZERO,
  version: "3.0.0",
  mode: "zoom",
  control: { hold: false, clickMode: null, enableTooltip: true },
  toolbar: { activeTab: "data" },
};
const V4_ZERO = {
  ...V3_ZERO,
  version: "4.0.0",
  measure: { mode: "one" },
  annotations: { visible: true },
};
const V5_ZERO = {
  key: V0_ZERO.key,
  remoteCreated: true,
  version: "5.0.0",
  viewport: V0_ZERO.viewport,
  selection: V0_ZERO.selection,
  mode: "zoom",
  control: { hold: false, clickMode: null, enableTooltip: true },
  toolbar: { activeTab: "data" },
  measure: { mode: "one" },
  annotations: { visible: true },
  selectedRules: [],
  hiddenLines: [],
};

describe("lineplot state migrations", () => {
  it.each([
    ["0.0.0", V0_ZERO],
    ["1.0.0", V1_ZERO],
    ["2.0.0", V2_ZERO],
    ["3.0.0", V3_ZERO],
    ["4.0.0", V4_ZERO],
    ["5.0.0", V5_ZERO],
  ])("should migrate state from %s to latest", (_version, state) => {
    const migrated = LinePlot.anyStateZ.parse(state);
    expect(migrated.version).toBe("5.0.0");
    expect(migrated.selectedRules).toEqual([]);
    expect(migrated.viewport).toBeDefined();
    expect(migrated.toolbar).toBeDefined();
  });

  it("should park v4 body into pendingUpload when remoteCreated is false", () => {
    const migrated = LinePlot.anyStateZ.parse({ ...V4_ZERO, remoteCreated: false });
    expect(migrated.pendingUpload).toBeDefined();
    expect(migrated.pendingUpload?.title).toBeDefined();
    expect(migrated.pendingUpload?.legend).toBeDefined();
    expect(migrated.pendingUpload?.axes).toBeDefined();
  });

  it("should leave pendingUpload undefined when remoteCreated is true", () => {
    const migrated = LinePlot.anyStateZ.parse({ ...V4_ZERO, remoteCreated: true });
    expect(migrated.pendingUpload).toBeUndefined();
  });

  it("should lift selected rules into selectedRules on the v4 to v5 migration", () => {
    const migrated = LinePlot.anyStateZ.parse({
      ...V4_ZERO,
      remoteCreated: true,
      rules: [
        {
          key: "r1",
          label: "",
          color: "",
          axis: "y1",
          lineWidth: 1,
          lineDash: 0,
          units: "",
          position: 0,
          selected: true,
        },
        {
          key: "r2",
          label: "",
          color: "",
          axis: "y1",
          lineWidth: 1,
          lineDash: 0,
          units: "",
          position: 0,
          selected: false,
        },
        {
          key: "r3",
          label: "",
          color: "",
          axis: "y1",
          lineWidth: 1,
          lineDash: 0,
          units: "",
          position: 0,
          selected: true,
        },
      ],
    });
    expect(migrated.selectedRules).toEqual(["r1", "r3"]);
  });
});

const KEY = "88aee41e-53b7-4a76-9df9-aceccc220089";
const LINE_KEY = "65538-88aee41e";

const zeroV0Axis = (key: string) => ({
  key,
  label: "",
  labelDirection: key.startsWith("x") ? "x" : "y",
  bounds: { lower: 0, upper: 0 },
  autoBounds: { lower: true, upper: true },
  tickSpacing: 75,
  labelLevel: "small",
});

// A legacy console export stores its body under the versioned console-state shape: a
// wrapped axes object, hex-string line colors, a version literal, and no top-level name.
// The strict legacy schemas must be tried before the lenient typed linePlotZ so the body
// (lines, channels) is migrated rather than silently dropped.
const LEGACY: Record<string, unknown> = {
  version: "0.0.0",
  key: KEY,
  remoteCreated: false,
  title: { level: "h4", visible: false },
  legend: { visible: true },
  channels: { x1: 0, x2: 0, y1: [65538], y2: [], y3: [], y4: [] },
  ranges: { x1: [], x2: [] },
  viewport: { renderTrigger: 0, zoom: { width: 1, height: 1 }, pan: { x: 0, y: 0 } },
  axes: {
    renderTrigger: 0,
    hasHadChannelSet: false,
    axes: {
      y1: zeroV0Axis("y1"),
      y2: zeroV0Axis("y2"),
      y3: zeroV0Axis("y3"),
      y4: zeroV0Axis("y4"),
      x1: zeroV0Axis("x1"),
      x2: zeroV0Axis("x2"),
    },
  },
  lines: [
    {
      key: LINE_KEY,
      color: "#FF0000",
      strokeWidth: 3,
      downsample: 1,
      downsampleMode: "decimate",
    },
  ],
  rules: [],
  selection: {
    box: { one: { x: 0, y: 0 }, two: { x: 0, y: 0 }, root: { x: "left", y: "top" } },
  },
};

const zeroTypedAxis = (key: lineplot.AxisKey): lineplot.Axis => ({
  key,
  label: "",
  labelDirection: key.startsWith("x") ? "x" : "y",
  labelLevel: "small",
  bounds: { lower: 0, upper: 0 },
  manualBounds: { lower: true, upper: true },
  tickSpacing: 0,
});

// A typed export is the server line plot the new export path writes, tagged with a layout
// type. It carries a name and a flat per-axis bundle the legacy schemas never match, so
// it parses directly.
const TYPED_EXPORT = {
  key: KEY,
  name: "My Plot",
  type: "lineplot",
  title: { level: "p", visible: false },
  legend: { hidden: true, position: { x: 0, y: 0 } },
  channels: { x1: 0, x2: 0, y1: [65538], y2: [], y3: [], y4: [] },
  ranges: { x1: [], x2: [] },
  axes: {
    x1: zeroTypedAxis("x1"),
    x2: zeroTypedAxis("x2"),
    y1: zeroTypedAxis("y1"),
    y2: zeroTypedAxis("y2"),
    y3: zeroTypedAxis("y3"),
    y4: zeroTypedAxis("y4"),
  },
  lines: [
    {
      key: LINE_KEY,
      color: "#FF0000",
      strokeWidth: 3,
      downsample: 1,
      downsampleMode: "decimate",
    },
  ],
  rules: [],
};

describe("lineplot import", () => {
  describe("parseImport", () => {
    it("should migrate a legacy console export, preserving lines and channels", () => {
      const out = LinePlot.parseImport(LEGACY, undefined);
      expect(out.lines).toHaveLength(1);
      expect(out.channels?.y1).toEqual([65538]);
    });

    it("should import a typed line plot export directly, preserving structure", () => {
      const out = LinePlot.parseImport(TYPED_EXPORT, undefined);
      expect(out.lines).toHaveLength(1);
      expect(out.channels?.y1).toEqual([65538]);
      expect(out.legend?.hidden).toBe(true);
    });

    it("should drop the source key from a legacy export so the server assigns a fresh one", () => {
      const out = LinePlot.parseImport(LEGACY, undefined);
      expect(out.key).toBeUndefined();
    });

    it("should drop the source key from a typed export so the server assigns a fresh one", () => {
      const out = LinePlot.parseImport(TYPED_EXPORT, undefined);
      expect(out.key).toBeUndefined();
    });
  });
});

describe("lineplot ingest", () => {
  it("creates the plot on the cluster and opens it as a tab", async () => {
    await awaitGranted(client, clientLineplot.TYPE_ONTOLOGY_ID, "update");
    const openTab = vi.fn<Panel.OpenTab>();
    const name = uniqueName("imported");
    await LinePlot.ingest(TYPED_EXPORT, {
      name,
      openTab,
      client,
      projectKey: await project(),
      fileName: "test.json",
    });
    expect(openTab).toHaveBeenCalledTimes(1);
    const spec = openTab.mock.calls[0][0];
    assert("resource" in spec, "expected a resource tab spec");
    if (typeof spec.resource === "string")
      throw new Error("expected a resolved resource, got an id string");
    expect(spec.resource.type).toBe("lineplot");
    const created = await client.lineplots.retrieve({ key: spec.resource.key });
    expect(created.name).toBe(name);
    expect(created.channels.y1).toEqual([65538]);
    const cached = client.lineplots.getCached({ key: spec.resource.key });
    expect(expectLive(cached).name).toBe(name);
  });

  it("rejects the import when the permission cache has no grant", async () => {
    await expect(
      LinePlot.ingest(TYPED_EXPORT, {
        name: "denied",
        openTab: vi.fn<Panel.OpenTab>(),
        client: null,
        projectKey: "project-1",
        fileName: "test.json",
      }),
    ).rejects.toThrow("You do not have permission to import line plots");
  });

  it("throws when a legacy state carries no body data", () => {
    const v5NoBody = {
      version: "5.0.0",
      key: "88aee41e-53b7-4a76-9df9-aceccc220089",
      remoteCreated: true,
      viewport: { renderTrigger: 0, zoom: dimensions.DECIMAL, pan: xy.ZERO },
      selection: { box: box.ZERO },
      mode: "zoom",
      control: { hold: false, clickMode: null, enableTooltip: true },
      toolbar: { activeTab: "data" },
      measure: { mode: "one" },
      annotations: { visible: true },
      selectedRules: [],
      hiddenLines: [],
    };
    expect(() => LinePlot.parseImport(v5NoBody, undefined)).toThrow(
      "Imported line plot has no body data",
    );
  });
});
