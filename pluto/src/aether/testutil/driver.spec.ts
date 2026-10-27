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
  [aetherTest.Leaf.TYPE]: aetherTest.Leaf,
  [aetherTest.Composite.TYPE]: aetherTest.Composite,
};

describe("createDriver", () => {
  it("exposes a root and a mock comms pair", () => {
    const d = aetherTest.createDriver(REGISTRY);
    expect(d.root).toBeDefined();
    expect(d.workerSide).toBeDefined();
    expect(d.mainSide).toBeDefined();
  });

  describe("update / find", () => {
    it("mounts a component from the registry and looks it up by path", () => {
      const d = aetherTest.createDriver(REGISTRY);
      d.update([aetherTest.ROOT_KEY, "a"], aetherTest.Leaf.TYPE, { value: 1 });
      const leaf = d.find<aetherTest.Leaf>([aetherTest.ROOT_KEY, "a"]);
      expect(leaf).toBeInstanceOf(aetherTest.Leaf);
      expect(leaf.key).toBe("a");
      expect(leaf.state).toEqual({ value: 1 });
    });

    it("mounts nested children under a composite", () => {
      const d = aetherTest.createDriver(REGISTRY);
      d.update([aetherTest.ROOT_KEY, "c"], aetherTest.Composite.TYPE, {});
      d.update([aetherTest.ROOT_KEY, "c", "leaf"], aetherTest.Leaf.TYPE, { value: 2 });
      expect(d.find<aetherTest.Leaf>([aetherTest.ROOT_KEY, "c", "leaf"]).state).toEqual(
        { value: 2 },
      );
    });

    it("re-runs afterUpdate when updating an existing component", () => {
      const d = aetherTest.createDriver(REGISTRY);
      d.update([aetherTest.ROOT_KEY, "a"], aetherTest.Leaf.TYPE, { value: 1 });
      d.update([aetherTest.ROOT_KEY, "a"], aetherTest.Leaf.TYPE, { value: 2 });
      const leaf = d.find<aetherTest.Leaf>([aetherTest.ROOT_KEY, "a"]);
      expect(leaf.state).toEqual({ value: 2 });
      expect(leaf.updateCalls).toHaveLength(2);
    });

    it("derives each component's key from its own path, not a later mutation", () => {
      // Regression: update must snapshot the path so a caller mutating its array (e.g.
      // pushing the next child key) cannot rewrite an already-mounted component's key.
      const d = aetherTest.createDriver(REGISTRY);
      const path = [aetherTest.ROOT_KEY, "c"];
      d.update(path, aetherTest.Composite.TYPE, {});
      path.push("leaf");
      d.update(path, aetherTest.Leaf.TYPE, { value: 1 });
      expect(d.find([aetherTest.ROOT_KEY, "c"]).key).toBe("c");
      expect(d.find([aetherTest.ROOT_KEY, "c", "leaf"]).key).toBe("leaf");
    });

    it("throws when no component exists at the path", () => {
      const d = aetherTest.createDriver(REGISTRY);
      expect(() => d.find([aetherTest.ROOT_KEY, "missing"])).toThrow();
    });

    it("throws when the type is not in the registry", () => {
      const d = aetherTest.createDriver(REGISTRY);
      expect(() => d.update([aetherTest.ROOT_KEY, "x"], "unregistered", {})).toThrow();
    });
  });

  describe("delete", () => {
    it("removes a component and its descendants", () => {
      const d = aetherTest.createDriver(REGISTRY);
      d.update([aetherTest.ROOT_KEY, "c"], aetherTest.Composite.TYPE, {});
      d.update([aetherTest.ROOT_KEY, "c", "leaf"], aetherTest.Leaf.TYPE, { value: 1 });
      const leaf = d.find<aetherTest.Leaf>([aetherTest.ROOT_KEY, "c", "leaf"]);
      d.delete([aetherTest.ROOT_KEY, "c"]);
      expect(leaf.deleteCallCount).toBe(1);
      expect(() => d.find([aetherTest.ROOT_KEY, "c"])).toThrow();
    });
  });
});
