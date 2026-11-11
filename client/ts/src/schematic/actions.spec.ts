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

const node = (key: string, x: number, y: number): schematic.Node => ({
  key,
  position: { x, y },
  zIndex: 0,
});

const edge = (
  key: string,
  srcNode: string,
  srcParam: string,
  tgtNode: string,
  tgtParam: string,
  style: Partial<schematic.Edge> = {},
): schematic.Edge =>
  schematic.edgeZ.parse({
    key,
    source: { node: srcNode, param: srcParam },
    target: { node: tgtNode, param: tgtParam },
    ...style,
  });

const empty = (overrides: Partial<schematic.Schematic> = {}): schematic.Schematic =>
  schematic.schematicZ.parse({ name: "Test Schematic", ...overrides });

const cfg = (raw: unknown): schematic.NodeConfig => schematic.nodeConfigZ.parse(raw);

const groupConfig = (
  members: string[],
  extra: Partial<schematic.GroupBoxNodeConfig> = {},
): schematic.NodeConfig => cfg({ variant: "group_box", members, ...extra });

const tank = (label: string, strokeColor?: unknown): schematic.NodeConfig =>
  cfg({
    variant: "tank",
    label: { label },
    ...(strokeColor != null && { strokeColor }),
  });

const apply = (
  state: schematic.Schematic,
  ...actions: schematic.Action[]
): schematic.Schematic => schematic.reduceAll(state, actions).next;

describe("schematic reducer", () => {
  describe("set of the name", () => {
    it("should set the schematic name to the value", () => {
      const state = empty({ name: "old" });
      const out = apply(state, schematic.set({ path: ["name"], value: "new" }));
      expect(out.name).toEqual("new");
    });
    it("should leave nodes, edges, and configs untouched", () => {
      const state = empty({
        name: "old",
        nodes: [node("n1", 0, 0)],
        edges: [edge("e1", "a", "o", "b", "i")],
        configs: { n1: tank("Pump") },
      });
      const out = apply(state, schematic.set({ path: ["name"], value: "new" }));
      expect(out.nodes).toEqual(state.nodes);
      expect(out.edges).toEqual(state.edges);
      expect(out.configs).toEqual(state.configs);
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
  });

  describe("setNode", () => {
    it("should append the node to the end of the slice when no node has the same key", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = apply(state, schematic.setNode({ node: node("n2", 1, 2) }));
      expect(out.nodes).toEqual([node("n1", 0, 0), node("n2", 1, 2)]);
    });
    it("should write config under the node's key when config is non-undefined", () => {
      const out = apply(
        empty(),
        schematic.setNode({
          node: node("n1", 0, 0),
          config: { variant: "tank", label: { label: "Pump" }, strokeColor: "#ff0000" },
        }),
      );
      expect(out.configs).toEqual({ n1: tank("Pump", "#ff0000") });
    });
    it("should leave configs untouched when the action's config is undefined", () => {
      const out = apply(empty(), schematic.setNode({ node: node("n1", 0, 0) }));
      expect(out.configs).toEqual({});
    });
    it("should replace an existing node in place when the key already exists, preserving slice index", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1), node("n3", 2, 2)],
      });
      const out = apply(state, schematic.setNode({ node: node("n2", 9, 9) }));
      expect(out.nodes).toHaveLength(3);
      expect(out.nodes[0]).toEqual(node("n1", 0, 0));
      expect(out.nodes[1]).toEqual(node("n2", 9, 9));
      expect(out.nodes[2]).toEqual(node("n3", 2, 2));
    });
  });

  describe("removeNode", () => {
    it("should remove the matching node and any config stored under its key", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1)],
        configs: { n1: tank("Pump"), n2: tank("Tank") },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.nodes).toEqual([node("n2", 1, 1)]);
      expect(out.configs).toEqual({ n2: tank("Tank") });
    });
    it("should remove the edges that connect to the node", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1), node("n3", 2, 2)],
        edges: [
          edge("e1", "n1", "out", "n2", "in"),
          edge("e2", "n3", "out", "n1", "in"),
          edge("e3", "n2", "out", "n3", "in", { variant: "electric" }),
        ],
      });
      const { next, targets } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      expect(next.edges).toEqual([
        edge("e3", "n2", "out", "n3", "in", { variant: "electric" }),
      ]);
      expect(targets).toEqual(["n1", "e1", "e2"]);
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({
        nodes: [node("n1", 0, 0)],
        configs: { n1: tank("Pump"), ghost: tank("Orphan") },
      });
      const out = apply(state, schematic.removeNode({ key: "ghost" }));
      expect(out.nodes).toEqual(state.nodes);
      expect(out.configs).toEqual(state.configs);
    });
  });

  describe("removeNode group member cascade", () => {
    it("should remove the deleted node's key from its group's members", () => {
      const state = empty({
        nodes: [node("g1", -20, -20), node("n1", 0, 0), node("n2", 100, 0)],
        configs: { g1: groupConfig(["n1", "n2"]) },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.configs.g1).toEqual(groupConfig(["n2"]));
    });

    it("should remove the key from every group that lists it", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("g2", 0, 0), node("n1", 0, 0)],
        configs: {
          g1: groupConfig(["n1", "a"]),
          g2: groupConfig(["b", "n1"]),
        },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.configs.g1).toEqual(groupConfig(["a"]));
      expect(out.configs.g2).toEqual(groupConfig(["b"]));
    });

    it("should remove duplicate entries of the key from a members list", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0), node("n2", 0, 0)],
        configs: { g1: groupConfig(["n1", "n2", "n1"]) },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.configs.g1).toEqual(groupConfig(["n2"]));
    });

    it("should splice a removed group out of the outer group that lists it", () => {
      const state = empty({
        nodes: [node("outer", 0, 0), node("inner", 0, 0), node("m1", 0, 0)],
        configs: {
          outer: groupConfig(["inner", "x"]),
          inner: groupConfig(["m1"]),
        },
      });
      const out = apply(state, schematic.removeNode({ key: "inner" }));
      expect(out.configs.outer).toEqual(groupConfig(["x"]));
      expect(out.configs.inner).toBeUndefined();
      expect(out.configs.m1).toBeUndefined();
    });

    it("should leave a former member's config untouched when its group is removed", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("m1", 0, 0)],
        configs: { g1: groupConfig(["m1"]), m1: tank("Pump") },
      });
      const out = apply(state, schematic.removeNode({ key: "g1" }));
      expect(out.configs.m1).toEqual(tank("Pump"));
      expect(out.configs.g1).toBeUndefined();
    });

    it("should only affect the removed node's direct group in a nested chain", () => {
      const state = empty({
        nodes: [node("outer", 0, 0), node("inner", 0, 0), node("n1", 0, 0)],
        configs: {
          outer: groupConfig(["inner"]),
          inner: groupConfig(["n1", "n2"]),
        },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.configs.outer).toEqual(groupConfig(["inner"]));
      expect(out.configs.inner).toEqual(groupConfig(["n2"]));
    });

    it("should only splice the direct group when the node is three groups deep", () => {
      const state = empty({
        nodes: [
          node("outer", 0, 0),
          node("mid", 0, 0),
          node("inner", 0, 0),
          node("n1", 0, 0),
        ],
        configs: {
          outer: groupConfig(["mid"]),
          mid: groupConfig(["inner"]),
          inner: groupConfig(["n1", "n2"]),
        },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.configs.outer).toEqual(groupConfig(["mid"]));
      expect(out.configs.mid).toEqual(groupConfig(["inner"]));
      expect(out.configs.inner).toEqual(groupConfig(["n2"]));
    });

    it("should cascade for each removal in a multi-remove batch", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0), node("n2", 0, 0)],
        configs: { g1: groupConfig(["n1", "n2"]) },
      });
      const out = apply(
        state,
        schematic.removeNode({ key: "n1" }),
        schematic.removeNode({ key: "n2" }),
      );
      expect(out.configs.g1).toEqual(groupConfig([]));
    });

    it("should preserve the group's other config fields when splicing", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: groupConfig(["n1", "n2"], { locked: true }) },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.configs.g1).toEqual(groupConfig(["n2"], { locked: true }));
    });

    it("should leave group configs untouched when the removed node is in no group", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0), node("loose", 5, 5)],
        configs: { g1: groupConfig(["n1"]) },
      });
      const out = apply(state, schematic.removeNode({ key: "loose" }));
      expect(out.configs.g1).toEqual(groupConfig(["n1"]));
    });

    it("should not cascade on a no-op removal, even for a listed key", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: groupConfig(["n1", "ghost"]) },
      });
      const out = apply(state, schematic.removeNode({ key: "ghost" }));
      expect(out.configs.g1).toEqual(groupConfig(["n1", "ghost"]));
    });

    it("should leave an empty members list untouched", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: groupConfig([]) },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.configs.g1).toEqual(groupConfig([]));
    });

    it("should drop a self-listing group's config with the node", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: groupConfig(["g1", "n1"]), n1: tank("Pump") },
      });
      const out = apply(state, schematic.removeNode({ key: "g1" }));
      expect(out.configs).toEqual({ n1: tank("Pump") });
    });

    it("should splice a membership cycle when one side is removed", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("g2", 0, 0)],
        configs: { g1: groupConfig(["g2"]), g2: groupConfig(["g1"]) },
      });
      const out = apply(state, schematic.removeNode({ key: "g1" }));
      expect(out.configs).toEqual({ g2: groupConfig([]) });
    });

    it("should converge to the same configs regardless of removal order", () => {
      const build = () =>
        empty({
          nodes: [node("g1", 0, 0), node("n1", 0, 0)],
          configs: { g1: groupConfig(["n1"]), n1: tank("Pump") },
        });
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
      expect(a.configs).toEqual(b.configs);
      expect(a.configs).toEqual({});
      expect(a.nodes).toEqual([]);
      expect(b.nodes).toEqual([]);
    });

    it("should splice the key from a large fan of groups", () => {
      const nodes = [node("n1", 0, 0)];
      const configs: Record<string, schematic.NodeConfig> = {};
      const expected: Record<string, schematic.NodeConfig> = {};
      for (let i = 0; i < 50; i++) {
        const key = `g${i}`;
        nodes.push(node(key, 0, 0));
        configs[key] = groupConfig(["n1", "other"]);
        expected[key] = groupConfig(["other"]);
      }
      const out = apply(empty({ nodes, configs }), schematic.removeNode({ key: "n1" }));
      expect(out.configs).toEqual(expected);
    });

    it("should let a members write applied after the removal win", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0), node("n2", 0, 0)],
        configs: { g1: groupConfig(["n1", "n2"]) },
      });
      const out = apply(
        state,
        schematic.removeNode({ key: "n1" }),
        schematic.setConfig({ key: "g1", config: groupConfig(["n1", "n2", "n3"]) }),
      );
      expect(out.configs.g1).toEqual(groupConfig(["n1", "n2", "n3"]));
    });

    it("should keep the splice when the removal lands after a members write", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0), node("n2", 0, 0)],
        configs: { g1: groupConfig(["n1", "n2"]) },
      });
      const out = apply(
        state,
        schematic.setConfig({ key: "g1", config: groupConfig(["n1", "n2", "n3"]) }),
        schematic.removeNode({ key: "n1" }),
      );
      expect(out.configs.g1).toEqual(groupConfig(["n2", "n3"]));
    });

    it("should report the affected group keys as targets", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("g2", 0, 0), node("n1", 0, 0)],
        configs: { g1: groupConfig(["n1"]), g2: groupConfig(["other"]) },
      });
      const { targets } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      expect(targets).toContain("n1");
      expect(targets).toContain("g1");
      expect(targets).not.toContain("g2");
    });

    it("should report only the removed key as a target when no group lists it", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("loose", 0, 0)],
        configs: { g1: groupConfig(["n1"]) },
      });
      const { targets } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "loose" }),
      ]);
      expect(targets).toEqual(["loose"]);
    });

    it("should leave the input state unmodified when cascading", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: groupConfig(["n1", "n2"]) },
      });
      const before = structuredClone(state);
      schematic.reduceAll(state, [schematic.removeNode({ key: "n1" })]);
      expect(state).toEqual(before);
    });
  });

  describe("setEdge", () => {
    it("should append an edge whose key is not yet present", () => {
      const state = empty({
        nodes: [node("a", 0, 0), node("b", 1, 1), node("c", 2, 2)],
        edges: [edge("e1", "a", "o", "b", "i")],
      });
      const out = apply(
        state,
        schematic.addEdge({ edge: edge("e2", "b", "o", "c", "i") }),
      );
      expect(out.edges).toHaveLength(2);
      expect(out.edges[1].key).toBe("e2");
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
    it("should remove the matching edge", () => {
      const state = empty({
        edges: [edge("e1", "a", "o", "b", "i"), edge("e2", "b", "o", "c", "i")],
      });
      const out = apply(state, schematic.removeEdge({ key: "e1" }));
      expect(out.edges).toEqual([edge("e2", "b", "o", "c", "i")]);
    });
    it("should be a no-op when the key does not match any edge", () => {
      const state = empty({ edges: [edge("e1", "a", "o", "b", "i")] });
      const out = apply(state, schematic.removeEdge({ key: "ghost" }));
      expect(out).toBe(state);
    });
  });

  describe("setEdge", () => {
    const nodes = [node("a", 0, 0), node("b", 1, 1)];
    it("should append an edge whose key is not yet present", () => {
      const styled = edge("e1", "a", "o", "b", "i", {
        variant: "hydraulic",
        strokeWidth: 6,
      });
      const out = apply(empty({ nodes }), schematic.setEdge({ edge: styled }));
      expect(out.edges).toEqual([styled]);
    });
    it("should replace the edge with the same key in place", () => {
      const styled = edge("e1", "a", "o", "b", "i", {
        segments: [{ direction: "x", length: 40 }],
      });
      const state = empty({
        nodes,
        edges: [edge("e1", "a", "o", "b", "i"), edge("e2", "b", "o", "a", "i")],
      });
      const out = apply(state, schematic.setEdge({ edge: styled }));
      expect(out.edges).toEqual([styled, edge("e2", "b", "o", "a", "i")]);
    });
    it.each([
      ["source", edge("e1", "ghost", "o", "a", "i")],
      ["target", edge("e1", "a", "o", "ghost", "i")],
    ])("should be a no-op when the %s node does not exist", (_, e) => {
      const state = empty({ nodes });
      expect(apply(state, schematic.setEdge({ edge: e }))).toBe(state);
    });
  });

  describe("setConfig", () => {
    it("should write the config entry under the given key", () => {
      const out = apply(
        empty({ nodes: [node("n1", 0, 0)] }),
        schematic.setConfig({
          key: "n1",
          config: { variant: "tank", label: { label: "Pump" } },
        }),
      );
      expect(out.configs).toEqual({ n1: tank("Pump") });
    });
    it("should replace an existing config entry whole", () => {
      const state = empty({
        nodes: [node("n1", 0, 0)],
        configs: { n1: tank("Old", "#ff0000") },
      });
      const out = apply(
        state,
        schematic.setConfig({
          key: "n1",
          config: { variant: "tank", label: { label: "New" } },
        }),
      );
      expect(out.configs).toEqual({ n1: tank("New") });
    });
    it("should replace an entry with one of another variant", () => {
      const state = empty({ nodes: [node("n1", 0, 0)], configs: { n1: tank("Pump") } });
      const out = apply(
        state,
        schematic.setConfig({ key: "n1", config: { variant: "cap" } }),
      );
      expect(out.configs).toEqual({ n1: cfg({ variant: "cap" }) });
    });
    it("should be a no-op when no node has the key", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.setConfig({
          key: "ghost",
          config: { variant: "tank", label: { label: "Floating" } },
        }),
      ]);
      expect(next).toBe(state);
      expect(inverse).toEqual([]);
    });
  });

  describe("path actions", () => {
    const set = (path: string[], value: unknown) => schematic.set({ path, value });

    it("should set a field of an element config from its wire form", () => {
      const state = empty({ nodes: [node("n1", 0, 0)], configs: { n1: tank("Pump") } });
      const { next, targets } = schematic.reduceAll(state, [
        set(["configs", "n1", "label", "max_inline_size"], 200),
      ]);
      expect(next.configs.n1).toEqual(
        cfg({ variant: "tank", label: { label: "Pump", maxInlineSize: 200 } }),
      );
      expect(targets).toEqual(["n1"]);
    });

    it("should return a wire-form inverse that restores the config", () => {
      const state = empty({ nodes: [node("n1", 0, 0)], configs: { n1: tank("Pump") } });
      const { next, inverse } = schematic.reduceAll(state, [
        set(["configs", "n1", "label", "max_inline_size"], 200),
      ]);
      expect(inverse).toEqual([
        schematic.set({
          path: ["configs", "n1", "label", "max_inline_size"],
          value: 150,
        }),
      ]);
      expect(apply(next, ...inverse)).toEqual(state);
    });

    it("should keep the other element configs by identity", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1)],
        configs: { n1: tank("Pump"), n2: tank("Valve") },
      });
      const out = apply(state, set(["configs", "n1", "label", "label"], "Tank"));
      expect(out.configs.n2).toBe(state.configs.n2);
    });

    it("should remove an element config", () => {
      const state = empty({ nodes: [node("n1", 0, 0)], configs: { n1: tank("Pump") } });
      const out = apply(state, schematic.remove({ path: ["configs"], key: "n1" }));
      expect(out.configs).toEqual({});
    });

    // An orphan config can exist in a schematic stored before the reducers removed
    // configs with their elements.
    it.each([
      ["a set of a field", set(["configs", "ghost", "label", "label"], "Tank")],
      ["a set of the whole config", set(["configs", "ghost"], { variant: "tank" })],
      [
        "a remove of a field",
        schematic.remove({ path: ["configs", "ghost", "label"], key: "label" }),
      ],
      [
        "a remove of the whole config",
        schematic.remove({ path: ["configs"], key: "ghost" }),
      ],
      [
        "an insert",
        schematic.insert({ path: ["configs", "ghost", "tags"], value: { key: "a" } }),
      ],
      ["a move", schematic.move({ path: ["configs", "ghost", "tags"], key: "a" })],
    ])(
      "should do nothing for %s of a node or edge that does not exist",
      (_, action) => {
        const state = empty({
          nodes: [node("n1", 0, 0)],
          configs: { ghost: tank("Orphan") },
        });
        const { next, inverse } = schematic.reduceAll(state, [action]);
        expect(next).toBe(state);
        expect(inverse).toEqual([]);
      },
    );

    it("should throw for a path to the element configs map", () => {
      expect(() => apply(empty(), set(["configs"], {}))).toThrow(
        "configs is not inside an entry of configs",
      );
    });

    it("should not classify a set of the name as a config edit", () => {
      expect(
        schematic.kindOf([set(["name"], "Other"), set(["configs", "n1", "label"], {})]),
      ).toEqual("transaction");
    });
  });

  describe("setChanged", () => {
    const encoded = (state: schematic.Schematic, actions: schematic.Action[]) =>
      actions.map((a) => schematic.encode(state, a));

    it("should keep an edit to another field that base does not hold", () => {
      const base = tank("Pump");
      const state = empty({
        nodes: [node("n1", 0, 0)],
        configs: { n1: tank("Pump", "#ff0000") },
      });
      const actions = schematic.setChanged({ key: "n1", base, next: tank("Valve") });
      const out = apply(state, ...encoded(state, actions));
      expect(out.configs.n1).toEqual(tank("Valve", "#ff0000"));
    });

    it("should name the entry and its variant in each action", () => {
      const actions = schematic.setChanged({
        key: "n1",
        base: tank("Pump"),
        next: tank("Valve"),
      });
      expect(actions).toEqual([
        schematic.set({
          path: ["configs", "n1", "label", "label"],
          value: "Valve",
          guards: [{ path: ["configs", "n1", "variant"], value: "tank" }],
        }),
      ]);
    });

    it("should remove a field that next does not hold", () => {
      const base = tank("Pump", "#ff0000");
      const state = empty({ nodes: [node("n1", 0, 0)], configs: { n1: base } });
      const actions = schematic.setChanged({ key: "n1", base, next: tank("Pump") });
      expect(apply(state, ...encoded(state, actions)).configs.n1).toEqual(tank("Pump"));
    });

    it("should replace the config when the symbol changes variant", () => {
      const base = tank("Pump");
      const next = cfg({ variant: "valve" });
      const state = empty({ nodes: [node("n1", 0, 0)], configs: { n1: base } });
      const actions = schematic.setChanged({ key: "n1", base, next });
      expect(apply(state, ...encoded(state, actions)).configs.n1).toEqual(next);
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
      expect(
        schematic.kindOf([schematic.setConfig({ key: "n1", config: base })]),
      ).toEqual("config");
    });

    it("should classify a config edit that rides on a move as a move", () => {
      const move = schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } });
      expect(schematic.kindOf([move, ...single])).toEqual("move");
    });

    it("should classify a set of the name by its own type", () => {
      expect(
        schematic.kindOf([schematic.set({ path: ["name"], value: "New" })]),
      ).toEqual("set");
    });

    it("should not classify a set of the name that rides on a move as a move", () => {
      const move = schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } });
      expect(
        schematic.kindOf([move, schematic.set({ path: ["name"], value: "New" })]),
      ).toEqual("transaction");
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
      const out = apply(
        empty(),
        schematic.setNode({ node: node("pump", 0, 0) }),
        schematic.setNode({ node: node("valve", 100, 0) }),
        schematic.setNode({ node: node("tank", 200, 0) }),
        schematic.addEdge({ edge: edge("e1", "pump", "out", "valve", "in") }),
        schematic.addEdge({ edge: edge("e2", "valve", "out", "tank", "in") }),
        schematic.setConfig({
          key: "pump",
          config: { variant: "tank", label: { label: "Main Pump" } },
        }),
        schematic.setEdge({
          edge: edge("e1", "pump", "out", "valve", "in", { variant: "electric" }),
        }),
      );
      expect(out.nodes).toHaveLength(3);
      expect(out.edges).toHaveLength(2);
      expect(out.edges[0].variant).toBe("electric");
      expect(out.configs).toEqual({ pump: tank("Main Pump") });
    });

    it("should drop the config and edges of a node that is removed and re-added", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1)],
        edges: [edge("e1", "n1", "o", "n2", "i")],
        configs: { n1: tank("v1") },
      });
      const out = apply(
        state,
        schematic.removeNode({ key: "n1" }),
        schematic.setNode({ node: node("n1", 50, 50) }),
      );
      expect(out.nodes).toHaveLength(2);
      expect(out.nodes[1]).toEqual(node("n1", 50, 50));
      expect(out.configs).toEqual({});
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
        actions.push(
          schematic.setConfig({
            key: `n${i}`,
            config: { variant: "tank", label: { label: `node ${i}` } },
          }),
        );
      actions.push(
        schematic.setEdge({
          edge: edge("e1", "n1", "out", "n2", "in", { variant: "electric" }),
        }),
      );
      const out = schematic.reduceAll(state, actions).next;
      expect(out.nodes).toHaveLength(5);
      expect(out.nodes[0].position).toEqual({ x: 0, y: 100 });
      expect(out.nodes[4].position).toEqual({ x: 400, y: 100 });
      expect(out.edges).toHaveLength(4);
      expect(out.edges[1].variant).toBe("electric");
      expect(Object.keys(out.configs)).toHaveLength(3);
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

    it("should reject a setConfig action whose config names no element variant", () => {
      expect(() =>
        schematic.actionZ.parse({
          type: "set_config",
          setConfig: { key: "orphan", config: { data: 1 } },
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
      expectRoundTrip(state, [schematic.set({ path: ["name"], value: "new" })]);
    });
    it("should round-trip when setting the same name", () => {
      const state = empty({ name: "same" });
      expectRoundTrip(state, [schematic.set({ path: ["name"], value: "same" })]);
    });
    it("should target the name so concurrent sets of it invalidate each other", () => {
      const state = empty({ name: "a" });
      const { targets } = schematic.reduceAll(state, [
        schematic.set({ path: ["name"], value: "b" }),
      ]);
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
        schematic.setNode({
          node: node("n2", 5, 5),
          config: { variant: "tank", label: { label: "x" } },
        }),
      ]);
    });
    it("should invert a replace with a setNode of the prior node and config", () => {
      const state = empty({
        nodes: [node("n1", 1, 1)],
        configs: { n1: tank("Old", "#ff0000") },
      });
      expectRoundTrip(state, [
        schematic.setNode({
          node: { ...node("n1", 9, 9) },
          config: { variant: "tank", label: { label: "New" } },
        }),
      ]);
    });
    it("should invert a replace with no new config by restoring the prior node", () => {
      const state = empty({
        nodes: [node("n1", 1, 1)],
        configs: { n1: tank("Old") },
      });
      expectRoundTrip(state, [schematic.setNode({ node: node("n1", 9, 9) })]);
    });
  });

  describe("removeNode", () => {
    it("should invert by re-inserting the node and its config (order not preserved)", () => {
      // setNode appends rather than inserting at a specific index, so a remove + undo
      // cycle can rearrange the nodes array. The contents are restored but the original
      // index is lost. Would be addressed by an InsertNode(node, idx) action.
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1)],
        configs: { n1: tank("Pump"), n2: tank("Tank") },
      });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(restored.configs).toEqual(state.configs);
      const byKey = (ns: schematic.Schematic["nodes"]) =>
        Object.fromEntries(ns.map((n) => [n.key, n]));
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
    it("should restore the connected edges and their style", () => {
      const state = empty({
        nodes: [node("n2", 1, 1), node("n1", 0, 0)],
        edges: [edge("e1", "n1", "out", "n2", "in", { variant: "data" })],
        configs: { n1: tank("Pump") },
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
      expect(restored.configs).toEqual(state.configs);
    });
  });

  describe("removeNode group member cascade", () => {
    it("should append a setConfig restoring the group's members to the inverse", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: groupConfig(["a", "n1", "b"]) },
      });
      const { inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      expect(inverse).toHaveLength(2);
      expect(inverse[1]).toEqual(
        schematic.setConfig({ key: "g1", config: groupConfig(["a", "n1", "b"]) }),
      );
    });

    it("should invert by restoring the node and its membership", () => {
      const state = empty({
        nodes: [node("g1", -20, -20), node("n1", 0, 0)],
        configs: { g1: groupConfig(["n1"]) },
      });
      expectRoundTrip(state, [schematic.removeNode({ key: "n1" })]);
    });

    it("should restore members in their original order", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: groupConfig(["a", "n1", "b"]) },
      });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(restored.configs.g1).toEqual(groupConfig(["a", "n1", "b"]));
    });

    it("should restore duplicate member entries on undo", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: groupConfig(["n1", "n1"]) },
      });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(restored.configs.g1).toEqual(groupConfig(["n1", "n1"]));
    });

    it("should restore membership in every group that listed the key", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("g2", 0, 0), node("n1", 0, 0)],
        configs: {
          g1: groupConfig(["n1", "a"]),
          g2: groupConfig(["b", "n1"]),
        },
      });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(restored.configs.g1).toEqual(groupConfig(["n1", "a"]));
      expect(restored.configs.g2).toEqual(groupConfig(["b", "n1"]));
    });

    it("should restore full membership after undoing a multi-remove batch", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0), node("n2", 0, 0)],
        configs: { g1: groupConfig(["n1", "n2"]) },
      });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.removeNode({ key: "n1" }),
        schematic.removeNode({ key: "n2" }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(restored.configs.g1).toEqual(groupConfig(["n1", "n2"]));
      const byKey = (ns: schematic.Schematic["nodes"]) =>
        Object.fromEntries(ns.map((n) => [n.key, n]));
      expect(byKey(restored.nodes)).toEqual(byKey(state.nodes));
    });

    it("should splice the member again when the removal is reapplied after undo", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: groupConfig(["n1", "n2"]) },
      });
      const action = schematic.removeNode({ key: "n1" });
      const { next, inverse } = schematic.reduceAll(state, [action]);
      const restored = schematic.reduceAll(next, inverse).next;
      const redone = schematic.reduceAll(restored, [action]).next;
      expect(redone.configs.g1).toEqual(groupConfig(["n2"]));
      expect(redone.configs.n1).toBeUndefined();
    });
  });

  describe("addEdge / removeEdge", () => {
    it("should invert addEdge with removeEdge", () => {
      const state = empty({ nodes: [node("a", 0, 0), node("b", 1, 1)] });
      expectRoundTrip(state, [
        schematic.addEdge({ edge: edge("e1", "a", "o", "b", "i") }),
      ]);
    });
    it("should invert removeEdge with addEdge and restore its style", () => {
      const state = empty({
        nodes: [node("a", 0, 0), node("b", 1, 1)],
        edges: [edge("e1", "a", "o", "b", "i", { strokeWidth: 5 })],
      });
      expectRoundTrip(state, [schematic.removeEdge({ key: "e1" })]);
    });
  });

  describe("setEdge", () => {
    it("should invert a new edge with removeEdge", () => {
      const state = empty({ nodes: [node("a", 0, 0), node("b", 1, 1)] });
      expectRoundTrip(state, [
        schematic.setEdge({ edge: edge("e1", "a", "o", "b", "i") }),
      ]);
    });
    it("should invert a replaced edge with its prior style", () => {
      const state = empty({
        nodes: [node("a", 0, 0), node("b", 1, 1)],
        edges: [edge("e1", "a", "o", "b", "i", { variant: "pneumatic" })],
      });
      expectRoundTrip(state, [
        schematic.setEdge({ edge: edge("e1", "a", "o", "b", "i", { strokeWidth: 9 }) }),
      ]);
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

  describe("setConfig", () => {
    it("should restore the prior entry whole", () => {
      const state = empty({
        nodes: [node("n1", 0, 0)],
        configs: { n1: tank("Old", "#ff0000") },
      });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.setConfig({
          key: "n1",
          config: { variant: "tank", label: { label: "New" } },
        }),
      ]);
      expect(inverse).toEqual([
        schematic.setConfig({ key: "n1", config: tank("Old", "#ff0000") }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      expect(restored.configs.n1).toEqual(tank("Old", "#ff0000"));
    });
    it("should produce an empty inverse when the key had no entry", () => {
      const { inverse } = schematic.reduceAll(empty({ nodes: [node("n1", 0, 0)] }), [
        schematic.setConfig({ key: "n1", config: tank("New") }),
      ]);
      expect(inverse).toEqual([]);
    });
  });

  describe("multi-action transactions", () => {
    it("should invert a build sequence", () => {
      expectRoundTrip(empty(), [
        schematic.setNode({ node: node("pump", 0, 0) }),
        schematic.setNode({ node: node("valve", 100, 0) }),
        schematic.addEdge({ edge: edge("e1", "pump", "out", "valve", "in") }),
        schematic.setEdge({
          edge: edge("e1", "pump", "out", "valve", "in", { variant: "electric" }),
        }),
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
      const state = empty({
        nodes: [node("n1", 0, 0)],
        configs: { n1: tank("Pump") },
      });
      expectRoundTrip(state, [
        schematic.removeNode({ key: "n1" }),
        schematic.setNode({
          node: node("n1", 50, 50),
          config: { variant: "tank", label: { label: "Pump" } },
        }),
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

  describe("coalesced setConfig undo (regression)", () => {
    it("should restore the original config after two rapid setConfig calls", () => {
      const initial = empty({
        nodes: [node("n1", 0, 0)],
        configs: {
          n1: cfg({ variant: "tank", label: { level: "p" }, strokeColor: "#ff0000" }),
        },
      });
      // Each dispatch carries the FULL form values, as the onChange of IndividualConfig
      // does: ({ values }) => dispatch(schematic.setConfig({ key, config: values })).
      const r1 = schematic.reduceAll(initial, [
        schematic.setConfig({
          key: "n1",
          config: { variant: "tank", label: { level: "h2" }, strokeColor: "#ff0000" },
        }),
      ]);
      const r2 = schematic.reduceAll(r1.next, [
        schematic.setConfig({
          key: "n1",
          config: { variant: "tank", label: { level: "h3" }, strokeColor: "#ff0000" },
        }),
      ]);
      // Matches pushOnto's coalescing: inverse = [next.inverse, ...top.inverse]
      const mergedInverse = [...r2.inverse, ...r1.inverse];
      const restored = schematic.reduceAll(r2.next, mergedInverse).next;
      expect(restored.configs.n1).toEqual(initial.configs.n1);
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
