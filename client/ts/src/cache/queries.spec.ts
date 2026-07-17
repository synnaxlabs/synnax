// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { cache } from "@/cache";
import { NotFoundError } from "@/errors";

const flush = () => new Promise<void>((resolve) => setTimeout(resolve, 0));

type Q = { k: string };
const qA: Q = { k: "a" };
const qB: Q = { k: "b" };

interface CreateArgs<D extends cache.Data> {
  fetch?: (query: Q) => Promise<D>;
  mount?: cache.QueriesParams<Q, D>["mount"];
  ensureStreaming?: () => Promise<void>;
}

const createQueries = <D extends cache.Data>({
  fetch = () => Promise.reject(new Error("no fetch configured")),
  mount,
  ensureStreaming,
}: CreateArgs<D> = {}) =>
  new cache.Queries<Q, D>({ name: "thing", fetch, mount, ensureStreaming });

describe("hashQuery", () => {
  it("collapses key orderings to the same hash", () => {
    expect(cache.hashQuery({ a: 1, b: 2 })).toEqual(cache.hashQuery({ b: 2, a: 1 }));
  });

  it("preserves array order", () => {
    expect(cache.hashQuery([1, 2, 3])).not.toEqual(cache.hashQuery([3, 2, 1]));
  });

  it("hashes nested objects recursively", () => {
    expect(cache.hashQuery({ a: { x: 1, y: 2 } })).toEqual(
      cache.hashQuery({ a: { y: 2, x: 1 } }),
    );
  });

  it("hashes null and primitives", () => {
    expect(cache.hashQuery(null)).toEqual("null");
    expect(cache.hashQuery(undefined)).toEqual("undefined");
    expect(cache.hashQuery(42)).toEqual("42");
    expect(cache.hashQuery("x")).toEqual('"x"');
  });

  it("disambiguates null, undefined, and absent fields", () => {
    const hNull = cache.hashQuery(null);
    const hUndef = cache.hashQuery(undefined);
    expect(hNull).not.toEqual(hUndef);

    const nestedNull = cache.hashQuery({ a: null });
    const nestedUndef = cache.hashQuery({ a: undefined });
    const absent = cache.hashQuery({});
    expect(nestedNull).not.toEqual(nestedUndef);
    expect(nestedNull).not.toEqual(absent);
    expect(nestedUndef).not.toEqual(absent);
  });

  it("hashes bigints without throwing and disambiguates from same-valued numbers", () => {
    expect(cache.hashQuery(42n)).toEqual("42n");
    expect(cache.hashQuery(42n)).not.toEqual(cache.hashQuery(42));
    expect(cache.hashQuery({ k: 42n })).toEqual('{"k":42n}');
    expect(() => cache.hashQuery({ k: 9007199254740993n })).not.toThrow();
  });

  it("delegates to primitive.Hashable.hash() for class instances", () => {
    class TaggedID {
      constructor(private readonly v: string) {}
      hash(): string {
        return `tag:${this.v}`;
      }
    }
    expect(cache.hashQuery({ id: new TaggedID("abc") })).toEqual('{"id":tag:abc}');
    expect(cache.hashQuery(new TaggedID("xyz"))).toEqual("tag:xyz");
  });

  it("produces stable hashes across instances representing the same value", () => {
    class Wrapper {
      constructor(private readonly v: number) {}
      hash(): string {
        return this.v.toString();
      }
    }
    expect(cache.hashQuery({ k: new Wrapper(7) })).toEqual(
      cache.hashQuery({ k: new Wrapper(7) }),
    );
  });
});

describe("Queries", () => {
  describe("retrieve", () => {
    it("fetches on the first read and serves the cache while subscribed", async () => {
      const fetch = vi.fn(async () => 7);
      const queries = createQueries({ fetch });
      queries.onChange(qA, vi.fn());
      expect(await queries.retrieve(qA)).toEqual(7);
      expect(await queries.retrieve(qA)).toEqual(7);
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("refetches when nothing subscribes to the query", async () => {
      const fetch = vi.fn(async () => 7);
      const queries = createQueries({ fetch });
      expect(await queries.retrieve(qA)).toEqual(7);
      expect(await queries.retrieve(qA)).toEqual(7);
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("dedupes concurrent reads of the same query into one fetch", async () => {
      let resolve: (v: number) => void = () => {};
      const fetch = vi.fn(
        () =>
          new Promise<number>((r) => {
            resolve = r;
          }),
      );
      const queries = createQueries({ fetch });
      const [a, b] = [queries.retrieve(qA), queries.retrieve(qA)];
      // The fetch starts after an internal microtask; wait for it to capture
      // the resolver.
      await flush();
      resolve(3);
      expect(await a).toEqual(3);
      expect(await b).toEqual(3);
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("fetches separately for distinct queries", async () => {
      const fetch = vi.fn(async ({ k }: Q) => k);
      const queries = createQueries({ fetch });
      expect(await queries.retrieve(qA)).toEqual("a");
      expect(await queries.retrieve(qB)).toEqual("b");
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("treats equivalent queries with key reorderings as the same entry", async () => {
      const fetch = vi.fn(async () => 7);
      const queries = new cache.Queries<Record<string, number>, number>({
        name: "thing",
        fetch,
      });
      queries.onChange({ a: 1, b: 2 }, vi.fn());
      await queries.retrieve({ a: 1, b: 2 });
      await queries.retrieve({ b: 2, a: 1 });
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("treats distinct Hashable instances of the same value as the same entry", async () => {
      class Wrapper {
        constructor(private readonly v: number) {}
        hash(): string {
          return this.v.toString();
        }
      }
      const fetch = vi.fn(async () => 7);
      const queries = new cache.Queries<{ k: Wrapper }, number>({
        name: "thing",
        fetch,
      });
      queries.onChange({ k: new Wrapper(42) }, vi.fn());
      await queries.retrieve({ k: new Wrapper(42) });
      await queries.retrieve({ k: new Wrapper(42) });
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("isolates entries across separate instances", async () => {
      const a = createQueries({ fetch: async () => 1 });
      const b = createQueries({ fetch: vi.fn(async () => 2) });
      await a.retrieve(qA);
      expect(await b.retrieve(qA)).toEqual(2);
    });

    it("rejects with the fetch error", async () => {
      const queries = createQueries({
        fetch: async () => {
          throw new Error("boom");
        },
      });
      await expect(queries.retrieve(qA)).rejects.toThrow("boom");
    });

    it("retries after a failed fetch instead of caching the error", async () => {
      let calls = 0;
      const queries = createQueries({
        fetch: async () => {
          calls++;
          if (calls === 1) throw new Error("boom");
          return 9;
        },
      });
      await expect(queries.retrieve(qA)).rejects.toThrow("boom");
      expect(await queries.retrieve(qA)).toEqual(9);
    });

    it("starts streaming without letting it block the fetch", async () => {
      const ensureStreaming = vi.fn(() => new Promise<void>(() => {}));
      const queries = createQueries({ fetch: async () => 1, ensureStreaming });
      expect(await queries.retrieve(qA)).toEqual(1);
      expect(ensureStreaming).toHaveBeenCalledTimes(1);
    });
  });

  describe("getCached", () => {
    it("returns undefined for unfetched, in-flight, and failed queries", async () => {
      let resolve: (v: number) => void = () => {};
      const queries = createQueries<number>({
        fetch: () =>
          new Promise<number>((r) => {
            resolve = r;
          }),
      });
      expect(queries.getCached(qA)).toBeUndefined();
      const promise = queries.retrieve(qA);
      expect(queries.getCached(qA)).toBeUndefined();
      await flush();
      resolve(4);
      await promise;
      expect(queries.getCached(qA)).toEqual({ variant: "changed", data: 4 });
    });

    it("returns a referentially stable answer between changes", async () => {
      const queries = createQueries<number>({ fetch: async () => 1 });
      const off = queries.onChange(qA, () => {});
      await queries.retrieve(qA);
      const first = queries.getCached(qA);
      expect(queries.getCached(qA)).toBe(first);
      off();
    });
  });

  describe("onChange", () => {
    it("fires with the new answer when a listener updates it", async () => {
      let update: cache.MountParams<Q, number>["update"] = () => {};
      const queries = createQueries<number>({
        fetch: async () => 1,
        mount: (params) => {
          update = params.update;
          return [];
        },
      });
      const handler = vi.fn();
      queries.onChange(qA, handler);
      await queries.retrieve(qA);
      update(2);
      expect(handler).toHaveBeenCalledWith({ variant: "changed", data: 2 });
      expect(queries.getCached(qA)).toEqual({ variant: "changed", data: 2 });
    });

    it("passes the previous answer to setter updates", async () => {
      let update: cache.MountParams<Q, number>["update"] = () => {};
      const queries = createQueries<number>({
        fetch: async () => 10,
        mount: (params) => {
          update = params.update;
          return [];
        },
      });
      queries.onChange(qA, vi.fn());
      await queries.retrieve(qA);
      update((prev) => (prev ?? 0) + 5);
      expect(queries.getCached(qA)).toEqual({ variant: "changed", data: 15 });
    });

    it("scopes notifications by query", async () => {
      let update: cache.MountParams<Q, number>["update"] = () => {};
      const queries = createQueries<number>({
        fetch: async () => 1,
        mount: (params) => {
          if (params.query.k === "a") update = params.update;
          return [];
        },
      });
      const aHandler = vi.fn();
      const bHandler = vi.fn();
      queries.onChange(qA, aHandler);
      queries.onChange(qB, bHandler);
      update(2);
      expect(aHandler).toHaveBeenCalledTimes(1);
      expect(bHandler).not.toHaveBeenCalled();
    });

    it("returns a destructor that removes the handler", () => {
      let update: cache.MountParams<Q, number>["update"] = () => {};
      const queries = createQueries<number>({
        fetch: async () => 1,
        mount: (params) => {
          update = params.update;
          return [];
        },
      });
      const handler = vi.fn();
      const other = vi.fn();
      queries.onChange(qA, other);
      const unsubscribe = queries.onChange(qA, handler);
      unsubscribe();
      update(2);
      expect(handler).not.toHaveBeenCalled();
      expect(other).toHaveBeenCalledTimes(1);
    });

    it("fires when a fetch settles the answer", async () => {
      const queries = createQueries<number>({ fetch: async () => 8 });
      const handler = vi.fn();
      queries.onChange(qA, handler);
      await queries.retrieve(qA);
      await flush();
      expect(handler).toHaveBeenCalledWith({ variant: "changed", data: 8 });
    });

    it("does not fire on fetch failure", async () => {
      const queries = createQueries<number>({
        fetch: async () => {
          throw new Error("boom");
        },
      });
      const handler = vi.fn();
      queries.onChange(qA, handler);
      await expect(queries.retrieve(qA)).rejects.toThrow("boom");
      await flush();
      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe("listener mounting", () => {
    it("mounts on the first subscriber and unmounts on the last", () => {
      const destructor = vi.fn();
      const mount = vi.fn(() => [destructor]);
      const queries = createQueries<number>({ fetch: async () => 1, mount });
      const offA = queries.onChange(qA, vi.fn());
      const offB = queries.onChange(qA, vi.fn());
      expect(mount).toHaveBeenCalledTimes(1);
      offA();
      expect(destructor).not.toHaveBeenCalled();
      offB();
      expect(destructor).toHaveBeenCalledTimes(1);
    });

    it("mounts separately per query", () => {
      const mount = vi.fn((_: cache.MountParams<Q, number>) => []);
      const queries = createQueries<number>({ fetch: async () => 1, mount });
      queries.onChange(qA, vi.fn());
      queries.onChange(qB, vi.fn());
      expect(mount).toHaveBeenCalledTimes(2);
      expect(mount.mock.calls[0][0].query).toEqual(qA);
      expect(mount.mock.calls[1][0].query).toEqual(qB);
    });

    it("remounts after a full unsubscribe", () => {
      const mount = vi.fn(() => []);
      const queries = createQueries<number>({ fetch: async () => 1, mount });
      const off = queries.onChange(qA, vi.fn());
      off();
      queries.onChange(qA, vi.fn());
      expect(mount).toHaveBeenCalledTimes(2);
    });
  });

  describe("deletion", () => {
    it("delivers the corpse through onChange and getCached", async () => {
      let remove: cache.MountParams<Q, number>["remove"] = () => {};
      const queries = createQueries<number>({
        fetch: async () => 6,
        mount: (params) => {
          remove = params.remove;
          return [];
        },
      });
      const handler = vi.fn();
      queries.onChange(qA, handler);
      await queries.retrieve(qA);
      remove();
      expect(handler).toHaveBeenCalledWith({ variant: "deleted", corpse: 6 });
      expect(queries.getCached(qA)).toEqual({ variant: "deleted", corpse: 6 });
    });

    it("prefers an explicit corpse over the cached answer", async () => {
      let remove: cache.MountParams<Q, number>["remove"] = () => {};
      const queries = createQueries<number>({
        fetch: async () => 6,
        mount: (params) => {
          remove = params.remove;
          return [];
        },
      });
      queries.onChange(qA, vi.fn());
      await queries.retrieve(qA);
      remove(99);
      expect(queries.getCached(qA)).toEqual({ variant: "deleted", corpse: 99 });
    });

    it("invalidates when there is no corpse to retain", () => {
      let remove: cache.MountParams<Q, number>["remove"] = () => {};
      const queries = createQueries<number>({
        fetch: async () => 6,
        mount: (params) => {
          remove = params.remove;
          return [];
        },
      });
      queries.onChange(qA, vi.fn());
      remove();
      expect(queries.getCached(qA)).toBeUndefined();
    });

    it("rejects retrieve with NotFoundError for a deleted answer", async () => {
      let remove: cache.MountParams<Q, number>["remove"] = () => {};
      const queries = createQueries<number>({
        fetch: async () => 6,
        mount: (params) => {
          remove = params.remove;
          return [];
        },
      });
      queries.onChange(qA, vi.fn());
      await queries.retrieve(qA);
      remove();
      await expect(queries.retrieve(qA)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  describe("invalidation", () => {
    it("drops the entry, notifies with undefined, and refetches on next read", async () => {
      let invalidate: cache.MountParams<Q, number>["invalidate"] = () => {};
      const fetch = vi.fn(async () => 1);
      const queries = createQueries<number>({
        fetch,
        mount: (params) => {
          invalidate = params.invalidate;
          return [];
        },
      });
      const handler = vi.fn();
      queries.onChange(qA, handler);
      await queries.retrieve(qA);
      invalidate();
      expect(handler).toHaveBeenLastCalledWith(undefined);
      expect(queries.getCached(qA)).toBeUndefined();
      await queries.retrieve(qA);
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("invalidates when an update produces null", async () => {
      let update: cache.MountParams<Q, number>["update"] = () => {};
      const queries = createQueries<number>({
        fetch: async () => 1,
        mount: (params) => {
          update = params.update;
          return [];
        },
      });
      queries.onChange(qA, vi.fn());
      await queries.retrieve(qA);
      update(null);
      expect(queries.getCached(qA)).toBeUndefined();
    });

    it("keeps an in-flight fetch when an update produces null with nothing cached", async () => {
      let resolve: (v: number) => void = () => {};
      let update: cache.MountParams<Q, number>["update"] = () => {};
      const queries = createQueries<number>({
        fetch: () =>
          new Promise<number>((r) => {
            resolve = r;
          }),
        mount: (params) => {
          update = params.update;
          return [];
        },
      });
      queries.onChange(qA, vi.fn());
      const promise = queries.retrieve(qA);
      await flush();
      update((prev) => prev);
      resolve(1);
      await expect(promise).resolves.toEqual(1);
      expect(queries.getCached(qA)).toEqual({ variant: "changed", data: 1 });
    });

    it("does not notify when invalidating an absent entry", () => {
      let invalidate: cache.MountParams<Q, number>["invalidate"] = () => {};
      const queries = createQueries<number>({
        fetch: async () => 1,
        mount: (params) => {
          invalidate = params.invalidate;
          return [];
        },
      });
      const handler = vi.fn();
      queries.onChange(qA, handler);
      invalidate();
      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe("identity-gated settle", () => {
    it("ignores a late resolution that follows a listener update", async () => {
      let resolve: (v: number) => void = () => {};
      let update: cache.MountParams<Q, number>["update"] = () => {};
      const queries = createQueries<number>({
        fetch: () =>
          new Promise<number>((r) => {
            resolve = r;
          }),
        mount: (params) => {
          update = params.update;
          return [];
        },
      });
      queries.onChange(qA, vi.fn());
      const promise = queries.retrieve(qA);
      await flush();
      update(99);
      resolve(1);
      await promise;
      await flush();
      expect(queries.getCached(qA)).toEqual({ variant: "changed", data: 99 });
    });
  });
});
