// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { bench, describe, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";

const MockSender = {
  send: vi.fn(),
};

const shouldNotCallCreate = () => {
  throw new Error("should not call create");
};

const complexSchema = z.object({
  id: z.string(),
  value: z.number(),
  metadata: z.object({
    timestamp: z.number(),
    tags: z.array(z.string()),
    config: z.record(z.string(), z.union([z.string(), z.number(), z.boolean()])),
  }),
  status: z.enum(["active", "inactive", "pending"]),
});

class BenchRoot extends aether.Composite<typeof complexSchema, {}, BenchL1> {
  schema = complexSchema;
  afterUpdate(ctx: aether.Context): void {
    ctx.set("rootTime", Date.now());
    ctx.set("rootValue", this.state.value);
  }
}

class BenchL1 extends aether.Composite<typeof complexSchema, {}, BenchL2> {
  schema = complexSchema;
  afterUpdate(ctx: aether.Context): void {
    const rootValue = ctx.getOptional<number>("rootValue") ?? 0;
    ctx.set("l1Value", rootValue * 2);
    ctx.set("l1Status", this.state.status);
  }
}

class BenchL2 extends aether.Composite<typeof complexSchema, {}, BenchL3> {
  schema = complexSchema;
  afterUpdate(ctx: aether.Context): void {
    const l1Value = ctx.getOptional<number>("l1Value") ?? 0;
    ctx.set("l2Value", l1Value * 1.5);
    ctx.set("l2Status", this.state.status);
  }
}

class BenchL3 extends aether.Leaf<typeof complexSchema, { computedValue: number }> {
  schema = complexSchema;
  afterUpdate(ctx: aether.Context): void {
    const l2Value = ctx.getOptional<number>("l2Value") ?? 0;
    this.internal.computedValue = l2Value + this.state.value;
  }
}

// Sparse-subscription variant: same tree shape, but the intermediate levels are pure
// structure (they neither read nor publish context) and only ~1/15 of the leaves read
// the root's value. A root update therefore re-runs only those subscribers, not the
// whole subtree — the case the broadcast model could not exploit.
class SparseRoot extends aether.Composite<typeof complexSchema, {}, SparseMid> {
  schema = complexSchema;
  afterUpdate(ctx: aether.Context): void {
    ctx.set("rootValue", this.state.value);
  }
}

class SparseMid extends aether.Composite<
  typeof complexSchema,
  {},
  SparseMid | SparseLeaf
> {
  schema = complexSchema;
}

class SparseLeaf extends aether.Leaf<typeof complexSchema, { computedValue: number }> {
  schema = complexSchema;
  afterUpdate(ctx: aether.Context): void {
    if (this.state.value % 15 !== 0) return;
    this.internal.computedValue = ctx.getOptional<number>("rootValue") ?? 0;
  }
}

// Add this function before the benchmark
function createBenchmarkTree() {
  const root = new BenchRoot({
    path: ["root"],
    type: "bench",
    sender: MockSender,
    instrumentation: alamos.NOOP,
    parent: null,
  });

  // Initialize the root
  root._updateState({
    path: ["root"],
    type: "bench",
    state: {
      id: "root-1",
      value: 100,
      metadata: {
        timestamp: Date.now(),
        tags: ["benchmark"],
        config: { setting1: "test", setting2: 42 },
      },
      status: "active",
    },
    create: shouldNotCallCreate,
  });

  // Create 15 L1 nodes, each with 15 L2 nodes, each with 15 L3 nodes
  for (let i = 0; i < 15; i++) {
    const l1Key = `l1-${i}`;
    root._updateState({
      path: ["root", l1Key],
      type: "bench",
      state: {
        id: l1Key,
        value: i * 10,
        metadata: {
          timestamp: Date.now(),
          tags: ["l1", `group-${i}`],
          config: { index: i },
        },
        status: "active",
      },
      create: (parent) =>
        new BenchL1({
          path: ["root", l1Key],
          type: "bench",
          sender: MockSender,
          instrumentation: alamos.NOOP,
          parent,
        }),
    });

    for (let j = 0; j < 15; j++) {
      const l2Key = `l2-${i}-${j}`;
      root._updateState({
        path: ["root", l1Key, l2Key],
        type: "bench",
        state: {
          id: l2Key,
          value: j * 5,
          metadata: {
            timestamp: Date.now(),
            tags: ["l2", `group-${i}-${j}`],
            config: { parentIndex: i, index: j },
          },
          status: "active",
        },
        create: (parent) =>
          new BenchL2({
            path: ["root", l1Key, l2Key],
            type: "bench",
            sender: MockSender,
            instrumentation: alamos.NOOP,
            parent,
          }),
      });

      for (let k = 0; k < 15; k++) {
        const l3Key = `l3-${i}-${j}-${k}`;
        root._updateState({
          path: ["root", l1Key, l2Key, l3Key],
          type: "bench",
          state: {
            id: l3Key,
            value: k,
            metadata: {
              timestamp: Date.now(),
              tags: ["l3"],
              config: { parentL1: i, parentL2: j, index: k },
            },
            status: "pending",
          },
          create: (parent) =>
            new BenchL3({
              path: ["root", l1Key, l2Key, l3Key],
              type: "bench",
              sender: MockSender,
              instrumentation: alamos.NOOP,
              parent,
            }),
        });
      }
    }
  }
  return root;
}

const sparseState = (id: string, value: number) => ({
  id,
  value,
  metadata: { timestamp: Date.now(), tags: [id], config: { value } },
  status: "active" as const,
});

function createSparseTree(): SparseRoot {
  const root = new SparseRoot({
    path: ["root"],
    type: "bench",
    sender: MockSender,
    instrumentation: alamos.NOOP,
    parent: null,
  });
  root._updateState({
    path: ["root"],
    type: "bench",
    state: sparseState("root", 100),
    create: shouldNotCallCreate,
  });
  for (let i = 0; i < 15; i++) {
    const l1Key = `l1-${i}`;
    root._updateState({
      path: ["root", l1Key],
      type: "bench",
      state: sparseState(l1Key, i),
      create: (parent) =>
        new SparseMid({
          path: ["root", l1Key],
          type: "bench",
          sender: MockSender,
          instrumentation: alamos.NOOP,
          parent,
        }),
    });
    for (let j = 0; j < 15; j++) {
      const l2Key = `l2-${i}-${j}`;
      root._updateState({
        path: ["root", l1Key, l2Key],
        type: "bench",
        state: sparseState(l2Key, j),
        create: (parent) =>
          new SparseMid({
            path: ["root", l1Key, l2Key],
            type: "bench",
            sender: MockSender,
            instrumentation: alamos.NOOP,
            parent,
          }),
      });
      for (let k = 0; k < 15; k++) {
        const l3Key = `l3-${i}-${j}-${k}`;
        root._updateState({
          path: ["root", l1Key, l2Key, l3Key],
          type: "bench",
          state: sparseState(l3Key, k),
          create: (parent) =>
            new SparseLeaf({
              path: ["root", l1Key, l2Key, l3Key],
              type: "bench",
              sender: MockSender,
              instrumentation: alamos.NOOP,
              parent,
            }),
        });
      }
    }
  }
  return root;
}

// Full-subscription worst case: every node reads its parent's value, so a root update
// touches all 3616 nodes either way. Guards against regression from the per-key
// bookkeeping overhead.
describe("deep tree updates (full subscription)", () => {
  let root: BenchRoot;
  bench(
    "should update the entire tree",
    () => {
      root._updateState({
        path: ["root"],
        type: "bench",
        state: {
          id: "root-1",
          value: 200,
          metadata: {
            timestamp: Date.now(),
            tags: ["benchmark", "updated"],
            config: { setting1: "test", setting2: 84 },
          },
          status: "active",
        },
        create: shouldNotCallCreate,
      });
    },
    {
      time: 1000,
      setup: () => {
        root = createBenchmarkTree();
      },
    },
  );
});

// Sparse-subscription case: same 3616-node tree, but only 225 leaves read the root's
// value. A root update re-runs only those subscribers, so this should be dramatically
// faster than the full-subscription case — the win the broadcast model could not get.
describe("deep tree updates (sparse subscription)", () => {
  let root: SparseRoot;
  bench(
    "should update only the subscribed leaves",
    () => {
      root._updateState({
        path: ["root"],
        type: "bench",
        state: sparseState("root", 200),
        create: shouldNotCallCreate,
      });
    },
    {
      time: 1000,
      setup: () => {
        root = createSparseTree();
      },
    },
  );
});
