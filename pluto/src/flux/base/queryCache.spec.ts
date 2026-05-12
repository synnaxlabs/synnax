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
import { errorResult, pendingResult, successResult } from "@/flux/result";

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

  describe("equal callback", () => {
    it("skips notification on success-to-success when data is equal", () => {
      const cache = new QueryCache();
      const equal = (a: { v: number }, b: { v: number }) => a.v === b.v;
      cache.set("k", successResult("retrieved", { v: 1 }), equal);
      const listener = vi.fn();
      cache.subscribe("k", listener);
      cache.set("k", successResult("retrieved", { v: 1 }), equal);
      expect(listener).not.toHaveBeenCalled();
    });

    it("notifies on success-to-success when data differs", () => {
      const cache = new QueryCache();
      const equal = (a: { v: number }, b: { v: number }) => a.v === b.v;
      cache.set("k", successResult("retrieved", { v: 1 }), equal);
      const listener = vi.fn();
      cache.subscribe("k", listener);
      cache.set("k", successResult("retrieved", { v: 2 }), equal);
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("ignores the equal callback for non-success transitions", () => {
      const cache = new QueryCache();
      const equal = vi.fn(() => true);
      cache.set("k", successResult("retrieved", 1));
      cache.set("k", errorResult("retrieve", new Error("x")), equal as any);
      expect(cache.get("k")?.variant).toEqual("error");
    });
  });

  describe("invalidate and clear", () => {
    it("removes the entry and notifies subscribers", () => {
      const cache = new QueryCache();
      const listener = vi.fn();
      cache.set("k", successResult<number>("retrieved", 1));
      cache.subscribe("k", listener);
      cache.invalidate("k");
      expect(cache.get("k")).toBeUndefined();
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("no-ops invalidate on an absent hash", () => {
      const cache = new QueryCache();
      const listener = vi.fn();
      cache.subscribe("k", listener);
      cache.invalidate("k");
      expect(listener).not.toHaveBeenCalled();
    });

    it("drops all entries on clear", () => {
      const cache = new QueryCache();
      cache.set("a", successResult<number>("retrieved", 1));
      cache.set("b", successResult<number>("retrieved", 2));
      cache.clear();
      expect(cache.get("a")).toBeUndefined();
      expect(cache.get("b")).toBeUndefined();
    });
  });
});
