// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { anyStateZ, parseImport } from "@/layered/service/arc/imex/import";

const LATEST_VERSION = "3.0.0";

const zeroGraph = {
  editable: true,
  fitViewOnResize: false,
  viewport: { position: { x: 0, y: 0 }, zoom: 1 },
  selected: [],
  nodes: [],
  edges: [],
};

// A legacy v1 console-state export: a versioned redux shape that parks node props inline
// under graph.props. The migration chain rewrites set_status and lifts each prop into the
// pendingUpload graph configs map (function type moving from "key" to "type").
const v1State = (props: Record<string, Record<string, unknown>>) => ({
  key: "test",
  version: "1.0.0",
  remoteCreated: false,
  graph: { ...zeroGraph, props },
  text: { raw: "x = 1" },
  mode: "graph",
});

describe("arc import", () => {
  describe("anyStateZ", () => {
    it("should migrate a v0 state to the latest version", () => {
      const v0 = {
        key: "test",
        version: "0.0.0",
        remoteCreated: false,
        graph: { ...zeroGraph, props: {} },
        text: { raw: "" },
        mode: "graph",
      };
      expect(anyStateZ.parse(v0).version).toBe(LATEST_VERSION);
    });

    it("should migrate a v1 state to the latest version", () => {
      expect(anyStateZ.parse(v1State({})).version).toBe(LATEST_VERSION);
    });

    it("should parse a latest-version state as-is", () => {
      const v3 = {
        key: "test",
        version: LATEST_VERSION,
        graph: {
          editable: true,
          fitViewOnResize: false,
          viewport: { position: { x: 0, y: 0 }, zoom: 1 },
          selected: [],
        },
      };
      expect(anyStateZ.parse(v3).version).toBe(LATEST_VERSION);
    });
  });

  describe("set_status rewrite into pendingUpload", () => {
    it("should rewrite a set_status node with all legacy fields", () => {
      const migrated = anyStateZ.parse(
        v1State({
          n1: {
            key: "set_status",
            statusKey: "alarm",
            variant: "warning",
            message: "overpressure",
          },
        }),
      );
      expect(migrated.pendingUpload?.graph.configs.n1).toEqual({
        type: "status.set",
        key_or_name: "alarm",
        variant: "warning",
        message: "overpressure",
      });
    });

    it("should default missing legacy fields when rewriting set_status", () => {
      const migrated = anyStateZ.parse(v1State({ n1: { key: "set_status" } }));
      expect(migrated.pendingUpload?.graph.configs.n1).toEqual({
        type: "status.set",
        key_or_name: "",
        variant: "success",
        message: "",
      });
    });

    it("should pass through non-set_status nodes, moving key to type", () => {
      const migrated = anyStateZ.parse(
        v1State({ n1: { key: "channel.read", channel: 42 } }),
      );
      expect(migrated.pendingUpload?.graph.configs.n1).toEqual({
        type: "channel.read",
        channel: 42,
      });
    });
  });

  describe("parseImport", () => {
    it("should build an Arc create payload from a legacy export", () => {
      const result = parseImport(
        v1State({ n1: { key: "channel.read", channel: 42 } }),
        "Imported Arc",
      );
      expect(result.name).toBe("Imported Arc");
      expect(result.mode).toBe("graph");
      expect(result.graph?.configs?.n1).toEqual({ type: "channel.read", channel: 42 });
      expect(result.text?.raw).toBe("x = 1");
    });
  });
});
