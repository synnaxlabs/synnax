// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, type Synnax as Client } from "@synnaxlabs/client";
import { describe, expect, it, vi } from "vitest";

import {
  ensureFetch,
  invalidate,
  type LocalCache,
  setSettled,
  subscribeReader,
} from "@/flux/suspend";

const newLocal = (): LocalCache<number> => ({
  epoch: 0,
  inFlight: new Map(),
  settled: new Map(),
  generations: new Map(),
  listeners: new Map(),
});

describe("setSettled", () => {
  it("evicts the oldest entry once the cap is exceeded", () => {
    const local = newLocal();
    for (let i = 0; i < 257; i++) setSettled(local, `q${i}`, { data: i });
    expect(local.settled.size).toEqual(256);
    expect(local.settled.has("q0")).toBe(false);
    expect(local.settled.has("q1")).toBe(true);
    expect(local.settled.has("q256")).toBe(true);
  });

  it("refreshes recency on re-insert so hot entries survive eviction", () => {
    const local = newLocal();
    for (let i = 0; i < 256; i++) setSettled(local, `q${i}`, { data: i });
    setSettled(local, "q0", { data: -1 });
    setSettled(local, "fresh", { data: 999 });
    expect(local.settled.has("q0")).toBe(true);
    expect(local.settled.get("q0")).toEqual({ data: -1 });
    expect(local.settled.has("q1")).toBe(false);
  });

  it("notifies listeners after the entry lands", () => {
    const local = newLocal();
    const listener = vi.fn(() => expect(local.settled.get("q")).toEqual({ data: 1 }));
    subscribeReader(local, "q", listener);
    setSettled(local, "q", { data: 1 });
    expect(listener).toHaveBeenCalledTimes(1);
  });

  it("notifies only the readers of the settled query", () => {
    const local = newLocal();
    const listener = vi.fn();
    subscribeReader(local, "other", listener);
    setSettled(local, "q", { data: 1 });
    expect(listener).not.toHaveBeenCalled();
  });
});

describe("subscribeReader", () => {
  it("drops the hash once its last reader leaves", () => {
    const local = newLocal();
    const release = subscribeReader(local, "q", vi.fn());
    release();
    expect(local.listeners.has("q")).toBe(false);
  });
});

describe("invalidate", () => {
  const params = { client: null as unknown as Client, query: { key: "q" } };
  const fetcher = (local: LocalCache<number>) => {
    const answers: Array<(n: number) => void> = [];
    const fetch = async (): Promise<number> =>
      await ensureFetch(params, {
        name: "number",
        retrieve: async () => await new Promise<number>((r) => answers.push(r)),
        local,
      });
    return { answers, fetch };
  };

  it("starts a new fetch instead of joining the one it discarded", async () => {
    const local = newLocal();
    const { answers, fetch } = fetcher(local);
    const first = fetch();
    invalidate(local, query.hash(params.query));
    const second = fetch();
    expect(answers).toHaveLength(2);
    answers[1](2);
    expect(await second).toBe(2);
    answers[0](1);
    expect(await first).toBe(2);
    expect([...local.settled.values()]).toEqual([{ data: 2 }]);
  });

  it("refetches for a reader waiting on a fetch it discarded", async () => {
    const local = newLocal();
    const { answers, fetch } = fetcher(local);
    const first = fetch();
    invalidate(local, query.hash(params.query));
    answers[0](1);
    await vi.waitFor(() => expect(answers).toHaveLength(2));
    answers[1](2);
    expect(await first).toBe(2);
    expect([...local.settled.values()]).toEqual([{ data: 2 }]);
  });

  it("drops the settled answer", () => {
    const local = newLocal();
    setSettled(local, "q", { data: 1 });
    invalidate(local, "q");
    expect(local.settled.has("q")).toBe(false);
  });
});
