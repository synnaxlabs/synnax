// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { noopBooleanSinkSpec, noopBooleanSourceSpec } from "@/telem/aether/noop";
import { telemTest } from "@/telem/aether/test";
import { TelemTest } from "@/telem/test";
import { renderAether } from "@/testutil/renderAether";
import { Toggle } from "@/vis/toggle";
import { toggle } from "@/vis/toggle/aether";

// Mounts the aether Toggle under the real provider stack with registered test telem.
// `h.component.toggle()` is called directly; the source's `setValue` drives the
// component's `enabled` state through its real subscription.
const setup = ({
  enabled = false,
  sourceValue = false,
}: { enabled?: boolean; sourceValue?: boolean } = {}) => {
  const sink = telemTest.sink<boolean>();
  const source = telemTest.source<boolean>(sourceValue);
  const h = renderAether(toggle.Toggle, {
    state: toggle.toggleStateZ.parse({
      enabled,
      sink: telemTest.booleanSinkSpec(sink),
      source: telemTest.booleanSourceSpec(source),
    }),
  });
  return { h, sink, source };
};

describe("toggle/aether/Toggle", () => {
  describe("schema", () => {
    it("should require an enabled value", () => {
      expect(() => toggle.toggleStateZ.parse({})).toThrow();
    });

    it("should default sink and source to noop", () => {
      const parsed = toggle.toggleStateZ.parse({ enabled: true });
      expect(parsed.enabled).toBe(true);
      expect(parsed.sink).toBeDefined();
      expect(parsed.source).toBeDefined();
    });
  });

  describe("toggle", () => {
    it("should command true when currently disabled", () => {
      const { h, sink } = setup({ sourceValue: false });
      h.component.toggle();
      expect(sink.values).toEqual([true]);
    });

    it("should command false when currently enabled", () => {
      const { h, sink } = setup({ sourceValue: true });
      h.component.toggle();
      expect(sink.values).toEqual([false]);
    });

    it("should command against the latest source value", () => {
      const { h, sink, source } = setup({ sourceValue: false });
      h.component.toggle();
      source.setValue(true);
      h.component.toggle();
      expect(sink.values).toEqual([true, false]);
    });

    it("should alternate values across repeated toggles with feedback", () => {
      const { h, sink, source } = setup({ sourceValue: false });
      h.component.toggle();
      source.setValue(true);
      h.component.toggle();
      source.setValue(false);
      h.component.toggle();
      source.setValue(true);
      expect(sink.values).toEqual([true, false, true]);
    });
  });

  describe("enabled state", () => {
    it("should initialize enabled from the source on mount", () => {
      const { h } = setup({ enabled: false, sourceValue: true });
      expect(h.state.enabled).toBe(true);
    });

    it("should enable when the source emits true", () => {
      const { h, source } = setup({ sourceValue: false });
      source.setValue(true);
      expect(h.state.enabled).toBe(true);
    });

    it("should disable when the source emits false", () => {
      const { h, source } = setup({ sourceValue: true });
      source.setValue(false);
      expect(h.state.enabled).toBe(false);
    });

    it("should leave enabled unchanged when the source repeats its value", () => {
      const { h, source } = setup({ sourceValue: false });
      source.setValue(false);
      expect(h.state.enabled).toBe(false);
    });

    it("should not command the sink when the source changes", () => {
      const { sink, source } = setup({ sourceValue: false });
      source.setValue(true);
      expect(sink.values).toEqual([]);
    });
  });

  describe("afterDelete", () => {
    it("should clean up the source and the sink", () => {
      const { h, sink, source } = setup({ sourceValue: false });
      const sinkCleanup = vi.spyOn(sink, "cleanup");
      const sourceCleanup = vi.spyOn(source, "cleanup");
      h.unmount();
      expect(sinkCleanup).toHaveBeenCalled();
      expect(sourceCleanup).toHaveBeenCalled();
    });

    it("should stop reacting to source emissions after delete", () => {
      const { h, source } = setup({ sourceValue: false });
      h.unmount();
      expect(() => source.setValue(true)).not.toThrow();
    });
  });

  // The aether Toggle carries the behavior above; Toggle.use is a thin React adapter
  // over its RPC method. This test covers only that adapter wiring.
  describe("Toggle.use (React adapter)", () => {
    const TestWrapper = TelemTest.createTestWrapper({ registry: toggle.REGISTRY });

    it("should expose a toggle fn and the enabled state", () => {
      const { result } = renderHook(
        () =>
          Toggle.use({
            aetherKey: "tgl",
            source: noopBooleanSourceSpec,
            sink: noopBooleanSinkSpec,
          }),
        { wrapper: TestWrapper },
      );
      expect(result.current.toggle).toBeDefined();
      expect(result.current.enabled).toBe(false);
    });
  });
});
