// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { anyStateZ } from "@/layered/service/lineplot/imex/import";

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
    const migrated = anyStateZ.parse(state);
    expect(migrated.version).toBe("5.0.0");
    expect(migrated.selectedRules).toEqual([]);
    expect(migrated.viewport).toBeDefined();
    expect(migrated.toolbar).toBeDefined();
  });

  it("should park v4 body into pendingUpload when remoteCreated is false", () => {
    const migrated = anyStateZ.parse({ ...V4_ZERO, remoteCreated: false });
    expect(migrated.pendingUpload).toBeDefined();
    expect(migrated.pendingUpload?.title).toBeDefined();
    expect(migrated.pendingUpload?.legend).toBeDefined();
    expect(migrated.pendingUpload?.axes).toBeDefined();
  });

  it("should leave pendingUpload undefined when remoteCreated is true", () => {
    const migrated = anyStateZ.parse({ ...V4_ZERO, remoteCreated: true });
    expect(migrated.pendingUpload).toBeUndefined();
  });

  it("should lift selected rules into selectedRules on the v4 to v5 migration", () => {
    const migrated = anyStateZ.parse({
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
