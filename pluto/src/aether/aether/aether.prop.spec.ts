// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";

// Property/oracle test for context propagation. The incremental engine (re-run only the
// subscribers of a changed key, cascading) is checked against a trivially-correct oracle
// — a full top-down recompute of the whole tree. For random trees and random update /
// delete sequences, every live node's resolved context must match the recompute; any
// missed propagation (a stale node) makes the two disagree.

const MockSender = { send: vi.fn() };
const NOOP = alamos.Instrumentation.NOOP;

const stateZ = z.object({ x: z.number(), id: z.string() });

const KEYS = ["a", "b", "c", "d"] as const;
type Key = (typeof KEYS)[number];
const KEY_WEIGHT: Record<Key, number> = { a: 1, b: 10, c: 100, d: 1000 };

interface Behavior {
  /** Keys always read. */
  reads: Key[];
  /** Keys read only when state.x is even — exercises dynamic subscribe/unsubscribe. */
  condReads: Key[];
  /** Keys published every afterUpdate (empty for leaves). Publishing is unconditional,
   * matching the real-world invariant that providers establish keys at mount. */
  publishes: Key[];
}

interface Observation {
  reads: Record<string, number>;
  published: Record<string, number>;
}

// Per-case shared state, re-pointed each iteration. Components read these module
// bindings at afterUpdate time, so reassigning them between cases is safe.
let behaviors = new Map<string, Behavior>();
let recorded = new Map<string, Observation>();

const activeReadKeys = (b: Behavior, x: number): Key[] => [
  ...new Set<Key>([...b.reads, ...(x % 2 === 0 ? b.condReads : [])]),
];

const publishedValue = (x: number, key: Key, reads: Record<string, number>): number => {
  let sum = 0;
  for (const v of Object.values(reads)) sum += v;
  return x + KEY_WEIGHT[key] + sum;
};

// Shared afterUpdate body for both the leaf and composite test components.
const runNode = (state: { x: number; id: string }, ctx: aether.Context): void => {
  const b = behaviors.get(state.id);
  if (b == null) throw new Error(`missing behavior for ${state.id}`);
  const reads: Record<string, number> = {};
  for (const k of activeReadKeys(b, state.x)) {
    const v = ctx.getOptional<number>(k);
    if (v != null) reads[k] = v;
  }
  const published: Record<string, number> = {};
  for (const k of b.publishes) {
    const v = publishedValue(state.x, k, reads);
    published[k] = v;
    ctx.set(k, v);
  }
  recorded.set(state.id, { reads, published });
};

class PropComposite extends aether.Composite<typeof stateZ> {
  schema = stateZ;
  afterUpdate(ctx: aether.Context): void {
    runNode(this.state, ctx);
  }
}

class PropLeaf extends aether.Leaf<typeof stateZ> {
  schema = stateZ;
  afterUpdate(ctx: aether.Context): void {
    runNode(this.state, ctx);
  }
}

const shouldNotCallCreate = () => {
  throw new Error("should not call create");
};

interface Node {
  id: string;
  key: string;
  fullPath: string[];
  isLeaf: boolean;
  parentId: string | null;
}

// Deterministic PRNG (mulberry32) so each case is reproducible from its seed.
const rng = (seed: number) => () => {
  seed = (seed + 0x6d2b79f5) | 0;
  let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
  t ^= t + Math.imul(t ^ (t >>> 7), 61 | t);
  return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
};

const pickSubset = (rand: () => number, max: number): Key[] =>
  KEYS.filter(() => rand() < 0.4).slice(0, max);

const genTree = (rand: () => number): Node[] => {
  const root: Node = {
    id: "n0",
    key: "n0",
    fullPath: ["n0"],
    isLeaf: false,
    parentId: null,
  };
  const nodes: Node[] = [root];
  const composites: Node[] = [root];
  const count = 5 + Math.floor(rand() * 16);
  for (let i = 1; i < count; i++) {
    const parent = composites[Math.floor(rand() * composites.length)];
    const isLeaf = rand() < 0.5;
    const key = `n${i}`;
    const node: Node = {
      id: key,
      key,
      fullPath: [...parent.fullPath, key],
      isLeaf,
      parentId: parent.id,
    };
    nodes.push(node);
    if (!isLeaf) composites.push(node);
  }
  for (const node of nodes)
    behaviors.set(node.id, {
      reads: pickSubset(rand, 2),
      condReads: pickSubset(rand, 1),
      publishes: node.isLeaf ? [] : pickSubset(rand, 2),
    });
  return nodes;
};

// Brute-force, obviously-correct reference: recompute every live node top-down.
const oracle = (
  nodes: Node[],
  live: Set<string>,
  states: Map<string, number>,
): Map<string, Observation> => {
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const liveNodes = nodes
    .filter((n) => live.has(n.id))
    .sort((a, b) => a.fullPath.length - b.fullPath.length);
  const pub = new Map<string, Record<string, number>>();
  const out = new Map<string, Observation>();
  for (const node of liveNodes) {
    const x = states.get(node.id) as number;
    const b = behaviors.get(node.id) as Behavior;
    const reads: Record<string, number> = {};
    for (const k of activeReadKeys(b, x)) {
      let anc = node.parentId == null ? null : byId.get(node.parentId);
      while (anc != null) {
        const v = pub.get(anc.id)?.[k];
        if (v !== undefined) {
          reads[k] = v;
          break;
        }
        anc = anc.parentId == null ? null : byId.get(anc.parentId);
      }
    }
    const published: Record<string, number> = {};
    for (const k of b.publishes) published[k] = publishedValue(x, k, reads);
    pub.set(node.id, published);
    out.set(node.id, { reads, published });
  }
  return out;
};

const construct = (node: Node, parent: aether.Component): aether.Component => {
  const props = {
    path: [node.key],
    type: node.isLeaf ? "leaf" : "node",
    sender: MockSender,
    instrumentation: NOOP,
    parent,
  };
  return node.isLeaf ? new PropLeaf(props) : new PropComposite(props);
};

describe("aether context propagation (property/oracle)", () => {
  it("keeps every node's resolved context equal to a full recompute", () => {
    const CASES = 300;
    for (let seed = 0; seed < CASES; seed++) {
      const rand = rng(seed + 1);
      behaviors = new Map();
      recorded = new Map();

      const nodes = genTree(rand);
      const byId = new Map(nodes.map((n) => [n.id, n]));
      const states = new Map<string, number>();
      const randX = () => Math.floor(rand() * 6);

      // Build the tree by routing creates through the root, exactly as production does.
      const root = new PropComposite({
        path: ["n0"],
        type: "node",
        sender: MockSender,
        instrumentation: NOOP,
        parent: null,
      });
      states.set("n0", randX());
      root._updateState({
        path: ["n0"],
        state: { x: states.get("n0") as number, id: "n0" },
        type: "node",
        create: shouldNotCallCreate,
      });
      for (const node of nodes.slice(1)) {
        states.set(node.id, randX());
        root._updateState({
          path: node.fullPath,
          state: { x: states.get(node.id) as number, id: node.id },
          type: node.isLeaf ? "leaf" : "node",
          create: (parent) => construct(node, parent),
        });
      }

      // Drive a random sequence of state updates and deletes.
      const live = new Set(nodes.map((n) => n.id));
      const ops = 10 + Math.floor(rand() * 11);
      const liveList = () => [...live].map((id) => byId.get(id) as Node);
      for (let m = 0; m < ops; m++) {
        const deletable = liveList().filter((n) => n.id !== "n0");
        if (rand() < 0.2 && deletable.length > 0) {
          const target = deletable[Math.floor(rand() * deletable.length)];
          root._delete(target.fullPath);
          for (const n of nodes)
            if (
              n.fullPath.length >= target.fullPath.length &&
              target.fullPath.every((seg, i) => n.fullPath[i] === seg)
            )
              live.delete(n.id);
        } else {
          const candidates = liveList();
          const target = candidates[Math.floor(rand() * candidates.length)];
          states.set(target.id, randX());
          root._updateState({
            path: target.fullPath,
            state: { x: states.get(target.id) as number, id: target.id },
            type: target.isLeaf ? "leaf" : "node",
            create: shouldNotCallCreate,
          });
        }
      }

      const expected = oracle(nodes, live, states);
      for (const id of live)
        expect(
          recorded.get(id),
          `seed ${seed}, node ${id}: incremental propagation diverged from full recompute`,
        ).toEqual(expected.get(id));
    }
  });
});
