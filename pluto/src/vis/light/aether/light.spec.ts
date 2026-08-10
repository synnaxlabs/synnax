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
import { light } from "@/vis/light/aether";

interface SetupOptions {
  enabled?: boolean;
  sourceValue?: boolean;
  stalenessTimeout?: number;
}

const setup = ({
  enabled = false,
  sourceValue = false,
  stalenessTimeout,
}: SetupOptions = {}) => {
  const source = telemTest.source<boolean>(sourceValue);
  const h = renderAether(light.Light, {
    state: light.stateZ.parse({
      enabled,
      source: telemTest.booleanSourceSpec(source),
      ...(stalenessTimeout != null ? { stalenessTimeout } : {}),
    }),
  });
  return { h, source };
};

describe("light/aether/Light", () => {
  describe("enabled", () => {
    it("should take its initial enabled state from the source", () => {
      const { h } = setup({ sourceValue: true });
      expect(h.state.enabled).toBe(true);
    });

    it("should follow the source", () => {
      const { h, source } = setup({ sourceValue: false });
      source.setValue(true);
      expect(h.state.enabled).toBe(true);
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
      source.setValue(true);
      vi.advanceTimersByTime(1250);
      expect(h.state.stale).toBe(true);
    });

    it("should stay live while the source keeps sending", () => {
      const { h, source } = setup({ stalenessTimeout: 5 });
      for (let i = 0; i < 10; i++) {
        vi.advanceTimersByTime(1000);
        source.setValue(i % 2 === 0);
      }
      expect(h.state.stale).toBe(false);
    });

    // A steady boolean channel still delivers samples, so it must not read as stale.
    it("should stay live while the source repeats the same value", () => {
      const { h, source } = setup({ sourceValue: true, stalenessTimeout: 5 });
      for (let i = 0; i < 10; i++) {
        vi.advanceTimersByTime(1000);
        source.setValue(true);
      }
      expect(h.state.stale).toBe(false);
    });

    it("should turn live again when the source sends after going stale", () => {
      const { h, source } = setup({ stalenessTimeout: 1 });
      source.setValue(true);
      vi.advanceTimersByTime(1250);
      expect(h.state.stale).toBe(true);
      source.setValue(false);
      expect(h.state.stale).toBe(false);
    });

    it("should keep reporting the last known enabled state while stale", () => {
      const { h, source } = setup({ sourceValue: false, stalenessTimeout: 1 });
      source.setValue(true);
      vi.advanceTimersByTime(1250);
      expect(h.state.stale).toBe(true);
      expect(h.state.enabled).toBe(true);
    });

    it("should release its registration on delete", () => {
      const { h } = setup({ stalenessTimeout: 1 });
      h.unmount();
      expect(vi.getTimerCount()).toEqual(0);
    });
  });
});
