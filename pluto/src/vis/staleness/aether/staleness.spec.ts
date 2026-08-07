// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { aether } from "@/aether/aether";
import { aetherTest } from "@/aether/test";
import { buildStack } from "@/testutil/providers";
import { Theming } from "@/theming";
import { staleness } from "@/vis/staleness/aether";

// A minimal source-backed component. Real leaves drive `received` from a telem
// subscription; this one exposes it so specs can emit samples directly.
class Leaf extends aether.Leaf<
  typeof staleness.stateZ,
  { registration: staleness.Registration }
> {
  static readonly TYPE = "stalenessTestLeaf";
  static readonly z = staleness.stateZ;
  schema = Leaf.z;

  readonly transitions: boolean[] = [];

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.registration = staleness.useRegistration(ctx, i.registration, {
      timeout: () => this.state.stalenessTimeout,
      onChange: (stale) => {
        this.transitions.push(stale);
        this.setState((p) => ({ ...p, stale }));
      },
    });
  }

  received(): void {
    this.internal.registration.received();
  }

  afterDelete(): void {
    this.internal.registration?.cleanup();
  }
}

// Registers but never releases, so specs can reach the Provider's cleanup backstop.
class LeakyLeaf extends aether.Leaf<
  typeof staleness.stateZ,
  { registration: staleness.Registration }
> {
  static readonly TYPE = "stalenessLeakyTestLeaf";
  static readonly z = staleness.stateZ;
  schema = LeakyLeaf.z;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.registration = staleness.useRegistration(ctx, i.registration, {
      timeout: () => this.state.stalenessTimeout,
      onChange: () => {},
    });
  }
}

interface SetupOptions {
  sweepInterval?: CrudeTimeSpan;
  timeouts?: number[];
}

// Mounts `timeouts.length` leaves under a single staleness Provider, so specs can
// assert on the sweep the whole tree shares.
const setup = ({ sweepInterval, timeouts = [5] }: SetupOptions = {}) => {
  const stack = buildStack({
    registry: { [Leaf.TYPE]: Leaf },
    staleness: sweepInterval != null ? { sweepInterval } : {},
  });
  const leaves = timeouts.map((stalenessTimeout, i) => {
    const path = [...stack.basePath, `leaf${i}`];
    stack.driver.update(path, Leaf.TYPE, staleness.stateZ.parse({ stalenessTimeout }));
    return stack.driver.find<Leaf>(path);
  });
  return {
    stack,
    leaves,
    leaf: leaves[0],
    deleteLeaf: (i: number) => stack.driver.delete([...stack.basePath, `leaf${i}`]),
    // basePath is the deepest mounted provider, which is staleness while render is off.
    setSweepInterval: (next: CrudeTimeSpan) =>
      stack.driver.update(stack.basePath, staleness.Provider.TYPE, {
        sweepInterval: next,
      }),
  };
};

describe("staleness", () => {
  let teardown: (() => void) | null = null;

  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    teardown?.();
    teardown = null;
    vi.useRealTimers();
  });

  const mount = (options?: SetupOptions) => {
    const res = setup(options);
    teardown = () => res.stack.driver.delete([aetherTest.ROOT_KEY]);
    return res;
  };

  describe("transitions", () => {
    it("should turn stale when no sample arrives within the timeout", () => {
      const { leaf } = mount({ timeouts: [5] });
      vi.advanceTimersByTime(5250);
      expect(leaf.transitions).toEqual([true]);
      expect(leaf.state.stale).toBe(true);
    });

    it("should stay live while samples keep arriving", () => {
      const { leaf } = mount({ timeouts: [5] });
      for (let i = 0; i < 10; i++) {
        vi.advanceTimersByTime(1000);
        leaf.received();
      }
      expect(leaf.transitions).toEqual([]);
    });

    it("should turn stale one window after the last sample", () => {
      const { leaf } = mount({ timeouts: [5] });
      vi.advanceTimersByTime(3000);
      leaf.received();
      vi.advanceTimersByTime(4750);
      expect(leaf.transitions).toEqual([]);
      vi.advanceTimersByTime(500);
      expect(leaf.transitions).toEqual([true]);
    });

    it("should clear staleness when a sample arrives again", () => {
      const { leaf } = mount({ timeouts: [5] });
      vi.advanceTimersByTime(5250);
      leaf.received();
      expect(leaf.transitions).toEqual([true, false]);
      expect(leaf.state.stale).toBe(false);
    });

    it("should report each transition once", () => {
      const { leaf } = mount({ timeouts: [5] });
      vi.advanceTimersByTime(10500);
      leaf.received();
      leaf.received();
      expect(leaf.transitions).toEqual([true, false]);
    });

    it("should turn a source live again when its timeout grows past the gap", () => {
      const { leaf } = mount({ timeouts: [5] });
      vi.advanceTimersByTime(5250);
      expect(leaf.transitions).toEqual([true]);
      leaf.setState((p) => ({ ...p, stalenessTimeout: 30 }));
      vi.advanceTimersByTime(250);
      expect(leaf.transitions).toEqual([true, false]);
    });

    it("should track each source independently", () => {
      const { leaves } = mount({ timeouts: [1, 10] });
      vi.advanceTimersByTime(1250);
      expect(leaves[0].transitions).toEqual([true]);
      expect(leaves[1].transitions).toEqual([]);
    });
  });

  describe("sweep", () => {
    it("should run one sweep for every source in the tree", () => {
      mount({ timeouts: [5, 5, 5, 5] });
      expect(vi.getTimerCount()).toEqual(1);
    });

    it("should not allocate timers as samples arrive", () => {
      const { leaf } = mount({ timeouts: [5] });
      const spy = vi.spyOn(globalThis, "setInterval");
      for (let i = 0; i < 100; i++) leaf.received();
      expect(spy).not.toHaveBeenCalled();
      expect(vi.getTimerCount()).toEqual(1);
    });

    it("should stop sweeping once the last source is deleted", () => {
      const { deleteLeaf } = mount({ timeouts: [5, 5] });
      deleteLeaf(0);
      expect(vi.getTimerCount()).toEqual(1);
      deleteLeaf(1);
      expect(vi.getTimerCount()).toEqual(0);
    });

    it("should stop sweeping when a deleted provider still holds a source", () => {
      const stack = buildStack({ registry: { [LeakyLeaf.TYPE]: LeakyLeaf } });
      teardown = () => stack.driver.delete([aetherTest.ROOT_KEY]);
      stack.driver.update(
        [...stack.basePath, "leaf"],
        LeakyLeaf.TYPE,
        staleness.stateZ.parse({}),
      );
      expect(vi.getTimerCount()).toEqual(1);
      stack.driver.delete(stack.basePath);
      expect(vi.getTimerCount()).toEqual(0);
    });

    it("should stop reporting transitions for a deleted source", () => {
      const { leaf, deleteLeaf } = mount({ timeouts: [5] });
      deleteLeaf(0);
      vi.advanceTimersByTime(10000);
      expect(leaf.transitions).toEqual([]);
    });
  });

  describe("sweep interval", () => {
    it("should default to a quarter second", () => {
      const { leaf } = mount({ timeouts: [1] });
      vi.advanceTimersByTime(1250);
      expect(leaf.transitions).toEqual([true]);
    });

    it("should read a bare number as milliseconds", () => {
      const { leaf } = mount({ sweepInterval: 1000, timeouts: [1] });
      // A quarter second sweep would have reported by now.
      vi.advanceTimersByTime(999);
      expect(leaf.transitions).toEqual([]);
      vi.advanceTimersByTime(1);
      expect(leaf.transitions).toEqual([true]);
    });

    it("should accept a TimeSpan", () => {
      const { leaf } = mount({ sweepInterval: TimeSpan.seconds(1), timeouts: [1] });
      vi.advanceTimersByTime(999);
      expect(leaf.transitions).toEqual([]);
      vi.advanceTimersByTime(1);
      expect(leaf.transitions).toEqual([true]);
    });

    it("should bound how late a transition is reported", () => {
      const { leaf } = mount({
        sweepInterval: TimeSpan.milliseconds(50),
        timeouts: [1],
      });
      vi.advanceTimersByTime(1050);
      expect(leaf.transitions).toEqual([true]);
    });

    it("should adopt a new interval without adding a timer", () => {
      const { leaf, setSweepInterval } = mount({
        sweepInterval: TimeSpan.seconds(10),
        timeouts: [1],
      });
      setSweepInterval(TimeSpan.milliseconds(100));
      vi.advanceTimersByTime(1100);
      expect(leaf.transitions).toEqual([true]);
      expect(vi.getTimerCount()).toEqual(1);
    });

    it("should keep sweeping on the old interval until it is changed", () => {
      const { leaf, setSweepInterval } = mount({
        sweepInterval: TimeSpan.seconds(10),
        timeouts: [1],
      });
      vi.advanceTimersByTime(5000);
      expect(leaf.transitions).toEqual([]);
      setSweepInterval(TimeSpan.milliseconds(100));
      vi.advanceTimersByTime(100);
      expect(leaf.transitions).toEqual([true]);
    });
  });

  describe("resolveColor", () => {
    const theme = Theming.themeZ.parse(Theming.SYNNAX_THEMES.synnaxDark);

    it("should resolve an unset color to the theme warning shade", () => {
      expect(staleness.resolveColor(color.ZERO, theme)).toEqual(
        theme.colors.warning.m1,
      );
    });

    it("should resolve a null color to the theme warning shade", () => {
      expect(staleness.resolveColor(undefined, theme)).toEqual(theme.colors.warning.m1);
    });

    it("should honor a configured color", () => {
      expect(staleness.resolveColor("#FF0000", theme)).toEqual(
        color.construct("#FF0000"),
      );
    });
  });

  describe("Provider", () => {
    it("should default the sweep interval", () => {
      const parsed = staleness.Provider.z.parse({});
      expect(parsed.sweepInterval.equals(staleness.DEFAULT_SWEEP_INTERVAL)).toBe(true);
    });

    // TimeSpan.z alone would read this as nanoseconds, leaving a sweep so short it pegs
    // the worker.
    it("should read a bare number as milliseconds", () => {
      const parsed = staleness.Provider.z.parse({ sweepInterval: 500 });
      expect(parsed.sweepInterval.equals(TimeSpan.milliseconds(500))).toBe(true);
    });

    it("should keep an explicit TimeSpan unit", () => {
      const parsed = staleness.Provider.z.parse({ sweepInterval: TimeSpan.seconds(2) });
      expect(parsed.sweepInterval.equals(TimeSpan.seconds(2))).toBe(true);
    });

    it("should recover a TimeSpan that crossed the worker boundary", () => {
      // Structured clone strips the prototype, leaving the bare value object.
      const cloned = structuredClone(TimeSpan.seconds(1));
      const parsed = staleness.Provider.z.parse({ sweepInterval: cloned });
      expect(parsed.sweepInterval).toBeInstanceOf(TimeSpan);
      expect(parsed.sweepInterval.equals(TimeSpan.seconds(1))).toBe(true);
    });
  });
});
