// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { Series, sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";
import type { z } from "zod";

import { calculation } from "@/calculation";
import { framer } from "@/framer";
import { query } from "@/query";
import { status } from "@/status";
import { PushStreamer } from "@/testutil";

const A = "0a7d3c2e-6f1b-4c5d-9e8f-1a2b3c4d5e6f";
const B = "1b8e4d3f-7a2c-4d6e-8f9a-2b3c4d5e6f70";
const C = "2c9f5e4a-8b3d-4e7f-9a0b-3c4d5e6f7a81";
const X = 1048577;
const Y = 1048578;
const Z = 1048579;

type Where = { key?: { eq?: string; in?: string[] }; output?: { in?: number[] } };

interface Held {
  /** Reads the Core's calculations now, as the Core answers the request. */
  read: () => void;
  /** Returns the read answer to the client. */
  release: () => void;
}

const createFakeCore = () => {
  const calcs = new Map<string, calculation.Calculation>();
  const heldOutputs: Held[] = [];
  const heldDeletes: Array<() => void> = [];
  const core = {
    calcs,
    heldOutputs,
    heldDeletes,
    holdOutputs: false,
    holdDelete: false,
    refuseOutputs: false,
    lookups: [] as number[][],
  };
  const select = (where: Where | undefined): calculation.Calculation[] =>
    [...calcs.values()].filter((c) => {
      if (where?.key != null) {
        const keys = where.key.in ?? (where.key.eq == null ? [] : [where.key.eq]);
        return keys.includes(c.key);
      }
      if (where?.output?.in != null) return where.output.in.includes(c.output);
      return true;
    });
  const send = async (
    target: string,
    req: { where?: Where; keys?: string[]; calculations?: unknown[] },
    _: z.ZodType,
    res: z.ZodType,
  ): Promise<unknown> => {
    switch (target) {
      case "/calculation/retrieve": {
        const answer = () => res.parse({ calculations: select(req.where) });
        if (req.where?.output?.in != null) core.lookups.push(req.where.output.in);
        if (core.refuseOutputs && req.where?.output != null)
          throw new Error("the Core refused the lookup");
        if (!core.holdOutputs || req.where?.output == null) return answer();
        return await new Promise((resolve) => {
          let reply: unknown;
          heldOutputs.push({
            read: () => {
              reply = answer();
            },
            release: () => resolve(reply ?? answer()),
          });
        });
      }
      case "/calculation/create": {
        const created = (
          req.calculations as Array<{ calculation: calculation.Calculation }>
        ).map(({ calculation: c }) => c);
        created.forEach((c) => calcs.set(c.key, c));
        return res.parse({ calculations: created });
      }
      case "/calculation/delete": {
        req.keys?.forEach((k) => calcs.delete(k));
        if (core.holdDelete) await new Promise<void>((r) => heldDeletes.push(r));
        return res.parse({});
      }
      default:
        throw new Error(`unexpected request to ${target}`);
    }
  };
  const unary = { use: () => {}, send } as unknown as UnaryClient;
  return { core, unary };
};

const createDetached = () => {
  const { core, unary } = createFakeCore();
  const cache = new query.Cache({ openStreamer: null, onError: () => {} });
  onTestFinished(async () => await cache.close());
  const statuses = new status.Client({ unary, cache });
  const calcs = new calculation.Client({ unary, cache, statuses });
  return { core, calcs, cache };
};

const createStreamed = async (recordCap?: number) => {
  const { core, unary } = createFakeCore();
  const stream = new PushStreamer();
  const cache = new query.Cache({
    openStreamer: async () => stream,
    breaker: { baseInterval: TimeSpan.milliseconds(1) },
    onError: () => {},
    recordCap,
  });
  onTestFinished(async () => await cache.close());
  const statuses = new status.Client({ unary, cache });
  const calcs = new calculation.Client({ unary, cache, statuses });
  await cache.ensureStreaming();
  return { core, calcs, stream };
};

const createdFrame = (
  key: string,
  output: number,
  seq: number,
  extra: Partial<calculation.Calculation> = {},
) =>
  new framer.Frame({
    [calculation.ACTION_CHANNEL_NAME]: new Series([
      {
        key,
        dispatchKey: `remote-${seq}`,
        seq,
        actions: [
          calculation.create({
            calculation: { key, output, expression: "return 1", ...extra },
          }),
        ],
      },
    ]),
  });

const deletedFrame = (key: string) =>
  new framer.Frame({ [calculation.DELETE_CHANNEL_NAME]: new Series([key]) });

describe("calculation.Client breaker", () => {
  describe("outputs", () => {
    it("should look a failed channel up again once a later lookup answers", async () => {
      const { core, calcs } = createDetached();
      core.calcs.set(A, {
        key: A,
        output: X,
        expression: "return 1",
        operations: [],
        reads: [],
      });
      core.refuseOutputs = true;
      await expect(calcs.outputs.retrieve([X])).rejects.toThrow();
      core.refuseOutputs = false;
      await calcs.outputs.retrieve([Y]);
      await expect
        .poll(() => calcs.outputs.get(X))
        .toMatchObject({ hidden: false, calculation: A });
    });

    it("should not look a channel up again once a lookup answered it", async () => {
      const { core, calcs } = createDetached();
      core.refuseOutputs = true;
      await expect(calcs.outputs.retrieve([X])).rejects.toThrow();
      core.refuseOutputs = false;
      await calcs.outputs.retrieve([Y]);
      await expect.poll(() => calcs.outputs.get(X)).toBeDefined();
      core.lookups.length = 0;
      await calcs.outputs.retrieve([Z]);
      await sleep.sleep(TimeSpan.milliseconds(100));
      expect(core.lookups).toEqual([[Z]]);
    });

    it("should not restore a calculation deleted while a lookup by output is in flight", async () => {
      const { core, calcs } = createDetached();
      core.holdOutputs = true;
      const lookup = calcs.outputs.retrieve([X]);
      await expect.poll(() => core.heldOutputs.length).toBe(1);
      await calcs.create({ key: A, output: X, expression: "return 1" });
      // The Core answers the lookup after the create and before the delete.
      core.heldOutputs[0].read();
      await calcs.delete(A);
      expect(calcs.outputs.get(X)).toMatchObject({ hidden: false, calculation: null });
      core.heldOutputs[0].release();
      await lookup;
      expect(calcs.outputs.get(X)).toMatchObject({ hidden: false, calculation: null });
      await expect(calcs.retrieve(A)).rejects.toThrow();
    });

    it("should drop a lookup by output that a cache reset outdates", async () => {
      const { core, calcs, cache } = createDetached();
      core.calcs.set(A, {
        key: A,
        output: X,
        expression: "return 1",
        operations: [],
        reads: [],
      });
      core.holdOutputs = true;
      const lookup = calcs.outputs.retrieve([X]);
      await expect.poll(() => core.heldOutputs.length).toBe(1);
      core.heldOutputs[0].read();
      // The Core behind the connection is replaced while the lookup is in flight.
      await cache.reset();
      core.heldOutputs[0].release();
      await lookup;
      expect(calcs.outputs.get(X)).toBeUndefined();
    });

    it("should not overwrite a newer cached calculation with a lookup by output", async () => {
      const { core, calcs, stream } = await createStreamed();
      core.holdOutputs = true;
      const lookup = calcs.outputs.retrieve([X]);
      await expect.poll(() => core.heldOutputs.length).toBe(1);
      core.calcs.set(A, {
        key: A,
        output: X,
        expression: "return 1",
        operations: [],
        reads: [],
      });
      stream.push(createdFrame(A, X, 1));
      await expect
        .poll(() => calcs.outputs.get(X))
        .toMatchObject({ hidden: false, calculation: A });
      // The Core answers the lookup, then the expression changes.
      core.heldOutputs[0].read();
      core.calcs.set(A, { ...core.calcs.get(A)!, expression: "return 2" });
      stream.push(
        new framer.Frame({
          [calculation.ACTION_CHANNEL_NAME]: new Series([
            {
              key: A,
              dispatchKey: "remote-2",
              seq: 2,
              actions: [calculation.set({ path: ["expression"], value: "return 2" })],
            },
          ]),
        }),
      );
      await expect
        .poll(async () => (await calcs.retrieve(A)).expression)
        .toBe("return 2");
      core.heldOutputs[0].release();
      await lookup;
      expect((await calcs.retrieve(A)).expression).toBe("return 2");
    });

    it("should keep a new calculation's output when the stream echoes an old delete", async () => {
      const { calcs, stream } = await createStreamed();
      await calcs.create({ key: A, output: X, expression: "return 1" });
      await calcs.delete(A);
      await calcs.create({ key: B, output: X, expression: "return 2" });
      expect(calcs.outputs.get(X)).toMatchObject({ hidden: false, calculation: B });
      // The echo of our delete lands after the create's reply.
      stream.push(deletedFrame(A));
      stream.push(createdFrame(B, X, 1, { expression: "return 2" }));
      // A later frame fences the two above.
      stream.push(createdFrame(C, Y, 1));
      await expect
        .poll(() => calcs.outputs.get(Y))
        .toMatchObject({ hidden: false, calculation: C });
      expect(calcs.outputs.get(X)).toMatchObject({ hidden: false, calculation: B });
    });

    it("should keep a remote calculation's output when our delete's reply lands after it", async () => {
      const { core, calcs, stream } = await createStreamed();
      await calcs.create({ key: A, output: X, expression: "return 1" });
      core.holdDelete = true;
      const deleting = calcs.delete(A);
      await expect.poll(() => core.heldDeletes.length).toBe(1);
      // The Core commits the delete, another client writes X, and both reach the
      // stream before our reply.
      stream.push(deletedFrame(A));
      stream.push(createdFrame(B, X, 1));
      await expect
        .poll(() => calcs.outputs.get(X))
        .toMatchObject({ hidden: false, calculation: B });
      core.heldDeletes[0]();
      await deleting;
      expect(calcs.outputs.get(X)).toMatchObject({ hidden: false, calculation: B });
    });

    it("should map a watched output to none when its trimmed calculation is deleted", async () => {
      const { calcs, stream } = await createStreamed(2);
      // A keyed reader holds the entry of X, as Pluto's useIsOutput does.
      const off = calcs.outputs.subscribe(() => {}, X);
      onTestFinished(() => off());
      const keys = [A, B, C];
      keys.forEach((key, i) => stream.push(createdFrame(key, X + i, 1)));
      await expect
        .poll(() => calcs.outputs.get(X + 2))
        .toMatchObject({ hidden: false, calculation: C });
      // A fourth calculation trims the three before it from the calculation table.
      const D = "3d0a6f5b-9c4e-4f80-8b1c-4d5e6f7a8b92";
      stream.push(createdFrame(D, X + 3, 1));
      await expect
        .poll(() => calcs.outputs.get(X + 3))
        .toMatchObject({ hidden: false, calculation: D });
      expect(calcs.outputs.get(X)).toMatchObject({ hidden: false, calculation: A });
      stream.push(deletedFrame(A));
      const E = "4e1b7a6c-0d5f-4a91-9c2d-5e6f7a8b9ca3";
      stream.push(createdFrame(E, X + 4, 1));
      await expect
        .poll(() => calcs.outputs.get(X + 4))
        .toMatchObject({ hidden: false, calculation: E });
      expect(calcs.outputs.get(X)).toMatchObject({ hidden: false, calculation: null });
    });
  });

  describe("status", () => {
    it("should not compose a status for a request that leaves it out", async () => {
      const { calcs, stream } = await createStreamed();
      stream.push(
        createdFrame(A, X, 1, {
          status: {
            key: "5c1e7a0b-2d3f-4e6a-8b9c-0d1e2f3a4b5c",
            name: "Calculation status",
            variant: "info",
            message: "carried by the create",
            description: "",
            time: TimeStamp.now(),
            labels: [],
            owner: calculation.resourceID(A),
            details: undefined,
          },
        }),
      );
      await expect
        .poll(() => calcs.outputs.get(X))
        .toMatchObject({ hidden: false, calculation: A });
      const [bare] = await calcs.retrieve({ where: { key: { in: [A] } } });
      expect(bare.status).toBeUndefined();
    });
  });
});
