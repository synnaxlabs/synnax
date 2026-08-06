// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, query, Synnax } from "@synnaxlabs/client";
import { type destructor, TimeStamp } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { flux } from "@/flux/aether";

type Query = {
  key: string;
};

type Data = {
  key: string;
  name: string;
};

const CLIENT = new Synnax({
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  cache: false,
});

type RetrieveMock = ReturnType<
  typeof vi.fn<(params: flux.RetrieveParams<Query>) => Promise<Data>>
>;

interface Harness {
  definition: flux.Definition<Query, Data>;
  retrieve: RetrieveMock;
  handlers: query.ChangeHandler<Data>[];
  cache: Map<string, query.Cached<Data>>;
  disconnects: ReturnType<typeof vi.fn<destructor.Destructor>>[];
}

const createHarness = (): Harness => {
  const handlers: query.ChangeHandler<Data>[] = [];
  const disconnects: ReturnType<typeof vi.fn<destructor.Destructor>>[] = [];
  const cache = new Map<string, query.Cached<Data>>();
  const retrieve = vi.fn(
    async ({ query: q }: flux.RetrieveParams<Query>): Promise<Data> => ({
      key: q.key,
      name: `name-${q.key}`,
    }),
  );
  const definition: flux.Definition<Query, Data> = {
    name: "test",
    retrieve,
    onChange: (_, handler) => {
      handlers.push(handler);
      const disconnect = vi.fn(() => {
        handlers.splice(handlers.indexOf(handler), 1);
      });
      disconnects.push(disconnect);
      return disconnect;
    },
    getCached: ({ query: q }) => cache.get(q.key),
  };
  return { definition, retrieve, handlers, cache, disconnects };
};

const flush = async () => await new Promise((resolve) => setTimeout(resolve, 0));

describe("flux.Retrieve", () => {
  let harness: Harness;
  let onChange: ReturnType<
    typeof vi.fn<(result: query.Cached<Data> | undefined) => void>
  >;
  let onError: ReturnType<typeof vi.fn<(error: Error) => void>>;

  beforeEach(() => {
    harness = createHarness();
    onChange = vi.fn();
    onError = vi.fn();
  });

  const create = () =>
    new flux.Retrieve<Query, Data>({
      definition: harness.definition,
      onChange,
      onError,
    });

  it("seeds synchronously from the cache without fetching", () => {
    const cached = { key: "a", name: "cached" };
    harness.cache.set("a", cached);
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    expect(observer.value).toEqual(cached);
    expect(harness.retrieve).not.toHaveBeenCalled();
    expect(onChange).toHaveBeenCalledWith(cached);
  });

  it("fetches on a cache miss and pushes the result", async () => {
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    expect(observer.value).toBeUndefined();
    await flush();
    expect(observer.value).toEqual({ key: "a", name: "name-a" });
    expect(onChange).toHaveBeenCalledWith({ key: "a", name: "name-a" });
  });

  it("resolves not-found to an undefined answer without erroring", async () => {
    harness.retrieve.mockRejectedValueOnce(new NotFoundError("nope"));
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    await flush();
    expect(observer.value).toBeUndefined();
    expect(onError).not.toHaveBeenCalled();
  });

  it("routes fetch failures to onError", async () => {
    const failure = new Error("boom");
    harness.retrieve.mockRejectedValueOnce(failure);
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    await flush();
    expect(onError).toHaveBeenCalledWith(failure);
    expect(observer.value).toBeUndefined();
  });

  it("pushes live subscription updates", async () => {
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    await flush();
    const next = { key: "a", name: "renamed" };
    harness.handlers[0](next);
    expect(observer.value).toEqual(next);
    expect(onChange).toHaveBeenLastCalledWith(next);
  });

  it("holds the previous answer across an invalidation push", async () => {
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    await flush();
    const held = observer.value;
    onChange.mockClear();
    harness.handlers[0](undefined);
    expect(observer.value).toEqual(held);
    expect(onChange).not.toHaveBeenCalled();
  });

  it("forwards a deletion push", async () => {
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    await flush();
    const corpse = { key: "a", name: "name-a" };
    const deleted = new query.Deleted(corpse, TimeStamp.now());
    harness.handlers[0](deleted);
    expect(observer.value).toBeUndefined();
    expect(query.Deleted.matches(observer.cached)).toBe(true);
    expect(onChange).toHaveBeenLastCalledWith(deleted);
  });

  it("ignores an update with an equal query and client", async () => {
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    await flush();
    observer.update(CLIENT, { key: "a" });
    expect(harness.retrieve).toHaveBeenCalledTimes(1);
    expect(harness.disconnects[0]).not.toHaveBeenCalled();
  });

  it("re-subscribes when the query changes", async () => {
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    await flush();
    observer.update(CLIENT, { key: "b" });
    expect(harness.disconnects[0]).toHaveBeenCalled();
    expect(harness.handlers).toHaveLength(1);
    await flush();
    expect(observer.value).toEqual({ key: "b", name: "name-b" });
  });

  it("drops a stale fetch that resolves after a query change", async () => {
    let resolveSlow: (data: Data) => void = () => {};
    harness.retrieve.mockImplementationOnce(
      async () => await new Promise<Data>((resolve) => (resolveSlow = resolve)),
    );
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    observer.update(CLIENT, { key: "b" });
    await flush();
    resolveSlow({ key: "a", name: "stale" });
    await flush();
    expect(observer.value).toEqual({ key: "b", name: "name-b" });
  });

  it("clears the answer when the client is null", () => {
    const cached = { key: "a", name: "cached" };
    harness.cache.set("a", cached);
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    observer.update(null, { key: "a" });
    expect(observer.value).toBeUndefined();
    expect(harness.disconnects[0]).toHaveBeenCalled();
  });

  it("stops pushing after close", async () => {
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    observer.close();
    await flush();
    expect(observer.value).toBeUndefined();
    expect(harness.disconnects[0]).toHaveBeenCalled();
  });

  it("refetches the current query on demand", async () => {
    const observer = create();
    observer.update(CLIENT, { key: "a" });
    await flush();
    observer.refetch();
    await flush();
    expect(harness.retrieve).toHaveBeenCalledTimes(2);
  });
});
