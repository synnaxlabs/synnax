// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { crdt, type record, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { arc } from "@/arc";

const node = (key: string, x: number, y: number): arc.graph.Node => ({
  key,
  position: { x, y },
});

const cfg = (type: string, params: record.Unknown = {}): record.Unknown => ({
  type,
  ...params,
});

const empty = (graph: Partial<arc.graph.Graph> = {}, name = "Test Arc"): arc.Arc =>
  arc.arcZ.parse({
    name,
    type: "automation",
    source: { mode: "graph", graph: arc.graph.graphZ.parse(graph) },
  });

const graphOf = (a: arc.Arc): arc.graph.Graph => {
  if (a.source.mode !== "graph") throw new Error("expected a graph-mode Arc");
  return a.source.graph;
};

const emptyText = (): arc.Arc =>
  arc.arcZ.parse({ name: "Test Arc", type: "automation", source: { mode: "text" } });

const textOf = (a: arc.Arc): arc.text.Text => {
  if (a.source.mode !== "text") throw new Error("expected a text-mode Arc");
  return a.source.text;
};

const apply = (state: arc.Arc, ...actions: arc.Action[]): arc.Arc =>
  arc.reduceAll(state, actions).next;

const setName = (name: string): arc.Action => arc.set({ path: ["name"], value: name });

describe("arc reducer", () => {
  describe("mode", () => {
    it.each([
      ["setNode", arc.setNode({ node: node("n1", 0, 0) })],
      ["removeNode", arc.removeNode({ key: "n1" })],
      ["removeEdge", arc.removeEdge({ key: "e1" })],
    ])("should reject the graph action %s on a text Arc", (_, action) => {
      expect(() => apply(emptyText(), action)).toThrow("cannot take a graph edit");
    });

    it("should reject a text action on a graph Arc", () => {
      const op = new crdt.Text(2).insert(0, "x")[0];
      expect(() => apply(empty(), arc.insertChar(op))).toThrow(
        "cannot take a text edit",
      );
    });
  });

  describe("set of the name", () => {
    it("should set the module name to the value", () => {
      expect(apply(empty({}, "old"), setName("new")).name).toEqual("new");
    });
    it("should set the name of a text Arc", () => {
      expect(apply(emptyText(), setName("new")).name).toEqual("new");
    });
    it("should leave the graph untouched", () => {
      const state = empty({
        nodes: [node("n1", 0, 0)],
        edges: [
          {
            key: "n1n2",
            source: { node: "n1", param: "out" },
            target: { node: "n2", param: "in" },
            kind: arc.ir.EdgeKind.continuous,
          },
        ],
      });
      const out = apply(state, setName("new"));
      expect(graphOf(out).nodes).toEqual(graphOf(state).nodes);
      expect(graphOf(out).edges).toEqual(graphOf(state).edges);
    });
  });

  describe("setDirectory", () => {
    it("should move the Arc into the directory", () => {
      const directory = uuid.create();
      expect(apply(empty(), arc.setDirectory({ directory })).directory).toEqual(
        directory,
      );
    });
  });

  describe("setNode", () => {
    it("should append the node when no node has the same key", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      const out = apply(state, arc.setNode({ node: node("n2", 1, 2) }));
      expect(graphOf(out).nodes).toEqual([node("n1", 0, 0), node("n2", 1, 2)]);
    });
    it("should replace an existing node in place, preserving slice index", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1), node("n3", 2, 2)],
      });
      const out = apply(state, arc.setNode({ node: node("n2", 9, 9) }));
      expect(graphOf(out).nodes).toHaveLength(3);
      expect(graphOf(out).nodes[1]).toEqual(node("n2", 9, 9));
    });
  });

  describe("setNodePosition", () => {
    it("should move the matching node to the new position", () => {
      const state = empty({ nodes: [node("n1", 0, 0), node("n2", 5, 5)] });
      const out = apply(
        state,
        arc.setNodePosition({ key: "n1", position: { x: 100, y: 200 } }),
      );
      expect(graphOf(out).nodes[0].position).toEqual({ x: 100, y: 200 });
      expect(graphOf(out).nodes[1].position).toEqual({ x: 5, y: 5 });
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expect(
        apply(state, arc.setNodePosition({ key: "ghost", position: { x: 9, y: 9 } })),
      ).toBe(state);
    });
  });

  describe("setNodeInputs", () => {
    it("should overwrite fields present in both the existing and payload configs", () => {
      const state = empty({
        nodes: [node("n1", 0, 0)],
        inputs: { n1: cfg("constant", { value: 1 }) },
      });
      const out = apply(
        state,
        arc.setNodeInputs({ key: "n1", inputs: cfg("constant", { value: 2 }) }),
      );
      expect(graphOf(out).inputs.n1).toEqual(cfg("constant", { value: 2 }));
    });
    it("should merge the config into the existing entry, preserving absent fields", () => {
      const state = empty({
        nodes: [node("n1", 0, 0)],
        inputs: { n1: cfg("constant", { value: 1 }) },
      });
      const out = apply(state, arc.setNodeInputs({ key: "n1", inputs: { value: 2 } }));
      expect(graphOf(out).inputs.n1).toEqual(cfg("constant", { value: 2 }));
    });
    it("should write the config even when no entry exists yet", () => {
      const out = apply(empty(), arc.setNodeInputs({ key: "n1", inputs: cfg("add") }));
      expect(graphOf(out).inputs.n1).toEqual(cfg("add"));
    });
  });

  describe("removeNode", () => {
    it("should remove the node, its config, and every connected edge", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1), node("n3", 2, 2)],
        edges: [
          {
            key: "n1n2",
            source: { node: "n1", param: "out" },
            target: { node: "n2", param: "in" },
            kind: arc.ir.EdgeKind.continuous,
          },
          {
            key: "n2n3",
            source: { node: "n2", param: "out" },
            target: { node: "n3", param: "in" },
            kind: arc.ir.EdgeKind.continuous,
          },
        ],
        inputs: { n2: cfg("add") },
      });
      const out = apply(state, arc.removeNode({ key: "n2" }));
      expect(graphOf(out).nodes).toEqual([node("n1", 0, 0), node("n3", 2, 2)]);
      expect(graphOf(out).edges).toEqual([]);
      expect(graphOf(out).inputs.n2).toBeUndefined();
    });
    it("should keep unrelated edges intact", () => {
      const state = empty({
        nodes: [node("n1", 0, 0), node("n2", 1, 1), node("n3", 2, 2)],
        edges: [
          {
            key: "n1n3",
            source: { node: "n1", param: "out" },
            target: { node: "n3", param: "in" },
            kind: arc.ir.EdgeKind.continuous,
          },
        ],
      });
      const out = apply(state, arc.removeNode({ key: "n2" }));
      expect(graphOf(out).edges).toEqual([
        {
          key: "n1n3",
          source: { node: "n1", param: "out" },
          target: { node: "n3", param: "in" },
          kind: arc.ir.EdgeKind.continuous,
        },
      ]);
    });
    it("should be a no-op when the key does not match any node", () => {
      const state = empty({ nodes: [node("n1", 0, 0)] });
      expect(apply(state, arc.removeNode({ key: "ghost" }))).toBe(state);
    });
  });

  describe("addEdge", () => {
    it("should append an edge with new source and target handles", () => {
      const state = empty({
        edges: [
          {
            key: "ab",
            source: { node: "a", param: "o" },
            target: { node: "b", param: "i" },
            kind: arc.ir.EdgeKind.continuous,
          },
        ],
      });
      const out = apply(
        state,
        arc.addEdge({
          edge: {
            key: "bc",
            source: { node: "b", param: "o" },
            target: { node: "c", param: "i" },
            kind: arc.ir.EdgeKind.continuous,
          },
        }),
      );
      expect(graphOf(out).edges).toEqual([
        {
          key: "ab",
          source: { node: "a", param: "o" },
          target: { node: "b", param: "i" },
          kind: arc.ir.EdgeKind.continuous,
        },
        {
          key: "bc",
          source: { node: "b", param: "o" },
          target: { node: "c", param: "i" },
          kind: arc.ir.EdgeKind.continuous,
        },
      ]);
    });
    it("should be a no-op when an edge with the same source and target exists", () => {
      const state = empty({
        edges: [
          {
            key: "ab",
            source: { node: "a", param: "o" },
            target: { node: "b", param: "i" },
            kind: arc.ir.EdgeKind.continuous,
          },
        ],
      });
      expect(
        apply(
          state,
          arc.addEdge({
            edge: {
              key: "ab",
              source: { node: "a", param: "o" },
              target: { node: "b", param: "i" },
              kind: arc.ir.EdgeKind.continuous,
            },
          }),
        ),
      ).toBe(state);
    });
  });

  describe("removeEdge", () => {
    it("should remove the edge matching the key", () => {
      const state = empty({
        edges: [
          {
            key: "ab",
            source: { node: "a", param: "o" },
            target: { node: "b", param: "i" },
            kind: arc.ir.EdgeKind.continuous,
          },
          {
            key: "bc",
            source: { node: "b", param: "o" },
            target: { node: "c", param: "i" },
            kind: arc.ir.EdgeKind.continuous,
          },
        ],
      });
      const out = apply(state, arc.removeEdge({ key: "ab" }));
      expect(graphOf(out).edges).toEqual([
        {
          key: "bc",
          source: { node: "b", param: "o" },
          target: { node: "c", param: "i" },
          kind: arc.ir.EdgeKind.continuous,
        },
      ]);
    });
    it("should be a no-op when no edge matches the key", () => {
      const state = empty({
        edges: [
          {
            key: "ab",
            source: { node: "a", param: "o" },
            target: { node: "b", param: "i" },
            kind: arc.ir.EdgeKind.continuous,
          },
        ],
      });
      expect(apply(state, arc.removeEdge({ key: "missing" }))).toBe(state);
    });
  });

  describe("reconnectEdge", () => {
    it("should rewrite the endpoints of the edge with the key, preserving key and kind", () => {
      const state = empty({
        edges: [
          {
            key: "ab",
            source: { node: "a", param: "o" },
            target: { node: "b", param: "i" },
            kind: arc.ir.EdgeKind.continuous,
          },
        ],
      });
      const out = apply(
        state,
        arc.reconnectEdge({
          key: "ab",
          source: { node: "a", param: "o" },
          target: { node: "c", param: "i" },
        }),
      );
      expect(graphOf(out).edges).toEqual([
        {
          key: "ab",
          source: { node: "a", param: "o" },
          target: { node: "c", param: "i" },
          kind: arc.ir.EdgeKind.continuous,
        },
      ]);
    });
    it("should be a no-op when no edge matches the key", () => {
      const state = empty({
        edges: [
          {
            key: "ab",
            source: { node: "a", param: "o" },
            target: { node: "b", param: "i" },
            kind: arc.ir.EdgeKind.continuous,
          },
        ],
      });
      expect(
        apply(
          state,
          arc.reconnectEdge({
            key: "missing",
            source: { node: "a", param: "o" },
            target: { node: "c", param: "i" },
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
      expect(graphOf(arc.reduceAll(state, actions).next).nodes[0].position).toEqual({
        x: 29,
        y: 58,
      });
    });
    it("should build a complete graph from an empty module", () => {
      const out = apply(
        empty(),
        arc.setNode({ node: node("src", 0, 0) }),
        arc.setNodeInputs({ key: "src", inputs: cfg("on", { channel: 1 }) }),
        arc.setNode({ node: node("sink", 200, 0) }),
        arc.setNodeInputs({ key: "sink", inputs: cfg("write", { channel: 2 }) }),
        arc.addEdge({
          edge: {
            key: "srcsink",
            source: { node: "src", param: "out" },
            target: { node: "sink", param: "in" },
            kind: arc.ir.EdgeKind.continuous,
          },
        }),
      );
      expect(graphOf(out).nodes).toHaveLength(2);
      expect(graphOf(out).edges).toHaveLength(1);
      expect(graphOf(out).inputs.src).toEqual(cfg("on", { channel: 1 }));
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

  // Node removal cascades to its config and connected edges; the inverse
  // re-inserts the node via setNode (which appends) plus setNodeInputs and
  // addEdge. Slice order is not preserved, so compare nodes by key.
  const expectGraphRoundTrip = (state: arc.Arc, actions: arc.Action[]) => {
    const { next, inverse } = arc.reduceAll(state, actions);
    const restored = arc.reduceAll(next, inverse).next;
    const byKey = (ns: arc.graph.Node[]) =>
      Object.fromEntries(ns.map((n) => [n.key, n]));
    expect(byKey(graphOf(restored).nodes)).toEqual(byKey(graphOf(state).nodes));
    expect(graphOf(restored).edges).toEqual(graphOf(state).edges);
    expect(graphOf(restored).inputs).toEqual(graphOf(state).inputs);
  };

  describe("set of the name", () => {
    it("should invert to restore the prior name", () => {
      expectRoundTrip(empty({}, "old"), [setName("new")]);
    });
    it("should target the name", () => {
      expect(arc.reduceAll(empty({}, "a"), [setName("b")]).targets).toEqual(["name"]);
    });
  });

  describe("setDirectory", () => {
    it("should invert to restore the prior directory", () => {
      const state = { ...empty(), directory: uuid.create() };
      expectRoundTrip(state, [arc.setDirectory({ directory: uuid.create() })]);
    });
    it("should produce an empty inverse when the Arc has no directory", () => {
      expect(
        arc.reduceAll(empty(), [arc.setDirectory({ directory: uuid.create() })])
          .inverse,
      ).toEqual([]);
    });
  });

  describe("setNode", () => {
    it("should invert an insert with a removeNode", () => {
      expectRoundTrip(empty({ nodes: [node("n1", 0, 0)] }), [
        arc.setNode({ node: node("n2", 5, 5) }),
      ]);
    });
    it("should invert a replace by restoring the prior node", () => {
      expectRoundTrip(empty({ nodes: [node("n1", 1, 1)] }), [
        arc.setNode({ node: node("n1", 9, 9) }),
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

  describe("setNodeInputs", () => {
    it("should invert by restoring the prior config", () => {
      expectGraphRoundTrip(
        empty({
          nodes: [node("n1", 0, 0)],
          inputs: { n1: cfg("constant", { value: 1 }) },
        }),
        [arc.setNodeInputs({ key: "n1", inputs: cfg("constant", { value: 2 }) })],
      );
    });
  });

  describe("removeNode", () => {
    it("should invert by re-inserting the node, its config, and connected edges", () => {
      expectGraphRoundTrip(
        empty({
          nodes: [node("n1", 0, 0), node("n2", 1, 1)],
          edges: [
            {
              key: "n1n2",
              source: { node: "n1", param: "out" },
              target: { node: "n2", param: "in" },
              kind: arc.ir.EdgeKind.continuous,
            },
          ],
          inputs: { n1: cfg("on", { channel: 1 }) },
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
      expectRoundTrip(empty(), [
        arc.addEdge({
          edge: {
            key: "ab",
            source: { node: "a", param: "o" },
            target: { node: "b", param: "i" },
            kind: arc.ir.EdgeKind.continuous,
          },
        }),
      ]);
    });
    it("should invert removeEdge with addEdge", () => {
      expectRoundTrip(
        empty({
          edges: [
            {
              key: "ab",
              source: { node: "a", param: "o" },
              target: { node: "b", param: "i" },
              kind: arc.ir.EdgeKind.continuous,
            },
          ],
        }),
        [arc.removeEdge({ key: "ab" })],
      );
    });
  });

  describe("isUndoable", () => {
    it("should report every Arc action as undoable", () => {
      expect(arc.isUndoable(setName("x"))).toBe(true);
      expect(arc.isUndoable(arc.setNode({ node: node("n1", 0, 0) }))).toBe(true);
      expect(
        arc.isUndoable(arc.setNodePosition({ key: "n1", position: { x: 0, y: 0 } })),
      ).toBe(true);
      expect(arc.isUndoable(arc.setNodeInputs({ key: "n1", inputs: cfg("add") }))).toBe(
        true,
      );
      expect(arc.isUndoable(arc.removeNode({ key: "n1" }))).toBe(true);
      expect(
        arc.isUndoable(
          arc.addEdge({
            edge: {
              key: "ab",
              source: { node: "a", param: "o" },
              target: { node: "b", param: "i" },
              kind: arc.ir.EdgeKind.continuous,
            },
          }),
        ),
      ).toBe(true);
      expect(arc.isUndoable(arc.removeEdge({ key: "ab" }))).toBe(true);
      expect(
        arc.isUndoable(
          arc.reconnectEdge({
            key: "ab",
            source: { node: "a", param: "o" },
            target: { node: "c", param: "i" },
          }),
        ),
      ).toBe(true);
    });
  });

  describe("text", () => {
    it("appends insert_char ops so the document materializes to the typed text", () => {
      const gen = new crdt.Text(2);
      const ops = gen.insert(0, "hi").map((op) => arc.insertChar(op));
      const { next } = arc.reduceAll(emptyText(), ops);
      const doc = new crdt.Text(3);
      doc.load(textOf(next).doc);
      expect(doc.toString()).toEqual("hi");
    });

    it("appends delete_char ops so deleted characters drop from the materialized text", () => {
      const gen = new crdt.Text(2);
      const insertOps = gen.insert(0, "hi").map((op) => arc.insertChar(op));
      const deleteOps = gen.delete(0, 1).map((op) => arc.deleteChar(op));
      const { next } = arc.reduceAll(emptyText(), [...insertOps, ...deleteOps]);
      const doc = new crdt.Text(3);
      doc.load(textOf(next).doc);
      expect(doc.toString()).toEqual("i");
    });

    it("is not undoable", () => {
      expect(arc.isUndoable(arc.insertChar(new crdt.Text(2).insert(0, "x")[0]))).toBe(
        false,
      );
    });

    it("forgets tombstoned characters without changing the materialized text", () => {
      const gen = new crdt.Text(2);
      const insertOps = gen.insert(0, "hi");
      const deleteOps = gen.delete(1, 1);
      let state = arc.reduceAll(emptyText(), [
        ...insertOps.map((op) => arc.insertChar(op)),
        ...deleteOps.map((op) => arc.deleteChar(op)),
      ]).next;
      expect(textOf(state).doc.inserts).toHaveLength(2);
      expect(textOf(state).doc.deletes).toHaveLength(1);

      state = arc.reduceAll(state, [
        arc.forgetChars({ ids: deleteOps.map((op) => op.id) }),
      ]).next;
      expect(textOf(state).doc.inserts).toHaveLength(1);
      expect(textOf(state).doc.deletes).toHaveLength(0);
      const doc = new crdt.Text(3);
      doc.load(textOf(state).doc);
      expect(doc.toString()).toEqual("h");
    });

    it("forget_chars is not undoable", () => {
      expect(arc.isUndoable(arc.forgetChars({ ids: [] }))).toBe(false);
    });
  });
});

describe("arc path actions", () => {
  const set = (path: string[], value: unknown) => arc.set({ path, value });

  it("should set a field of a node's inputs and keep its case", () => {
    const state = empty({ inputs: { n1: cfg("constant", { minValue: 1 }) } });
    const { next, targets } = arc.reduceAll(state, [
      set(["graph", "inputs", "n1", "minValue"], 2),
    ]);
    expect(graphOf(next).inputs.n1).toEqual(cfg("constant", { minValue: 2 }));
    expect(targets).toEqual(["n1"]);
  });

  it("should return an inverse that restores the inputs", () => {
    const state = empty({ inputs: { n1: cfg("constant", { value: 1 }) } });
    const { next, inverse } = arc.reduceAll(state, [
      set(["graph", "inputs", "n1", "value"], 2),
    ]);
    expect(arc.reduceAll(next, inverse).next).toEqual(state);
  });

  it("should throw for a path outside the node inputs", () => {
    expect(() => apply(empty(), set(["graph", "nodes"], []))).toThrow(
      "graph.nodes is not inside an entry of graph.inputs",
    );
  });
});
