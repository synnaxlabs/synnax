// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { Jumps } from "@/schematic/edge/common/jumps";
import { type Path } from "@/schematic/edge/common/path";

const map = (entries: Record<string, Path.Crossing[]>): Map<string, Path.Crossing[]> =>
  new Map(Object.entries(entries));

describe("jumps store", () => {
  it("returns a stable empty reference for an unknown key", () => {
    const store = Jumps.createStore();
    expect(store.get("missing")).toBe(store.get("missing"));
    expect(store.get("missing")).toHaveLength(0);
  });

  it("exposes committed hops and keeps the reference stable between reads", () => {
    const store = Jumps.createStore();
    store.commit(map({ a: [{ x: 1, y: 2, radius: 5 }] }));
    expect(store.get("a")).toEqual([{ x: 1, y: 2, radius: 5 }]);
    expect(store.get("a")).toBe(store.get("a"));
  });

  it("preserves the array reference across a commit that did not change the key", () => {
    const store = Jumps.createStore();
    store.commit(map({ a: [{ x: 1, y: 2, radius: 5 }] }));
    const before = store.get("a");
    store.commit(
      map({ a: [{ x: 1, y: 2, radius: 5 }], b: [{ x: 3, y: 4, radius: 5 }] }),
    );
    expect(store.get("a")).toBe(before);
  });

  it("swaps the reference and notifies only when a key's hops change", () => {
    const store = Jumps.createStore();
    store.commit(map({ a: [{ x: 1, y: 2, radius: 5 }] }));
    const before = store.get("a");
    const onA = vi.fn();
    store.subscribe("a", onA);
    store.commit(map({ a: [{ x: 9, y: 9, radius: 5 }] }));
    expect(store.get("a")).not.toBe(before);
    expect(onA).toHaveBeenCalledTimes(1);
  });

  it("notifies a key when only a hop's radius changes", () => {
    const store = Jumps.createStore();
    store.commit(map({ a: [{ x: 1, y: 2, radius: 5 }] }));
    const onA = vi.fn();
    store.subscribe("a", onA);
    store.commit(map({ a: [{ x: 1, y: 2, radius: 8 }] }));
    expect(onA).toHaveBeenCalledTimes(1);
  });

  it("does not notify a key whose hops are unchanged", () => {
    const store = Jumps.createStore();
    store.commit(
      map({ a: [{ x: 1, y: 2, radius: 5 }], b: [{ x: 3, y: 4, radius: 5 }] }),
    );
    const onA = vi.fn();
    store.subscribe("a", onA);
    store.commit(
      map({ a: [{ x: 1, y: 2, radius: 5 }], b: [{ x: 7, y: 7, radius: 5 }] }),
    );
    expect(onA).not.toHaveBeenCalled();
  });

  it("notifies and clears a key dropped from the next commit", () => {
    const store = Jumps.createStore();
    store.commit(map({ a: [{ x: 1, y: 2, radius: 5 }] }));
    const onA = vi.fn();
    store.subscribe("a", onA);
    store.commit(map({}));
    expect(onA).toHaveBeenCalledTimes(1);
    expect(store.get("a")).toHaveLength(0);
  });

  it("stops notifying after unsubscribe", () => {
    const store = Jumps.createStore();
    const onA = vi.fn();
    const unsubscribe = store.subscribe("a", onA);
    unsubscribe();
    store.commit(map({ a: [{ x: 1, y: 2, radius: 5 }] }));
    expect(onA).not.toHaveBeenCalled();
  });
});
