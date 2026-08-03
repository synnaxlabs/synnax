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
import { Composite } from "@/aether/test/Composite";
import { createDriver, ROOT_KEY } from "@/aether/test/driver";
import { Leaf } from "@/aether/test/Leaf";

const REGISTRY: aether.ComponentRegistry = {
  [Leaf.TYPE]: Leaf,
  [Composite.TYPE]: Composite,
};

describe("createDriver", () => {
  it("exposes a root and a mock comms pair", () => {
    const d = createDriver(REGISTRY);
    expect(d.root).toBeDefined();
    expect(d.workerSide).toBeDefined();
    expect(d.mainSide).toBeDefined();
  });

  describe("update / find", () => {
    it("mounts a component from the registry and looks it up by path", () => {
      const d = createDriver(REGISTRY);
      d.update([ROOT_KEY, "a"], Leaf.TYPE, { value: 1 });
      const leaf = d.find<Leaf>([ROOT_KEY, "a"]);
      expect(leaf).toBeInstanceOf(Leaf);
      expect(leaf.key).toBe("a");
      expect(leaf.state).toEqual({ value: 1 });
    });

    it("mounts nested children under a composite", () => {
      const d = createDriver(REGISTRY);
      d.update([ROOT_KEY, "c"], Composite.TYPE, {});
      d.update([ROOT_KEY, "c", "leaf"], Leaf.TYPE, { value: 2 });
      expect(d.find<Leaf>([ROOT_KEY, "c", "leaf"]).state).toEqual({ value: 2 });
    });

    it("re-runs afterUpdate when updating an existing component", () => {
      const d = createDriver(REGISTRY);
      d.update([ROOT_KEY, "a"], Leaf.TYPE, { value: 1 });
      d.update([ROOT_KEY, "a"], Leaf.TYPE, { value: 2 });
      const leaf = d.find<Leaf>([ROOT_KEY, "a"]);
      expect(leaf.state).toEqual({ value: 2 });
      expect(leaf.updateCalls).toHaveLength(2);
    });

    it("derives each component's key from its own path, not a later mutation", () => {
      // Regression: update must snapshot the path so a caller mutating its array (e.g.
      // pushing the next child key) cannot rewrite an already-mounted component's key.
      const d = createDriver(REGISTRY);
      const path = [ROOT_KEY, "c"];
      d.update(path, Composite.TYPE, {});
      path.push("leaf");
      d.update(path, Leaf.TYPE, { value: 1 });
      expect(d.find([ROOT_KEY, "c"]).key).toBe("c");
      expect(d.find([ROOT_KEY, "c", "leaf"]).key).toBe("leaf");
    });

    it("throws when no component exists at the path", () => {
      const d = createDriver(REGISTRY);
      expect(() => d.find([ROOT_KEY, "missing"])).toThrow();
    });

    it("throws when the type is not in the registry", () => {
      const d = createDriver(REGISTRY);
      expect(() => d.update([ROOT_KEY, "x"], "unregistered", {})).toThrow();
    });
  });

  describe("delete", () => {
    it("removes a component and its descendants", () => {
      const d = createDriver(REGISTRY);
      d.update([ROOT_KEY, "c"], Composite.TYPE, {});
      d.update([ROOT_KEY, "c", "leaf"], Leaf.TYPE, { value: 1 });
      const leaf = d.find<Leaf>([ROOT_KEY, "c", "leaf"]);
      d.delete([ROOT_KEY, "c"]);
      expect(leaf.deleteCallCount).toBe(1);
      expect(() => d.find([ROOT_KEY, "c"])).toThrow();
    });
  });
});
