// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { hashQuery, QueryCache } from "@/flux/base/queryCache";
import { pendingResult, successResult } from "@/flux/result";

const flush = () => new Promise<void>((resolve) => setTimeout(resolve, 0));

describe("hashQuery", () => {
  it("collapses key orderings to the same hash", () => {
    expect(hashQuery({ a: 1, b: 2 })).toEqual(hashQuery({ b: 2, a: 1 }));
  });

  it("preserves array order", () => {
    expect(hashQuery([1, 2, 3])).not.toEqual(hashQuery([3, 2, 1]));
  });

  it("hashes nested objects recursively", () => {
    expect(hashQuery({ a: { x: 1, y: 2 } })).toEqual(hashQuery({ a: { y: 2, x: 1 } }));
  });

  it("hashes null and primitives", () => {
    expect(hashQuery(null)).toEqual("null");
    expect(hashQuery(undefined)).toEqual("null");
    expect(hashQuery(42)).toEqual("42");
    expect(hashQuery("x")).toEqual('"x"');
  });

  it("delegates to primitive.Hashable.hash() for class instances", () => {
    class TaggedID {
      constructor(private readonly v: string) {}
      hash(): string {
        return `tag:${this.v}`;
      }
    }
    expect(hashQuery({ id: new TaggedID("abc") })).toEqual('{"id":tag:abc}');
    expect(hashQuery(new TaggedID("xyz"))).toEqual("tag:xyz");
  });

  it("produces stable hashes across instances representing the same value", () => {
    class Wrapper {
      constructor(private readonly v: number) {}
      hash(): string {
        return this.v.toString();
      }
    }
    expect(hashQuery({ k: new Wrapper(7) })).toEqual(hashQuery({ k: new Wrapper(7) }));
  });
});

describe("QueryCache", () => {
  describe("get / set", () => {
    it("returns undefined for an absent hash", () => {
      const cache = new QueryCache();
      expect(cache.get("missing")).toBeUndefined();
    });

    it("returns the stored result", () => {
      const cache = new QueryCache();
      const result = successResult<number>("retrieved x", 7);
      cache.set("k", result);
      expect(cache.get("k")).toBe(result);
    });
  });

  describe("subscribe", () => {
    it("notifies subscribers on set", () => {
      const cache = new QueryCache();
      const listener = vi.fn();
      cache.subscribe("k", listener);
      cache.set("k", successResult<number>("retrieved", 1));
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("scopes notifications by hash", () => {
      const cache = new QueryCache();
      const aListener = vi.fn();
      const bListener = vi.fn();
      cache.subscribe("a", aListener);
      cache.subscribe("b", bListener);
      cache.set("a", successResult<number>("retrieved", 1));
      expect(aListener).toHaveBeenCalledTimes(1);
      expect(bListener).not.toHaveBeenCalled();
    });

    it("returns a destructor that removes the listener", () => {
      const cache = new QueryCache();
      const listener = vi.fn();
      const unsubscribe = cache.subscribe("k", listener);
      unsubscribe();
      cache.set("k", successResult<number>("retrieved", 1));
      expect(listener).not.toHaveBeenCalled();
    });
  });

  describe("auto-transition on settle", () => {
    it("replaces a loading entry with a success entry when the promise resolves", async () => {
      const cache = new QueryCache();
      let resolve: (v: number) => void = () => {};
      const promise = new Promise<number>((r) => {
        resolve = r;
      });
      cache.set("k", pendingResult<number>("Number", promise));
      resolve(42);
      await flush();
      const entry = cache.get<number>("k");
      expect(entry?.variant).toEqual("success");
      expect(entry?.variant === "success" && entry.data).toEqual(42);
    });

    it("replaces a loading entry with an error entry when the promise rejects", async () => {
      const cache = new QueryCache();
      let reject: (reason: unknown) => void = () => {};
      const promise = new Promise<number>((_, r) => {
        reject = r;
      });
      cache.set("k", pendingResult<number>("Number", promise));
      reject(new Error("boom"));
      await flush();
      const entry = cache.get<number>("k");
      expect(entry?.variant).toEqual("error");
    });

    it("transitions to error when the promise rejects with a non-Error value", async () => {
      const cache = new QueryCache();
      let reject: (reason: unknown) => void = () => {};
      const promise = new Promise<number>((_, r) => {
        reject = r;
      });
      cache.set("k", pendingResult<number>("Number", promise));
      reject("oops");
      await flush();
      const entry = cache.get<number>("k");
      expect(entry?.variant).toEqual("error");
    });

    it("notifies subscribers on the settle transition", async () => {
      const cache = new QueryCache();
      const listener = vi.fn();
      let resolve: (v: number) => void = () => {};
      const promise = new Promise<number>((r) => {
        resolve = r;
      });
      cache.subscribe("k", listener);
      cache.set("k", pendingResult<number>("Number", promise));
      listener.mockClear();
      resolve(1);
      await flush();
      expect(listener).toHaveBeenCalledTimes(1);
    });
  });

  describe("identity-gated replacement", () => {
    it("ignores a late promise resolution that follows a listener push", async () => {
      const cache = new QueryCache();
      let resolve: (v: number) => void = () => {};
      const promise = new Promise<number>((r) => {
        resolve = r;
      });
      cache.set("k", pendingResult<number>("Number", promise));
      cache.set("k", successResult<number>("listener", 99));
      resolve(1);
      await flush();
      const entry = cache.get<number>("k");
      expect(entry?.variant === "success" && entry.data).toEqual(99);
    });

    it("ignores a late promise rejection that follows a listener push", async () => {
      const cache = new QueryCache();
      let reject: (reason: unknown) => void = () => {};
      const promise = new Promise<number>((_, r) => {
        reject = r;
      });
      cache.set("k", pendingResult<number>("Number", promise));
      cache.set("k", successResult<number>("listener", 5));
      reject(new Error("ignored"));
      await flush();
      const entry = cache.get<number>("k");
      expect(entry?.variant).toEqual("success");
    });
  });

  describe("invalidate", () => {
    it("removes the entry and notifies subscribers", () => {
      const cache = new QueryCache();
      const listener = vi.fn();
      cache.set("k", successResult<number>("retrieved", 1));
      cache.subscribe("k", listener);
      cache.invalidate("k");
      expect(cache.get("k")).toBeUndefined();
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("no-ops on an absent hash", () => {
      const cache = new QueryCache();
      const listener = vi.fn();
      cache.subscribe("k", listener);
      cache.invalidate("k");
      expect(listener).not.toHaveBeenCalled();
    });
  });
});
