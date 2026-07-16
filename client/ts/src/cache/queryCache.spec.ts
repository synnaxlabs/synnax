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

const flush = () => new Promise<void>((resolve) => setTimeout(resolve, 0));

const successState = <D extends cache.Data>(data: D): cache.QueryState<D> => ({
  variant: "success",
  data,
});
const loadingState = <D extends cache.Data>(
  promise: Promise<D>,
): cache.QueryState<D> => ({ variant: "loading", promise });

type Q = { k: string };
const qA: Q = { k: "a" };
const qB: Q = { k: "b" };

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

describe("QueryCache", () => {
  describe("get / set", () => {
    it("returns undefined for an absent query", () => {
      const qc = new cache.QueryCache<Q, number>();
      expect(qc.get(qA)).toBeUndefined();
    });

    it("returns the stored result", () => {
      const qc = new cache.QueryCache<Q, number>();
      const result = successState(7);
      qc.set(qA, result);
      expect(qc.get(qA)).toBe(result);
    });

    it("treats equivalent queries with key reorderings as the same entry", () => {
      const qc = new cache.QueryCache<Record<string, number>, number>();
      const result = successState(7);
      qc.set({ a: 1, b: 2 }, result);
      expect(qc.get({ b: 2, a: 1 })).toBe(result);
    });

    it("treats distinct Hashable instances of the same value as the same entry", () => {
      class Wrapper {
        constructor(private readonly v: number) {}
        hash(): string {
          return this.v.toString();
        }
      }
      const qc = new cache.QueryCache<{ k: Wrapper }, number>();
      const result = successState(7);
      qc.set({ k: new Wrapper(42) }, result);
      expect(qc.get({ k: new Wrapper(42) })).toBe(result);
    });

    it("isolates entries across separate QueryCache instances", () => {
      const a = new cache.QueryCache<Q, number>();
      const b = new cache.QueryCache<Q, number>();
      a.set(qA, successState(1));
      expect(b.get(qA)).toBeUndefined();
    });
  });

  describe("subscribe", () => {
    it("notifies subscribers on set", () => {
      const qc = new cache.QueryCache<Q, number>();
      const listener = vi.fn();
      qc.subscribe(qA, listener);
      qc.set(qA, successState(1));
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("scopes notifications by query", () => {
      const qc = new cache.QueryCache<Q, number>();
      const aListener = vi.fn();
      const bListener = vi.fn();
      qc.subscribe(qA, aListener);
      qc.subscribe(qB, bListener);
      qc.set(qA, successState(1));
      expect(aListener).toHaveBeenCalledTimes(1);
      expect(bListener).not.toHaveBeenCalled();
    });

    it("returns a destructor that removes the listener", () => {
      const qc = new cache.QueryCache<Q, number>();
      const listener = vi.fn();
      const unsubscribe = qc.subscribe(qA, listener);
      unsubscribe();
      qc.set(qA, successState(1));
      expect(listener).not.toHaveBeenCalled();
    });
  });

  describe("auto-transition on settle", () => {
    it("replaces a loading entry with a success entry when the promise resolves", async () => {
      const qc = new cache.QueryCache<Q, number>();
      let resolve: (v: number) => void = () => {};
      const promise = new Promise<number>((r) => {
        resolve = r;
      });
      qc.set(qA, loadingState(promise));
      resolve(42);
      await flush();
      const entry = qc.get(qA);
      expect(entry?.variant).toEqual("success");
      expect(entry?.variant === "success" && entry.data).toEqual(42);
    });

    it("replaces a loading entry with an error entry when the promise rejects", async () => {
      const qc = new cache.QueryCache<Q, number>();
      let reject: (reason: unknown) => void = () => {};
      const promise = new Promise<number>((_, r) => {
        reject = r;
      });
      qc.set(qA, loadingState(promise));
      reject(new Error("boom"));
      await flush();
      const entry = qc.get(qA);
      expect(entry?.variant).toEqual("error");
    });

    it("transitions to error when the promise rejects with a non-Error value", async () => {
      const qc = new cache.QueryCache<Q, number>();
      let reject: (reason: unknown) => void = () => {};
      const promise = new Promise<number>((_, r) => {
        reject = r;
      });
      qc.set(qA, loadingState(promise));
      reject("oops");
      await flush();
      const entry = qc.get(qA);
      expect(entry?.variant).toEqual("error");
    });

    it("notifies subscribers on the settle transition", async () => {
      const qc = new cache.QueryCache<Q, number>();
      const listener = vi.fn();
      let resolve: (v: number) => void = () => {};
      const promise = new Promise<number>((r) => {
        resolve = r;
      });
      qc.subscribe(qA, listener);
      qc.set(qA, loadingState(promise));
      listener.mockClear();
      resolve(1);
      await flush();
      expect(listener).toHaveBeenCalledTimes(1);
    });
  });

  describe("identity-gated replacement", () => {
    it("ignores a late promise resolution that follows a listener push", async () => {
      const qc = new cache.QueryCache<Q, number>();
      let resolve: (v: number) => void = () => {};
      const promise = new Promise<number>((r) => {
        resolve = r;
      });
      qc.set(qA, loadingState(promise));
      qc.set(qA, successState(99));
      resolve(1);
      await flush();
      const entry = qc.get(qA);
      expect(entry?.variant === "success" && entry.data).toEqual(99);
    });

    it("ignores a late promise rejection that follows a listener push", async () => {
      const qc = new cache.QueryCache<Q, number>();
      let reject: (reason: unknown) => void = () => {};
      const promise = new Promise<number>((_, r) => {
        reject = r;
      });
      qc.set(qA, loadingState(promise));
      qc.set(qA, successState(5));
      reject(new Error("ignored"));
      await flush();
      const entry = qc.get(qA);
      expect(entry?.variant).toEqual("success");
    });
  });

  describe("invalidate", () => {
    it("removes the entry and notifies subscribers", () => {
      const qc = new cache.QueryCache<Q, number>();
      const listener = vi.fn();
      qc.set(qA, successState(1));
      qc.subscribe(qA, listener);
      qc.invalidate(qA);
      expect(qc.get(qA)).toBeUndefined();
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("no-ops on an absent query", () => {
      const qc = new cache.QueryCache<Q, number>();
      const listener = vi.fn();
      qc.subscribe(qA, listener);
      qc.invalidate(qA);
      expect(listener).not.toHaveBeenCalled();
    });
  });
});
