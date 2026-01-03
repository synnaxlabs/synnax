// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { noopBooleanSinkSpec, noopBooleanSourceSpec } from "@/telem/aether/noop";
import { telemTest } from "@/telem/aether/test";
import { TelemTest } from "@/telem/test";
import { Toggle } from "@/vis/toggle";
import { toggle } from "@/vis/toggle/aether";

const TestWrapper = TelemTest.createTestWrapper({ registry: toggle.REGISTRY });

describe("Toggle", () => {
  it("should return toggle function and enabled state", () => {
    const { result } = renderHook(
      () =>
        Toggle.use({
          aetherKey: "test-toggle",
          source: noopBooleanSourceSpec,
          sink: noopBooleanSinkSpec,
        }),
      { wrapper: TestWrapper },
    );

    expect(result.current.toggle).toBeDefined();
    expect(result.current.enabled).toBe(false);
  });

  describe("with TestSink", () => {
    let sink: telemTest.TestSink<boolean>;

    beforeEach(() => {
      sink = telemTest.sink<boolean>();
    });

    afterEach(() => {
      sink.cleanup();
    });

    it("should send inverted value to sink when toggled", () => {
      const { result } = renderHook(
        () =>
          Toggle.use({
            aetherKey: "test-toggle-sink",
            source: noopBooleanSourceSpec,
            sink: telemTest.booleanSinkSpec(sink),
          }),
        { wrapper: TestWrapper },
      );

      act(() => {
        result.current.toggle();
      });

      expect(sink.lastValue).toBe(true);
      expect(sink.values).toEqual([true]);
    });
  });

  describe("with TestSource", () => {
    let source: telemTest.TestSource<boolean>;

    beforeEach(() => {
      source = telemTest.source(false);
    });

    afterEach(() => {
      source.cleanup();
    });

    it("should reflect source value in enabled state", async () => {
      const { result } = renderHook(
        () =>
          Toggle.use({
            aetherKey: "test-toggle-source",
            source: telemTest.booleanSourceSpec(source),
            sink: noopBooleanSinkSpec,
          }),
        { wrapper: TestWrapper },
      );

      expect(result.current.enabled).toBe(false);

      act(() => {
        source.setValue(true);
      });

      expect(result.current.enabled).toBe(true);
    });
  });

  describe("with TestSource and TestSink", () => {
    let source: telemTest.TestSource<boolean>;
    let sink: telemTest.TestSink<boolean>;

    beforeEach(() => {
      source = telemTest.source(false);
      sink = telemTest.sink<boolean>();
    });

    afterEach(() => {
      source.cleanup();
      sink.cleanup();
    });

    it("should toggle from current source state", () => {
      const { result } = renderHook(
        () =>
          Toggle.use({
            aetherKey: "test-toggle-both",
            source: telemTest.booleanSourceSpec(source),
            sink: telemTest.booleanSinkSpec(sink),
          }),
        { wrapper: TestWrapper },
      );

      expect(result.current.enabled).toBe(false);

      act(() => {
        result.current.toggle();
      });

      expect(sink.lastValue).toBe(true);

      act(() => {
        source.setValue(true);
      });

      expect(result.current.enabled).toBe(true);

      act(() => {
        result.current.toggle();
      });

      expect(sink.lastValue).toBe(false);
      expect(sink.values).toEqual([true, false]);
    });

    it("should alternate values on sequential toggles with feedback", () => {
      const { result } = renderHook(
        () =>
          Toggle.use({
            aetherKey: "test-toggle-sequential",
            source: telemTest.booleanSourceSpec(source),
            sink: telemTest.booleanSinkSpec(sink),
          }),
        { wrapper: TestWrapper },
      );

      act(() => {
        result.current.toggle();
        source.setValue(true);
      });
      act(() => {
        result.current.toggle();
        source.setValue(false);
      });
      act(() => {
        result.current.toggle();
        source.setValue(true);
      });

      expect(sink.values).toEqual([true, false, true]);
    });
  });
});
