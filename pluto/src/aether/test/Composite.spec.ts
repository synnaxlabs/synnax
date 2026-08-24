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
import { Composite, compositeStateZ } from "@/aether/test/Composite";
import { createDriver, ROOT_KEY } from "@/aether/test/driver";
import { Leaf } from "@/aether/test/Leaf";

const REGISTRY: aether.ComponentRegistry = {
  [Composite.TYPE]: Composite,
  [Leaf.TYPE]: Leaf,
};

describe("aetherTest.Composite", () => {
  it("accepts any record-shaped state", () => {
    expect(compositeStateZ.parse({ a: 1 })).toEqual({ a: 1 });
  });

  it("records lifecycle calls like Leaf", () => {
    const d = createDriver(REGISTRY);
    d.update([ROOT_KEY, "c"], Composite.TYPE, {});
    const c = d.find<Composite>([ROOT_KEY, "c"]);
    expect(c.updateCalls).toHaveLength(1);
    d.delete([ROOT_KEY, "c"]);
    expect(c.deleteCallCount).toBe(1);
  });

  it("hosts arbitrary descendants", () => {
    const d = createDriver(REGISTRY);
    d.update([ROOT_KEY, "c"], Composite.TYPE, {});
    d.update([ROOT_KEY, "c", "leaf"], Leaf.TYPE, { value: 7 });
    expect(d.find<Leaf>([ROOT_KEY, "c", "leaf"]).state).toEqual({ value: 7 });
  });

  it("tears down descendants when the parent is deleted", () => {
    const d = createDriver(REGISTRY);
    d.update([ROOT_KEY, "c"], Composite.TYPE, {});
    d.update([ROOT_KEY, "c", "leaf"], Leaf.TYPE, { value: 1 });
    const leaf = d.find<Leaf>([ROOT_KEY, "c", "leaf"]);
    d.delete([ROOT_KEY, "c"]);
    expect(leaf.deleteCallCount).toBe(1);
  });
});
