// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { base } from "@/flux/base";
import { pendingResult, successResult } from "@/flux/result";

const flush = () => new Promise<void>((resolve) => setTimeout(resolve, 0));

type Q = { k: string };
const qA: Q = { k: "a" };
const qB: Q = { k: "b" };

describe("hashQuery", () => {
  it("collapses key orderings to the same hash", () => {
    expect(base.hashQuery({ a: 1, b: 2 })).toEqual(base.hashQuery({ b: 2, a: 1 }));
  });

  it("preserves array order", () => {
    expect(base.hashQuery([1, 2, 3])).not.toEqual(base.hashQuery([3, 2, 1]));
  });

  it("hashes nested objects recursively", () => {
    expect(base.hashQuery({ a: { x: 1, y: 2 } })).toEqual(
      base.hashQuery({ a: { y: 2, x: 1 } }),
    );
  });

  it("hashes null and primitives", () => {
    expect(base.hashQuery(null)).toEqual("null");
    expect(base.hashQuery(undefined)).toEqual("undefined");
    expect(base.hashQuery(42)).toEqual("42");
    expect(base.hashQuery("x")).toEqual('"x"');
  });

  it("disambiguates null, undefined, and absent fields", () => {
    const hNull = base.hashQuery(null);
    const hUndef = base.hashQuery(undefined);
    expect(hNull).not.toEqual(hUndef);

    const nestedNull = base.hashQuery({ a: null });
    const nestedUndef = base.hashQuery({ a: undefined });
    const absent = base.hashQuery({});
    expect(nestedNull).not.toEqual(nestedUndef);
    expect(nestedNull).not.toEqual(absent);
    expect(nestedUndef).not.toEqual(absent);
  });

  it("hashes bigints without throwing and disambiguates from same-valued numbers", () => {
    expect(base.hashQuery(42n)).toEqual("42n");
    expect(base.hashQuery(42n)).not.toEqual(base.hashQuery(42));
    expect(base.hashQuery({ k: 42n })).toEqual('{"k":42n}');
    expect(() => base.hashQuery({ k: 9007199254740993n })).not.toThrow();
  });

  it("delegates to primitive.Hashable.hash() for class instances", () => {
    class TaggedID {
      constructor(private readonly v: string) {}
      hash(): string {
        return `tag:${this.v}`;
      }
    }
    expect(base.hashQuery({ id: new TaggedID("abc") })).toEqual('{"id":tag:abc}');
    expect(base.hashQuery(new TaggedID("xyz"))).toEqual("tag:xyz");
  });

  it("produces stable hashes across instances representing the same value", () => {
    class Wrapper {
      constructor(private readonly v: number) {}
      hash(): string {
        return this.v.toString();
      }
    }
    expect(base.hashQuery({ k: new Wrapper(7) })).toEqual(
      base.hashQuery({ k: new Wrapper(7) }),
    );
  });
});

describe("QueryCache", () => {
  describe("get / set", () => {
    it("returns undefined for an absent query", () => {
      const cache = new base.QueryCache<Q, number>();
      expect(cache.get(qA)).toBeUndefined();
    });

    it("returns the stored result", () => {
      const cache = new base.QueryCache<Q, number>();
      const result = successResult<number>("retrieved x", 7);
      cache.set(qA, result);
      expect(cache.get(qA)).toBe(result);
    });

    it("treats equivalent queries with key reorderings as the same entry", () => {
      const cache = new base.QueryCache<Record<string, number>, number>();
      const result = successResult<number>("retrieved", 7);
      cache.set({ a: 1, b: 2 }, result);
      expect(cache.get({ b: 2, a: 1 })).toBe(result);
    });

    it("treats distinct Hashable instances of the same value as the same entry", () => {
      class Wrapper {
        constructor(private readonly v: number) {}
        hash(): string {
          return this.v.toString();
        }
      }
      const cache = new base.QueryCache<{ k: Wrapper }, number>();
      const result = successResult<number>("retrieved", 7);
      cache.set({ k: new Wrapper(42) }, result);
      expect(cache.get({ k: new Wrapper(42) })).toBe(result);
    });

    it("isolates entries across separate QueryCache instances", () => {
      const a = new base.QueryCache<Q, number>();
      const b = new base.QueryCache<Q, number>();
      a.set(qA, successResult<number>("retrieved", 1));
      expect(b.get(qA)).toBeUndefined();
    });
  });

  describe("subscribe", () => {
    it("notifies subscribers on set", () => {
      const cache = new base.QueryCache<Q, number>();
      const listener = vi.fn();
      cache.subscribe(qA, listener);
      cache.set(qA, successResult<number>("retrieved", 1));
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("scopes notifications by query", () => {
      const cache = new base.QueryCache<Q, number>();
      const aListener = vi.fn();
      const bListener = vi.fn();
      cache.subscribe(qA, aListener);
      cache.subscribe(qB, bListener);
      cache.set(qA, successResult<number>("retrieved", 1));
      expect(aListener).toHaveBeenCalledTimes(1);
      expect(bListener).not.toHaveBeenCalled();
    });

    it("returns a destructor that removes the listener", () => {
      const cache = new base.QueryCache<Q, number>();
      const listener = vi.fn();
      const unsubscribe = cache.subscribe(qA, listener);
      unsubscribe();
      cache.set(qA, successResult<number>("retrieved", 1));
      expect(listener).not.toHaveBeenCalled();
    });
  });

  describe("auto-transition on settle", () => {
    it("replaces a loading entry with a success entry when the promise resolves", async () => {
      const cache = new base.QueryCache<Q, number>();
      let resolve: (v: number) => void = () => {};
      const promise = new Promise<number>((r) => {
        resolve = r;
      });
      cache.set(qA, pendingResult<number>("Number", promise));
      resolve(42);
      await flush();
      const entry = cache.get(qA);
      expect(entry?.variant).toEqual("success");
      expect(entry?.variant === "success" && entry.data).toEqual(42);
    });

    it("replaces a loading entry with an error entry when the promise rejects", async () => {
      const cache = new base.QueryCache<Q, number>();
      let reject: (reason: unknown) => void = () => {};
      const promise = new Promise<number>((_, r) => {
        reject = r;
      });
      cache.set(qA, pendingResult<number>("Number", promise));
      reject(new Error("boom"));
      await flush();
      const entry = cache.get(qA);
      expect(entry?.variant).toEqual("error");
    });

    it("transitions to error when the promise rejects with a non-Error value", async () => {
      const cache = new base.QueryCache<Q, number>();
      let reject: (reason: unknown) => void = () => {};
      const promise = new Promise<number>((_, r) => {
        reject = r;
      });
      cache.set(qA, pendingResult<number>("Number", promise));
      reject("oops");
      await flush();
      const entry = cache.get(qA);
      expect(entry?.variant).toEqual("error");
    });

    it("notifies subscribers on the settle transition", async () => {
      const cache = new base.QueryCache<Q, number>();
      const listener = vi.fn();
      let resolve: (v: number) => void = () => {};
      const promise = new Promise<number>((r) => {
        resolve = r;
      });
      cache.subscribe(qA, listener);
      cache.set(qA, pendingResult<number>("Number", promise));
      listener.mockClear();
      resolve(1);
      await flush();
      expect(listener).toHaveBeenCalledTimes(1);
    });
  });

  describe("identity-gated replacement", () => {
    it("ignores a late promise resolution that follows a listener push", async () => {
      const cache = new base.QueryCache<Q, number>();
      let resolve: (v: number) => void = () => {};
      const promise = new Promise<number>((r) => {
        resolve = r;
      });
      cache.set(qA, pendingResult<number>("Number", promise));
      cache.set(qA, successResult<number>("listener", 99));
      resolve(1);
      await flush();
      const entry = cache.get(qA);
      expect(entry?.variant === "success" && entry.data).toEqual(99);
    });

    it("ignores a late promise rejection that follows a listener push", async () => {
      const cache = new base.QueryCache<Q, number>();
      let reject: (reason: unknown) => void = () => {};
      const promise = new Promise<number>((_, r) => {
        reject = r;
      });
      cache.set(qA, pendingResult<number>("Number", promise));
      cache.set(qA, successResult<number>("listener", 5));
      reject(new Error("ignored"));
      await flush();
      const entry = cache.get(qA);
      expect(entry?.variant).toEqual("success");
    });
  });

  describe("invalidate", () => {
    it("removes the entry and notifies subscribers", () => {
      const cache = new base.QueryCache<Q, number>();
      const listener = vi.fn();
      cache.set(qA, successResult<number>("retrieved", 1));
      cache.subscribe(qA, listener);
      cache.invalidate(qA);
      expect(cache.get(qA)).toBeUndefined();
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("no-ops on an absent query", () => {
      const cache = new base.QueryCache<Q, number>();
      const listener = vi.fn();
      cache.subscribe(qA, listener);
      cache.invalidate(qA);
      expect(listener).not.toHaveBeenCalled();
    });
  });
});
