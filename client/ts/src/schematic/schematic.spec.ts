// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { NotFoundError, ValidationError } from "@/errors";
import { query } from "@/query";
import { schematic } from "@/schematic";
import { createTestClient, expectDeleted } from "@/testutil";

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
      const retrieved = await client.schematics.retrieve(schem.key);
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
      const res = await client.schematics.retrieve(schem.key);
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
      await expect(client.schematics.retrieve(schem.key)).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("config case preservation", () => {
    test("preserves arbitrary key casing within config values", async () => {
      const proj = await client.projects.create({ name: "CaseTest", layout: {} });
      const schem = await client.schematics.create(proj.key, {
        name: "CaseTest",
        configs: {
          n1: {
            camelCaseKey: "value1",
            PascalCaseKey: "value2",
            snake_case_key: "value3",
            nested: {
              innerCamelCase: 123,
              InnerPascalCase: { deepKey: true },
            },
          },
        },
      });
      const retrieved = await client.schematics.retrieve(schem.key);
      const config = retrieved.configs.n1 as Record<string, unknown>;
      expect(config.camelCaseKey).toEqual("value1");
      expect(config.PascalCaseKey).toEqual("value2");
      expect(config.snake_case_key).toEqual("value3");
      expect((config.nested as Record<string, unknown>).innerCamelCase).toEqual(123);
      expect(
        (
          (config.nested as Record<string, unknown>).InnerPascalCase as Record<
            string,
            unknown
          >
        ).deepKey,
      ).toEqual(true);
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
          client.schematics.dispatch(schem2.key, [
            schematic.setNode({
              node: { key: "n1", position: { x: 0, y: 0 } },
            }),
          ]),
        ).rejects.toThrow(ValidationError);
      });
    });
  });

  describe("dispatch", () => {
    test("setNodePosition moves the matching node", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.setNode({ node: { key: "n1", position: { x: 0, y: 0 } } }),
      ]);
      await client.schematics.dispatch(schem.key, [
        schematic.setNodePosition({ key: "n1", position: { x: 100, y: 200 } }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.nodes).toHaveLength(1);
      expect(res.nodes[0].position).toEqual({ x: 100, y: 200 });
    });

    test("setNode inserts a node and writes its config", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.setNode({
          node: { key: "n1", position: { x: 1, y: 2 } },
          config: { label: "Pump" },
        }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.nodes).toHaveLength(1);
      expect(res.nodes[0]).toMatchObject({ key: "n1", position: { x: 1, y: 2 } });
      expect(res.configs.n1.label).toBe("Pump");
    });

    test("removeNode removes the node and drops its config", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.setNode({
          node: { key: "n1", position: { x: 0, y: 0 } },
          config: { label: "Pump" },
        }),
        schematic.setNode({
          node: { key: "n2", position: { x: 1, y: 1 } },
          config: { label: "Tank" },
        }),
      ]);
      await client.schematics.dispatch(schem.key, [
        schematic.removeNode({ key: "n1" }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.nodes).toHaveLength(1);
      expect(res.nodes[0]).toMatchObject({ key: "n2", position: { x: 1, y: 1 } });
      expect(res.configs).toEqual({ n2: { label: "Tank" } });
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
      await client.schematics.dispatch(schem.key, [
        schematic.addEdge({ edge: e("e1", "a", "o", "b", "i") }),
        schematic.addEdge({ edge: e("e2", "b", "o", "c", "i") }),
      ]);
      await client.schematics.dispatch(schem.key, [
        schematic.addEdge({ edge: e("e2", "x", "y", "z", "w") }),
        schematic.addEdge({ edge: e("e3", "c", "o", "d", "i") }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.edges).toEqual([
        e("e1", "a", "o", "b", "i"),
        e("e2", "b", "o", "c", "i"),
        e("e3", "c", "o", "d", "i"),
      ]);
    });

    test("removeEdge removes the matching edge", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.addEdge({
          edge: {
            key: "e1",
            source: { node: "a", param: "o" },
            target: { node: "b", param: "i" },
          },
        }),
      ]);
      await client.schematics.dispatch(schem.key, [
        schematic.removeEdge({ key: "e1" }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.edges).toEqual([]);
    });

    test("setConfig upserts config under the given key", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.setConfig({ key: "n1", config: { label: "Original" } }),
        schematic.setConfig({ key: "n1", config: { label: "Replaced" } }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.configs.n1.label).toBe("Replaced");
    });

    test("applies a multi-action sequence atomically", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.setNode({
          node: { key: "pump", position: { x: 0, y: 0 } },
        }),
        schematic.setNode({
          node: { key: "valve", position: { x: 100, y: 0 } },
        }),
        schematic.addEdge({
          edge: {
            key: "e1",
            source: { node: "pump", param: "out" },
            target: { node: "valve", param: "in" },
          },
        }),
        schematic.setConfig({ key: "pump", config: { label: "Main Pump" } }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.nodes).toHaveLength(2);
      expect(res.edges).toHaveLength(1);
      expect(res.configs.pump.label).toBe("Main Pump");
    });

    test("converges to the final position after a 30-action drag storm", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.setNode({
          node: { key: "pump", position: { x: 0, y: 0 } },
        }),
      ]);
      const actions = Array.from({ length: 30 }, (_, i) =>
        schematic.setNodePosition({ key: "pump", position: { x: i, y: i * 2 } }),
      );
      await client.schematics.dispatch(schem.key, actions);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.nodes[0].position).toEqual({ x: 29, y: 58 });
    });

    test("preserves arbitrary key casing within config values through dispatch", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.setConfig({
          key: "n1",
          config: {
            camelCaseKey: "v1",
            PascalCaseKey: "v2",
            snake_case_key: "v3",
            nested: { innerCamelCase: 1, InnerPascalCase: { deepKey: true } },
          },
        }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      const config = res.configs.n1;
      expect(config.camelCaseKey).toBe("v1");
      expect(config.PascalCaseKey).toBe("v2");
      expect(config.snake_case_key).toBe("v3");
      const nested = config.nested as Record<string, unknown>;
      expect(nested.innerCamelCase).toBe(1);
      expect((nested.InnerPascalCase as Record<string, unknown>).deepKey).toBe(true);
    });
  });
});

describe("store", () => {
  it("tombstones deletes from live delete signals", async () => {
    await client.cache.ensureStreaming();
    const project = await client.projects.create({ name: `sch-${id.create()}` });
    const created = await client.schematics.create(project.key, {
      name: `schematic-${id.create()}`,
    });
    await client.schematics.delete(created.key);
    await expect
      .poll(() => query.Deleted.matches(client.schematics.getCached(created.key)))
      .toBe(true);
    const cached = expectDeleted(client.schematics.getCached(created.key));
    expect(cached.corpse.name).toEqual(created.name);
  });
});
