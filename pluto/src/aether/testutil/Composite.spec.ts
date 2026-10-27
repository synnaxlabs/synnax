// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type aether } from "@/aether/aether";
import { aetherTest } from "@/aether/testutil";

const REGISTRY: aether.ComponentRegistry = {
  [aetherTest.Composite.TYPE]: aetherTest.Composite,
  [aetherTest.Leaf.TYPE]: aetherTest.Leaf,
};

describe("aetherTest.Composite", () => {
  it("accepts any record-shaped state", () => {
    expect(aetherTest.compositeStateZ.parse({ a: 1 })).toEqual({ a: 1 });
  });

  it("records lifecycle calls like Leaf", () => {
    const d = aetherTest.createDriver(REGISTRY);
    d.update([aetherTest.ROOT_KEY, "c"], aetherTest.Composite.TYPE, {});
    const c = d.find<aetherTest.Composite>([aetherTest.ROOT_KEY, "c"]);
    expect(c.updateCalls).toHaveLength(1);
    d.delete([aetherTest.ROOT_KEY, "c"]);
    expect(c.deleteCallCount).toBe(1);
  });

  it("hosts arbitrary descendants", () => {
    const d = aetherTest.createDriver(REGISTRY);
    d.update([aetherTest.ROOT_KEY, "c"], aetherTest.Composite.TYPE, {});
    d.update([aetherTest.ROOT_KEY, "c", "leaf"], aetherTest.Leaf.TYPE, { value: 7 });
    expect(d.find<aetherTest.Leaf>([aetherTest.ROOT_KEY, "c", "leaf"]).state).toEqual({
      value: 7,
    });
  });

  it("tears down descendants when the parent is deleted", () => {
    const d = aetherTest.createDriver(REGISTRY);
    d.update([aetherTest.ROOT_KEY, "c"], aetherTest.Composite.TYPE, {});
    d.update([aetherTest.ROOT_KEY, "c", "leaf"], aetherTest.Leaf.TYPE, { value: 1 });
    const leaf = d.find<aetherTest.Leaf>([aetherTest.ROOT_KEY, "c", "leaf"]);
    d.delete([aetherTest.ROOT_KEY, "c"]);
    expect(leaf.deleteCallCount).toBe(1);
  });
});
