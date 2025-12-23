// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { noopBooleanSinkSpec } from "@/telem/aether/noop";
import { telemTest } from "@/telem/aether/test";
import { TelemTest } from "@/telem/test";
import { Button } from "@/vis/button";
import { button } from "@/vis/button/aether";

const TestWrapper = TelemTest.createTestWrapper({ registry: button.REGISTRY });

describe("Button", () => {
  it("should return onMouseDown and onMouseUp handlers", () => {
    const { result } = renderHook(
      () =>
        Button.use({
          aetherKey: "test-button",
          sink: noopBooleanSinkSpec,
          mode: "fire",
        }),
      { wrapper: TestWrapper },
    );

    expect(result.current.onMouseDown).toBeDefined();
    expect(result.current.onMouseUp).toBeDefined();
    expect(result.current.onClick).toBeDefined();
  });

  it("should allow calling onMouseDown and onMouseUp", () => {
    const { result } = renderHook(
      () =>
        Button.use({
          aetherKey: "test-button-2",
          sink: noopBooleanSinkSpec,
          mode: "momentary",
        }),
      { wrapper: TestWrapper },
    );

    result.current.onMouseDown();
    result.current.onMouseUp();
  });

  describe("with TestSink", () => {
    let sink: telemTest.TestSink<boolean>;

    beforeEach(() => {
      sink = telemTest.sink<boolean>();
    });

    afterEach(() => {
      sink.cleanup();
    });

    it("should call sink with true on mouseDown in momentary mode", () => {
      const { result } = renderHook(
        () =>
          Button.use({
            aetherKey: "test-button-sink",
            sink: telemTest.booleanSinkSpec(sink),
            mode: "momentary",
          }),
        { wrapper: TestWrapper },
      );

      result.current.onMouseDown();

      expect(sink.lastValue).toBe(true);
      expect(sink.values).toEqual([true]);
    });

    it("should call sink with false on mouseUp in momentary mode", () => {
      const { result } = renderHook(
        () =>
          Button.use({
            aetherKey: "test-button-sink-2",
            sink: telemTest.booleanSinkSpec(sink),
            mode: "momentary",
          }),
        { wrapper: TestWrapper },
      );

      result.current.onMouseDown();
      result.current.onMouseUp();

      expect(sink.lastValue).toBe(false);
      expect(sink.values).toEqual([true, false]);
    });

    it("should call sink with true on mouseUp in fire mode", () => {
      const { result } = renderHook(
        () =>
          Button.use({
            aetherKey: "test-button-fire",
            sink: telemTest.booleanSinkSpec(sink),
            mode: "fire",
          }),
        { wrapper: TestWrapper },
      );

      result.current.onMouseDown();
      expect(sink.values).toEqual([]);

      result.current.onMouseUp();
      expect(sink.lastValue).toBe(true);
      expect(sink.values).toEqual([true]);
    });

    it("should call sink with true then false on mouseDown in pulse mode", () => {
      const { result } = renderHook(
        () =>
          Button.use({
            aetherKey: "test-button-pulse",
            sink: telemTest.booleanSinkSpec(sink),
            mode: "pulse",
          }),
        { wrapper: TestWrapper },
      );

      result.current.onMouseDown();

      expect(sink.values).toEqual([true, false]);
    });
  });
});
