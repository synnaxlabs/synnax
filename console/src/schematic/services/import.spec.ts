// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { parseImport } from "@/schematic/services/import";

const configsOf = (s: { configs?: unknown }): record.Unknown =>
  typeof s.configs === "object" && s.configs != null ? { ...s.configs } : {};

const LEGACY_V2 = {
  version: "2.0.0",
  key: "88aee41e-53b7-4a76-9df9-aceccc220089",
  type: "schematic",
  name: "Schematic",
  editable: true,
  fitViewOnResize: false,
  snapshot: false,
  remoteCreated: false,
  control: "released",
  viewport: { position: { x: 0, y: 0 }, zoom: 1 },
  viewportMode: "select",
  legend: { visible: false, position: { x: 50, y: 50, units: { x: "px", y: "px" } } },
  nodes: [
    {
      key: "n1",
      position: { x: -300, y: -3.5 },
      type: "custom",
      width: 230,
      height: 112,
      zIndex: 4,
    },
  ],
  edges: [],
  props: { n1: { key: "valve", color: [28, 28, 28, 1] } },
};

const LEGACY_V2_TELEM = {
  ...LEGACY_V2,
  nodes: [
    { key: "n1", position: { x: 0, y: 0 }, type: "custom" },
    { key: "n2", position: { x: 10, y: 0 }, type: "custom" },
    { key: "n3", position: { x: 20, y: 0 }, type: "custom" },
  ],
  props: {
    n1: {
      key: "solenoidValve",
      color: [28, 28, 28, 1],
      normallyOpen: false,
      control: {
        show: true,
        chip: { sink: { props: { authority: 50 } } },
        indicator: { statusSource: { props: {} } },
      },
      source: {
        type: "source-pipeline",
        props: {
          outlet: "valueStream",
          segments: {
            valueStream: { type: "stream-channel-value", props: { channel: 101 } },
          },
        },
      },
      sink: {
        type: "sink-pipeline",
        props: {
          inlet: "setpoint",
          segments: {
            setpoint: { type: "boolean-numeric-converter-sink", props: {} },
            setter: { type: "controlled-numeric-telem-sink", props: { channel: 102 } },
          },
        },
      },
    },
    n2: {
      key: "value",
      color: [28, 28, 28, 1],
      telem: {
        type: "source-pipeline",
        props: {
          outlet: "stringifier",
          segments: {
            valueStream: { type: "stream-channel-value", props: { channel: 103 } },
            rollingAverage: { type: "rolling-average", props: { windowSize: 5 } },
            stringifier: { type: "stringify-number", props: { precision: 2 } },
          },
        },
      },
    },
    n3: {
      key: "value",
      color: [28, 28, 28, 1],
      telem: {
        type: "source-pipeline",
        props: {
          outlet: "stringifier",
          segments: {
            valueStream: { type: "stream-channel-value", props: { channel: 0 } },
            stringifier: { type: "stringify-number", props: { precision: 2 } },
          },
        },
      },
    },
  },
};

const TYPED_EXPORT = {
  key: "88aee41e-53b7-4a76-9df9-aceccc220089",
  name: "Schematic",
  type: "schematic",
  version: "6.0.0",
  snapshot: false,
  nodes: [{ key: "n1", position: { x: 0, y: 0 } }],
  edges: [],
  configs: { n1: { variant: "valve", color: [28, 28, 28, 1] } },
};

describe("schematic import", () => {
  describe("parseImport", () => {
    it("should migrate a legacy console export, preserving every symbol config", () => {
      const out = parseImport(LEGACY_V2, undefined);
      expect(out.nodes).toHaveLength(1);
      expect(configsOf(out).n1).toMatchObject({ variant: "valve" });
    });

    it("should import a typed schematic export directly, preserving configs", () => {
      const out = parseImport(TYPED_EXPORT, undefined);
      expect(out.nodes).toHaveLength(1);
      expect(configsOf(out).n1).toMatchObject({ variant: "valve" });
    });

    it("should not silently drop configs by parsing a legacy file as a typed one", () => {
      const out = parseImport(LEGACY_V2, undefined);
      expect(configsOf(out)).not.toEqual({});
    });

    it("should convert multi-word legacy variants to snake_case", () => {
      const out = parseImport(LEGACY_V2_TELEM, undefined);
      expect(configsOf(out).n1).toMatchObject({ variant: "solenoid_valve" });
    });

    it("should extract state and command channels from legacy source/sink specs", () => {
      const out = parseImport(LEGACY_V2_TELEM, undefined);
      const n1 = configsOf(out).n1 as record.Unknown;
      expect(n1).toMatchObject({ stateChannel: 101, commandChannel: 102 });
      expect(n1).not.toHaveProperty("source");
      expect(n1).not.toHaveProperty("sink");
    });

    it("should extract control authority from the legacy chip sink", () => {
      const out = parseImport(LEGACY_V2_TELEM, undefined);
      const n1 = configsOf(out).n1 as record.Unknown;
      expect(n1.control).toMatchObject({ show: true, authority: 50 });
      expect(n1.control).not.toHaveProperty("chip");
      expect(n1.control).not.toHaveProperty("indicator");
    });

    it("should extract value channel and rolling average from legacy telem specs", () => {
      const out = parseImport(LEGACY_V2_TELEM, undefined);
      const n2 = configsOf(out).n2 as record.Unknown;
      expect(n2).toMatchObject({ variant: "value", channel: 103, rollingAverage: 5 });
      expect(n2).not.toHaveProperty("telem");
    });

    it("should omit channel arguments for the legacy zero-channel sentinel", () => {
      const out = parseImport(LEGACY_V2_TELEM, undefined);
      const n3 = configsOf(out).n3 as record.Unknown;
      expect(n3).not.toHaveProperty("channel");
      expect(n3).not.toHaveProperty("telem");
    });

    it("should produce a payload that passes the create schema", () => {
      const out = parseImport(LEGACY_V2_TELEM, undefined);
      expect(() => schematic.newZ.parse(out)).not.toThrow();
    });
  });
});
