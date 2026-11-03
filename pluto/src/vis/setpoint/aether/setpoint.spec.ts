// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { telemTest } from "@/telem/aether/testutil";
import { TelemTest } from "@/telem/testutil";
import { renderAether } from "@/testutil/internal";
import { Setpoint } from "@/vis/setpoint";
import { setpoint } from "@/vis/setpoint/aether";

// Mounts the aether Setpoint under the real provider stack with registered test telem.
// The source's `setValue` drives the component's `value` through its real subscription.
const setup = (sourceValue: number = NaN) => {
  const sink = telemTest.sink<number>();
  const source = telemTest.source<number>(sourceValue);
  const h = renderAether(setpoint.Setpoint, {
    state: setpoint.stateZ.parse({
      sink: telemTest.numberSinkSpec(sink),
      source: telemTest.numberSourceSpec(source),
    }),
  });
  return { h, sink, source };
};

describe("setpoint/aether/Setpoint", () => {
  describe("set", () => {
    it("should command the sink with the value", () => {
      const { h, sink } = setup();
      h.component.set(42);
      expect(sink.values).toEqual([42]);
    });
  });

  describe("value", () => {
    it("should be null before the source reports a sample", () => {
      const { h } = setup();
      expect(h.state.value).toBeNull();
    });

    it("should initialize from the source on mount", () => {
      const { h } = setup(12);
      expect(h.state.value).toBe(12);
    });

    it("should follow the source as it changes", () => {
      const { h, source } = setup(12);
      source.setValue(15);
      expect(h.state.value).toBe(15);
    });

    it("should keep a zero from the source", () => {
      const { h, source } = setup(12);
      source.setValue(0);
      expect(h.state.value).toBe(0);
    });

    it("should not command the sink when the source changes", () => {
      const { sink, source } = setup(12);
      source.setValue(15);
      expect(sink.values).toEqual([]);
    });
  });

  describe("afterDelete", () => {
    it("should clean up the source and the sink", () => {
      const { h, sink, source } = setup();
      const sinkCleanup = vi.spyOn(sink, "cleanup");
      const sourceCleanup = vi.spyOn(source, "cleanup");
      h.unmount();
      expect(sinkCleanup).toHaveBeenCalled();
      expect(sourceCleanup).toHaveBeenCalled();
    });
  });

  describe("Setpoint.use (React adapter)", () => {
    const TestWrapper = TelemTest.createTestWrapper({ registry: setpoint.REGISTRY });

    it("should expose the value the source holds", async () => {
      const sink = telemTest.numberSinkSpec(telemTest.sink<number>());
      const source = telemTest.numberSourceSpec(telemTest.source<number>(7));
      const { result } = renderHook(
        () => Setpoint.use({ aetherKey: "sp", sink, source }),
        { wrapper: TestWrapper },
      );
      await waitFor(() => expect(result.current.value).toBe(7));
    });
  });
});
