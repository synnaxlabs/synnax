// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { parseImport } from "@/session/schematic/migrations";

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
  });
});
