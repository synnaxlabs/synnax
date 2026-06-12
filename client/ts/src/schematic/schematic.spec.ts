// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { NotFoundError, ValidationError } from "@/errors";
import { schematic } from "@/schematic";
import { createTestClient } from "@/testutil/client";

const newProjectSchematic = async (client: ReturnType<typeof createTestClient>) => {
  const proj = await client.projects.create({ name: "dispatch", layout: {} });
  const schem = await client.schematics.create(proj.key, {
    name: "dispatch",
  });
  return { proj, schem };
};

const client = createTestClient();

describe("Schematic", () => {
  describe("create", () => {
    test("create one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schem = await client.schematics.create(proj.key, {
        name: "Schematic",
      });
      expect(schem.name).toEqual("Schematic");
      expect(schem.key).not.toEqual(uuid.ZERO);
      const retrieved = await client.schematics.retrieve({ key: schem.key });
      expect(retrieved.key).toEqual(schem.key);
    });
  });

  describe("rename", () => {
    test("rename one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schem = await client.schematics.create(proj.key, {
        name: "Schematic",
      });
      await client.schematics.rename(schem.key, "Schematic2");
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.name).toEqual("Schematic2");
    });
  });

  describe("delete", () => {
    test("delete one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schem = await client.schematics.create(proj.key, {
        name: "Schematic",
      });
      await client.schematics.delete(schem.key);
      await expect(client.schematics.retrieve({ key: schem.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("config case preservation", () => {
    test("preserves element key casing and round-trips telem args", async () => {
      const proj = await client.projects.create({ name: "CaseTest", layout: {} });
      const schem = await client.schematics.create(proj.key, {
        name: "CaseTest",
        configs: {
          myNode_A1: { variant: "value", channel: 42, rollingAverage: 5 },
        },
      });
      const retrieved = await client.schematics.retrieve({ key: schem.key });
      expect(retrieved.configs).toHaveProperty("myNode_A1");
      const config = retrieved.configs.myNode_A1;
      expect(config.variant).toBe("value");
      if (config.variant !== "value") return;
      expect(config.channel).toBe(42);
      expect(config.rollingAverage).toBe(5);
    });
  });

  describe("copy", () => {
    test("copy one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schem = await client.schematics.create(proj.key, {
        name: "Schematic",
      });
      const schem2 = await client.schematics.copy({
        key: schem.key,
        name: "Schematic2",
        snapshot: false,
      });
      expect(schem2.name).toEqual("Schematic2");
      expect(schem2.key).not.toEqual(uuid.ZERO);
    });

    describe("snapshot", () => {
      it("should not allow the caller to edit the snapshot", async () => {
        const proj = await client.projects.create({
          name: "Schematic",
          layout: { one: 1 },
        });
        const schem = await client.schematics.create(proj.key, {
          name: "Schematic",
        });
        const schem2 = await client.schematics.copy({
          key: schem.key,
          name: "Schematic2",
          snapshot: true,
        });
        await expect(
          client.schematics.dispatch(schem2.key, "sess-1", [
            schematic.setNode({ node: { key: "n1", position: { x: 0, y: 0 } } }),
          ]),
        ).rejects.toThrow(ValidationError);
      });
    });
  });

  describe("dispatch", () => {
    test("setNodePosition moves the matching node", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setNode({ node: { key: "n1", position: { x: 0, y: 0 } } }),
      ]);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setNodePosition({ key: "n1", position: { x: 100, y: 200 } }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes).toHaveLength(1);
      expect(res.nodes[0].position).toEqual({ x: 100, y: 200 });
    });

    test("setNode inserts a node and writes its config", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setNode({
          node: { key: "n1", position: { x: 1, y: 2 } },
          config: { variant: "tank", label: { label: "Pump" } },
        }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes).toHaveLength(1);
      expect(res.nodes[0]).toMatchObject({ key: "n1", position: { x: 1, y: 2 } });
      expect(res.configs.n1).toMatchObject({
        variant: "tank",
        label: { label: "Pump" },
      });
    });

    test("removeNode removes the node and drops its config", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setNode({
          node: { key: "n1", position: { x: 0, y: 0 } },
          config: { variant: "tank", label: { label: "Pump" } },
        }),
        schematic.setNode({
          node: { key: "n2", position: { x: 1, y: 1 } },
          config: { variant: "tank", label: { label: "Tank" } },
        }),
      ]);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.removeNode({ key: "n1" }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes).toHaveLength(1);
      expect(res.nodes[0]).toMatchObject({ key: "n2", position: { x: 1, y: 1 } });
      expect(Object.keys(res.configs)).toEqual(["n2"]);
      expect(res.configs.n2).toMatchObject({
        variant: "tank",
        label: { label: "Tank" },
      });
    });

    test("addEdge appends new edges and is a no-op on duplicate keys", async () => {
      const { schem } = await newProjectSchematic(client);
      const e = (
        key: string,
        srcNode: string,
        srcParam: string,
        tgtNode: string,
        tgtParam: string,
      ) => ({
        key,
        source: { node: srcNode, param: srcParam },
        target: { node: tgtNode, param: tgtParam },
      });
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.addEdge({ edge: e("e1", "a", "o", "b", "i") }),
        schematic.addEdge({ edge: e("e2", "b", "o", "c", "i") }),
      ]);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.addEdge({ edge: e("e2", "x", "y", "z", "w") }),
        schematic.addEdge({ edge: e("e3", "c", "o", "d", "i") }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.edges).toEqual([
        e("e1", "a", "o", "b", "i"),
        e("e2", "b", "o", "c", "i"),
        e("e3", "c", "o", "d", "i"),
      ]);
    });

    test("removeEdge removes the matching edge", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.addEdge({
          edge: {
            key: "e1",
            source: { node: "a", param: "o" },
            target: { node: "b", param: "i" },
          },
        }),
      ]);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.removeEdge({ key: "e1" }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.edges).toEqual([]);
    });

    test("setConfig upserts config under the given key", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setConfig({
          key: "n1",
          config: { variant: "tank", label: { label: "Original" } },
        }),
        schematic.setConfig({ key: "n1", config: { label: { label: "Replaced" } } }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.configs.n1).toMatchObject({
        variant: "tank",
        label: { label: "Replaced" },
      });
    });

    test("applies a multi-action sequence atomically", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setNode({ node: { key: "pump", position: { x: 0, y: 0 } } }),
        schematic.setNode({ node: { key: "valve", position: { x: 100, y: 0 } } }),
        schematic.addEdge({
          edge: {
            key: "e1",
            source: { node: "pump", param: "out" },
            target: { node: "valve", param: "in" },
          },
        }),
        schematic.setConfig({
          key: "pump",
          config: { variant: "tank", label: { label: "Main Pump" } },
        }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes).toHaveLength(2);
      expect(res.edges).toHaveLength(1);
      expect(res.configs.pump).toMatchObject({
        variant: "tank",
        label: { label: "Main Pump" },
      });
    });

    test("converges to the final position after a 30-action drag storm", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setNode({ node: { key: "pump", position: { x: 0, y: 0 } } }),
      ]);
      const actions = Array.from({ length: 30 }, (_, i) =>
        schematic.setNodePosition({ key: "pump", position: { x: i, y: i * 2 } }),
      );
      await client.schematics.dispatch(schem.key, "sess-1", actions);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes[0].position).toEqual({ x: 29, y: 58 });
    });

    test("round-trips telem args through dispatch", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, "sess-1", [
        schematic.setConfig({
          key: "valveNode_B2",
          config: { variant: "valve", stateChannel: 12, commandChannel: 13 },
        }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.configs).toHaveProperty("valveNode_B2");
      const config = res.configs.valveNode_B2;
      expect(config.variant).toBe("valve");
      if (config.variant !== "valve") return;
      expect(config.stateChannel).toBe(12);
      expect(config.commandChannel).toBe(13);
    });
  });
});
