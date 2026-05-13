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
import { successResult } from "@/flux/result";

const noop = vi.fn();

const makeClient = () =>
  new base.Client({
    client: null,
    storeConfig: {},
    handleError: noop,
    handleAsyncError: noop,
  });

describe("Client.getCache", () => {
  it("lazily constructs a cache on first call for a key", () => {
    const client = makeClient();
    const cache = client.getCache<{ k: string }, number>("a");
    expect(cache).toBeDefined();
    expect(cache.get({ k: "x" })).toBeUndefined();
  });

  it("returns the same instance on subsequent calls with the same key", () => {
    const client = makeClient();
    const first = client.getCache<{ k: string }, number>("a");
    const second = client.getCache<{ k: string }, number>("a");
    expect(first).toBe(second);
  });

  it("returns distinct instances for distinct keys", () => {
    const client = makeClient();
    const a = client.getCache<{ k: string }, number>("a");
    const b = client.getCache<{ k: string }, number>("b");
    expect(a).not.toBe(b);
  });

  it("isolates entries across keys on the same client", () => {
    const client = makeClient();
    const a = client.getCache<{ k: string }, number>("a");
    const b = client.getCache<{ k: string }, number>("b");
    a.set({ k: "x" }, successResult<number>("retrieved", 1));
    expect(b.get({ k: "x" })).toBeUndefined();
  });

  it("isolates caches across separate Client instances", () => {
    const c1 = makeClient();
    const c2 = makeClient();
    const a1 = c1.getCache<{ k: string }, number>("a");
    const a2 = c2.getCache<{ k: string }, number>("a");
    a1.set({ k: "x" }, successResult<number>("retrieved", 1));
    expect(a2.get({ k: "x" })).toBeUndefined();
  });

  it("preserves entries across getCache calls for the same key", () => {
    const client = makeClient();
    const first = client.getCache<{ k: string }, number>("a");
    const result = successResult<number>("retrieved", 7);
    first.set({ k: "x" }, result);
    const second = client.getCache<{ k: string }, number>("a");
    expect(second.get({ k: "x" })).toBe(result);
  });
});
