// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { telemTest } from "@/telem/aether/testutil";
import { renderAether } from "@/testutil/internal";
import { symbolNode } from "@/vis/symbolNode/aether";

const TIMEOUT = TimeSpan.seconds(1).nanoseconds;

// Mounts the aether SymbolNode with an active and a level source and one sink.
const setup = () => {
  const active = telemTest.source<number>(NaN);
  const level = telemTest.source<number>(NaN);
  const sink = telemTest.sink<number>();
  const h = renderAether(symbolNode.SymbolNode, {
    state: {
      stalenessTimeout: TIMEOUT,
      sources: {
        active: telemTest.numberSourceSpec(active),
        level: telemTest.numberSourceSpec(level),
      },
      sinks: { active: telemTest.numberSinkSpec(sink) },
    },
  });
  return { h, active, level, sink };
};

describe("symbolNode/aether/SymbolNode", () => {
  describe("readings", () => {
    it("should read null for a source with no data", () => {
      const { h } = setup();
      expect(h.state.readings).toEqual({
        active: { value: null, stale: false },
        level: { value: null, stale: false },
      });
    });

    it("should read the value of each source by its key", () => {
      const { h, active, level } = setup();
      active.setValue(1);
      level.setValue(42);
      expect(h.state.readings).toEqual({
        active: { value: 1, stale: false },
        level: { value: 42, stale: false },
      });
    });

    it("should read NaN as null", () => {
      const { h, active } = setup();
      active.setValue(1);
      active.setValue(NaN);
      expect(h.state.readings.active.value).toBeNull();
    });

    it("should drop the reading of a source that the state removes", () => {
      const { h, active } = setup();
      h.setState((p) => ({
        ...p,
        sources: { active: telemTest.numberSourceSpec(active) },
      }));
      expect(Object.keys(h.state.readings)).toEqual(["active"]);
    });

    it("should read the new source of a key whose spec changes", () => {
      const { h, active } = setup();
      active.setValue(1);
      const next = telemTest.source<number>(5);
      h.setState((p) => ({
        ...p,
        sources: { ...p.sources, active: telemTest.numberSourceSpec(next) },
      }));
      expect(h.state.readings.active).toEqual({ value: 5, stale: false });
    });

    it("should read a source whose key names an object prototype member", () => {
      const source = telemTest.source<number>(3);
      const h = renderAether(symbolNode.SymbolNode, {
        state: { sources: { constructor: telemTest.numberSourceSpec(source) } },
      });
      expect(h.state.readings).toEqual({ constructor: { value: 3, stale: false } });
      const cleanup = vi.spyOn(source, "cleanup");
      h.setState((p) => ({ ...p, sources: {} }));
      expect(cleanup).toHaveBeenCalled();
      expect(h.state.readings).toEqual({});
    });

    it("should stop reading a source that the state removes", () => {
      const { h, active, level } = setup();
      const cleanup = vi.spyOn(level, "cleanup");
      h.setState((p) => ({
        ...p,
        sources: { active: telemTest.numberSourceSpec(active) },
      }));
      expect(cleanup).toHaveBeenCalled();
    });
  });

  describe("staleness", () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    it("should turn stale only the source that stops sending", () => {
      const { h, active, level } = setup();
      active.setValue(1);
      level.setValue(1);
      for (let i = 0; i < 5; i++) {
        vi.advanceTimersByTime(500);
        level.setValue(i);
      }
      expect(h.state.readings.active.stale).toBe(true);
      expect(h.state.readings.level.stale).toBe(false);
    });

    it("should read a new source of a stale key as live", () => {
      const { h, active } = setup();
      active.setValue(1);
      vi.advanceTimersByTime(1250);
      h.setState((p) => ({
        ...p,
        sources: {
          ...p.sources,
          active: telemTest.numberSourceSpec(telemTest.source<number>(2)),
        },
      }));
      expect(h.state.readings.active).toEqual({ value: 2, stale: false });
    });

    it("should keep the last value of a stale source", () => {
      const { h, active } = setup();
      active.setValue(1);
      vi.advanceTimersByTime(1250);
      expect(h.state.readings.active).toEqual({ value: 1, stale: true });
    });

    it("should release each registration on delete", () => {
      const { h } = setup();
      h.unmount();
      vi.advanceTimersByTime(1250);
      expect(vi.getTimerCount()).toEqual(0);
    });
  });

  describe("set", () => {
    it("should write the value to the sink of the key", () => {
      const { h, sink } = setup();
      h.component.set("active", 1);
      expect(sink.values).toEqual([1]);
    });

    it("should throw for a key with no sink", () => {
      const { h } = setup();
      expect(() => h.component.set("level", 1)).toThrow(
        "symbol node has no sink level",
      );
    });

    it("should throw for a sink that the state removes", () => {
      const { h, sink } = setup();
      const cleanup = vi.spyOn(sink, "cleanup");
      h.setState((p) => ({ ...p, sinks: {} }));
      expect(cleanup).toHaveBeenCalled();
      expect(() => h.component.set("active", 1)).toThrow(
        "symbol node has no sink active",
      );
    });

    it("should not write when a source changes", () => {
      const { active, sink } = setup();
      active.setValue(1);
      expect(sink.values).toEqual([]);
    });
  });

  describe("afterDelete", () => {
    it("should clean up the sources and the sinks", () => {
      const { h, active, level, sink } = setup();
      const cleanups = [active, level, sink].map((t) => vi.spyOn(t, "cleanup"));
      h.unmount();
      for (const cleanup of cleanups) expect(cleanup).toHaveBeenCalled();
    });
  });
});
