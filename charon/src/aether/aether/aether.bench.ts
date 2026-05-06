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

// Add this function before the benchmark
function createBenchmarkTree() {
  const root = new BenchRoot({
    key: "root",
    type: "bench",
    sender: MockSender,
    instrumentation: alamos.NOOP,
    parentCtxValues: null,
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
      create: (parentCtxValues) =>
        new BenchL1({
          key: l1Key,
          type: "bench",
          sender: MockSender,
          instrumentation: alamos.NOOP,
          parentCtxValues,
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
        create: (parentCtxValues) =>
          new BenchL2({
            key: l2Key,
            type: "bench",
            sender: MockSender,
            instrumentation: alamos.NOOP,
            parentCtxValues,
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
          create: (parentCtxValues) =>
            new BenchL3({
              key: l3Key,
              type: "bench",
              sender: MockSender,
              instrumentation: alamos.NOOP,
              parentCtxValues,
            }),
        });
      }
    }
  }
  return root;
}

describe("deep tree updates", () => {
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
