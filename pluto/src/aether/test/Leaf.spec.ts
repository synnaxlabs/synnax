// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { createDriver, ROOT_KEY } from "@/aether/test/driver";
import { Leaf, leafStateZ } from "@/aether/test/Leaf";

const mount = () => {
  const driver = createDriver({ [Leaf.TYPE]: Leaf });
  return driver;
};

describe("aetherTest.Leaf", () => {
  it("accepts any record-shaped state", () => {
    expect(leafStateZ.parse({ a: 1, b: "x" })).toEqual({ a: 1, b: "x" });
  });

  it("records each afterUpdate with its state and prevState", () => {
    const d = mount();
    d.update([ROOT_KEY, "l"], Leaf.TYPE, { value: 1 });
    d.update([ROOT_KEY, "l"], Leaf.TYPE, { value: 2 });
    const leaf = d.find<Leaf>([ROOT_KEY, "l"]);
    expect(leaf.updateCalls).toHaveLength(2);
    expect(leaf.updateCalls[0].state).toEqual({ value: 1 });
    expect(leaf.updateCalls[1].prevState).toEqual({ value: 1 });
    expect(leaf.updateCalls[1].state).toEqual({ value: 2 });
  });

  it("counts afterDelete invocations", () => {
    const d = mount();
    d.update([ROOT_KEY, "l"], Leaf.TYPE, {});
    const leaf = d.find<Leaf>([ROOT_KEY, "l"]);
    expect(leaf.deleteCallCount).toBe(0);
    d.delete([ROOT_KEY, "l"]);
    expect(leaf.deleteCallCount).toBe(1);
  });
});
