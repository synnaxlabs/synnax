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

import { isUndoable, reduceAll } from "@/schematic/actions";
import {
  type Action,
  actionZ,
  addEdge,
  removeEdge,
  removeNode,
  setConfig,
  setNode,
  setNodeMeasured,
  setNodePosition,
} from "@/schematic/actions.gen";
import { type Edge, type Node, type Schematic } from "@/schematic/types.gen";

const node = (key: string, x: number, y: number): Node => ({
  key,
  position: { x, y },
});

const edge = (
  key: string,
  srcNode: string,
  srcParam: string,
  tgtNode: string,
  tgtParam: string,
): Edge => ({
  key,
  source: { node: srcNode, param: srcParam },
  target: { node: tgtNode, param: tgtParam },
});

const empty = (overrides: Partial<Schematic> = {}): Schematic => ({
  key: "00000000-0000-0000-0000-000000000000",
  name: "",
  snapshot: false,
  nodes: [],
  edges: [],
  configs: {},
  ...overrides,
});

const apply = (state: Schematic, ...actions: Action[]): Schematic =>
  reduceAll(state, actions).next;

describe("schematic reducer", () => {
  describe("setNodePosition", () => {
    it("should move the matching node to the new position", () => {
      const state = empty({ nodes: [node("n1", 0, 0), node("n2", 5, 5)] });
      const out = apply(
        state,
        setNodePosition({ key: "n1", position: { x: 100, y: 200 } }),
      );
      expect(out.nodes).toEqual([node("n1", 100, 200), node("n2", 5, 5)]);
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = apply(
        state,
        setNodePosition({ key: "ghost", position: { x: 9, y: 9 } }),
      );
      expect(out.nodes).toEqual([node("n1", 0, 0)]);
    });
    it("should only move the first matching node when keys are duplicated", () => {
      const state = empty({ nodes: [node("dup", 0, 0), node("dup", 1, 1)] });
      const out = apply(
        state,
        setNodePosition({ key: "dup", position: { x: 9, y: 9 } }),
      );
      expect(out.nodes[0].position).toEqual({ x: 9, y: 9 });
      expect(out.nodes[1].position).toEqual({ x: 1, y: 1 });
    });
  });

  describe("setNode", () => {
    it("should append the node to the end of the slice when no node has the same key", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = apply(state, setNode({ node: node("n2", 1, 2) }));
      expect(out.nodes).toEqual([node("n1", 0, 0), node("n2", 1, 2)]);
    });
    it("should write config under the node's key when config is non-undefined", () => {
      const out = apply(
        empty(),
        setNode({ node: node("n1", 0, 0), config: { label: "Pump", color: "#f00" } }),
      );
      expect(out.configs).toEqual({ n1: { label: "Pump", color: "#f00" } });
    });
    it("should leave configs untouched when the action's config is undefined", () => {
      const out = apply(empty(), setNode({ node: node("n1", 0, 0) }));
      expect(out.configs).toEqual({});
    });
    it("should replace an existing node in place when the key already exists, preserving slice index", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1), node("n3", 2, 2)],
      });
      const out = apply(state, setNode({ node: node("n2", 9, 9) }));
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
      const out = apply(state, removeNode({ key: "n1" }));
      expect(out.nodes).toEqual([node("n2", 1, 1)]);
      expect(out.configs).toEqual({ n2: { label: "Tank" } });
    });
    it("should leave existing edges intact even when they reference the removed node", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1)],
        edges: [edge("e1", "n1", "out", "n2", "in")],
      });
      const out = apply(state, removeNode({ key: "n1" }));
      expect(out.edges).toHaveLength(1);
      expect(out.edges[0].source.node).toBe("n1");
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({
        nodes: [node("n1", 0, 0)],
        configs: { n1: { label: "Pump" } },
      });
      const out = apply(state, removeNode({ key: "ghost" }));
      expect(out.nodes).toEqual(state.nodes);
      expect(out.configs).toEqual(state.configs);
    });
  });

  describe("setEdge", () => {
    it("should append an edge whose key is not yet present", () => {
      const state = empty({ edges: [edge("e1", "a", "o", "b", "i")] });
      const out = apply(state, addEdge({ edge: edge("e2", "b", "o", "c", "i") }));
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
      const out = apply(state, addEdge({ edge: edge("e2", "x", "y", "z", "w") }));
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
      const out = apply(state, removeEdge({ key: "e1" }));
      expect(out.edges).toEqual([edge("e2", "b", "o", "c", "i")]);
    });
    it("should be a no-op when the key does not match any edge", () => {
      const state = empty({ edges: [edge("e1", "a", "o", "b", "i")] });
      const out = apply(state, removeEdge({ key: "ghost" }));
      expect(out.edges).toEqual(state.edges);
    });
  });

  describe("setConfig", () => {
    it("should write the config entry under the given key", () => {
      const out = apply(empty(), setConfig({ key: "n1", config: { label: "Pump" } }));
      expect(out.configs).toEqual({ n1: { label: "Pump" } });
    });
    it("should merge payload fields into an existing config entry", () => {
      const state = empty({
        configs: { n1: { label: "Old", color: "#ff0000" } },
      });
      const out = apply(state, setConfig({ key: "n1", config: { label: "New" } }));
      expect(out.configs).toEqual({ n1: { label: "New", color: "#ff0000" } });
    });
    it("should accept a key that does not match any node or edge", () => {
      const out = apply(empty(), setConfig({ key: "orphan", config: { data: 1 } }));
      expect(out.configs).toEqual({ orphan: { data: 1 } });
    });
    it("should override the payload color with the source node's color when the new entry is for an edge", () => {
      const state = empty({
        edges: [edge("e1", "src", "o", "tgt", "i")],
        configs: { src: { color: [0, 1, 0, 1] } },
      });
      const out = apply(
        state,
        setConfig({ key: "e1", config: { variant: "pipe", color: [0, 0, 0, 0] } }),
      );
      expect(out.configs.e1).toEqual({ variant: "pipe", color: [0, 1, 0, 1] });
    });
    it("should inherit the source node's color when the payload omits color", () => {
      const state = empty({
        edges: [edge("e1", "src", "o", "tgt", "i")],
        configs: { src: { color: [0, 1, 0, 1] } },
      });
      const out = apply(state, setConfig({ key: "e1", config: { variant: "pipe" } }));
      expect(out.configs.e1).toEqual({ variant: "pipe", color: [0, 1, 0, 1] });
    });
    it("should leave the payload untouched when the source node has a zero color", () => {
      const state = empty({
        edges: [edge("e1", "src", "o", "tgt", "i")],
        configs: { src: { color: [0, 0, 0, 0] } },
      });
      const out = apply(
        state,
        setConfig({ key: "e1", config: { variant: "pipe", color: [0, 0, 0, 0] } }),
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
        setConfig({ key: "e1", config: { variant: "pipe", color: [0, 0, 0, 0] } }),
      );
      expect(out.configs.e1).toEqual({ variant: "pipe", color: [0, 0, 0, 0] });
    });
    it("should leave the payload untouched when the source node has no config", () => {
      const state = empty({
        edges: [edge("e1", "src", "o", "tgt", "i")],
      });
      const out = apply(
        state,
        setConfig({ key: "e1", config: { variant: "pipe", color: [0, 0, 0, 0] } }),
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
        setConfig({ key: "e1", config: { variant: "electric" } }),
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
        addEdge({ edge: edge("e1", "src", "o", "tgt", "i") }),
        setConfig({
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
      reduceAll(state, [setNodePosition({ key: "n1", position: { x: 9, y: 9 } })]);
      expect(state).toEqual(before);
    });
    it("should return a new state object when any action mutates", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = reduceAll(state, [
        setNodePosition({ key: "n1", position: { x: 9, y: 9 } }),
      ]);
      expect(out.next).not.toBe(state);
      expect(out.next.nodes).not.toBe(state.nodes);
    });
    it("should return the same state object when an action is a no-op", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = reduceAll(state, [
        setNodePosition({ key: "ghost", position: { x: 9, y: 9 } }),
      ]);
      expect(out.next).toBe(state);
    });
  });

  describe("real-world scenarios", () => {
    it("should converge to the final position after a 30-action drag storm", () => {
      const state = empty({ nodes: [node("pump", 0, 0)] });
      const actions: Action[] = [];
      for (let i = 0; i < 30; i++)
        actions.push(setNodePosition({ key: "pump", position: { x: i, y: i * 2 } }));
      const out = reduceAll(state, actions).next;
      expect(out.nodes[0].position).toEqual({ x: 29, y: 58 });
    });

    it("should build a complete graph from an empty schematic", () => {
      const out = apply(
        empty(),
        setNode({ node: node("pump", 0, 0) }),
        setNode({ node: node("valve", 100, 0) }),
        setNode({ node: node("tank", 200, 0) }),
        addEdge({ edge: edge("e1", "pump", "out", "valve", "in") }),
        addEdge({ edge: edge("e2", "valve", "out", "tank", "in") }),
        setConfig({ key: "pump", config: { label: "Main Pump" } }),
        setConfig({ key: "e1", config: { variant: "pipe" } }),
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
        removeNode({ key: "n1" }),
        setNode({ node: node("n1", 50, 50) }),
      );
      expect(out.nodes).toHaveLength(2);
      expect(out.nodes[1]).toEqual(node("n1", 50, 50));
      expect(out.configs).toEqual({});
      expect(out.edges).toHaveLength(1);
      expect(out.edges[0].source.node).toBe("n1");
    });

    it("should converge an idempotent action sequence to the same state as a single application", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const a = setNodePosition({ key: "n1", position: { x: 10, y: 20 } });
      expect(reduceAll(state, [a]).next).toEqual(reduceAll(state, [a, a, a]).next);
    });

    it("should apply a 50-action editor session and converge to a coherent schematic", () => {
      const state = empty();
      const actions: Action[] = [];
      for (let i = 0; i < 5; i++)
        actions.push(setNode({ node: node(`n${i}`, i * 100, 0) }));
      for (let i = 0; i < 5; i++) {
        actions.push(
          setNodePosition({ key: `n${i}`, position: { x: i * 100, y: 50 } }),
        );
        actions.push(
          setNodePosition({ key: `n${i}`, position: { x: i * 100, y: 100 } }),
        );
      }
      for (let i = 0; i < 4; i++)
        actions.push(
          addEdge({
            edge: edge(`e${i}`, `n${i}`, "out", `n${i + 1}`, "in"),
          }),
        );
      for (let i = 0; i < 3; i++)
        actions.push(setConfig({ key: `n${i}`, config: { label: `node ${i}` } }));
      actions.push(setConfig({ key: "e1", config: { variant: "electric" } }));
      const out = reduceAll(state, actions).next;
      expect(out.nodes).toHaveLength(5);
      expect(out.nodes[0].position).toEqual({ x: 0, y: 100 });
      expect(out.nodes[4].position).toEqual({ x: 400, y: 100 });
      expect(out.edges).toHaveLength(4);
      expect(Object.keys(out.configs)).toHaveLength(4);
    });

    it("should leave state untouched when given an empty action list", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expect(reduceAll(state, []).next).toBe(state);
    });
  });

  describe("zod parsing", () => {
    it("should reject an action with an unknown discriminator with a ZodError", () => {
      expect(() => actionZ.parse({ type: "unknown", payload: { foo: 1 } })).toThrow(
        z.ZodError,
      );
    });

    it("should reject a setNodePosition action missing required fields with a ZodError", () => {
      expect(() =>
        actionZ.parse({ type: "set_node_position", setNodePosition: {} }),
      ).toThrow(z.ZodError);
    });

    it("should accept a fully populated setNodePosition action", () => {
      const a = setNodePosition({ key: "n1", position: { x: 1, y: 2 } });
      expect(actionZ.parse(a)).toEqual(a);
    });
  });
});

describe("schematic reducer inverses", () => {
  const expectRoundTrip = (state: Schematic, actions: Action[]) => {
    const { next, inverse } = reduceAll(state, actions);
    const restored = reduceAll(next, inverse).next;
    expect(restored).toEqual(state);
  };

  // Acknowledges the documented gap in setConfig's inverse: keys newly added
  // by a SetConfig action cannot be removed by the inverse (SetConfig is a
  // merge, not a replace). Asserts that nodes and edges round-trip cleanly,
  // and that every config key present in the original is restored to its
  // original value. Keys absent from the original may persist as phantom
  // entries on the restored state.
  const expectUserVisibleRoundTrip = (state: Schematic, actions: Action[]) => {
    const { next, inverse } = reduceAll(state, actions);
    const restored = reduceAll(next, inverse).next;
    expect(restored.nodes).toEqual(state.nodes);
    expect(restored.edges).toEqual(state.edges);
    for (const [k, v] of Object.entries(state.configs))
      expect(restored.configs[k]).toEqual(v);
  };

  describe("setNodePosition", () => {
    it("should invert to restore the prior position", () => {
      const state = empty({ nodes: [node("n1", 1, 2)] });
      expectRoundTrip(state, [
        setNodePosition({ key: "n1", position: { x: 99, y: 100 } }),
      ]);
    });
    it("should produce an empty inverse for a no-op", () => {
      const state = empty({ nodes: [node("n1", 1, 2)] });
      const { inverse } = reduceAll(state, [
        setNodePosition({ key: "ghost", position: { x: 9, y: 9 } }),
      ]);
      expect(inverse).toEqual([]);
    });
  });

  describe("setNode", () => {
    it("should invert an insert with a removeNode", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expectRoundTrip(state, [
        setNode({ node: node("n2", 5, 5), config: { label: "x" } }),
      ]);
    });
    it("should invert a replace with a setNode of the prior node and config", () => {
      const state = empty({
        nodes: [node("n1", 1, 1)],
        configs: { n1: { label: "Old", color: "#ff0000" } },
      });
      expectRoundTrip(state, [
        setNode({ node: { ...node("n1", 9, 9) }, config: { label: "New" } }),
      ]);
    });
    it("should invert a replace with no new config by restoring the prior node", () => {
      const state = empty({
        nodes: [node("n1", 1, 1)],
        configs: { n1: { label: "Old" } },
      });
      expectRoundTrip(state, [setNode({ node: node("n1", 9, 9) })]);
    });
  });

  describe("removeNode", () => {
    it("should invert by re-inserting the node and its config (order not preserved)", () => {
      // setNode appends rather than inserting at a specific index, so a
      // remove + undo cycle can rearrange the nodes array. The contents are
      // restored but the original index is lost. Would be addressed by an
      // InsertNode(node, idx) action.
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1)],
        configs: { n1: { label: "Pump" }, n2: { label: "Tank" } },
      });
      const { next, inverse } = reduceAll(state, [removeNode({ key: "n1" })]);
      const restored = reduceAll(next, inverse).next;
      expect(restored.configs).toEqual(state.configs);
      const byKey = (ns: Schematic["nodes"]) =>
        Object.fromEntries(ns.map((n) => [n.key, n]));
      expect(byKey(restored.nodes)).toEqual(byKey(state.nodes));
    });
    it("should invert a single-node removal cleanly when order is unambiguous", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expectRoundTrip(state, [removeNode({ key: "n1" })]);
    });
    it("should produce an empty inverse for a no-op removal", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const { inverse } = reduceAll(state, [removeNode({ key: "ghost" })]);
      expect(inverse).toEqual([]);
    });
  });

  describe("addEdge / removeEdge", () => {
    it("should invert addEdge with removeEdge", () => {
      const state = empty();
      expectRoundTrip(state, [addEdge({ edge: edge("e1", "a", "o", "b", "i") })]);
    });
    it("should invert removeEdge with addEdge", () => {
      const state = empty({ edges: [edge("e1", "a", "o", "b", "i")] });
      expectRoundTrip(state, [removeEdge({ key: "e1" })]);
    });
    it("should produce an empty inverse for a duplicate addEdge", () => {
      const state = empty({ edges: [edge("e1", "a", "o", "b", "i")] });
      const { inverse } = reduceAll(state, [
        addEdge({ edge: edge("e1", "x", "y", "z", "w") }),
      ]);
      expect(inverse).toEqual([]);
    });
  });

  describe("setConfig", () => {
    it("should restore overwritten fields and leave phantom new fields", () => {
      const state = empty({
        configs: { n1: { label: "Old", color: "#ff0000" } },
      });
      const { next, inverse } = reduceAll(state, [
        setConfig({ key: "n1", config: { label: "New", count: 1 } }),
      ]);
      const restored = reduceAll(next, inverse).next;
      // Overwritten fields are restored to their original values:
      expect(restored.configs.n1).toMatchObject({ label: "Old", color: "#ff0000" });
      // Newly-added fields persist as phantom entries — documented limitation
      // until a ReplaceConfig action exists.
      expect(restored.configs.n1).toHaveProperty("count", 1);
    });
    it("should produce an empty inverse when no key in the payload was previously present", () => {
      const state = empty({ configs: { n1: { label: "Old" } } });
      const { inverse } = reduceAll(state, [
        setConfig({ key: "n1", config: { count: 1 } }),
      ]);
      expect(inverse).toEqual([]);
    });
  });

  describe("setNodeMeasured", () => {
    it("should be reported as not undoable", () => {
      expect(
        isUndoable(setNodeMeasured({ key: "n1", measured: { width: 10, height: 20 } })),
      ).toBe(false);
    });
    it("should produce an empty inverse", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const { inverse } = reduceAll(state, [
        setNodeMeasured({ key: "n1", measured: { width: 10, height: 20 } }),
      ]);
      expect(inverse).toEqual([]);
    });
    it("should report no targets so it does not invalidate concurrent undoables", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const { targets } = reduceAll(state, [
        setNodeMeasured({ key: "n1", measured: { width: 10, height: 20 } }),
      ]);
      expect(targets).toEqual([]);
    });
  });

  describe("multi-action transactions", () => {
    it("should invert a build sequence (nodes and edges restored; phantom edge config persists)", () => {
      // Inverse cannot strip the e1 config that setConfig added — see
      // expectUserVisibleRoundTrip's docstring for the limitation.
      expectUserVisibleRoundTrip(empty(), [
        setNode({ node: node("pump", 0, 0) }),
        setNode({ node: node("valve", 100, 0) }),
        addEdge({ edge: edge("e1", "pump", "out", "valve", "in") }),
        setConfig({ key: "e1", config: { variant: "pipe" } }),
      ]);
    });
    it("should invert a multi-step move to the original positions", () => {
      const state = empty({ nodes: [node("n1", 0, 0), node("n2", 1, 1)] });
      expectRoundTrip(state, [
        setNodePosition({ key: "n1", position: { x: 10, y: 20 } }),
        setNodePosition({ key: "n2", position: { x: 30, y: 40 } }),
      ]);
    });
    it("should invert a remove + re-add by restoring the original schematic", () => {
      const state = empty({
        nodes: [node("n1", 0, 0)],
        configs: { n1: { label: "Pump" } },
      });
      expectRoundTrip(state, [
        removeNode({ key: "n1" }),
        setNode({ node: node("n1", 50, 50), config: { label: "Pump" } }),
      ]);
    });
  });

  describe("isUndoable", () => {
    it("should return true for every action other than setNodeMeasured", () => {
      expect(isUndoable(setNodePosition({ key: "n1", position: { x: 0, y: 0 } }))).toBe(
        true,
      );
      expect(isUndoable(setNode({ node: node("n1", 0, 0) }))).toBe(true);
      expect(isUndoable(removeNode({ key: "n1" }))).toBe(true);
      expect(isUndoable(addEdge({ edge: edge("e1", "a", "o", "b", "i") }))).toBe(true);
      expect(isUndoable(removeEdge({ key: "e1" }))).toBe(true);
      expect(isUndoable(setConfig({ key: "n1", config: {} }))).toBe(true);
    });
  });

  describe("single-action reduce against a draft", () => {
    it("should return the inverse and mutate the draft via reduceAll", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const { next, inverse } = reduceAll(state, [
        setNodePosition({ key: "n1", position: { x: 10, y: 20 } }),
      ]);
      expect(next.nodes[0].position).toEqual({ x: 10, y: 20 });
      expect(inverse).toEqual([
        setNodePosition({ key: "n1", position: { x: 0, y: 0 } }),
      ]);
    });
  });
});
