// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it, vi } from "vitest";

import { Arc } from "@/feature/arc";
import { type Layout } from "@/platform/layout";
import { createGrantedFluxStore, createTestFluxStore, uniqueName } from "@/testutil";

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
      expect(Arc.anyStateZ.parse(v0).version).toBe(LATEST_VERSION);
    });

    it("should migrate a v1 state to the latest version", () => {
      expect(Arc.anyStateZ.parse(v1State({})).version).toBe(LATEST_VERSION);
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
      expect(Arc.anyStateZ.parse(v3).version).toBe(LATEST_VERSION);
    });
  });

  describe("set_status rewrite into pendingUpload", () => {
    it("should rewrite a set_status node with all legacy fields", () => {
      const migrated = Arc.anyStateZ.parse(
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
      const migrated = Arc.anyStateZ.parse(v1State({ n1: { key: "set_status" } }));
      expect(migrated.pendingUpload?.graph.configs.n1).toEqual({
        type: "status.set",
        key_or_name: "",
        variant: "success",
        message: "",
      });
    });

    it("should pass through non-set_status nodes, moving key to type", () => {
      const migrated = Arc.anyStateZ.parse(
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
      const result = Arc.parseImport(
        v1State({ n1: { key: "channel.read", channel: 42 } }),
        "Imported Arc",
      );
      expect(result.name).toBe("Imported Arc");
      expect(result.mode).toBe("graph");
      expect(result.graph?.configs?.n1).toEqual({ type: "channel.read", channel: 42 });
      expect(result.text?.raw).toBe("x = 1");
    });

    it("should parse a current typed Arc export through arcZ", () => {
      const current = arc.arcZ.parse({
        key: "0f8b2c9e-1234-4f2a-9c3d-abcdefabcdef",
        name: "Typed Arc",
        mode: "text",
        text: { raw: "x = 1" },
      });
      const result = Arc.parseImport(current);
      expect(result.name).toBe("Typed Arc");
      expect(result.mode).toBe("text");
      expect(result).not.toHaveProperty("key");
    });

    it("should prefer the fallback name over a typed export's name", () => {
      const current = arc.arcZ.parse({
        key: "0f8b2c9e-1234-4f2a-9c3d-abcdefabcdef",
        name: "Typed Arc",
        mode: "graph",
        graph: { nodes: [], edges: [] },
      });
      expect(Arc.parseImport(current, "Renamed").name).toBe("Renamed");
    });

    it("should reject a latest-version console state with no parked graph", () => {
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
      expect(() => Arc.parseImport(v3)).toThrow("Imported Arc has no graph data");
    });
  });

  describe("ingest", () => {
    const client = createTestClient();

    it("should create the arc on the cluster and place its editor layout", async () => {
      const store = await createGrantedFluxStore(
        client,
        arc.TYPE_ONTOLOGY_ID,
        "update",
      );
      const placeLayout = vi.fn<Layout.Placer>();
      const name = uniqueName("imported");
      const id = await Arc.ingest(
        v1State({ n1: { key: "channel.read", channel: 1 } }),
        {
          layout: { name },
          placeLayout,
          store,
          client,
          projectKey: "project-1",
        },
      );
      expect(placeLayout).toHaveBeenCalledTimes(1);
      if (id == null) throw new Error("ingest returned no ontology id");
      const created = await client.arcs.retrieve({ key: id.key });
      expect(created.name).toBe(name);
      expect(store.arcs.get(id.key)?.name).toBe(name);
    });

    it("should reject the import when the permission cache has no grant", async () => {
      const store = createTestFluxStore(null);
      await expect(
        Arc.ingest(v1State({}), {
          layout: { name: "denied" },
          placeLayout: vi.fn<Layout.Placer>(),
          store,
          client: null,
          projectKey: "project-1",
        }),
      ).rejects.toThrow("You do not have permission to import Arc automations");
    });
  });
});
