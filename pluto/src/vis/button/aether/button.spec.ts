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

import { noopBooleanSinkSpec } from "@/telem/aether/noop";
import { telemTest } from "@/telem/aether/test";
import { TelemTest } from "@/telem/test";
import { renderAether } from "@/testutil/renderAether";
import { Button } from "@/vis/button";
import { button } from "@/vis/button/aether";

// Mounts the aether Button under the real provider stack with a registered test sink.
// Handlers are called directly on the typed `h.component`, and the writes they make are
// asserted against the test sink.
const setup = (mode?: button.Mode) => {
  const sink = telemTest.sink<boolean>();
  const h = renderAether(button.Button, {
    state: button.buttonStateZ.parse({
      sink: telemTest.booleanSinkSpec(sink),
      ...(mode != null ? { mode } : {}),
    }),
  });
  return { h, sink };
};

describe("button/aether/Button", () => {
  describe("schema", () => {
    it("should default to fire mode", () => {
      expect(button.buttonStateZ.parse({}).mode).toBe("fire");
    });

    it("should accept a mode override", () => {
      expect(button.buttonStateZ.parse({ mode: "pulse" }).mode).toBe("pulse");
    });
  });

  describe("fire mode", () => {
    it("should write true on mouse up", () => {
      const { h, sink } = setup("fire");
      h.component.onMouseUp();
      expect(sink.values).toEqual([true]);
    });

    it("should do nothing on mouse down", () => {
      const { h, sink } = setup("fire");
      h.component.onMouseDown();
      expect(sink.values).toEqual([]);
    });

    it("should write a single true across a full press", () => {
      const { h, sink } = setup("fire");
      h.component.onMouseDown();
      h.component.onMouseUp();
      expect(sink.values).toEqual([true]);
    });
  });

  describe("momentary mode", () => {
    it("should write true on mouse down", () => {
      const { h, sink } = setup("momentary");
      h.component.onMouseDown();
      expect(sink.values).toEqual([true]);
    });

    it("should write false on mouse up", () => {
      const { h, sink } = setup("momentary");
      h.component.onMouseUp();
      expect(sink.values).toEqual([false]);
    });

    it("should write true then false across a full press", () => {
      const { h, sink } = setup("momentary");
      h.component.onMouseDown();
      h.component.onMouseUp();
      expect(sink.values).toEqual([true, false]);
    });
  });

  describe("pulse mode", () => {
    it("should write true then false on mouse down", () => {
      const { h, sink } = setup("pulse");
      h.component.onMouseDown();
      expect(sink.values).toEqual([true, false]);
    });

    it("should do nothing on mouse up", () => {
      const { h, sink } = setup("pulse");
      h.component.onMouseUp();
      expect(sink.values).toEqual([]);
    });
  });

  describe("mode changes", () => {
    it("should apply a new mode after a state update", () => {
      const { h, sink } = setup("fire");
      h.component.onMouseDown();
      expect(sink.values).toEqual([]);
      h.setState((p) => ({ ...p, mode: "momentary" }));
      h.component.onMouseDown();
      expect(sink.values).toEqual([true]);
    });
  });

  describe("afterDelete", () => {
    it("should clean up the sink", () => {
      const { h, sink } = setup("fire");
      const cleanupSpy = vi.spyOn(sink, "cleanup");
      h.unmount();
      expect(cleanupSpy).toHaveBeenCalled();
    });
  });

  // The aether Button carries the behavior above; Button.use is a thin React adapter
  // over its RPC methods. These tests cover only that adapter wiring.
  describe("Button.use (React adapter)", () => {
    const TestWrapper = TelemTest.createTestWrapper({ registry: button.REGISTRY });

    it("should expose handlers with onClick aliasing mouse up", () => {
      const { result } = renderHook(
        () => Button.use({ aetherKey: "btn", sink: noopBooleanSinkSpec, mode: "fire" }),
        { wrapper: TestWrapper },
      );
      expect(result.current.onMouseDown).toBeDefined();
      expect(result.current.onMouseUp).toBeDefined();
      expect(result.current.onClick).toBe(result.current.onMouseUp);
    });
  });
});
