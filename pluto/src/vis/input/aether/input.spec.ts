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
import { Input } from "@/vis/input";
import { input } from "@/vis/input/aether";

// Mounts the aether Input under the real provider stack with registered test telem.
// The source's `setValue` drives the component's `value` through its real subscription.
const setup = (sourceValue: string = "") => {
  const sink = telemTest.sink<string>();
  const source = telemTest.source<string>(sourceValue);
  const h = renderAether(input.Input, {
    state: input.stateZ.parse({
      sink: telemTest.stringSinkSpec(sink),
      source: telemTest.stringSourceSpec(source),
    }),
  });
  return { h, sink, source };
};

describe("input/aether/Input", () => {
  describe("set", () => {
    it("should command the sink with the value", () => {
      const { h, sink } = setup();
      h.component.set("open");
      expect(sink.values).toEqual(["open"]);
    });
  });

  describe("value", () => {
    it("should be null before the source reports a sample", () => {
      const { h } = setup();
      expect(h.state.value).toBeNull();
    });

    it("should initialize from the source on mount", () => {
      const { h } = setup("closed");
      expect(h.state.value).toBe("closed");
    });

    it("should follow the source as it changes", () => {
      const { h, source } = setup("closed");
      source.setValue("open");
      expect(h.state.value).toBe("open");
    });

    it("should not command the sink when the source changes", () => {
      const { sink, source } = setup("closed");
      source.setValue("open");
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

  describe("Input.use (React adapter)", () => {
    const TestWrapper = TelemTest.createTestWrapper({ registry: input.REGISTRY });

    it("should expose the value the source holds", async () => {
      const sink = telemTest.stringSinkSpec(telemTest.sink<string>());
      const source = telemTest.stringSourceSpec(telemTest.source<string>("closed"));
      const { result } = renderHook(
        () => Input.use({ aetherKey: "in", sink, source }),
        { wrapper: TestWrapper },
      );
      await waitFor(() => expect(result.current.value).toBe("closed"));
    });
  });
});
