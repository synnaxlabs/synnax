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
): schematic.Edge => ({
  key,
  source: { node: srcNode, param: srcParam },
  target: { node: tgtNode, param: tgtParam },
});

const groupConfig = (
  members: unknown[],
  extra: Record<string, unknown> = {},
): Record<string, unknown> => ({ variant: "groupBox", members, ...extra });

const empty = (overrides: Partial<schematic.Schematic> = {}): schematic.Schematic =>
  schematic.schematicZ.parse({ name: "Test Schematic", ...overrides });

const apply = (
  state: schematic.Schematic,
  ...actions: schematic.Action[]
): schematic.Schematic => schematic.reduceAll(state, actions).next;

describe("schematic reducer", () => {
  describe("rename", () => {
    it("should set the schematic name to the payload name", () => {
      const state = empty({ name: "old" });
      const out = apply(state, schematic.rename({ name: "new" }));
      expect(out.name).toEqual("new");
    });
    it("should accept an empty name", () => {
      const state = empty({ name: "old" });
      const out = apply(state, schematic.rename({ name: "" }));
      expect(out.name).toEqual("");
    });
    it("should leave nodes, edges, and configs untouched", () => {
      const state = empty({
        name: "old",
        nodes: [node("n1", 0, 0)],
        edges: [edge("e1", "a", "o", "b", "i")],
        configs: { n1: { label: "Pump" } },
      });
      const out = apply(state, schematic.rename({ name: "new" }));
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
          config: { label: "Pump", color: "#f00" },
        }),
      );
      expect(out.configs).toEqual({ n1: { label: "Pump", color: "#f00" } });
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
        configs: { n1: { label: "Pump" }, n2: { label: "Tank" } },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.nodes).toEqual([node("n2", 1, 1)]);
      expect(out.configs).toEqual({ n2: { label: "Tank" } });
    });
    it("should leave existing edges intact even when they reference the removed node", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1)],
        edges: [edge("e1", "n1", "out", "n2", "in")],
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.edges).toHaveLength(1);
      expect(out.edges[0].source.node).toBe("n1");
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({
        nodes: [node("n1", 0, 0)],
        configs: { n1: { label: "Pump" } },
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
        configs: { g1: groupConfig(["m1"]), m1: { label: "Pump" } },
      });
      const out = apply(state, schematic.removeNode({ key: "g1" }));
      expect(out.configs.m1).toEqual({ label: "Pump" });
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

    it("should not touch a non-group config that carries a members field", () => {
      const state = empty({
        nodes: [node("t1", 0, 0), node("n1", 0, 0)],
        configs: { t1: { variant: "table", members: ["n1"] } },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.configs.t1).toEqual({ variant: "table", members: ["n1"] });
    });

    it("should skip a group config whose members is not an array", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: { variant: "groupBox", members: "n1" } },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.configs.g1).toEqual({ variant: "groupBox", members: "n1" });
    });

    it("should not cascade on a no-op removal, even for a listed key", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: groupConfig(["n1", "ghost"]) },
      });
      const out = apply(state, schematic.removeNode({ key: "ghost" }));
      expect(out.configs.g1).toEqual(groupConfig(["n1", "ghost"]));
    });

    it("should skip a config whose variant is not a string", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: { variant: 42, members: ["n1"] } },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.configs.g1).toEqual({ variant: 42, members: ["n1"] });
    });

    it("should preserve non-string member entries", () => {
      const state = empty({
        nodes: [node("g1", 0, 0), node("n1", 0, 0)],
        configs: { g1: groupConfig([42, "n1", null]) },
      });
      const out = apply(state, schematic.removeNode({ key: "n1" }));
      expect(out.configs.g1).toEqual(groupConfig([42, null]));
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
        configs: { g1: groupConfig(["g1", "n1"]), n1: { label: "Pump" } },
      });
      const out = apply(state, schematic.removeNode({ key: "g1" }));
      expect(out.configs).toEqual({ n1: { label: "Pump" } });
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
          configs: { g1: groupConfig(["n1"]), n1: { label: "Pump" } },
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
      const configs: Record<string, Record<string, unknown>> = {};
      const expected: Record<string, Record<string, unknown>> = {};
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
        schematic.setConfig({ key: "g1", config: { members: ["n1", "n2", "n3"] } }),
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
        schematic.setConfig({ key: "g1", config: { members: ["n1", "n2", "n3"] } }),
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
      const state = empty({ edges: [edge("e1", "a", "o", "b", "i")] });
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
      expect(out.edges).toEqual(state.edges);
    });
  });

  describe("setConfig", () => {
    it("should write the config entry under the given key", () => {
      const out = apply(
        empty(),
        schematic.setConfig({ key: "n1", config: { label: "Pump" } }),
      );
      expect(out.configs).toEqual({ n1: { label: "Pump" } });
    });
    it("should merge payload fields into an existing config entry", () => {
      const state = empty({
        configs: { n1: { label: "Old", color: "#ff0000" } },
      });
      const out = apply(
        state,
        schematic.setConfig({ key: "n1", config: { label: "New" } }),
      );
      expect(out.configs).toEqual({ n1: { label: "New", color: "#ff0000" } });
    });
    it("should accept a key that does not match any node or edge", () => {
      const out = apply(
        empty(),
        schematic.setConfig({ key: "orphan", config: { data: 1 } }),
      );
      expect(out.configs).toEqual({ orphan: { data: 1 } });
    });
    it("should override the payload color with the source node's color when the new entry is for an edge", () => {
      const state = empty({
        edges: [edge("e1", "src", "o", "tgt", "i")],
        configs: { src: { color: [0, 1, 0, 1] } },
      });
      const out = apply(
        state,
        schematic.setConfig({
          key: "e1",
          config: { variant: "pipe", color: [0, 0, 0, 0] },
        }),
      );
      expect(out.configs.e1).toEqual({ variant: "pipe", color: [0, 1, 0, 1] });
    });
    it("should inherit the source node's color when the payload omits color", () => {
      const state = empty({
        edges: [edge("e1", "src", "o", "tgt", "i")],
        configs: { src: { color: [0, 1, 0, 1] } },
      });
      const out = apply(
        state,
        schematic.setConfig({ key: "e1", config: { variant: "pipe" } }),
      );
      expect(out.configs.e1).toEqual({ variant: "pipe", color: [0, 1, 0, 1] });
    });
    it("should leave the payload untouched when the source node has a zero color", () => {
      const state = empty({
        edges: [edge("e1", "src", "o", "tgt", "i")],
        configs: { src: { color: [0, 0, 0, 0] } },
      });
      const out = apply(
        state,
        schematic.setConfig({
          key: "e1",
          config: { variant: "pipe", color: [0, 0, 0, 0] },
        }),
      );
      expect(out.configs.e1).toEqual({ variant: "pipe", color: [0, 0, 0, 0] });
    });
    it("should leave the payload untouched when the source node has no color", () => {
      const state = empty({
        edges: [edge("e1", "src", "o", "tgt", "i")],
        configs: { src: { label: "Pump" } },
      });
      const out = apply(
        state,
        schematic.setConfig({
          key: "e1",
          config: { variant: "pipe", color: [0, 0, 0, 0] },
        }),
      );
      expect(out.configs.e1).toEqual({ variant: "pipe", color: [0, 0, 0, 0] });
    });
    it("should leave the payload untouched when the source node has no config", () => {
      const state = empty({
        edges: [edge("e1", "src", "o", "tgt", "i")],
      });
      const out = apply(
        state,
        schematic.setConfig({
          key: "e1",
          config: { variant: "pipe", color: [0, 0, 0, 0] },
        }),
      );
      expect(out.configs.e1).toEqual({ variant: "pipe", color: [0, 0, 0, 0] });
    });
    it("should not override the color when merging into an existing edge config", () => {
      const state = empty({
        edges: [edge("e1", "src", "o", "tgt", "i")],
        configs: {
          src: { color: [0, 1, 0, 1] },
          e1: { variant: "pipe", color: [0, 0, 0, 0] },
        },
      });
      const out = apply(
        state,
        schematic.setConfig({ key: "e1", config: { variant: "electric" } }),
      );
      expect(out.configs.e1).toEqual({ variant: "electric", color: [0, 0, 0, 0] });
    });
    it("should inherit the source color end-to-end when addEdge is followed by setConfig in one batch", () => {
      const state = empty({
        nodes: [node("src", 0, 0), node("tgt", 100, 0)],
        configs: { src: { color: [0, 1, 0, 1] } },
      });
      const out = apply(
        state,
        schematic.addEdge({ edge: edge("e1", "src", "o", "tgt", "i") }),
        schematic.setConfig({
          key: "e1",
          config: { variant: "pipe", color: [0, 0, 0, 0], segments: [] },
        }),
      );
      expect(out.configs.e1).toEqual({
        variant: "pipe",
        color: [0, 1, 0, 1],
        segments: [],
      });
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
        schematic.setConfig({ key: "pump", config: { label: "Main Pump" } }),
        schematic.setConfig({ key: "e1", config: { variant: "pipe" } }),
      );
      expect(out.nodes).toHaveLength(3);
      expect(out.edges).toHaveLength(2);
      expect(out.configs).toEqual({
        pump: { label: "Main Pump" },
        e1: { variant: "pipe" },
      });
    });

    it("should drop config but keep dangling edges when a node is removed and re-added", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1)],
        edges: [edge("e1", "n1", "o", "n2", "i")],
        configs: { n1: { label: "v1" } },
      });
      const out = apply(
        state,
        schematic.removeNode({ key: "n1" }),
        schematic.setNode({ node: node("n1", 50, 50) }),
      );
      expect(out.nodes).toHaveLength(2);
      expect(out.nodes[1]).toEqual(node("n1", 50, 50));
      expect(out.configs).toEqual({});
      expect(out.edges).toHaveLength(1);
      expect(out.edges[0].source.node).toBe("n1");
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
          schematic.setConfig({ key: `n${i}`, config: { label: `node ${i}` } }),
        );
      actions.push(schematic.setConfig({ key: "e1", config: { variant: "electric" } }));
      const out = schematic.reduceAll(state, actions).next;
      expect(out.nodes).toHaveLength(5);
      expect(out.nodes[0].position).toEqual({ x: 0, y: 100 });
      expect(out.nodes[4].position).toEqual({ x: 400, y: 100 });
      expect(out.edges).toHaveLength(4);
      expect(Object.keys(out.configs)).toHaveLength(4);
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

  // Acknowledges the documented gap in setConfig's inverse: keys newly added by a
  // SetConfig action cannot be removed by the inverse (SetConfig is a merge, not a
  // replace). Asserts that nodes and edges round-trip cleanly, and that every config
  // key present in the original is restored to its original value. Keys absent from the
  // original may persist as phantom entries on the restored state.
  const expectUserVisibleRoundTrip = (
    state: schematic.Schematic,
    actions: schematic.Action[],
  ) => {
    const { next, inverse } = schematic.reduceAll(state, actions);
    const restored = schematic.reduceAll(next, inverse).next;
    expect(restored.nodes).toEqual(state.nodes);
    expect(restored.edges).toEqual(state.edges);
    for (const [k, v] of Object.entries(state.configs))
      expect(restored.configs[k]).toEqual(v);
  };

  describe("rename", () => {
    it("should invert to restore the prior name", () => {
      const state = empty({ name: "old" });
      expectRoundTrip(state, [schematic.rename({ name: "new" })]);
    });
    it("should round-trip when renaming to the same name", () => {
      const state = empty({ name: "same" });
      expectRoundTrip(state, [schematic.rename({ name: "same" })]);
    });
    it("should report the schematic key as a target so concurrent renames invalidate each other", () => {
      const state = empty({ name: "a" });
      const { targets } = schematic.reduceAll(state, [schematic.rename({ name: "b" })]);
      expect(targets).toEqual([state.key]);
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
        schematic.setNode({ node: node("n2", 5, 5), config: { label: "x" } }),
      ]);
    });
    it("should invert a replace with a setNode of the prior node and config", () => {
      const state = empty({
        nodes: [node("n1", 1, 1)],
        configs: { n1: { label: "Old", color: "#ff0000" } },
      });
      expectRoundTrip(state, [
        schematic.setNode({ node: { ...node("n1", 9, 9) }, config: { label: "New" } }),
      ]);
    });
    it("should invert a replace with no new config by restoring the prior node", () => {
      const state = empty({
        nodes: [node("n1", 1, 1)],
        configs: { n1: { label: "Old" } },
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
        configs: { n1: { label: "Pump" }, n2: { label: "Tank" } },
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
        schematic.setConfig({ key: "g1", config: { members: ["a", "n1", "b"] } }),
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
      const state = empty();
      expectRoundTrip(state, [
        schematic.addEdge({ edge: edge("e1", "a", "o", "b", "i") }),
      ]);
    });
    it("should invert removeEdge with addEdge", () => {
      const state = empty({ edges: [edge("e1", "a", "o", "b", "i")] });
      expectRoundTrip(state, [schematic.removeEdge({ key: "e1" })]);
    });
    it("should produce an empty inverse for a duplicate addEdge", () => {
      const state = empty({ edges: [edge("e1", "a", "o", "b", "i")] });
      const { inverse } = schematic.reduceAll(state, [
        schematic.addEdge({ edge: edge("e1", "x", "y", "z", "w") }),
      ]);
      expect(inverse).toEqual([]);
    });
  });

  describe("setConfig", () => {
    it("should restore overwritten fields and leave phantom new fields", () => {
      const state = empty({
        configs: { n1: { label: "Old", color: "#ff0000" } },
      });
      const { next, inverse } = schematic.reduceAll(state, [
        schematic.setConfig({ key: "n1", config: { label: "New", count: 1 } }),
      ]);
      const restored = schematic.reduceAll(next, inverse).next;
      // Overwritten fields are restored to their original values:
      expect(restored.configs.n1).toMatchObject({ label: "Old", color: "#ff0000" });
      // Newly-added fields persist as phantom entries — documented limitation
      // until a ReplaceConfig action exists.
      expect(restored.configs.n1).toHaveProperty("count", 1);
    });
    it("should produce an empty inverse when no key in the payload was previously present", () => {
      const state = empty({ configs: { n1: { label: "Old" } } });
      const { inverse } = schematic.reduceAll(state, [
        schematic.setConfig({ key: "n1", config: { count: 1 } }),
      ]);
      expect(inverse).toEqual([]);
    });
  });

  describe("multi-action transactions", () => {
    it("should invert a build sequence (nodes and edges restored; phantom edge config persists)", () => {
      // Inverse cannot strip the e1 config that setConfig added — see
      // expectUserVisibleRoundTrip's docstring for the limitation.
      expectUserVisibleRoundTrip(empty(), [
        schematic.setNode({ node: node("pump", 0, 0) }),
        schematic.setNode({ node: node("valve", 100, 0) }),
        schematic.addEdge({ edge: edge("e1", "pump", "out", "valve", "in") }),
        schematic.setConfig({ key: "e1", config: { variant: "pipe" } }),
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
        configs: { n1: { label: "Pump" } },
      });
      expectRoundTrip(state, [
        schematic.removeNode({ key: "n1" }),
        schematic.setNode({ node: node("n1", 50, 50), config: { label: "Pump" } }),
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
        configs: { n1: { label: { level: "p" }, scale: 1, color: "red" } },
      });
      // Each dispatch carries the FULL form values, matching IndividualConfig's
      // onChange: ({ values }) => dispatch(schematic.setConfig({ key, config: values })).
      const r1 = schematic.reduceAll(initial, [
        schematic.setConfig({
          key: "n1",
          config: { label: { level: "h2" }, scale: 1, color: "red" },
        }),
      ]);
      const r2 = schematic.reduceAll(r1.next, [
        schematic.setConfig({
          key: "n1",
          config: { label: { level: "h3" }, scale: 1, color: "red" },
        }),
      ]);
      // Matches pushOnto's coalescing: inverse = [next.inverse, ...top.inverse]
      const mergedInverse = [...r2.inverse, ...r1.inverse];
      const restored = schematic.reduceAll(r2.next, mergedInverse).next;
      expect(restored.configs.n1).toEqual(initial.configs.n1);
    });
  });
});
