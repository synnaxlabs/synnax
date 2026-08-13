// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type LocalCache, setSettled } from "@/flux/suspend";

const newLocal = (): LocalCache<number> => ({
  epoch: 0,
  inFlight: new Map(),
  settled: new Map(),
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
});
