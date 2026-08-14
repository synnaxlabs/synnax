// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { telemTest } from "@/telem/aether/test";
import { renderAether } from "@/testutil/renderAether";
import { stateIndicator } from "@/vis/stateIndicator/aether";

const OPTIONS = [
  { key: "closed", value: 0 },
  { key: "open", value: 1 },
];

interface SetupOptions {
  sourceValue?: number;
  stalenessTimeout?: number;
}

const setup = ({ sourceValue = 0, stalenessTimeout }: SetupOptions = {}) => {
  const source = telemTest.source<number>(sourceValue);
  const h = renderAether(stateIndicator.StateIndicator, {
    state: stateIndicator.stateZ.parse({
      options: OPTIONS,
      source: telemTest.numberSourceSpec(source),
      ...(stalenessTimeout != null ? { stalenessTimeout } : {}),
    }),
  });
  return { h, source };
};

describe("stateIndicator/aether/StateIndicator", () => {
  describe("matched option", () => {
    it("should match the option the source value maps to", () => {
      const { h } = setup({ sourceValue: 1 });
      expect(h.state.key).toBe("open");
    });

    it("should report a null key for an unmapped value", () => {
      const { h, source } = setup({ sourceValue: 1 });
      source.setValue(7);
      expect(h.state.key).toBeNull();
    });
  });

  describe("staleness", () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    it("should start out live", () => {
      const { h } = setup({ stalenessTimeout: 1 });
      expect(h.state.stale).toBe(false);
    });

    it("should stay live before the source has ever sent", () => {
      const { h } = setup({ stalenessTimeout: 1 });
      vi.advanceTimersByTime(10000);
      expect(h.state.stale).toBe(false);
    });

    it("should turn stale when the source stops sending", () => {
      const { h, source } = setup({ stalenessTimeout: 1 });
      source.setValue(1);
      vi.advanceTimersByTime(1250);
      expect(h.state.stale).toBe(true);
    });

    it("should stay live while the source keeps sending", () => {
      const { h, source } = setup({ stalenessTimeout: 5 });
      for (let i = 0; i < 10; i++) {
        vi.advanceTimersByTime(1000);
        source.setValue(i % 2);
      }
      expect(h.state.stale).toBe(false);
    });

    // A channel parked on one state still delivers samples, so it must not read stale.
    it("should stay live while the source repeats the same value", () => {
      const { h, source } = setup({ sourceValue: 1, stalenessTimeout: 5 });
      for (let i = 0; i < 10; i++) {
        vi.advanceTimersByTime(1000);
        source.setValue(1);
      }
      expect(h.state.stale).toBe(false);
    });

    it("should turn live again when the source sends after going stale", () => {
      const { h, source } = setup({ stalenessTimeout: 1 });
      source.setValue(1);
      vi.advanceTimersByTime(1250);
      expect(h.state.stale).toBe(true);
      source.setValue(0);
      expect(h.state.stale).toBe(false);
    });

    // The fill encodes the state, so a stale indicator must keep the matched option.
    it("should keep the matched option while stale", () => {
      const { h, source } = setup({ stalenessTimeout: 1 });
      source.setValue(1);
      vi.advanceTimersByTime(1250);
      expect(h.state.stale).toBe(true);
      expect(h.state.key).toBe("open");
    });

    it("should release its registration on delete", () => {
      const { h } = setup({ stalenessTimeout: 1 });
      h.unmount();
      expect(vi.getTimerCount()).toEqual(0);
    });
  });
});
