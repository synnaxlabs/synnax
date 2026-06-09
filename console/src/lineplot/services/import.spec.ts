// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { parseImport } from "@/lineplot/services/import";
import * as v0 from "@/lineplot/types/v0";

const KEY = "88aee41e-53b7-4a76-9df9-aceccc220089";
const LINE_KEY = "65538-88aee41e";

// A legacy console export stores its body under the versioned console-state shape: a
// wrapped axes object, hex-string line colors, a version literal, and no top-level
// name. The strict legacy schemas must be tried before the lenient typed linePlotZ so
// the body (lines, channels) is migrated rather than silently dropped.
const LEGACY: Record<string, unknown> = {
  ...v0.ZERO_STATE,
  key: KEY,
  channels: { ...v0.ZERO_CHANNELS_STATE, y1: [65538] },
  lines: [
    {
      key: LINE_KEY,
      color: "#FF0000",
      strokeWidth: 3,
      downsample: 1,
      downsampleMode: "decimate",
    },
  ],
};

const zeroAxis = (key: lineplot.AxisKey): lineplot.Axis => ({
  key,
  label: "",
  labelDirection: "x",
  labelLevel: "small",
  bounds: { lower: 0, upper: 0 },
  autoBounds: { lower: false, upper: false },
  tickSpacing: 0,
});

// A typed export is the server line plot the new export path writes, tagged with a
// layout type. It carries a name and a flat per-axis bundle the legacy schemas never
// match, so it parses directly.
const TYPED_EXPORT = {
  key: KEY,
  name: "My Plot",
  type: "lineplot",
  title: { level: "p", visible: false },
  legend: { visible: false, position: { x: 0, y: 0 } },
  channels: { x1: 0, x2: 0, y1: [65538], y2: [], y3: [], y4: [] },
  ranges: { x1: [], x2: [] },
  axes: {
    x1: zeroAxis("x1"),
    x2: zeroAxis("x2"),
    y1: zeroAxis("y1"),
    y2: zeroAxis("y2"),
    y3: zeroAxis("y3"),
    y4: zeroAxis("y4"),
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
      const out = parseImport(LEGACY, undefined);
      expect(out.lines).toHaveLength(1);
      expect(out.channels?.y1).toEqual([65538]);
    });

    it("should not silently drop the body by parsing a legacy file as a typed one", () => {
      const out = parseImport(LEGACY, undefined);
      expect(out.lines).not.toHaveLength(0);
    });

    it("should import a typed line plot export directly, preserving structure", () => {
      const out = parseImport(TYPED_EXPORT, undefined);
      expect(out.lines).toHaveLength(1);
      expect(out.channels?.y1).toEqual([65538]);
    });

    it("should drop the source key from a legacy export so the server assigns a fresh one", () => {
      const out = parseImport(LEGACY, undefined);
      expect(out.key).toBeUndefined();
    });

    it("should drop the source key from a typed export so the server assigns a fresh one", () => {
      const out = parseImport(TYPED_EXPORT, undefined);
      expect(out.key).toBeUndefined();
    });
  });
});
