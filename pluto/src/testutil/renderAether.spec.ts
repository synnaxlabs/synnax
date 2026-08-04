// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { aetherTest } from "@/aether/test";
import { alamos } from "@/alamos/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { renderAether } from "@/testutil/renderAether";
import { theming } from "@/theming/aether";
import { SYNNAX_DARK } from "@/theming/base/theme";

const { Leaf, Composite } = aetherTest;

describe("renderAether", () => {
  describe("Leaf component under test", () => {
    it("mounts the component and parses its state", () => {
      const h = renderAether(Leaf, { state: { value: 1 } });
      expect(h.state).toEqual({ value: 1 });
    });

    it("types component as its concrete class without a cast", () => {
      const h = renderAether(Leaf, { state: { value: 1 } });
      expect(h.component.updateCalls).toHaveLength(1);
      expect(h.component.updateCalls[0].state).toEqual({ value: 1 });
    });

    it("applies setState and re-runs afterUpdate", () => {
      const h = renderAether(Leaf, { state: { value: 1 } });
      h.setState({ value: 2 });
      expect(h.state).toEqual({ value: 2 });
      expect(h.component.updateCalls).toHaveLength(2);
      expect(h.component.updateCalls[1].prevState).toEqual({ value: 1 });
    });

    it("accepts a setter function in setState", () => {
      const h = renderAether(Leaf, { state: { count: 1 } });
      h.setState((prev) => ({ count: (prev.count as number) + 1 }));
      expect(h.state).toEqual({ count: 2 });
    });

    it("records afterDelete on unmount", () => {
      const h = renderAether(Leaf, { state: { value: 1 } });
      const leaf = h.component;
      h.unmount();
      expect(leaf.deleteCallCount).toBe(1);
    });
  });

  describe("Composite component under test", () => {
    it("mounts and exposes child operations", () => {
      const h = renderAether(Composite, { state: {} });
      h.setChildState("c1", Leaf.TYPE, { value: 42 });
      const child = h.child<aetherTest.Leaf>("c1");
      expect(child.state).toEqual({ value: 42 });
      expect(child.updateCalls).toHaveLength(1);
    });

    it("allows replacing a child's state via setChildState", () => {
      const h = renderAether(Composite, { state: {} });
      h.setChildState("c1", Leaf.TYPE, { value: 1 });
      h.setChildState("c1", Leaf.TYPE, { value: 2 });
      expect(h.child<aetherTest.Leaf>("c1").state).toEqual({ value: 2 });
    });

    it("deletes a child via deleteChild", () => {
      const h = renderAether(Composite, { state: {} });
      h.setChildState("c1", Leaf.TYPE, { value: 1 });
      const child = h.child<aetherTest.Leaf>("c1");
      h.deleteChild("c1");
      expect(child.deleteCallCount).toBe(1);
      expect(() => h.child("c1")).toThrow();
    });

    it("seeds children passed in options.children", () => {
      const h = renderAether(Composite, {
        state: {},
        children: { c1: { type: Leaf.TYPE, state: { value: 1 } } },
      });
      expect(h.child<aetherTest.Leaf>("c1").state).toEqual({ value: 1 });
    });
  });

  describe("provider stack", () => {
    it("exposes each provider instance for direct inspection", () => {
      const h = renderAether(Leaf, { state: {} });
      expect(h.providers.alamos).toBeInstanceOf(alamos.Provider);
      expect(h.providers.status).toBeInstanceOf(status.Aggregator);
      expect(h.providers.synnax).toBeInstanceOf(synnax.Provider);
      expect(h.providers.theming).toBeInstanceOf(theming.Provider);
    });

    it("applies a theming override to the theming provider state", () => {
      const h = renderAether(Leaf, {
        state: {},
        theming: { theme: SYNNAX_DARK, fontURLs: [] },
      });
      expect(h.providers.theming?.state.theme.key).toBe("synnaxDark");
    });

    it("applies a synnax override (props) to the synnax provider state", () => {
      const h = renderAether(Leaf, {
        state: {},
        synnax: { props: null },
      });
      expect(h.providers.synnax?.state.props).toBeNull();
    });

    it("drops a provider from the stack when toggled off", () => {
      const h = renderAether(Leaf, { state: {}, telem: false });
      expect(h.providers.telem).toBeNull();
    });
  });

  describe("registry composition", () => {
    it("accepts additional types via the registry option", () => {
      class CustomLeaf extends Leaf {
        static readonly TYPE = "Custom";
      }
      const h = renderAether(Composite, {
        state: {},
        registry: { [CustomLeaf.TYPE]: CustomLeaf },
      });
      h.setChildState("c1", CustomLeaf.TYPE, { v: 1 });
      expect(h.child("c1")).toBeInstanceOf(CustomLeaf);
    });
  });

  describe("automatic teardown", () => {
    let mounted: aetherTest.Leaf;

    it("leaves the mount alive during the test", () => {
      const h = renderAether(Leaf, { state: { value: 1 } });
      mounted = h.component;
      expect(mounted.deleteCallCount).toBe(0);
    });

    it("tears the previous mount down after the test, without an explicit unmount", () => {
      expect(mounted.deleteCallCount).toBe(1);
    });

    it("is idempotent with an explicit unmount", () => {
      const h = renderAether(Leaf, { state: { value: 1 } });
      h.unmount();
      h.unmount();
      expect(h.component.deleteCallCount).toBe(1);
    });
  });
});
