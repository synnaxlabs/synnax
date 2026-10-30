// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type Synnax as Client } from "@synnaxlabs/client";
import { act, render, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";
import { Flux } from "@/flux";
import { type LocalCache, setSettled, subscribeReader } from "@/flux/suspend";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const newLocal = (): LocalCache<number> => ({
  epoch: 0,
  inFlight: new Map(),
  settled: new Map(),
  generations: new Map(),
  fetches: new Map(),
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

interface Answer {
  resolve: (n: number) => void;
  reject: (cause: Error) => void;
}

interface MountReadersParams {
  domainCached?: boolean;
  retryUnreachable?: boolean;
}

const WARM = { key: "warm" };
const QUERY = { key: "query" };
const READERS = ["first", "second", "third"];

const client = {
  connection: { status: { details: { epoch: 0 } } },
} as unknown as Client;
const Wrapper = createSynnaxWrapper({ client });

// Each reader commits on WARM before it reads QUERY. A committed reader keeps its
// pending fetch across a suspension, so it shows that fetch's answer. A new mount would
// drop it and show whatever settled instead.
const mountReaders = async ({
  domainCached = false,
  retryUnreachable = false,
}: MountReadersParams = {}) => {
  const answers: Answer[] = [];
  const cache: { value?: number } = {};
  const { use, useInvalidate } = Flux.createRetrieve<{ key: string }, number>({
    name: "Number",
    retrieve: async ({ query: { key } }) => {
      if (key === WARM.key) return 0;
      return await new Promise<number>((resolve, reject) =>
        answers.push({ resolve, reject }),
      );
    },
    getCached: ({ query: { key } }) =>
      domainCached && key === QUERY.key ? cache.value : undefined,
    retryUnreachable,
  });
  const reads = new Map<string, () => void>();
  let discard: (q: { key: string }) => void = () => {};
  const Value = ({ q, id }: { q: { key: string }; id: string }): ReactElement => (
    <div data-testid={id}>{use(q)}</div>
  );
  const Reader = ({ id }: { id: string }): ReactElement => {
    const [q, setQ] = useState(WARM);
    reads.set(id, () => setQ(QUERY));
    return (
      <Errors.SuspenseBoundary loading={null}>
        <Value q={q} id={id} />
      </Errors.SuspenseBoundary>
    );
  };
  const Discard = (): null => {
    discard = useInvalidate();
    return null;
  };
  let utils!: ReturnType<typeof render>;
  await act(async () => {
    utils = render(
      <Wrapper>
        <Discard />
        {READERS.map((id) => (
          <Reader key={id} id={id} />
        ))}
      </Wrapper>,
    );
  });
  const expectValue = async (id: string, value: number): Promise<void> => {
    await waitFor(() => expect(utils.getByTestId(id).textContent).toBe(`${value}`));
  };
  await Promise.all(READERS.map(async (id) => await expectValue(id, 0)));
  const read = async (id: string): Promise<void> => {
    await act(async () => reads.get(id)?.());
  };
  return {
    answers,
    cache,
    read,
    expectValue,
    discard: () => discard(QUERY),
    // A new reader takes the settled answer without a fetch.
    expectSettled: async (value: number): Promise<void> => {
      const fetches = answers.length;
      await read("third");
      await expectValue("third", value);
      expect(answers).toHaveLength(fetches);
    },
  };
};

describe("invalidate", () => {
  it("starts a new fetch instead of joining the one it discarded", async () => {
    const { answers, read, expectValue, discard, expectSettled } = await mountReaders();
    await read("first");
    discard();
    await read("second");
    expect(answers).toHaveLength(2);
    answers[1].resolve(2);
    await expectValue("second", 2);
    answers[0].resolve(1);
    await expectValue("first", 2);
    await expectSettled(2);
  });

  it("refetches for a reader waiting on a fetch it discarded", async () => {
    const { answers, read, expectValue, discard, expectSettled } = await mountReaders();
    await read("first");
    discard();
    answers[0].resolve(1);
    await vi.waitFor(() => expect(answers).toHaveLength(2));
    answers[1].resolve(2);
    await expectValue("first", 2);
    await expectSettled(2);
  });

  describe("with a domain cache", () => {
    it("should give a discarded fetch's reader the cached replacement", async () => {
      const { answers, cache, read, expectValue, discard } = await mountReaders({
        domainCached: true,
      });
      await read("first");
      discard();
      await read("second");
      cache.value = 2;
      answers[1].resolve(2);
      await expectValue("second", 2);
      answers[0].resolve(1);
      await expectValue("first", 2);
    });

    it("should give the cached replacement when that fetch fails", async () => {
      const { answers, cache, read, expectValue, discard } = await mountReaders({
        domainCached: true,
      });
      await read("first");
      discard();
      await read("second");
      cache.value = 2;
      answers[1].resolve(2);
      await expectValue("second", 2);
      answers[0].reject(new Error("stale"));
      await expectValue("first", 2);
    });
  });

  describe("during a recovery", () => {
    // A dead connection sends the fetch to a retry, whose attempt is the recovery.
    const startRecovery = async () => {
      const readers = await mountReaders({ retryUnreachable: true });
      await readers.read("first");
      readers.answers[0].reject(new DisconnectedError());
      await vi.waitFor(() => expect(readers.answers).toHaveLength(2));
      readers.discard();
      return { ...readers, recovery: readers.answers[1] };
    };

    it("should give the replacement when the recovery succeeds late", async () => {
      const { answers, recovery, read, expectValue, expectSettled } =
        await startRecovery();
      await read("second");
      answers[2].resolve(2);
      await expectValue("second", 2);
      recovery.resolve(1);
      await expectValue("first", 2);
      await expectSettled(2);
    });

    it("should give the replacement when the recovery fails late", async () => {
      const { answers, recovery, read, expectValue, expectSettled } =
        await startRecovery();
      await read("second");
      answers[2].resolve(2);
      await expectValue("second", 2);
      recovery.reject(new Error("stale"));
      await expectValue("first", 2);
      await expectSettled(2);
    });

    it("should refetch when the discarded recovery succeeds", async () => {
      const { answers, recovery, expectValue, expectSettled } = await startRecovery();
      recovery.resolve(1);
      await vi.waitFor(() => expect(answers).toHaveLength(3));
      answers[2].resolve(2);
      await expectValue("first", 2);
      await expectSettled(2);
    });

    it("should refetch when the discarded recovery fails", async () => {
      const { answers, recovery, expectValue, expectSettled } = await startRecovery();
      recovery.reject(new Error("stale"));
      await vi.waitFor(() => expect(answers).toHaveLength(3));
      answers[2].resolve(2);
      await expectValue("first", 2);
      await expectSettled(2);
    });
  });

  it("drops the settled answer", async () => {
    const { answers, read, expectValue, discard } = await mountReaders();
    await read("first");
    answers[0].resolve(1);
    await expectValue("first", 1);
    discard();
    await read("second");
    expect(answers).toHaveLength(2);
    answers[1].resolve(2);
    await expectValue("second", 2);
  });
});
