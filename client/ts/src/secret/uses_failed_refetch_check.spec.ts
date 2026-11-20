// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { Series, TimeSpan, uuid } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";
import { type z } from "zod";

import { NotFoundError } from "@/errors";
import { framer } from "@/framer";
import { query } from "@/query";
import { secret } from "@/secret";
import { PushStreamer } from "@/testutil";

const usesOf = (key: secret.Key) => [{ type: "certificate", key }];
const movedUsesOf = (key: secret.Key) => [{ type: "certificate", key: `${key}-moved` }];

interface RetrieveRequest {
  where: { key: { eq?: secret.Key; in?: secret.Key[] } };
  includeUses?: boolean;
}

interface Gate {
  reached: boolean;
  release: (err: Error | null) => void;
}

const createFake = async () => {
  const usesRequests: secret.Key[][] = [];
  const errors: unknown[] = [];
  const core = {
    usesOf,
    failing: null as ((keys: secret.Key[]) => Error | null) | null,
    gate: null as Gate | null,
    gated: null as Promise<Error | null> | null,
  };
  const send = async (
    target: string,
    req: RetrieveRequest,
    _: z.ZodType,
    res: z.ZodType,
  ) => {
    if (target !== "/secret/retrieve")
      throw new Error(`unexpected request to ${target}`);
    const { eq, in: keys = eq == null ? [] : [eq] } = req.where.key;
    if (req.includeUses === true) {
      usesRequests.push(keys);
      if (core.gated != null) {
        const gated = core.gated;
        core.gated = null;
        core.gate!.reached = true;
        const err = await gated;
        if (err != null) throw err;
      }
      const err = core.failing?.(keys) ?? null;
      if (err != null) throw err;
    }
    return res.parse({
      secrets: keys.map((key) => ({
        key,
        name: key,
        source: "stored",
        updated: 0,
        ...(req.includeUses === true && { uses: core.usesOf(key) }),
      })),
    });
  };
  const unary = { use: () => {}, send } as unknown as UnaryClient;
  let stream = new PushStreamer();
  let opened = 0;
  const cache = new query.Cache({
    openStreamer: async () => {
      opened++;
      if (opened > 1) stream = new PushStreamer();
      return stream;
    },
    breaker: { baseInterval: TimeSpan.milliseconds(1) },
    onError: (err) => errors.push(err),
  });
  onTestFinished(async () => await cache.close());
  const secrets = new secret.Client({ unary, cache });
  await cache.ensureStreaming();
  const change = (...keys: secret.Key[]): void =>
    stream.push(
      new framer.Frame({
        [query.REFERENCE_CHANGE_CHANNEL_NAME]: new Series(
          keys.map((key) => `${secret.TYPE_RESOURCE_ID.type}:${key}`),
        ),
      }),
    );
  const remove = (key: secret.Key): void =>
    stream.push(new framer.Frame({ [secret.DELETE_CHANNEL_NAME]: new Series([key]) }));
  const watch = async (...keys: secret.Key[]) => {
    for (const key of keys) {
      const q = { key, includeUses: true };
      onTestFinished(secrets.watch(q).onChange(() => {}));
      await secrets.retrieve(q);
    }
  };
  const gateNext = (): Gate => {
    let release: (err: Error | null) => void = () => {};
    core.gated = new Promise((resolve) => (release = resolve));
    const gate: Gate = { reached: false, release: (err) => release(err) };
    core.gate = gate;
    return gate;
  };
  const cachedUses = (key: secret.Key) => {
    const cached = secrets.watch({ key, includeUses: true }).getCached();
    return query.isLive(cached) ? cached.uses : undefined;
  };
  const reopened = () => opened;
  return {
    secrets,
    core,
    usesRequests,
    errors,
    change,
    remove,
    watch,
    gateNext,
    cachedUses,
    drop: () => stream.drop(),
    reopened,
  };
};

const sorted = (keys: secret.Key[]) => [...keys].sort();

describe("Secret uses after a failed refetch", () => {
  it("should refetch both keys when a change for another key lands during a failed refetch", async () => {
    const f = await createFake();
    const [a, b] = [uuid.create(), uuid.create()];
    await f.watch(a, b);
    const gate = f.gateNext();
    f.change(a);
    await expect.poll(() => gate.reached).toBe(true);
    f.core.usesOf = movedUsesOf;
    f.change(b);
    gate.release(new Error("internal server error"));
    await expect.poll(() => f.cachedUses(b)).toEqual(movedUsesOf(b));
    f.change(b);
    await expect.poll(() => f.cachedUses(a)).toEqual(movedUsesOf(a));
  });

  it("should refetch a key whose change lands again during its failed refetch", async () => {
    const f = await createFake();
    const [a] = [uuid.create()];
    await f.watch(a);
    const gate = f.gateNext();
    f.change(a);
    await expect.poll(() => gate.reached).toBe(true);
    f.core.usesOf = movedUsesOf;
    f.change(a);
    gate.release(new Error("internal server error"));
    await expect.poll(() => f.cachedUses(a)).toEqual(movedUsesOf(a));
  });

  it("should not refetch a key deleted during its failed refetch", async () => {
    const f = await createFake();
    const [a, b] = [uuid.create(), uuid.create()];
    await f.watch(a, b);
    const gate = f.gateNext();
    f.change(a);
    await expect.poll(() => gate.reached).toBe(true);
    f.remove(a);
    await expect.poll(() => f.cachedUses(a)).toBeUndefined();
    gate.release(new Error("internal server error"));
    const n = f.usesRequests.length;
    f.core.usesOf = movedUsesOf;
    f.change(b);
    await expect.poll(() => f.cachedUses(b)).toEqual(movedUsesOf(b));
    expect(f.usesRequests.slice(n).filter((keys) => keys.includes(b))).toEqual([[b]]);
  });

  it("should refetch every key after a reconnect during a failed refetch", async () => {
    const f = await createFake();
    const [a, b] = [uuid.create(), uuid.create()];
    await f.watch(a, b);
    const gate = f.gateNext();
    f.change(a);
    await expect.poll(() => gate.reached).toBe(true);
    f.core.usesOf = movedUsesOf;
    f.drop();
    await expect.poll(() => f.reopened()).toBeGreaterThan(1);
    gate.release(new Error("internal server error"));
    await expect.poll(() => f.cachedUses(a)).toEqual(movedUsesOf(a));
    await expect.poll(() => f.cachedUses(b)).toEqual(movedUsesOf(b));
  });

  it("should keep the survivors of a refetch that finds a deleted key", async () => {
    const f = await createFake();
    const [a, b] = [uuid.create(), uuid.create()];
    await f.watch(a, b);
    f.core.usesOf = movedUsesOf;
    f.core.failing = (keys) =>
      keys.includes(a) ? new NotFoundError(`secret ${a} not found`) : null;
    f.change(a, b);
    await expect.poll(() => f.cachedUses(b)).toEqual(movedUsesOf(b));
    expect(f.errors).toEqual([]);
    const n = f.usesRequests.length;
    f.change(b);
    await expect.poll(() => f.usesRequests.length).toBeGreaterThan(n);
    expect(f.usesRequests.slice(n)).toEqual([[b]]);
  });

  it("should refetch every key of a split refetch whose half fails", async () => {
    const f = await createFake();
    const [a, b, c] = [uuid.create(), uuid.create(), uuid.create()];
    await f.watch(a, b, c);
    f.core.usesOf = movedUsesOf;
    let failB = true;
    f.core.failing = (keys) => {
      if (keys.includes(a)) return new NotFoundError(`secret ${a} not found`);
      if (failB && keys.length === 1 && keys[0] === b) return new Error("internal");
      return null;
    };
    f.change(a, b, c);
    await expect.poll(() => f.errors.length).toBeGreaterThan(0);
    failB = false;
    const n = f.usesRequests.length;
    f.change(c);
    await expect.poll(() => f.cachedUses(b)).toEqual(movedUsesOf(b));
    await expect.poll(() => f.cachedUses(c)).toEqual(movedUsesOf(c));
    expect(sorted(f.usesRequests[n])).toEqual(sorted([a, b, c]));
  });
});
