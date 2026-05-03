// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import {
  type Action,
  actionZ,
  addNode,
  reduce,
  reduceAll,
  removeEdge,
  removeNode,
  setAuthority,
  setEdge,
  setLegend,
  setNodePosition,
  setProps,
} from "@/schematic/actions.gen";
import { ZERO_LEGEND } from "@/schematic/client";
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
  authority: 1,
  legend: ZERO_LEGEND,
  nodes: [],
  edges: [],
  props: {},
  ...overrides,
});

describe("schematic reducer", () => {
  describe("setNodePosition", () => {
    it("should move the matching node to the new position", () => {
      const state = empty({ nodes: [node("n1", 0, 0), node("n2", 5, 5)] });
      const out = reduceAll(state, [
        setNodePosition({ key: "n1", position: { x: 100, y: 200 } }),
      ]);
      expect(out.nodes).toEqual([node("n1", 100, 200), node("n2", 5, 5)]);
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = reduceAll(state, [
        setNodePosition({ key: "ghost", position: { x: 9, y: 9 } }),
      ]);
      expect(out.nodes).toEqual([node("n1", 0, 0)]);
    });
    it("should only move the first matching node when keys are duplicated", () => {
      const state = empty({ nodes: [node("dup", 0, 0), node("dup", 1, 1)] });
      const out = reduceAll(state, [
        setNodePosition({ key: "dup", position: { x: 9, y: 9 } }),
      ]);
      expect(out.nodes[0].position).toEqual({ x: 9, y: 9 });
      expect(out.nodes[1].position).toEqual({ x: 1, y: 1 });
    });
  });

  describe("addNode", () => {
    it("should append the node to the end of the slice", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = reduceAll(state, [addNode({ node: node("n2", 1, 2) })]);
      expect(out.nodes).toEqual([node("n1", 0, 0), node("n2", 1, 2)]);
    });
    it("should write props under the node's key when props is non-undefined", () => {
      const out = reduceAll(empty(), [
        addNode({ node: node("n1", 0, 0), props: { label: "Pump", color: "#f00" } }),
      ]);
      expect(out.props).toEqual({ n1: { label: "Pump", color: "#f00" } });
    });
    it("should leave props untouched when the action's props is undefined", () => {
      const out = reduceAll(empty(), [addNode({ node: node("n1", 0, 0) })]);
      expect(out.props).toEqual({});
    });
    it("should append a duplicate-key node, locking current behavior", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = reduceAll(state, [addNode({ node: node("n1", 9, 9) })]);
      expect(out.nodes).toHaveLength(2);
      expect(out.nodes[0]).toEqual(node("n1", 0, 0));
      expect(out.nodes[1]).toEqual(node("n1", 9, 9));
    });
  });

  describe("removeNode", () => {
    it("should remove the matching node and any props stored under its key", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1)],
        props: { n1: { label: "Pump" }, n2: { label: "Tank" } },
      });
      const out = reduceAll(state, [removeNode({ key: "n1" })]);
      expect(out.nodes).toEqual([node("n2", 1, 1)]);
      expect(out.props).toEqual({ n2: { label: "Tank" } });
    });
    it("should leave existing edges intact even when they reference the removed node", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1)],
        edges: [edge("e1", "n1", "out", "n2", "in")],
      });
      const out = reduceAll(state, [removeNode({ key: "n1" })]);
      expect(out.edges).toHaveLength(1);
      expect(out.edges[0].source.node).toBe("n1");
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({
        nodes: [node("n1", 0, 0)],
        props: { n1: { label: "Pump" } },
      });
      const out = reduceAll(state, [removeNode({ key: "ghost" })]);
      expect(out.nodes).toEqual(state.nodes);
      expect(out.props).toEqual(state.props);
    });
  });

  describe("setEdge", () => {
    it("should append an edge whose key is not yet present", () => {
      const state = empty({ edges: [edge("e1", "a", "o", "b", "i")] });
      const out = reduceAll(state, [setEdge({ edge: edge("e2", "b", "o", "c", "i") })]);
      expect(out.edges).toHaveLength(2);
      expect(out.edges[1].key).toBe("e2");
    });
    it("should replace an existing edge in place, preserving slice index", () => {
      const state = empty({
        edges: [
          edge("e1", "a", "o", "b", "i"),
          edge("e2", "b", "o", "c", "i"),
          edge("e3", "c", "o", "d", "i"),
        ],
      });
      const out = reduceAll(state, [setEdge({ edge: edge("e2", "x", "y", "z", "w") })]);
      expect(out.edges).toHaveLength(3);
      expect(out.edges[0].key).toBe("e1");
      expect(out.edges[1]).toEqual(edge("e2", "x", "y", "z", "w"));
      expect(out.edges[2].key).toBe("e3");
    });
  });

  describe("removeEdge", () => {
    it("should remove the matching edge", () => {
      const state = empty({
        edges: [edge("e1", "a", "o", "b", "i"), edge("e2", "b", "o", "c", "i")],
      });
      const out = reduceAll(state, [removeEdge({ key: "e1" })]);
      expect(out.edges).toEqual([edge("e2", "b", "o", "c", "i")]);
    });
    it("should be a no-op when the key does not match any edge", () => {
      const state = empty({ edges: [edge("e1", "a", "o", "b", "i")] });
      const out = reduceAll(state, [removeEdge({ key: "ghost" })]);
      expect(out.edges).toEqual(state.edges);
    });
  });

  describe("setProps", () => {
    it("should write the props entry under the given key", () => {
      const out = reduceAll(empty(), [
        setProps({ key: "n1", props: { label: "Pump" } }),
      ]);
      expect(out.props).toEqual({ n1: { label: "Pump" } });
    });
    it("should overwrite an existing props entry", () => {
      const state = empty({ props: { n1: { label: "Old" } } });
      const out = reduceAll(state, [setProps({ key: "n1", props: { label: "New" } })]);
      expect(out.props).toEqual({ n1: { label: "New" } });
    });
    it("should accept a key that does not match any node or edge", () => {
      const out = reduceAll(empty(), [setProps({ key: "orphan", props: { data: 1 } })]);
      expect(out.props).toEqual({ orphan: { data: 1 } });
    });
  });

  describe("setAuthority", () => {
    it("should replace the authority value", () => {
      const out = reduceAll(empty({ authority: 1 }), [setAuthority({ value: 200 })]);
      expect(out.authority).toBe(200);
    });
  });

  describe("setLegend", () => {
    it("should replace the legend wholesale", () => {
      const newLegend = {
        visible: true,
        position: { x: 0, y: 0 },
        colors: { on: color.construct("#ff0000") },
      };
      const out = reduceAll(empty(), [setLegend({ legend: newLegend })]);
      expect(out.legend).toEqual(newLegend);
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
      expect(out).not.toBe(state);
      expect(out.nodes).not.toBe(state.nodes);
    });
    it("should return the same state object when an action is a no-op", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = reduceAll(state, [
        setNodePosition({ key: "ghost", position: { x: 9, y: 9 } }),
      ]);
      expect(out).toBe(state);
    });
  });

  describe("real-world scenarios", () => {
    it("should converge to the final position after a 30-action drag storm", () => {
      const state = empty({ nodes: [node("pump", 0, 0)] });
      const actions: Action[] = [];
      for (let i = 0; i < 30; i++)
        actions.push(setNodePosition({ key: "pump", position: { x: i, y: i * 2 } }));
      const out = reduceAll(state, actions);
      expect(out.nodes[0].position).toEqual({ x: 29, y: 58 });
    });

    it("should build a complete graph from an empty schematic", () => {
      const out = reduceAll(empty(), [
        addNode({ node: node("pump", 0, 0) }),
        addNode({ node: node("valve", 100, 0) }),
        addNode({ node: node("tank", 200, 0) }),
        setEdge({ edge: edge("e1", "pump", "out", "valve", "in") }),
        setEdge({ edge: edge("e2", "valve", "out", "tank", "in") }),
        setProps({ key: "pump", props: { label: "Main Pump" } }),
        setProps({ key: "e1", props: { variant: "pipe" } }),
      ]);
      expect(out.nodes).toHaveLength(3);
      expect(out.edges).toHaveLength(2);
      expect(out.props).toEqual({
        pump: { label: "Main Pump" },
        e1: { variant: "pipe" },
      });
    });

    it("should drop props but keep dangling edges when a node is removed and re-added", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1)],
        edges: [edge("e1", "n1", "o", "n2", "i")],
        props: { n1: { label: "v1" } },
      });
      const out = reduceAll(state, [
        removeNode({ key: "n1" }),
        addNode({ node: node("n1", 50, 50) }),
      ]);
      expect(out.nodes).toHaveLength(2);
      expect(out.nodes[1]).toEqual(node("n1", 50, 50));
      expect(out.props).toEqual({});
      expect(out.edges).toHaveLength(1);
      expect(out.edges[0].source.node).toBe("n1");
    });

    it("should converge an idempotent action sequence to the same state as a single application", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const a = setNodePosition({ key: "n1", position: { x: 10, y: 20 } });
      expect(reduceAll(state, [a])).toEqual(reduceAll(state, [a, a, a]));
    });

    it("should apply a 50-action editor session and converge to a coherent schematic", () => {
      const state = empty();
      const actions: Action[] = [];
      for (let i = 0; i < 5; i++)
        actions.push(addNode({ node: node(`n${i}`, i * 100, 0) }));
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
          setEdge({
            edge: edge(`e${i}`, `n${i}`, "out", `n${i + 1}`, "in"),
          }),
        );
      for (let i = 0; i < 3; i++)
        actions.push(setProps({ key: `n${i}`, props: { label: `node ${i}` } }));
      actions.push(setProps({ key: "e1", props: { variant: "electric" } }));
      actions.push(setAuthority({ value: 255 }));
      actions.push(
        setLegend({
          legend: { visible: true, position: { x: 0, y: 0 }, colors: {} },
        }),
      );
      const out = reduceAll(state, actions);
      expect(out.nodes).toHaveLength(5);
      expect(out.nodes[0].position).toEqual({ x: 0, y: 100 });
      expect(out.nodes[4].position).toEqual({ x: 400, y: 100 });
      expect(out.edges).toHaveLength(4);
      expect(Object.keys(out.props)).toHaveLength(4);
      expect(out.authority).toBe(255);
      expect(out.legend.visible).toBe(true);
    });

    it("should leave state untouched when given an empty action list", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expect(reduceAll(state, [])).toBe(state);
    });
  });

  describe("single-action reduce", () => {
    it("should apply a single action without wrapping it in an array", () => {
      const out = reduce(empty(), addNode({ node: node("n1", 1, 2) }));
      expect(out.nodes).toEqual([node("n1", 1, 2)]);
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
