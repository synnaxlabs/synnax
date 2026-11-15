// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { schematic } from "@/schematic";

const nodeCfg = (raw: unknown): schematic.NodeConfig =>
  schematic.nodeConfigZ.parse(raw);

const edgeCfg = (raw: unknown): schematic.EdgeConfig =>
  schematic.edgeConfigZ.parse(raw);

const vent = (): schematic.NodeConfig => nodeCfg({ variant: "vent" });

const pipe = (): schematic.EdgeConfig => edgeCfg({ variant: "pipe" });

const node = (
  key: string,
  x: number,
  y: number,
  config: schematic.NodeConfig = vent(),
): schematic.Node => ({ key, position: { x, y }, zIndex: 0, config });

const edge = (
  key: string,
  srcNode: string,
  srcParam: string,
  tgtNode: string,
  tgtParam: string,
  config: schematic.EdgeConfig = pipe(),
): schematic.Edge => ({
  key,
  source: { node: srcNode, param: srcParam },
  target: { node: tgtNode, param: tgtParam },
  config,
});

const empty = (overrides: Partial<schematic.Schematic> = {}): schematic.Schematic =>
  schematic.schematicZ.parse({ name: "Test Schematic", ...overrides });

const groupConfig = (
  members: string[],
  extra: Partial<schematic.GroupBoxNodeConfig> = {},
): schematic.NodeConfig => nodeCfg({ variant: "group_box", members, ...extra });

const group = (
  key: string,
  members: string[],
  extra: Partial<schematic.GroupBoxNodeConfig> = {},
): schematic.Node => node(key, 0, 0, groupConfig(members, extra));

const tank = (label: string, strokeColor?: unknown): schematic.NodeConfig =>
  nodeCfg({
    variant: "tank",
    label: { label },
    ...(strokeColor != null && { strokeColor }),
  });

const apply = (
  state: schematic.Schematic,
  ...actions: schematic.Action[]
): schematic.Schematic => schematic.reduceAll(state, actions).next;

const set = (path: string[], value: unknown): schematic.Action =>
  schematic.set({ path, value });

const encoded = (state: schematic.Schematic, actions: schematic.Action[]) =>
  actions.map((a) => schematic.encode(state, a));

const byKey = (ns: schematic.Schematic["nodes"]) =>
  Object.fromEntries(ns.map((n) => [n.key, n]));

describe("schematic reducer", () => {
  describe("set of the name", () => {
    it("should set the schematic name to the value", () => {
      const state = empty({ name: "old" });
      const out = apply(state, set(["name"], "new"));
      expect(out.name).toEqual("new");
    });
    it("should leave nodes and edges untouched", () => {
      const state = empty({
        name: "old",
        nodes: [node("a", 0, 0, tank("Pump")), node("b", 1, 1)],
        edges: [edge("e1", "a", "o", "b", "i")],
      });
      const out = apply(state, set(["name"], "new"));
      expect(out.nodes).toEqual(state.nodes);
      expect(out.edges).toEqual(state.edges);
    });
  });

  describe("setNodePosition", () => {
    it("should move the matching node to the new position", () => {
      const state = empty({ nodes: [node("n1", 0, 0), node("n2", 5, 5)] });
      const out = apply(
        state,
        schematic.setNodePosition({ key: "n1", position: { x: 100, y: 200 } }),
      );
      expect(out.nodes).toEqual([node("n1", 100, 200), node("n2", 5, 5)]);
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = apply(
        state,
        schematic.setNodePosition({ key: "ghost", position: { x: 9, y: 9 } }),
      );
      expect(out.nodes).toEqual([node("n1", 0, 0)]);
    });
    it("should only move the first matching node when keys are duplicated", () => {
      const state = empty({ nodes: [node("dup", 0, 0), node("dup", 1, 1)] });
      const out = apply(
        state,
        schematic.setNodePosition({ key: "dup", position: { x: 9, y: 9 } }),
      );
      expect(out.nodes[0].position).toEqual({ x: 9, y: 9 });
      expect(out.nodes[1].position).toEqual({ x: 1, y: 1 });
    });
    it("should keep the node's config", () => {
      const state = empty({ nodes: [node("n1", 0, 0, tank("Pump"))] });
      const out = apply(
        state,
        schematic.setNodePosition({ key: "n1", position: { x: 9, y: 9 } }),
      );
      expect(out.nodes[0].config).toBe(state.nodes[0].config);
    });
  });

  describe("setNode", () => {
    it("should append the node to the end of the slice when no node has the same key", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = apply(state, schematic.setNode({ node: node("n2", 1, 2) }));
      expect(out.nodes).toEqual([node("n1", 0, 0), node("n2", 1, 2)]);
    });
    it("should store the config the node carries", () => {
      const out = apply(
        empty(),
        schematic.setNode({
          node: {
            key: "n1",
            position: { x: 0, y: 0 },
            config: {
              variant: "tank",
              label: { label: "Pump" },
              strokeColor: "#ff0000",
            },
          },
        }),
      );
      expect(out.nodes[0].config).toEqual(tank("Pump", "#ff0000"));
    });
    it("should replace an existing node and its config at the same index", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1, tank("Old")), node("n3", 2, 2)],
      });
      const out = apply(
        state,
        schematic.setNode({ node: node("n2", 9, 9, tank("New")) }),
      );
      expect(out.nodes).toEqual([
        node("n1", 0, 0),
        node("n2", 9, 9, tank("New")),
        node("n3", 2, 2),
      ]);
    });
  });

  describe("removeNode", () => {
    it("should remove the matching node and its config", () => {
      const state = empty({
        nodes: [node("n1", 0, 0, tank("Pump")), node("n2", 1, 1, tank("Tank"))],
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.nodes).toEqual([node("n2", 1, 1, tank("Tank"))]);
      expect(schematic.configsOf(out)).toEqual({ n2: tank("Tank") });
    });
    it("should remove the edges that connect to the node", () => {
      const electric = edgeCfg({ variant: "electric" });
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1), node("n3", 2, 2)],
        edges: [
          edge("e1", "n1", "out", "n2", "in"),
          edge("e2", "n3", "out", "n1", "in"),
          edge("e3", "n2", "out", "n3", "in", electric),
        ],
      });
      const { next, targets } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      expect(next.edges).toEqual([edge("e3", "n2", "out", "n3", "in", electric)]);
      expect(targets).toEqual(["n1", "e1", "e2"]);
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({ nodes: [node("n1", 0, 0, tank("Pump"))] });
      const out = apply(state, schematic.removeNode({ key: "ghost" }));
      expect(out).toBe(state);
    });
  });

  describe("removeNode group member cascade", () => {
    it("should remove the deleted node's key from its group's members", () => {
      const state = empty({
        nodes: [
          node("g1", -20, -20, groupConfig(["n1", "n2"])),
          node("n1", 0, 0),
          node("n2", 100, 0),
        ],
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(schematic.configsOf(out).g1).toEqual(groupConfig(["n2"]));
    });

    it("should remove the key from every group that lists it", () => {
      const state = empty({
        nodes: [group("g1", ["n1", "a"]), group("g2", ["b", "n1"]), node("n1", 0, 0)],
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(schematic.configsOf(out).g1).toEqual(groupConfig(["a"]));
      expect(schematic.configsOf(out).g2).toEqual(groupConfig(["b"]));
    });

    it("should remove duplicate entries of the key from a members list", () => {
      const state = empty({
        nodes: [group("g1", ["n1", "n2", "n1"]), node("n1", 0, 0), node("n2", 0, 0)],
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(schematic.configsOf(out).g1).toEqual(groupConfig(["n2"]));
    });

    it("should splice a removed group out of the outer group that lists it", () => {
      const state = empty({
        nodes: [
          group("outer", ["inner", "x"]),
          group("inner", ["m1"]),
          node("m1", 0, 0, tank("Pump")),
        ],
      });
      const out = apply(state, schematic.removeNode({ key: "inner" }));
      expect(schematic.configsOf(out)).toEqual({
        outer: groupConfig(["x"]),
        m1: tank("Pump"),
      });
    });

    it("should leave a former member's config untouched when its group is removed", () => {
      const state = empty({
        nodes: [group("g1", ["m1"]), node("m1", 0, 0, tank("Pump"))],
      });
      const out = apply(state, schematic.removeNode({ key: "g1" }));
      expect(schematic.configsOf(out)).toEqual({ m1: tank("Pump") });
    });

    it("should only affect the removed node's direct group in a nested chain", () => {
      const state = empty({
        nodes: [
          group("outer", ["inner"]),
          group("inner", ["n1", "n2"]),
          node("n1", 0, 0),
        ],
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(schematic.configsOf(out).outer).toEqual(groupConfig(["inner"]));
      expect(schematic.configsOf(out).inner).toEqual(groupConfig(["n2"]));
    });

    it("should only splice the direct group when the node is three groups deep", () => {
      const state = empty({
        nodes: [
          group("outer", ["mid"]),
          group("mid", ["inner"]),
          group("inner", ["n1", "n2"]),
          node("n1", 0, 0),
        ],
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(schematic.configsOf(out).outer).toEqual(groupConfig(["mid"]));
      expect(schematic.configsOf(out).mid).toEqual(groupConfig(["inner"]));
      expect(schematic.configsOf(out).inner).toEqual(groupConfig(["n2"]));
    });

    it("should cascade for each removal in a multi-remove batch", () => {
      const state = empty({
        nodes: [group("g1", ["n1", "n2"]), node("n1", 0, 0), node("n2", 0, 0)],
      });
      const out = apply(
        state,
        schematic.removeNode({ key: "n1" }),
        schematic.removeNode({ key: "n2" }),
      );
      expect(schematic.configsOf(out).g1).toEqual(groupConfig([]));
    });

    it("should preserve the group's other config fields when splicing", () => {
      const state = empty({
        nodes: [group("g1", ["n1", "n2"], { locked: true }), node("n1", 0, 0)],
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(schematic.configsOf(out).g1).toEqual(
        groupConfig(["n2"], { locked: true }),
      );
    });

    it("should leave group configs untouched when the removed node is in no group", () => {
      const state = empty({
        nodes: [group("g1", ["n1"]), node("n1", 0, 0), node("loose", 5, 5)],
      });
      const out = apply(state, schematic.removeNode({ key: "loose" }));
      expect(schematic.configsOf(out).g1).toBe(state.nodes[0].config);
    });

    it("should not cascade on a no-op removal, even for a listed key", () => {
      const state = empty({ nodes: [group("g1", ["n1", "ghost"]), node("n1", 0, 0)] });
      const out = apply(state, schematic.removeNode({ key: "ghost" }));
      expect(schematic.configsOf(out).g1).toEqual(groupConfig(["n1", "ghost"]));
    });

    it("should leave an empty members list untouched", () => {
      const state = empty({ nodes: [group("g1", []), node("n1", 0, 0)] });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(schematic.configsOf(out).g1).toEqual(groupConfig([]));
    });

    it("should drop a self-listing group with its node", () => {
      const state = empty({
        nodes: [group("g1", ["g1", "n1"]), node("n1", 0, 0, tank("Pump"))],
      });
      const out = apply(state, schematic.removeNode({ key: "g1" }));
      expect(schematic.configsOf(out)).toEqual({ n1: tank("Pump") });
    });

    it("should splice a membership cycle when one side is removed", () => {
      const state = empty({ nodes: [group("g1", ["g2"]), group("g2", ["g1"])] });
      const out = apply(state, schematic.removeNode({ key: "g1" }));
      expect(schematic.configsOf(out)).toEqual({ g2: groupConfig([]) });
    });

    it("should converge to the same nodes regardless of removal order", () => {
      const build = () =>
        empty({ nodes: [group("g1", ["n1"]), node("n1", 0, 0, tank("Pump"))] });
      const a = apply(
        build(),
        schematic.removeNode({ key: "n1" }),
        schematic.removeNode({ key: "g1" }),
      );
      const b = apply(
        build(),
        schematic.removeNode({ key: "g1" }),
        schematic.removeNode({ key: "n1" }),
      );
      expect(a.nodes).toEqual([]);
      expect(b.nodes).toEqual([]);
    });

    it("should splice the key from a large fan of groups", () => {
      const nodes = [node("n1", 0, 0)];
      const expected: Record<string, schematic.ElementConfig> = {};
      for (let i = 0; i < 50; i++) {
        const key = `g${i}`;
        nodes.push(group(key, ["n1", "other"]));
        expected[key] = groupConfig(["other"]);
      }
      const out = apply(empty({ nodes }), schematic.removeNode({ key: "n1" }));
      expect(schematic.configsOf(out)).toEqual(expected);
    });

    it("should let a members write applied after the removal win", () => {
      const state = empty({
        nodes: [group("g1", ["n1", "n2"]), node("n1", 0, 0), node("n2", 0, 0)],
      });
      const out = apply(
        state,
        schematic.removeNode({ key: "n1" }),
        set(["nodes", "g1", "config", "members"], ["n1", "n2", "n3"]),
      );
      expect(schematic.configsOf(out).g1).toEqual(groupConfig(["n1", "n2", "n3"]));
    });

    it("should keep the splice when the removal lands after a members write", () => {
      const state = empty({
        nodes: [group("g1", ["n1", "n2"]), node("n1", 0, 0), node("n2", 0, 0)],
      });
      const out = apply(
        state,
        set(["nodes", "g1", "config", "members"], ["n1", "n2", "n3"]),
        schematic.removeNode({ key: "n1" }),
      );
      expect(schematic.configsOf(out).g1).toEqual(groupConfig(["n2", "n3"]));
    });

    it("should report the affected group keys as targets", () => {
      const state = empty({
        nodes: [group("g1", ["n1"]), group("g2", ["other"]), node("n1", 0, 0)],
      });
      const { targets } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      expect(targets).toContain("n1");
      expect(targets).toContain("g1");
      expect(targets).not.toContain("g2");
    });

    it("should report only the removed key as a target when no group lists it", () => {
      const state = empty({ nodes: [group("g1", ["n1"]), node("loose", 0, 0)] });
      const { targets } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "loose" }),
      ]);
      expect(targets).toEqual(["loose"]);
    });

    it("should leave the input state unmodified when cascading", () => {
      const state = empty({ nodes: [group("g1", ["n1", "n2"]), node("n1", 0, 0)] });
      const before = structuredClone(state);
      schematic.reduceAll(state, [schematic.removeNode({ key: "n1" })]);
      expect(state).toEqual(before);
    });
  });

  describe("addEdge", () => {
    it("should append an edge whose key is not yet present", () => {
      const state = empty({
        nodes: [node("a", 0, 0), node("b", 1, 1), node("c", 2, 2)],
        edges: [edge("e1", "a", "o", "b", "i")],
      });
      const electric = edgeCfg({ variant: "electric" });
      const out = apply(
        state,
        schematic.addEdge({ edge: edge("e2", "b", "o", "c", "i", electric) }),
      );
      expect(out.edges).toHaveLength(2);
      expect(out.edges[1]).toEqual(edge("e2", "b", "o", "c", "i", electric));
    });
    it("should be a no-op when an edge with the same key already exists", () => {
      const original = edge("e2", "b", "o", "c", "i");
      const state = empty({
        edges: [
          edge("e1", "a", "o", "b", "i"),
          original,
          edge("e3", "c", "o", "d", "i"),
        ],
      });
      const out = apply(
        state,
        schematic.addEdge({ edge: edge("e2", "x", "y", "z", "w") }),
      );
      expect(out.edges).toHaveLength(3);
      expect(out.edges[0].key).toBe("e1");
      expect(out.edges[1]).toEqual(original);
      expect(out.edges[2].key).toBe("e3");
    });
    it.each([
      ["source", edge("e1", "ghost", "o", "a", "i")],
      ["target", edge("e1", "a", "o", "ghost", "i")],
    ])("should be a no-op when the %s node does not exist", (_, e) => {
      const state = empty({ nodes: [node("a", 0, 0)] });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.addEdge({ edge: e }),
      ]);
      expect(next).toBe(state);
      expect(inverse).toEqual([]);
    });
  });

  describe("removeEdge", () => {
    it("should remove the matching edge and its config", () => {
      const electric = edgeCfg({ variant: "electric" });
      const state = empty({
        edges: [
          edge("e1", "a", "o", "b", "i"),
          edge("e2", "b", "o", "c", "i", electric),
        ],
      });
      const out = apply(state, schematic.removeEdge({ key: "e1" }));
      expect(out.edges).toEqual([edge("e2", "b", "o", "c", "i", electric)]);
    });
    it("should be a no-op when the key does not match any edge", () => {
      const state = empty({ edges: [edge("e1", "a", "o", "b", "i")] });
      const out = apply(state, schematic.removeEdge({ key: "ghost" }));
      expect(out).toBe(state);
    });
  });

  describe("path actions", () => {
    it("should set a field of a node config from its wire form", () => {
      const state = empty({ nodes: [node("n1", 0, 0, tank("Pump"))] });
      const { next, targets } = schematic.reduceAll(state, [
        set(["nodes", "n1", "config", "label", "max_inline_size"], 200),
      ]);
      expect(next.nodes[0].config).toEqual(
        nodeCfg({ variant: "tank", label: { label: "Pump", maxInlineSize: 200 } }),
      );
      expect(targets).toEqual(["n1"]);
    });

    it("should set a field of an edge config", () => {
      const state = empty({
        nodes: [node("a", 0, 0), node("b", 1, 1)],
        edges: [edge("e1", "a", "o", "b", "i")],
      });
      const segments = [{ direction: "x", length: 10 }];
      const { next, targets } = schematic.reduceAll(state, [
        set(["edges", "e1", "config", "segments"], segments),
      ]);
      expect(next.edges[0].config).toEqual(edgeCfg({ variant: "pipe", segments }));
      expect(targets).toEqual(["e1"]);
    });

    it("should replace a node config whole", () => {
      const state = empty({ nodes: [node("n1", 0, 0, tank("Old", "#ff0000"))] });
      const next = tank("New");
      const out = apply(state, set(["nodes", "n1", "config"], next));
      expect(out.nodes[0].config).toEqual(next);
    });

    it("should return a wire-form inverse that restores the config", () => {
      const state = empty({ nodes: [node("n1", 0, 0, tank("Pump"))] });
      const { next, inverse } = schematic.reduceAll(state, [
        set(["nodes", "n1", "config", "label", "max_inline_size"], 200),
      ]);
      expect(inverse).toEqual([
        set(["nodes", "n1", "config", "label", "max_inline_size"], 150),
      ]);
      expect(apply(next, ...inverse)).toEqual(state);
    });

    it("should keep the other configs by identity", () => {
      const state = empty({
        nodes: [node("n1", 0, 0, tank("Pump")), node("n2", 1, 1, tank("Valve"))],
      });
      const out = apply(
        state,
        set(["nodes", "n1", "config", "label", "label"], "Tank"),
      );
      expect(out.nodes[1].config).toBe(state.nodes[1].config);
    });

    it.each([
      ["a set of a field", set(["nodes", "ghost", "config", "label", "label"], "Tank")],
      [
        "a set of the whole config",
        set(["nodes", "ghost", "config"], { variant: "tank" }),
      ],
      [
        "a remove of a field",
        schematic.remove({ path: ["nodes", "ghost", "config", "label"], key: "label" }),
      ],
      [
        "an insert",
        schematic.insert({
          path: ["nodes", "ghost", "config", "tags"],
          value: { key: "a" },
        }),
      ],
      [
        "a move",
        schematic.move({ path: ["nodes", "ghost", "config", "tags"], key: "a" }),
      ],
      ["a set of an edge field", set(["edges", "ghost", "config", "segments"], [])],
    ])(
      "should do nothing for %s of a node or edge that does not exist",
      (_, action) => {
        const state = empty({ nodes: [node("n1", 0, 0, tank("Pump"))] });
        const { next, inverse } = schematic.reduceAll(state, [action]);
        expect(next).toBe(state);
        expect(inverse).toEqual([]);
      },
    );

    it.each([
      ["the node list", "nodes", set(["nodes"], [])],
      ["a whole node", "nodes", set(["nodes", "n1"], node("n1", 0, 0))],
      ["a node's position", "nodes", set(["nodes", "n1", "position"], { x: 1, y: 1 })],
      ["a node's key", "nodes", set(["nodes", "n1", "key"], "n2")],
      ["a node by remove", "nodes", schematic.remove({ path: ["nodes"], key: "n1" })],
      [
        "a node by insert",
        "nodes",
        schematic.insert({ path: ["nodes"], value: node("n2", 0, 0) }),
      ],
      [
        "an edge's source",
        "edges",
        set(["edges", "e1", "source"], { node: "b", param: "o" }),
      ],
    ])("should throw for a path to %s", (_, field, action) => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("b", 1, 1)],
        edges: [edge("e1", "n1", "o", "b", "i")],
      });
      expect(() => apply(state, action)).toThrow(
        `[Schematic] - a path into ${field} must go into the config of one entry`,
      );
    });

    it("should not classify a set of the name as a config edit", () => {
      expect(
        schematic.kindOf([
          set(["name"], "Other"),
          set(["nodes", "n1", "config", "label"], {}),
        ]),
      ).toEqual("transaction");
    });
  });

  describe("setChanged", () => {
    it("should keep an edit to another field that base does not hold", () => {
      const base = tank("Pump");
      const state = empty({ nodes: [node("n1", 0, 0, tank("Pump", "#ff0000"))] });
      const actions = schematic.setChanged({ key: "n1", base, next: tank("Valve") });
      const out = apply(state, ...encoded(state, actions));
      expect(out.nodes[0].config).toEqual(tank("Valve", "#ff0000"));
    });

    it("should name the node, its config, and its variant in each action", () => {
      const actions = schematic.setChanged({
        key: "n1",
        base: tank("Pump"),
        next: tank("Valve"),
      });
      expect(actions).toEqual([
        schematic.set({
          path: ["nodes", "n1", "config", "label", "label"],
          value: "Valve",
          guards: [{ path: ["nodes", "n1", "config", "variant"], value: "tank" }],
        }),
      ]);
    });

    it("should name the edge list for an edge config", () => {
      const segments = [{ direction: "x" as const, length: 5 }];
      const actions = schematic.setChanged({
        key: "e1",
        base: pipe(),
        next: edgeCfg({ variant: "pipe", segments }),
      });
      expect(actions).toEqual([
        schematic.set({
          path: ["edges", "e1", "config", "segments"],
          value: segments,
          guards: [{ path: ["edges", "e1", "config", "variant"], value: "pipe" }],
        }),
      ]);
    });

    it("should remove a field that next does not hold", () => {
      const base = tank("Pump", "#ff0000");
      const state = empty({ nodes: [node("n1", 0, 0, base)] });
      const actions = schematic.setChanged({ key: "n1", base, next: tank("Pump") });
      const out = apply(state, ...encoded(state, actions));
      expect(out.nodes[0].config).toEqual(tank("Pump"));
    });

    it("should replace the config when the symbol changes variant", () => {
      const base = tank("Pump");
      const next = nodeCfg({ variant: "valve" });
      const state = empty({ nodes: [node("n1", 0, 0, base)] });
      const actions = schematic.setChanged({ key: "n1", base, next });
      expect(apply(state, ...encoded(state, actions)).nodes[0].config).toEqual(next);
    });

    it("should list no action when nothing differs", () => {
      const base = tank("Pump");
      expect(schematic.setChanged({ key: "n1", base, next: tank("Pump") })).toEqual([]);
    });
  });

  describe("kindOf", () => {
    const base = tank("Pump");
    const single = schematic.setChanged({ key: "n1", base, next: tank("Valve") });
    const both = schematic.setChanged({
      key: "n1",
      base,
      next: tank("Valve", "#ff0000"),
    });

    it("should classify config edits as one kind for any number of fields", () => {
      expect(schematic.kindOf(single)).toEqual("config");
      expect(schematic.kindOf(both)).toEqual("config");
      expect(schematic.kindOf([set(["nodes", "n1", "config"], base)])).toEqual(
        "config",
      );
    });

    it("should classify an edge config edit as a config edit", () => {
      expect(
        schematic.kindOf([set(["edges", "e1", "config", "segments"], [])]),
      ).toEqual("config");
    });

    it("should classify a config edit that rides on a move as a move", () => {
      const move = schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } });
      expect(schematic.kindOf([move, ...single])).toEqual("move");
    });

    it("should classify a set of the name by its own type", () => {
      expect(schematic.kindOf([set(["name"], "New")])).toEqual("set");
    });

    it("should not classify a set of the name that rides on a move as a move", () => {
      const move = schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } });
      expect(schematic.kindOf([move, set(["name"], "New")])).toEqual("transaction");
    });
  });

  describe("immutability", () => {
    it("should leave the input state object unmodified", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const before = structuredClone(state);
      schematic.reduceAll(state, [
        schematic.setNodePosition({ key: "n1", position: { x: 9, y: 9 } }),
      ]);
      expect(state).toEqual(before);
    });
    it("should return a new state object when any action mutates", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = schematic.reduceAll(state, [
        schematic.setNodePosition({ key: "n1", position: { x: 9, y: 9 } }),
      ]);
      expect(out.next).not.toBe(state);
      expect(out.next.nodes).not.toBe(state.nodes);
    });
    it("should return the same state object when an action is a no-op", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = schematic.reduceAll(state, [
        schematic.setNodePosition({ key: "ghost", position: { x: 9, y: 9 } }),
      ]);
      expect(out.next).toBe(state);
    });
  });

  describe("real-world scenarios", () => {
    it("should converge to the final position after a 30-action drag storm", () => {
      const state = empty({ nodes: [node("pump", 0, 0)] });
      const actions: schematic.Action[] = [];
      for (let i = 0; i < 30; i++)
        actions.push(
          schematic.setNodePosition({ key: "pump", position: { x: i, y: i * 2 } }),
        );
      const out = schematic.reduceAll(state, actions).next;
      expect(out.nodes[0].position).toEqual({ x: 29, y: 58 });
    });

    it("should build a complete graph from an empty schematic", () => {
      const electric = edgeCfg({ variant: "electric" });
      const out = apply(
        empty(),
        schematic.setNode({ node: node("pump", 0, 0, tank("Main Pump")) }),
        schematic.setNode({ node: node("valve", 100, 0) }),
        schematic.setNode({ node: node("tank", 200, 0) }),
        schematic.addEdge({ edge: edge("e1", "pump", "out", "valve", "in") }),
        schematic.addEdge({ edge: edge("e2", "valve", "out", "tank", "in", electric) }),
      );
      expect(schematic.configsOf(out)).toEqual({
        pump: tank("Main Pump"),
        valve: vent(),
        tank: vent(),
        e1: pipe(),
        e2: electric,
      });
    });

    it("should drop the edges of a node that is removed and re-added", () => {
      const state = empty({
        nodes: [node("n1", 0, 0, tank("v1")), node("n2", 1, 1)],
        edges: [edge("e1", "n1", "o", "n2", "i")],
      });
      const out = apply(
        state,
        schematic.removeNode({ key: "n1" }),
        schematic.setNode({ node: node("n1", 50, 50) }),
      );
      expect(out.nodes).toEqual([node("n2", 1, 1), node("n1", 50, 50)]);
      expect(out.edges).toEqual([]);
    });

    it("should converge an idempotent action sequence to the same state as a single application", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const a = schematic.setNodePosition({ key: "n1", position: { x: 10, y: 20 } });
      expect(schematic.reduceAll(state, [a]).next).toEqual(
        schematic.reduceAll(state, [a, a, a]).next,
      );
    });

    it("should apply a 50-action editor session and converge to a coherent schematic", () => {
      const state = empty();
      const actions: schematic.Action[] = [];
      for (let i = 0; i < 5; i++)
        actions.push(schematic.setNode({ node: node(`n${i}`, i * 100, 0) }));
      for (let i = 0; i < 5; i++) {
        actions.push(
          schematic.setNodePosition({ key: `n${i}`, position: { x: i * 100, y: 50 } }),
        );
        actions.push(
          schematic.setNodePosition({ key: `n${i}`, position: { x: i * 100, y: 100 } }),
        );
      }
      for (let i = 0; i < 4; i++)
        actions.push(
          schematic.addEdge({
            edge: edge(`e${i}`, `n${i}`, "out", `n${i + 1}`, "in"),
          }),
        );
      for (let i = 0; i < 3; i++)
        actions.push(set(["nodes", `n${i}`, "config"], tank(`node ${i}`)));
      actions.push(set(["edges", "e1", "config"], edgeCfg({ variant: "electric" })));
      const out = schematic.reduceAll(state, actions).next;
      expect(out.nodes).toHaveLength(5);
      expect(out.nodes[0].position).toEqual({ x: 0, y: 100 });
      expect(out.nodes[4].position).toEqual({ x: 400, y: 100 });
      expect(out.edges).toHaveLength(4);
      expect(out.nodes.map((n) => n.config.variant)).toEqual([
        "tank",
        "tank",
        "tank",
        "vent",
        "vent",
      ]);
      expect(out.edges.map((e) => e.config.variant)).toEqual([
        "pipe",
        "electric",
        "pipe",
        "pipe",
      ]);
    });

    it("should leave state untouched when given an empty action list", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expect(schematic.reduceAll(state, []).next).toBe(state);
    });
  });

  describe("zod parsing", () => {
    it("should reject an action with an unknown discriminator with a ZodError", () => {
      expect(() =>
        schematic.actionZ.parse({ type: "unknown", payload: { foo: 1 } }),
      ).toThrow(z.ZodError);
    });

    it("should reject a setNodePosition action missing required fields with a ZodError", () => {
      expect(() =>
        schematic.actionZ.parse({ type: "set_node_position", setNodePosition: {} }),
      ).toThrow(z.ZodError);
    });

    it("should reject a setNode action whose node holds no config", () => {
      expect(() =>
        schematic.actionZ.parse({
          type: "set_node",
          setNode: { node: { key: "n1", position: { x: 0, y: 0 } } },
        }),
      ).toThrow(z.ZodError);
    });

    it("should reject a setNode action whose config names no node variant", () => {
      expect(() =>
        schematic.actionZ.parse({
          type: "set_node",
          setNode: {
            node: { key: "n1", position: { x: 0, y: 0 }, config: { variant: "pipe" } },
          },
        }),
      ).toThrow(z.ZodError);
    });

    it("should accept a fully populated setNodePosition action", () => {
      const a = schematic.setNodePosition({ key: "n1", position: { x: 1, y: 2 } });
      expect(schematic.actionZ.parse(a)).toEqual(a);
    });
  });
});

describe("schematic reducer inverses", () => {
  const expectRoundTrip = (state: schematic.Schematic, actions: schematic.Action[]) => {
    const { next, inverse } = schematic.reduceAll(state, actions);
    const restored = schematic.reduceAll(next, inverse).next;
    expect(restored).toEqual(state);
  };

  describe("set of the name", () => {
    it("should invert to restore the prior name", () => {
      const state = empty({ name: "old" });
      expectRoundTrip(state, [set(["name"], "new")]);
    });
    it("should round-trip when setting the same name", () => {
      const state = empty({ name: "same" });
      expectRoundTrip(state, [set(["name"], "same")]);
    });
    it("should target the name so concurrent sets of it invalidate each other", () => {
      const state = empty({ name: "a" });
      const { targets } = schematic.reduceAll(state, [set(["name"], "b")]);
      expect(targets).toEqual(["name"]);
    });
  });

  describe("setNodePosition", () => {
    it("should invert to restore the prior position", () => {
      const state = empty({ nodes: [node("n1", 1, 2)] });
      expectRoundTrip(state, [
        schematic.setNodePosition({ key: "n1", position: { x: 99, y: 100 } }),
      ]);
    });
    it("should produce an empty inverse for a no-op", () => {
      const state = empty({ nodes: [node("n1", 1, 2)] });
      const { inverse } = schematic.reduceAll(state, [
        schematic.setNodePosition({ key: "ghost", position: { x: 9, y: 9 } }),
      ]);
      expect(inverse).toEqual([]);
    });
  });

  describe("setNode", () => {
    it("should invert an insert with a removeNode", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expectRoundTrip(state, [
        schematic.setNode({ node: node("n2", 5, 5, tank("x")) }),
      ]);
    });
    it("should invert a replace with a setNode of the prior node and config", () => {
      const state = empty({ nodes: [node("n1", 1, 1, tank("Old", "#ff0000"))] });
      const { inverse } = schematic.reduceAll(state, [
        schematic.setNode({ node: node("n1", 9, 9, tank("New")) }),
      ]);
      expect(inverse).toEqual([schematic.setNode({ node: state.nodes[0] })]);
      expectRoundTrip(state, [
        schematic.setNode({ node: node("n1", 9, 9, tank("New")) }),
      ]);
    });
  });

  describe("removeNode", () => {
    it("should invert by re-inserting the node and its config (order not preserved)", () => {
      // setNode appends rather than inserting at a specific index, so a remove + undo
      // cycle can rearrange the nodes array. The contents are restored but the original
      // index is lost. Would be addressed by an InsertNode(node, idx) action.
      const state = empty({
        nodes: [node("n1", 0, 0, tank("Pump")), node("n2", 1, 1, tank("Tank"))],
      });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(byKey(restored.nodes)).toEqual(byKey(state.nodes));
    });
    it("should invert a single-node removal cleanly when order is unambiguous", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expectRoundTrip(state, [schematic.removeNode({ key: "n1" })]);
    });
    it("should produce an empty inverse for a no-op removal", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const { inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "ghost" }),
      ]);
      expect(inverse).toEqual([]);
    });
    it("should restore the connected edges and their configs", () => {
      const state = empty({
        nodes: [node("n2", 1, 1), node("n1", 0, 0, tank("Pump"))],
        edges: [edge("e1", "n1", "out", "n2", "in", edgeCfg({ variant: "electric" }))],
      });
      expectRoundTrip(state, [schematic.removeNode({ key: "n1" })]);
    });
    it("should restore an edge between two nodes removed in one batch", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1)],
        edges: [edge("e1", "n1", "out", "n2", "in")],
      });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
        schematic.removeNode({ key: "n2" }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(restored.edges).toEqual(state.edges);
    });
  });

  describe("removeNode group member cascade", () => {
    it("should restore the group's members with a guarded set", () => {
      const state = empty({ nodes: [group("g1", ["a", "n1", "b"]), node("n1", 0, 0)] });
      const { inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      expect(inverse).toEqual([
        schematic.setNode({ node: node("n1", 0, 0) }),
        schematic.set({
          path: ["nodes", "g1", "config", "members"],
          value: ["a", "n1", "b"],
          guards: [{ path: ["nodes", "g1", "config", "variant"], value: "group_box" }],
        }),
      ]);
    });

    it("should leave a group that changed variant before the undo alone", () => {
      const state = empty({ nodes: [group("g1", ["n1"]), node("n1", 0, 0)] });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      const changed = apply(
        next,
        schematic.setNode({ node: node("g1", 0, 0, tank("T")) }),
      );
      const restored = apply(changed, ...inverse);
      expect(schematic.configsOf(restored)).toEqual({ g1: tank("T"), n1: vent() });
    });

    it("should invert by restoring the node and its membership", () => {
      const state = empty({
        nodes: [node("g1", -20, -20, groupConfig(["n1"])), node("n1", 0, 0)],
      });
      expectRoundTrip(state, [schematic.removeNode({ key: "n1" })]);
    });

    it("should restore members in their original order", () => {
      const state = empty({ nodes: [group("g1", ["a", "n1", "b"]), node("n1", 0, 0)] });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(schematic.configsOf(restored).g1).toEqual(groupConfig(["a", "n1", "b"]));
    });

    it("should restore duplicate member entries on undo", () => {
      const state = empty({ nodes: [group("g1", ["n1", "n1"]), node("n1", 0, 0)] });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(schematic.configsOf(restored).g1).toEqual(groupConfig(["n1", "n1"]));
    });

    it("should restore membership in every group that listed the key", () => {
      const state = empty({
        nodes: [group("g1", ["n1", "a"]), group("g2", ["b", "n1"]), node("n1", 0, 0)],
      });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(schematic.configsOf(restored).g1).toEqual(groupConfig(["n1", "a"]));
      expect(schematic.configsOf(restored).g2).toEqual(groupConfig(["b", "n1"]));
    });

    it("should restore full membership after undoing a multi-remove batch", () => {
      const state = empty({
        nodes: [group("g1", ["n1", "n2"]), node("n1", 0, 0), node("n2", 0, 0)],
      });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
        schematic.removeNode({ key: "n2" }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(byKey(restored.nodes)).toEqual(byKey(state.nodes));
    });

    it("should splice the member again when the removal is reapplied after undo", () => {
      const state = empty({ nodes: [group("g1", ["n1", "n2"]), node("n1", 0, 0)] });
      const action = schematic.removeNode({ key: "n1" });
      const { next, inverse } = schematic.reduceAll(state, [action]);
      const restored = schematic.reduceAll(next, inverse).next;
      const redone = schematic.reduceAll(restored, [action]).next;
      expect(schematic.configsOf(redone)).toEqual({ g1: groupConfig(["n2"]) });
    });
  });

  describe("addEdge / removeEdge", () => {
    it("should invert addEdge with removeEdge", () => {
      const state = empty({ nodes: [node("a", 0, 0), node("b", 1, 1)] });
      expectRoundTrip(state, [
        schematic.addEdge({ edge: edge("e1", "a", "o", "b", "i") }),
      ]);
    });
    it("should invert removeEdge with addEdge and restore its config", () => {
      const state = empty({
        nodes: [node("a", 0, 0), node("b", 1, 1)],
        edges: [edge("e1", "a", "o", "b", "i", edgeCfg({ variant: "electric" }))],
      });
      expectRoundTrip(state, [schematic.removeEdge({ key: "e1" })]);
    });
    it("should produce an empty inverse for a duplicate addEdge", () => {
      const state = empty({
        nodes: [node("a", 0, 0), node("b", 1, 1)],
        edges: [edge("e1", "a", "o", "b", "i")],
      });
      const { inverse } = schematic.reduceAll(state, [
        schematic.addEdge({ edge: edge("e1", "b", "o", "a", "i") }),
      ]);
      expect(inverse).toEqual([]);
    });
  });

  describe("set of a whole config", () => {
    it("should restore the prior config whole", () => {
      const state = empty({ nodes: [node("n1", 0, 0, tank("Old", "#ff0000"))] });
      const { next, inverse } = schematic.reduceAll(state, [
        set(["nodes", "n1", "config"], { variant: "tank", label: { label: "New" } }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(restored.nodes[0].config).toEqual(tank("Old", "#ff0000"));
    });
  });

  describe("multi-action transactions", () => {
    it("should invert a build sequence", () => {
      expectRoundTrip(empty(), [
        schematic.setNode({ node: node("pump", 0, 0) }),
        schematic.setNode({ node: node("valve", 100, 0) }),
        schematic.addEdge({ edge: edge("e1", "pump", "out", "valve", "in") }),
      ]);
    });
    it("should invert a multi-step move to the original positions", () => {
      const state = empty({ nodes: [node("n1", 0, 0), node("n2", 1, 1)] });
      expectRoundTrip(state, [
        schematic.setNodePosition({ key: "n1", position: { x: 10, y: 20 } }),
        schematic.setNodePosition({ key: "n2", position: { x: 30, y: 40 } }),
      ]);
    });
    it("should invert a remove + re-add by restoring the original schematic", () => {
      const state = empty({ nodes: [node("n1", 0, 0, tank("Pump"))] });
      expectRoundTrip(state, [
        schematic.removeNode({ key: "n1" }),
        schematic.setNode({ node: node("n1", 50, 50, tank("Pump")) }),
      ]);
    });
  });

  describe("single-action reduce against a draft", () => {
    it("should return the inverse and mutate the draft via schematic.reduceAll", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.setNodePosition({ key: "n1", position: { x: 10, y: 20 } }),
      ]);
      expect(next.nodes[0].position).toEqual({ x: 10, y: 20 });
      expect(inverse).toEqual([
        schematic.setNodePosition({ key: "n1", position: { x: 0, y: 0 } }),
      ]);
    });
  });

  describe("coalesced config edit undo (regression)", () => {
    it("should restore the original config after two rapid config edits", () => {
      const level = (l: string) =>
        nodeCfg({ variant: "tank", label: { level: l }, strokeColor: "#ff0000" });
      const initial = empty({ nodes: [node("n1", 0, 0, level("p"))] });
      const edit = (state: schematic.Schematic, from: string, to: string) =>
        schematic.reduceAll(
          state,
          encoded(
            state,
            schematic.setChanged({ key: "n1", base: level(from), next: level(to) }),
          ),
        );
      const r1 = edit(initial, "p", "h2");
      const r2 = edit(r1.next, "h2", "h3");
      // Matches pushOnto's coalescing: inverse = [next.inverse, ...top.inverse]
      const mergedInverse = [...r2.inverse, ...r1.inverse];
      const restored = schematic.reduceAll(r2.next, mergedInverse).next;
      expect(restored.nodes[0].config).toEqual(initial.nodes[0].config);
    });
  });
});

describe("group and project patches", () => {
  const G1 = "11111111-1111-4111-8111-111111111111";
  const G2 = "22222222-2222-4222-8222-222222222222";
  it("should move the document into the group", () => {
    const doc = empty();
    expect(
      schematic.reduceAll(doc, [schematic.set({ path: ["group"], value: G1 })]).next
        .group,
    ).toEqual(G1);
  });
  it("should move the document out of its group", () => {
    const doc = empty({ group: G1 });
    expect(
      schematic.reduceAll(doc, [schematic.remove({ key: "group" })]).next.group,
    ).toBeUndefined();
  });
  it("should restore the previous group through its inverse", () => {
    const doc = empty({ group: G1 });
    const { next, inverse } = schematic.reduceAll(doc, [
      schematic.set({ path: ["group"], value: G2 }),
    ]);
    expect(schematic.reduceAll(next, inverse).next.group).toEqual(G1);
  });
  it("should move the document into the project", () => {
    const doc = empty({ project: G1 });
    expect(
      schematic.reduceAll(doc, [schematic.set({ path: ["project"], value: G2 })]).next
        .project,
    ).toEqual(G2);
  });
});
