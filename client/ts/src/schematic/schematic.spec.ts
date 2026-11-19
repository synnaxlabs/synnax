// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it, test, vi } from "vitest";

import { NotFoundError } from "@/index";
import { query } from "@/query";
import { schematic } from "@/schematic";
import { createTestClient, expectDeleted } from "@/testutil";

const newProjectSchematic = async (client: ReturnType<typeof createTestClient>) => {
  const proj = await client.projects.create({ name: "dispatch" });
  const schem = await client.schematics.create({ project: proj.key, name: "dispatch" });
  return { proj, schem };
};

const nodeCfg = (raw: unknown): schematic.NodeConfig =>
  schematic.nodeConfigZ.parse(raw);

const tank = (extra: object = {}): schematic.NodeConfig =>
  nodeCfg({ variant: "tank", ...extra });

const group = (): schematic.NodeConfig => nodeCfg({ variant: "group_box" });

const node = (
  key: string,
  config: schematic.NodeConfig = nodeCfg({ variant: "vent" }),
  position = { x: 0, y: 0 },
): schematic.Node => ({ key, position, zIndex: 0, config });

const edge = (
  key: string,
  srcNode: string,
  srcParam: string,
  tgtNode: string,
  tgtParam: string,
): schematic.Edge => ({
  key,
  source: { node: srcNode, param: srcParam },
  target: { node: tgtNode, param: tgtParam },
  config: schematic.edgeConfigZ.parse({ variant: "pipe" }),
});

const setNodes = (...keys: string[]): schematic.Action[] =>
  keys.map((key) => schematic.setNode({ node: node(key) }));

const setParent = (key: string, parent: string): schematic.Action =>
  schematic.set({ path: ["nodes", key, "parent"], value: parent });

const client = createTestClient();

describe("Schematic", () => {
  describe("create", () => {
    test("create one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
      });
      const schem = await client.schematics.create({
        project: proj.key,
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
      });
      const schem = await client.schematics.create({
        project: proj.key,
        name: "Schematic",
      });
      await client.schematics.rename(schem.key, "Schematic2");
      const res = await client.schematics.retrieve(schem.key);
      expect(res.name).toEqual("Schematic2");
    });
  });

  describe("project", () => {
    test("retrieve the schematics of a project", async () => {
      const proj = await client.projects.create({ name: "Schematic" });
      const other = await client.projects.create({ name: "Schematic" });
      const v = await client.schematics.create({
        project: proj.key,
        name: "Schematic",
      });
      expect(v.project).toEqual(proj.key);
      const owned = await client.schematics.retrieve({ projects: [proj.key] });
      expect(owned.map(({ key }) => key)).toEqual([v.key]);
      const unowned = await client.schematics.retrieve({ projects: [other.key] });
      expect(unowned).toEqual([]);
    });

    test("move to another project", async () => {
      const proj = await client.projects.create({ name: "Schematic" });
      const dest = await client.projects.create({ name: "Schematic" });
      const v = await client.schematics.create({
        project: proj.key,
        name: "Schematic",
      });
      await client.batch.send(client.schematics.setProjectRequest(v.key, dest.key));
      const res = await client.schematics.retrieve([v.key]);
      expect(res[0].project).toEqual(dest.key);
    });

    test("delete with its project", async () => {
      const proj = await client.projects.create({ name: "Schematic" });
      const v = await client.schematics.create({
        project: proj.key,
        name: "Schematic",
      });
      await client.projects.delete(proj.key);
      await vi.waitFor(
        async () =>
          await expect(client.schematics.retrieve(v.key)).rejects.toThrow(
            NotFoundError,
          ),
      );
    });
  });

  describe("delete", () => {
    test("delete one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
      });
      const schem = await client.schematics.create({
        project: proj.key,
        name: "Schematic",
      });
      await client.schematics.delete(schem.key);
      await expect(client.schematics.retrieve(schem.key)).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("config case preservation", () => {
    test("preserves element key casing and round-trips telem args", async () => {
      const ch = await client.channels.create({
        name: `value_${id.create()}`,
        dataType: "float32",
        virtual: true,
      });
      const proj = await client.projects.create({ name: "CaseTest" });
      const schem = await client.schematics.create({
        project: proj.key,
        name: "CaseTest",
        nodes: [
          {
            key: "myNode_A1",
            position: { x: 0, y: 0 },
            config: { variant: "value", channel: ch.key, rollingAverage: 5 },
          },
        ],
      });
      const retrieved = await client.schematics.retrieve({ key: schem.key });
      expect(retrieved.nodes.map((n) => n.key)).toEqual(["myNode_A1"]);
      const { config } = retrieved.nodes[0];
      expect(config.variant).toBe("value");
      if (config.variant !== "value") return;
      expect(config.channel).toBe(ch.key);
      expect(config.rollingAverage).toBe(5);
    });
  });

  describe("configsOf", () => {
    const doc = (): schematic.Schematic =>
      schematic.schematicZ.parse({
        key: uuid.create(),
        name: "configs",
        nodes: [node("n1", tank()), node("n2")],
        edges: [edge("e1", "n1", "out", "n2", "in")],
      });

    it("should return the config of every node and edge by key", () => {
      const s = doc();
      expect(client.schematics.configsOf(s)).toEqual({
        n1: s.nodes[0].config,
        n2: s.nodes[1].config,
        e1: s.edges[0].config,
      });
    });

    it("should return the same record for the same schematic", () => {
      const s = doc();
      expect(client.schematics.configsOf(s)).toBe(client.schematics.configsOf(s));
    });

    it("should build a new record for a new schematic", () => {
      const s = doc();
      const next = { ...s, nodes: [s.nodes[0]] };
      expect(client.schematics.configsOf(next)).not.toBe(
        client.schematics.configsOf(s),
      );
      expect(client.schematics.configsOf(next)).not.toHaveProperty("n2");
    });
  });

  describe("copy", () => {
    test("copy one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
      });
      const schem = await client.schematics.create({
        project: proj.key,
        name: "Schematic",
      });
      const schem2 = await client.schematics.copy({
        key: schem.key,
        name: "Schematic2",
      });
      expect(schem2.name).toEqual("Schematic2");
      expect(schem2.key).not.toEqual(uuid.ZERO);
    });
  });

  describe("dispatch", () => {
    test("setNodePosition moves the matching node", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, setNodes("n1"));
      await client.schematics.dispatch(schem.key, [
        schematic.setNodePosition({ key: "n1", position: { x: 100, y: 200 } }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.nodes).toHaveLength(1);
      expect(res.nodes[0].position).toEqual({ x: 100, y: 200 });
    });

    test("setNode inserts a node with its config", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.setNode({
          node: {
            key: "n1",
            position: { x: 1, y: 2 },
            config: { variant: "tank", label: { label: "Pump" } },
          },
        }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.nodes).toEqual([
        node("n1", tank({ label: { label: "Pump" } }), { x: 1, y: 2 }),
      ]);
    });

    test("setNode replaces an existing node and its config", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.setNode({ node: node("n1", tank({ label: { label: "Original" } })) }),
        schematic.setNode({ node: node("n1", tank({ label: { label: "Replaced" } })) }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes).toEqual([node("n1", tank({ label: { label: "Replaced" } }))]);
    });

    test("removeNode removes the node with its config", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.setNode({ node: node("n1", tank({ label: { label: "Pump" } })) }),
        schematic.setNode({
          node: node("n2", tank({ label: { label: "Tank" } }), { x: 1, y: 1 }),
        }),
      ]);
      await client.schematics.dispatch(schem.key, [
        schematic.removeNode({ key: "n1" }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.nodes).toEqual([
        node("n2", tank({ label: { label: "Tank" } }), { x: 1, y: 1 }),
      ]);
    });

    test("removeNode promotes the children of a group on the Core", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        ...setNodes("n1"),
        schematic.setNode({ node: node("outer", group()) }),
        schematic.setNode({ node: { ...node("inner", group()), parent: "outer" } }),
        setParent("n1", "inner"),
      ]);
      await client.schematics.dispatch(schem.key, [
        schematic.removeNode({ key: "inner" }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.nodes).toEqual([
        { ...node("n1"), parent: "outer" },
        node("outer", group()),
      ]);
    });

    test("rejects a parent that is not a group box", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, setNodes("n1", "n2"));
      await expect(
        client.schematics.dispatch(schem.key, [setParent("n1", "n2")]),
      ).rejects.toThrow("nodes.0.parent: n2 is not a group box of the schematic");
    });

    test("an undo of a group removal restores the parents on the Core", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        ...setNodes("n1", "n2"),
        schematic.setNode({ node: node("g1", group()) }),
        setParent("n1", "g1"),
        setParent("n2", "g1"),
      ]);
      const before = await client.schematics.retrieve(schem.key);
      const { inverse } = schematic.reduceAll(before, [
        schematic.removeNode({ key: "g1" }),
      ]);
      await client.schematics.dispatch(schem.key, [
        schematic.removeNode({ key: "g1" }),
      ]);
      await client.schematics.dispatch(schem.key, inverse);
      const res = await client.schematics.retrieve(schem.key);
      const byKey = (ns: schematic.Node[]) =>
        Object.fromEntries(ns.map((n) => [n.key, n]));
      expect(byKey(res.nodes)).toEqual(byKey(before.nodes));
    });

    test("the client and Core reducers agree across group removals", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        ...setNodes("n1", "n2", "n3"),
        schematic.setNode({ node: node("outer", group()) }),
        schematic.setNode({ node: { ...node("mid", group()), parent: "outer" } }),
        schematic.setNode({ node: { ...node("inner", group()), parent: "mid" } }),
        setParent("n1", "inner"),
        setParent("n2", "mid"),
        setParent("n3", "outer"),
      ]);
      let local = await client.schematics.retrieve(schem.key);
      const batches: schematic.Action[][] = [
        [schematic.removeNode({ key: "n1" })],
        [schematic.removeNode({ key: "mid" })],
        [
          schematic.removeNode({ key: "outer" }),
          schematic.removeNode({ key: "inner" }),
        ],
        [schematic.removeNode({ key: "ghost" })],
      ];
      for (const batch of batches) {
        local = schematic.reduceAll(local, batch).next;
        await client.schematics.dispatch(schem.key, batch);
        const remote = await client.schematics.retrieve(schem.key);
        expect(remote.nodes).toEqual(local.nodes);
        expect(remote.edges).toEqual(local.edges);
      }
    });

    test("addEdge appends new edges and is a no-op on duplicate keys", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        ...setNodes("a", "b", "c", "d"),
        schematic.addEdge({ edge: edge("e1", "a", "o", "b", "i") }),
        schematic.addEdge({ edge: edge("e2", "b", "o", "c", "i") }),
      ]);
      await client.schematics.dispatch(schem.key, [
        schematic.addEdge({ edge: edge("e2", "x", "y", "z", "w") }),
        schematic.addEdge({ edge: edge("e3", "c", "o", "d", "i") }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.edges).toEqual([
        edge("e1", "a", "o", "b", "i"),
        edge("e2", "b", "o", "c", "i"),
        edge("e3", "c", "o", "d", "i"),
      ]);
    });

    test("removeEdge removes the matching edge", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        ...setNodes("a", "b"),
        schematic.addEdge({ edge: edge("e1", "a", "o", "b", "i") }),
      ]);
      expect((await client.schematics.retrieve(schem.key)).edges).toHaveLength(1);
      await client.schematics.dispatch(schem.key, [
        schematic.removeEdge({ key: "e1" }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.edges).toEqual([]);
    });

    test("a set of an edge config field reaches the Core", async () => {
      const { schem } = await newProjectSchematic(client);
      const segments = [{ direction: "x", length: 10 }];
      await client.schematics.dispatch(schem.key, [
        ...setNodes("a", "b"),
        schematic.addEdge({ edge: edge("e1", "a", "o", "b", "i") }),
        schematic.set({ path: ["edges", "e1", "config", "segments"], value: segments }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.edges[0].config).toEqual(
        schematic.edgeConfigZ.parse({ variant: "pipe", segments }),
      );
    });

    test("rejects a path into a node outside its config", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, setNodes("n1"));
      await expect(
        client.schematics.dispatch(schem.key, [
          schematic.set({ path: ["nodes", "n1", "position"], value: { x: 1, y: 1 } }),
        ]),
      ).rejects.toThrow("a path into nodes must go into the config of one entry");
    });

    test("sets and removes the parent of a node", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        ...setNodes("n1"),
        schematic.setNode({ node: node("g1", group()) }),
        setParent("n1", "g1"),
      ]);
      expect((await client.schematics.retrieve(schem.key)).nodes[0].parent).toBe("g1");
      await client.schematics.dispatch(schem.key, [
        schematic.remove({ path: ["nodes", "n1"], key: "parent" }),
      ]);
      expect(
        (await client.schematics.retrieve(schem.key)).nodes[0].parent,
      ).toBeUndefined();
    });

    test("setChanged keeps a concurrent edit to another field", async () => {
      const { schem } = await newProjectSchematic(client);
      const base = tank({ label: { label: "Original" } });
      await client.schematics.dispatch(schem.key, [
        schematic.setNode({ node: node("n1", base) }),
      ]);
      const red = { strokeColor: "#ff0000" };
      const replaced = { label: { label: "Replaced" } };
      await client.schematics.dispatch(
        schem.key,
        schematic.setChanged({
          key: "n1",
          base,
          next: tank({ label: { label: "Original" }, ...red }),
        }),
      );
      await client.schematics.dispatch(
        schem.key,
        schematic.setChanged({ key: "n1", base, next: tank(replaced) }),
      );
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes[0].config).toEqual(tank({ ...replaced, ...red }));
    });

    test("setChanged removes a field that next does not hold", async () => {
      const { schem } = await newProjectSchematic(client);
      const base = tank({ strokeColor: "#ff0000" });
      await client.schematics.dispatch(schem.key, [
        schematic.setNode({ node: node("n1", base) }),
      ]);
      await client.schematics.dispatch(
        schem.key,
        schematic.setChanged({ key: "n1", base, next: tank() }),
      );
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes[0].config).toEqual(tank());
    });

    test("applies a multi-action sequence atomically", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.setNode({
          node: node("pump", tank({ label: { label: "Main Pump" } })),
        }),
        schematic.setNode({ node: node("valve", undefined, { x: 100, y: 0 }) }),
        schematic.addEdge({ edge: edge("e1", "pump", "out", "valve", "in") }),
      ]);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.nodes).toHaveLength(2);
      expect(res.edges).toHaveLength(1);
      expect(res.nodes[0].config).toEqual(tank({ label: { label: "Main Pump" } }));
    });

    test("converges to the final position after a 30-action drag storm", async () => {
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, setNodes("pump"));
      const actions = Array.from({ length: 30 }, (_, i) =>
        schematic.setNodePosition({ key: "pump", position: { x: i, y: i * 2 } }),
      );
      await client.schematics.dispatch(schem.key, actions);
      const res = await client.schematics.retrieve(schem.key);
      expect(res.nodes[0].position).toEqual({ x: 29, y: 58 });
    });

    test("round-trips telem args through dispatch", async () => {
      const [state, command] = await client.channels.create([
        { name: `state_${id.create()}`, dataType: "float32", virtual: true },
        { name: `command_${id.create()}`, dataType: "float32", virtual: true },
      ]);
      const { schem } = await newProjectSchematic(client);
      await client.schematics.dispatch(schem.key, [
        schematic.setNode({
          node: {
            key: "valveNode_B2",
            position: { x: 0, y: 0 },
            config: {
              variant: "valve",
              stateChannel: state.key,
              commandChannel: command.key,
            },
          },
        }),
      ]);
      const res = await client.schematics.retrieve({ key: schem.key });
      expect(res.nodes.map((n) => n.key)).toEqual(["valveNode_B2"]);
      const { config } = res.nodes[0];
      expect(config.variant).toBe("valve");
      if (config.variant !== "valve") return;
      expect(config.stateChannel).toBe(state.key);
      expect(config.commandChannel).toBe(command.key);
    });
  });
});

describe("store", () => {
  it("tombstones deletes from live delete signals", async () => {
    await client.connect();
    const project = await client.projects.create({ name: `sch-${id.create()}` });
    const created = await client.schematics.create({
      project: project.key,
      name: `schematic-${id.create()}`,
    });
    await client.schematics.delete(created.key);
    await expect
      .poll(() =>
        query.Deleted.matches(client.schematics.watch(created.key).getCached()),
      )
      .toBe(true);
    const cached = expectDeleted(client.schematics.watch(created.key).getCached());
    expect(cached.corpse.name).toEqual(created.name);
  });
});
