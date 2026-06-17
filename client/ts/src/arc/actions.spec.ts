// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { arc } from "@/arc";

const node = (
  key: string,
  x: number,
  y: number,
  config: record.Unknown = {},
): arc.graph.Node => ({ key, type: "stage", config, position: { x, y } });

const edge = (
  srcNode: string,
  srcParam: string,
  tgtNode: string,
  tgtParam: string,
): arc.ir.Edge => ({
  source: { node: srcNode, param: srcParam },
  target: { node: tgtNode, param: tgtParam },
  kind: arc.ir.EdgeKind.continuous,
});

const empty = (graph: Partial<arc.graph.Graph> = {}, name = ""): arc.Arc => ({
  key: "arc-key",
  name,
  mode: "graph",
  text: { raw: "" },
  graph: {
    viewport: { position: { x: 0, y: 0 }, zoom: 1 },
    functions: [],
    edges: [],
    nodes: [],
    ...graph,
  },
});

const apply = (state: arc.Arc, ...actions: arc.Action[]): arc.Arc =>
  arc.reduceAll(state, actions).next;

describe("arc reducer", () => {
  describe("rename", () => {
    it("should set the module name to the payload name", () => {
      expect(apply(empty({}, "old"), arc.rename({ name: "new" })).name).toEqual("new");
    });
    it("should leave the graph untouched", () => {
      const state = empty({
        nodes: [node("n1", 0, 0)],
        edges: [edge("n1", "out", "n2", "in")],
      });
      const out = apply(state, arc.rename({ name: "new" }));
      expect(out.graph.nodes).toEqual(state.graph.nodes);
      expect(out.graph.edges).toEqual(state.graph.edges);
    });
  });

  describe("setMode", () => {
    it("should switch the representation mode", () => {
      expect(apply(empty(), arc.setMode({ mode: "text" })).mode).toEqual("text");
    });
    it("should be a no-op when the mode is unchanged", () => {
      const state = empty();
      expect(apply(state, arc.setMode({ mode: "graph" }))).toBe(state);
    });
  });

  describe("setNode", () => {
    it("should append the node when no node has the same key", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = apply(state, arc.setNode({ node: node("n2", 1, 2) }));
      expect(out.graph.nodes).toEqual([node("n1", 0, 0), node("n2", 1, 2)]);
    });
    it("should replace an existing node in place, preserving slice index", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1), node("n3", 2, 2)],
      });
      const out = apply(state, arc.setNode({ node: node("n2", 9, 9) }));
      expect(out.graph.nodes).toHaveLength(3);
      expect(out.graph.nodes[1]).toEqual(node("n2", 9, 9));
    });
  });

  describe("setNodePosition", () => {
    it("should move the matching node to the new position", () => {
      const state = empty({ nodes: [node("n1", 0, 0), node("n2", 5, 5)] });
      const out = apply(
        state,
        arc.setNodePosition({ key: "n1", position: { x: 100, y: 200 } }),
      );
      expect(out.graph.nodes[0].position).toEqual({ x: 100, y: 200 });
      expect(out.graph.nodes[1].position).toEqual({ x: 5, y: 5 });
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expect(
        apply(state, arc.setNodePosition({ key: "ghost", position: { x: 9, y: 9 } })),
      ).toBe(state);
    });
  });

  describe("setNodeConfig", () => {
    it("should replace the configuration of the matching node", () => {
      const state = empty({ nodes: [node("n1", 0, 0, { gain: 1 })] });
      const out = apply(state, arc.setNodeConfig({ key: "n1", config: { offset: 2 } }));
      expect(out.graph.nodes[0].config).toEqual({ offset: 2 });
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expect(apply(state, arc.setNodeConfig({ key: "ghost", config: {} }))).toBe(state);
    });
  });

  describe("removeNode", () => {
    it("should remove the matching node and every edge connected to it", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1), node("n3", 2, 2)],
        edges: [edge("n1", "out", "n2", "in"), edge("n2", "out", "n3", "in")],
      });
      const out = apply(state, arc.removeNode({ key: "n2" }));
      expect(out.graph.nodes).toEqual([node("n1", 0, 0), node("n3", 2, 2)]);
      expect(out.graph.edges).toEqual([]);
    });
    it("should keep unrelated edges intact", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1), node("n3", 2, 2)],
        edges: [edge("n1", "out", "n3", "in")],
      });
      const out = apply(state, arc.removeNode({ key: "n2" }));
      expect(out.graph.edges).toEqual([edge("n1", "out", "n3", "in")]);
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expect(apply(state, arc.removeNode({ key: "ghost" }))).toBe(state);
    });
  });

  describe("addEdge", () => {
    it("should append an edge with new source and target handles", () => {
      const state = empty({ edges: [edge("a", "o", "b", "i")] });
      const out = apply(state, arc.addEdge({ edge: edge("b", "o", "c", "i") }));
      expect(out.graph.edges).toEqual([
        edge("a", "o", "b", "i"),
        edge("b", "o", "c", "i"),
      ]);
    });
    it("should be a no-op when an edge with the same source and target exists", () => {
      const state = empty({ edges: [edge("a", "o", "b", "i")] });
      expect(apply(state, arc.addEdge({ edge: edge("a", "o", "b", "i") }))).toBe(state);
    });
  });

  describe("removeEdge", () => {
    it("should remove the edge matching the source and target handles", () => {
      const state = empty({
        edges: [edge("a", "o", "b", "i"), edge("b", "o", "c", "i")],
      });
      const out = apply(
        state,
        arc.removeEdge({
          source: { node: "a", param: "o" },
          target: { node: "b", param: "i" },
        }),
      );
      expect(out.graph.edges).toEqual([edge("b", "o", "c", "i")]);
    });
    it("should be a no-op when no edge matches the handles", () => {
      const state = empty({ edges: [edge("a", "o", "b", "i")] });
      expect(
        apply(
          state,
          arc.removeEdge({
            source: { node: "x", param: "o" },
            target: { node: "y", param: "i" },
          }),
        ),
      ).toBe(state);
    });
  });

  describe("immutability", () => {
    it("should leave the input state object unmodified", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const before = structuredClone(state);
      arc.reduceAll(state, [
        arc.setNodePosition({ key: "n1", position: { x: 9, y: 9 } }),
      ]);
      expect(state).toEqual(before);
    });
    it("should return a new state object when an action mutates", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const { next } = arc.reduceAll(state, [
        arc.setNodePosition({ key: "n1", position: { x: 9, y: 9 } }),
      ]);
      expect(next).not.toBe(state);
    });
    it("should return the same state object when an action is a no-op", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expect(
        arc.reduceAll(state, [
          arc.setNodePosition({ key: "ghost", position: { x: 9, y: 9 } }),
        ]).next,
      ).toBe(state);
    });
  });

  describe("real-world scenarios", () => {
    it("should converge to the final position after a 30-action drag storm", () => {
      const state = empty({ nodes: [node("n", 0, 0)] });
      const actions: arc.Action[] = [];
      for (let i = 0; i < 30; i++)
        actions.push(arc.setNodePosition({ key: "n", position: { x: i, y: i * 2 } }));
      expect(arc.reduceAll(state, actions).next.graph.nodes[0].position).toEqual({
        x: 29,
        y: 58,
      });
    });
    it("should build a complete graph from an empty module", () => {
      const out = apply(
        empty(),
        arc.setMode({ mode: "graph" }),
        arc.setNode({ node: node("src", 0, 0) }),
        arc.setNode({ node: node("op", 100, 0) }),
        arc.setNode({ node: node("sink", 200, 0) }),
        arc.addEdge({ edge: edge("src", "out", "op", "in") }),
        arc.addEdge({ edge: edge("op", "out", "sink", "in") }),
      );
      expect(out.graph.nodes).toHaveLength(3);
      expect(out.graph.edges).toHaveLength(2);
    });
    it("should leave state untouched when given an empty action list", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expect(arc.reduceAll(state, []).next).toBe(state);
    });
  });

  describe("zod parsing", () => {
    it("should reject an action with an unknown discriminator with a ZodError", () => {
      expect(() => arc.actionZ.parse({ type: "unknown", payload: {} })).toThrow(
        z.ZodError,
      );
    });
    it("should accept a fully populated setNodePosition action", () => {
      const a = arc.setNodePosition({ key: "n1", position: { x: 1, y: 2 } });
      expect(arc.actionZ.parse(a)).toEqual(a);
    });
  });
});

describe("arc reducer inverses", () => {
  const expectRoundTrip = (state: arc.Arc, actions: arc.Action[]) => {
    const { next, inverse } = arc.reduceAll(state, actions);
    expect(arc.reduceAll(next, inverse).next).toEqual(state);
  };

  // Node removal cascades to connected edges, and the inverse re-inserts the
  // node via setNode (which appends) plus addEdge for each removed edge. Slice
  // order is not preserved, so compare by key.
  const expectGraphRoundTrip = (state: arc.Arc, actions: arc.Action[]) => {
    const { next, inverse } = arc.reduceAll(state, actions);
    const restored = arc.reduceAll(next, inverse).next;
    const byKey = (ns: arc.graph.Node[]) =>
      Object.fromEntries(ns.map((n) => [n.key, n]));
    expect(byKey(restored.graph.nodes)).toEqual(byKey(state.graph.nodes));
    expect(restored.graph.edges).toEqual(state.graph.edges);
  };

  describe("rename", () => {
    it("should invert to restore the prior name", () => {
      expectRoundTrip(empty({}, "old"), [arc.rename({ name: "new" })]);
    });
    it("should report the arc key as a target", () => {
      const state = empty({}, "a");
      expect(arc.reduceAll(state, [arc.rename({ name: "b" })]).targets).toEqual([
        state.key,
      ]);
    });
  });

  describe("setMode", () => {
    it("should invert to restore the prior mode", () => {
      expectRoundTrip(empty(), [arc.setMode({ mode: "text" })]);
    });
    it("should produce an empty inverse for a no-op", () => {
      expect(arc.reduceAll(empty(), [arc.setMode({ mode: "graph" })]).inverse).toEqual(
        [],
      );
    });
  });

  describe("setNode", () => {
    it("should invert an insert with a removeNode", () => {
      expectRoundTrip(empty({ nodes: [node("n1", 0, 0)] }), [
        arc.setNode({ node: node("n2", 5, 5) }),
      ]);
    });
    it("should invert a replace by restoring the prior node", () => {
      expectRoundTrip(empty({ nodes: [node("n1", 1, 1, { gain: 2 })] }), [
        arc.setNode({ node: node("n1", 9, 9, { gain: 3 }) }),
      ]);
    });
  });

  describe("setNodePosition", () => {
    it("should invert to restore the prior position", () => {
      expectRoundTrip(empty({ nodes: [node("n1", 1, 2)] }), [
        arc.setNodePosition({ key: "n1", position: { x: 99, y: 100 } }),
      ]);
    });
    it("should produce an empty inverse for a no-op", () => {
      expect(
        arc.reduceAll(empty({ nodes: [node("n1", 1, 2)] }), [
          arc.setNodePosition({ key: "ghost", position: { x: 9, y: 9 } }),
        ]).inverse,
      ).toEqual([]);
    });
  });

  describe("setNodeConfig", () => {
    it("should invert by restoring the prior config (clean replace, no phantom keys)", () => {
      expectRoundTrip(empty({ nodes: [node("n1", 0, 0, { gain: 1, offset: 5 })] }), [
        arc.setNodeConfig({ key: "n1", config: { gain: 2 } }),
      ]);
    });
  });

  describe("removeNode", () => {
    it("should invert by re-inserting the node and its connected edges", () => {
      expectGraphRoundTrip(
        empty({
          nodes: [node("n1", 0, 0), node("n2", 1, 1)],
          edges: [edge("n1", "out", "n2", "in")],
        }),
        [arc.removeNode({ key: "n1" })],
      );
    });
    it("should produce an empty inverse for a no-op removal", () => {
      expect(
        arc.reduceAll(empty({ nodes: [node("n1", 0, 0)] }), [
          arc.removeNode({ key: "ghost" }),
        ]).inverse,
      ).toEqual([]);
    });
  });

  describe("addEdge / removeEdge", () => {
    it("should invert addEdge with removeEdge", () => {
      expectRoundTrip(empty(), [arc.addEdge({ edge: edge("a", "o", "b", "i") })]);
    });
    it("should invert removeEdge with addEdge", () => {
      expectRoundTrip(empty({ edges: [edge("a", "o", "b", "i")] }), [
        arc.removeEdge({
          source: { node: "a", param: "o" },
          target: { node: "b", param: "i" },
        }),
      ]);
    });
    it("should produce an empty inverse for a duplicate addEdge", () => {
      expect(
        arc.reduceAll(empty({ edges: [edge("a", "o", "b", "i")] }), [
          arc.addEdge({ edge: edge("a", "o", "b", "i") }),
        ]).inverse,
      ).toEqual([]);
    });
  });

  describe("isUndoable", () => {
    it("should report every Arc action as undoable", () => {
      expect(arc.isUndoable(arc.rename({ name: "x" }))).toBe(true);
      expect(arc.isUndoable(arc.setMode({ mode: "text" }))).toBe(true);
      expect(arc.isUndoable(arc.setNode({ node: node("n1", 0, 0) }))).toBe(true);
      expect(
        arc.isUndoable(arc.setNodePosition({ key: "n1", position: { x: 0, y: 0 } })),
      ).toBe(true);
      expect(arc.isUndoable(arc.setNodeConfig({ key: "n1", config: {} }))).toBe(true);
      expect(arc.isUndoable(arc.removeNode({ key: "n1" }))).toBe(true);
      expect(arc.isUndoable(arc.addEdge({ edge: edge("a", "o", "b", "i") }))).toBe(
        true,
      );
      expect(
        arc.isUndoable(
          arc.removeEdge({
            source: { node: "a", param: "o" },
            target: { node: "b", param: "i" },
          }),
        ),
      ).toBe(true);
    });
  });

  describe("multi-action transactions", () => {
    it("should invert a build sequence by restoring the original graph", () => {
      expectGraphRoundTrip(empty(), [
        arc.setNode({ node: node("src", 0, 0) }),
        arc.setNode({ node: node("op", 100, 0) }),
        arc.addEdge({ edge: edge("src", "out", "op", "in") }),
      ]);
    });
    it("should invert a multi-step move to the original positions", () => {
      expectRoundTrip(empty({ nodes: [node("n1", 0, 0), node("n2", 1, 1)] }), [
        arc.setNodePosition({ key: "n1", position: { x: 10, y: 20 } }),
        arc.setNodePosition({ key: "n2", position: { x: 30, y: 40 } }),
      ]);
    });
  });
});
