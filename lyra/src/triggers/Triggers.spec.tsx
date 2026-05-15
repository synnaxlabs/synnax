// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { mockBoundingClientRect } from "@/testutil/dom";
import { Triggers } from "@/triggers";

describe("Triggers", () => {
  describe("filter", () => {
    describe("not loose", () => {
      it("Should return an empty list when no triggers match", () => {
        expect(
          Triggers.filter(
            [
              ["A", "B"],
              ["A", "C"],
            ],
            [["A", "D"]],
          ),
        ).toEqual([]);
      });
      it("Should return a list of triggers that match", () => {
        expect(
          Triggers.filter(
            [
              ["A", "B"],
              ["A", "C"],
            ],
            [["A", "B"]],
          ),
        ).toEqual([["A", "B"]]);
      });
      it("Should not match loose triggers", () => {
        expect(Triggers.filter([["A"], ["A", "C"]], [["A", "B"]])).toEqual([]);
      });
      it("Should match multiple triggers", () => {
        expect(
          Triggers.filter(
            [
              ["A", "B"],
              ["A", "C"],
            ],
            [
              ["A", "B"],
              ["A", "C"],
            ],
          ),
        ).toEqual([
          ["A", "B"],
          ["A", "C"],
        ]);
      });
    });
    describe("loose", () => {
      it("Should return an empty list when no triggers match", () => {
        expect(
          Triggers.filter(
            [
              ["A", "B"],
              ["A", "C"],
            ],
            [["A", "D"]],
            { loose: true },
          ),
        ).toEqual([]);
      });
      it("Should return a list of triggers that match", () => {
        expect(
          Triggers.filter([["A"], ["A", "C"]], [["A", "B"]], { loose: true }),
        ).toEqual([["A"]]);
      });
      it("should return an empty list when no triggers match", () => {
        expect(
          Triggers.filter(
            [
              ["A", "B"],
              ["A", "C"],
            ],
            [["A"]],
            { loose: true },
          ),
        ).toEqual([]);
      });
    });
    describe("Triggers.purge", () => {
      it("Should correctly removed triggers from a list", () => {
        expect(
          Triggers.purge(
            [
              ["A", "B"],
              ["A", "C"],
            ],
            [["A", "B"]],
          ),
        ).toEqual([["A", "C"]]);
      });
    });
    describe("Diff", () => {
      it("Should correctly diff two lists of triggers", () => {
        expect(
          Triggers.diff(
            [
              ["A", "B"],
              ["A", "C"],
              ["A", "E"],
            ],
            [
              ["A", "B"],
              ["A", "C"],
              ["A", "D"],
            ],
          ),
        ).toEqual([[["A", "E"]], [["A", "D"]]]);
      });
    });
    describe("match", () => {
      it("should match the trigger correctly", () => {
        expect(Triggers.match);
      });
    });
    describe("Config", () => {
      describe("determineMode", () => {
        it("should select the matching mode with the highest complexity", () => {
          const config: Triggers.ModeConfig<"a" | "b"> = {
            defaultMode: "a",
            a: [["Shift"]],
            b: [["Shift", "Control"]],
          };
          expect(Triggers.determineMode(config, [["Shift", "Control"]])).toEqual("b");
          expect(Triggers.determineMode(config, [["Shift"]])).toEqual("a");
        });
        it("should correctly match loose trigers", () => {
          const config: Triggers.ModeConfig<"a" | "b"> = {
            defaultMode: "a",
            a: [["Shift"]],
            b: [["Shift", "Control"]],
          };
          expect(
            Triggers.determineMode(config, [["Shift", "Control"]], { loose: true }),
          ).toEqual("b");
          expect(Triggers.determineMode(config, [["Shift"]], { loose: true })).toEqual(
            "a",
          );
        });
      });
    });
  });

  describe("use", () => {
    it("should handle single key triggers", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["A"]],
        });
        return <div>Hello</div>;
      };
      const Wrapper = () => (
        <Triggers.Provider>
          <C />
        </Triggers.Provider>
      );
      render(<Wrapper />);
      fireEvent.keyDown(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenCalledOnce();
      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["A"]],
        prevTriggers: [],
        cursor: { x: 0, y: 0 },
        stage: "start",
      } satisfies Triggers.UseEvent);
      fireEvent.keyUp(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenCalledTimes(2);
      expect(callback).toHaveBeenLastCalledWith({
        target: document.body,
        triggers: [["A"]],
        prevTriggers: [["A"]],
        cursor: { x: 0, y: 0 },
        stage: "end",
      } satisfies Triggers.UseEvent);
    });

    it("should handle multi-key combinations", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["Control", "A"]],
        });
        return <div>Hello</div>;
      };
      render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      // Press Control first
      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      expect(callback).not.toHaveBeenCalled();

      // Then press A
      fireEvent.keyDown(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["Control", "A"]],
        prevTriggers: [["Control"]],
        cursor: { x: 0, y: 0 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      // Release A first
      fireEvent.keyUp(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenLastCalledWith({
        target: document.body,
        triggers: [["Control", "A"]],
        prevTriggers: [["Control", "A"]],
        cursor: { x: 0, y: 0 },
        stage: "end",
      } satisfies Triggers.UseEvent);

      // Release Control
      fireEvent.keyUp(document.body, { code: "ControlLeft" });
    });

    it("should handle mouse triggers", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["MouseLeft"]],
        });
        return <div>Hello</div>;
      };
      render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      fireEvent.mouseDown(document.body, { button: 0 });
      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["MouseLeft"]],
        prevTriggers: [],
        cursor: { x: 0, y: 0 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      fireEvent.mouseUp(document.body, { button: 0 });
      expect(callback).toHaveBeenLastCalledWith({
        target: document.body,
        triggers: [["MouseLeft"]],
        prevTriggers: [["MouseLeft"]],
        cursor: { x: 0, y: 0 },
        stage: "end",
      } satisfies Triggers.UseEvent);
    });

    it("should handle double key presses", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["A", "A"]],
          double: true,
        });
        return <div>Hello</div>;
      };
      render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      // First press
      fireEvent.keyDown(document.body, { code: "KeyA" });
      fireEvent.keyUp(document.body, { code: "KeyA" });

      // Quick second press (within 300ms)
      fireEvent.keyDown(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["A", "A"]],
        prevTriggers: [],
        cursor: { x: 0, y: 0 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      fireEvent.keyUp(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenLastCalledWith({
        target: document.body,
        triggers: [["A", "A"]],
        prevTriggers: [["A", "A"]],
        cursor: { x: 0, y: 0 },
        stage: "end",
      } satisfies Triggers.UseEvent);
    });

    it("should handle loose matching", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["Control"]],
          loose: true,
        });
        return <div>Hello</div>;
      };
      render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      // Control + A should trigger because of loose matching
      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyA" });

      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["Control"]],
        prevTriggers: [],
        cursor: { x: 0, y: 0 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      fireEvent.keyUp(document.body, { code: "KeyA" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });

      expect(callback).toHaveBeenLastCalledWith({
        target: document.body,
        triggers: [["Control"]],
        prevTriggers: [["Control"]],
        cursor: { x: 0, y: 0 },
        stage: "end",
      } satisfies Triggers.UseEvent);
    });

    it("should handle multiple simultaneous triggers", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [
            ["Control", "A"],
            ["Control", "B"],
          ],
        });
        return <div>Hello</div>;
      };
      render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      // Try Control + A
      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyA" });

      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["Control", "A"]],
        prevTriggers: [["Control"]],
        cursor: { x: 0, y: 0 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      fireEvent.keyUp(document.body, { code: "KeyA" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });

      // Try Control + B
      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyB" });

      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["Control", "A"]],
        prevTriggers: [["Control", "A"]],
        cursor: { x: 0, y: 0 },
        stage: "end",
      } satisfies Triggers.UseEvent);

      fireEvent.keyUp(document.body, { code: "KeyB" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });
    });
  });

  describe("region-based triggers", () => {
    it("should only trigger when cursor is in the specified region", async () => {
      Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
      const callback = vi.fn();
      const regionRef = { current: document.createElement("div") };
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["A"]],
          region: regionRef,
        });
        return <div ref={regionRef}>Target Region</div>;
      };
      render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      // Simulate mouse moving into region
      fireEvent.mouseMove(regionRef.current, { clientX: 10, clientY: 10 });

      // Key press should trigger because cursor is in region
      fireEvent.keyDown(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["A"]],
        prevTriggers: [],
        cursor: { x: 10, y: 10 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      // Move cursor out of region
      fireEvent.mouseMove(document.body, { clientX: -10, clientY: -10 });

      // Key press should not trigger because cursor is outside region
      fireEvent.keyDown(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenCalledTimes(1); // Still only called once

      fireEvent.keyUp(document.body, { code: "KeyA" });
    });

    it("should handle regionMustBeElement correctly", async () => {
      vi.useFakeTimers();
      const callback = vi.fn();
      Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
      const regionRef = { current: document.createElement("div") };
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["A"]],
          region: regionRef,
          regionMustBeElement: true,
        });
        return <div ref={regionRef}>Target Region</div>;
      };
      render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      // // Move cursor into region but trigger on body
      fireEvent.mouseMove(regionRef.current, { clientX: 10, clientY: 10 });
      fireEvent.keyDown(document.body, { code: "KeyA" });
      fireEvent.keyUp(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenCalledTimes(1);
      fireEvent.mouseMove(regionRef.current, { clientX: 10, clientY: 10 });

      vi.advanceTimersByTime(500);

      // Trigger directly on region element
      fireEvent.keyDown(regionRef.current, { code: "KeyA" });
      expect(callback).toHaveBeenCalledTimes(2);
      expect(callback).toHaveBeenLastCalledWith({
        target: regionRef.current,
        triggers: [["A"]],
        prevTriggers: [],
        cursor: { x: 10, y: 10 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      fireEvent.keyUp(regionRef.current, { code: "KeyA" });
    });

    it("should handle mouse triggers with regions", async () => {
      vi.useFakeTimers();
      const callback = vi.fn();
      Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
      const regionRef = { current: document.createElement("div") };
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["MouseLeft"]],
          region: regionRef,
        });
        return <div ref={regionRef}>Target Region</div>;
      };
      render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      // // Mouse click outside region
      fireEvent.mouseMove(document.body, { clientX: -10, clientY: -10 });
      fireEvent.mouseDown(document.body, { button: 0 });
      fireEvent.mouseUp(document.body, { button: 0 });
      expect(callback).toHaveBeenCalledOnce();

      vi.advanceTimersByTime(500);

      // Mouse click inside region
      fireEvent.mouseMove(regionRef.current, { clientX: 10, clientY: 10 });
      fireEvent.mouseDown(regionRef.current, { button: 0 });
      expect(callback).toHaveBeenCalledWith({
        target: regionRef.current,
        triggers: [["MouseLeft"]],
        prevTriggers: [],
        cursor: { x: 10, y: 10 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      fireEvent.mouseUp(regionRef.current, { button: 0 });
    });
  });

  describe("input element behavior", () => {
    beforeEach(() => {
      vi.useFakeTimers();
      Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    });

    afterEach(() => {
      vi.clearAllTimers();
      vi.useRealTimers();
    });

    it("should ignore alphanumeric keys in input elements without modifiers", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["A"]],
        });
        return <input type="text" data-testid="input" />;
      };
      const { getByTestId } = render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      const input = getByTestId("input");
      fireEvent.mouseMove(input, { clientX: 10, clientY: 10 });
      fireEvent.keyDown(input, { code: "KeyA" });
      vi.advanceTimersByTime(500);
      expect(callback).not.toHaveBeenCalled();
    });

    it("should handle ctrl+key combinations in input elements", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["Control", "A"]],
        });
        return <input type="text" data-testid="input" />;
      };
      const { getByTestId } = render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      const input = getByTestId("input");
      fireEvent.mouseMove(input, { clientX: 10, clientY: 10 });

      // Press Control first
      fireEvent.keyDown(input, { code: "ControlLeft" });
      expect(callback).not.toHaveBeenCalled();

      // Then press A with Control held
      fireEvent.keyDown(input, { code: "KeyA", ctrlKey: true });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target: input,
        triggers: [["Control", "A"]],
        prevTriggers: [["Control"]],
        cursor: { x: 10, y: 10 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      // Release in correct order
      fireEvent.keyUp(input, { code: "KeyA", ctrlKey: true });
      fireEvent.keyUp(input, { code: "ControlLeft" });
    });

    it("should handle contenteditable elements", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["A"], ["Control", "B"]],
        });
        return (
          <div data-testid="editable" contentEditable>
            Editable content
          </div>
        );
      };
      const { getByTestId } = render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      const editable = getByTestId("editable");
      fireEvent.mouseMove(editable, { clientX: 10, clientY: 10 });

      // Regular key should not trigger
      fireEvent.keyDown(editable, { code: "KeyA" });
      vi.advanceTimersByTime(500);
      expect(callback).not.toHaveBeenCalled();

      // Ctrl+key should trigger
      fireEvent.keyDown(editable, { code: "ControlLeft" });
      fireEvent.keyDown(editable, { code: "KeyB", ctrlKey: true });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target: editable,
        triggers: [["Control", "B"]],
        prevTriggers: [["Control"]],
        cursor: { x: 10, y: 10 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      fireEvent.keyUp(editable, { code: "KeyB", ctrlKey: true });
      fireEvent.keyUp(editable, { code: "ControlLeft" });
    });

    it("should handle non-alphanumeric keys in input elements", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["Escape"], ["ArrowUp"]],
        });
        return <input type="text" data-testid="input" />;
      };
      const { getByTestId } = render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      const input = getByTestId("input");
      fireEvent.mouseMove(input, { clientX: 10, clientY: 10 });

      // Escape should trigger
      fireEvent.keyDown(input, { code: "Escape" });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target: input,
        triggers: [["Escape"]],
        prevTriggers: [],
        cursor: { x: 10, y: 10 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      fireEvent.keyUp(input, { code: "Escape" });
      vi.advanceTimersByTime(500);

      // Arrow keys should trigger
      fireEvent.keyDown(input, { code: "ArrowUp" });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target: input,
        triggers: [["ArrowUp"]],
        prevTriggers: [],
        cursor: { x: 10, y: 10 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      fireEvent.keyUp(input, { code: "ArrowUp" });
    });
  });

  describe("meta key handling", () => {
    beforeEach(() => {
      vi.useFakeTimers();
      Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    });

    afterEach(() => {
      vi.clearAllTimers();
      vi.useRealTimers();
    });

    it("should treat Meta (Command) key as Control", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["Control", "A"]],
        });
        return <div data-testid="target">Hello</div>;
      };
      const { getByTestId } = render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      const target = getByTestId("target");
      fireEvent.mouseMove(target, { clientX: 10, clientY: 10 });

      // Press Meta (Command) first
      fireEvent.keyDown(target, { code: "MetaLeft" });
      expect(callback).not.toHaveBeenCalled();

      // Then press A with Meta held
      fireEvent.keyDown(target, { code: "KeyA", metaKey: true });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target,
        triggers: [["Control", "A"]],
        prevTriggers: [["Control"]],
        cursor: { x: 10, y: 10 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      // Release in correct order
      fireEvent.keyUp(target, { code: "KeyA", metaKey: true });
      fireEvent.keyUp(target, { code: "MetaLeft" });
    });

    it("should handle both Meta and Control for the same trigger", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["Control", "S"]],
        });
        return <div data-testid="target">Hello</div>;
      };
      const { getByTestId } = render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      const target = getByTestId("target");
      fireEvent.mouseMove(target, { clientX: 10, clientY: 10 });

      // Test with Control
      fireEvent.keyDown(target, { code: "ControlLeft" });
      fireEvent.keyDown(target, { code: "KeyS", ctrlKey: true });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target,
        triggers: [["Control", "S"]],
        prevTriggers: [["Control"]],
        cursor: { x: 10, y: 10 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      fireEvent.keyUp(target, { code: "KeyS", ctrlKey: true });
      fireEvent.keyUp(target, { code: "ControlLeft" });
      vi.advanceTimersByTime(500);

      // Test with Meta
      fireEvent.keyDown(target, { code: "MetaLeft" });
      fireEvent.keyDown(target, { code: "KeyS", metaKey: true });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target,
        triggers: [["Control", "S"]],
        prevTriggers: [["Control"]],
        cursor: { x: 10, y: 10 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      fireEvent.keyUp(target, { code: "KeyS", metaKey: true });
      fireEvent.keyUp(target, { code: "MetaLeft" });
    });

    it("should handle right Meta key the same as left Meta key", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["Control", "X"]],
        });
        return <div data-testid="target">Hello</div>;
      };
      const { getByTestId } = render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      const target = getByTestId("target");
      fireEvent.mouseMove(target, { clientX: 10, clientY: 10 });

      // Test with right Meta key
      fireEvent.keyDown(target, { code: "MetaRight" });
      fireEvent.keyDown(target, { code: "KeyX", metaKey: true });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target,
        triggers: [["Control", "X"]],
        prevTriggers: [["Control"]],
        cursor: { x: 10, y: 10 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      fireEvent.keyUp(target, { code: "KeyX", metaKey: true });
      fireEvent.keyUp(target, { code: "MetaRight" });
    });

    it("should handle Safari's sticky shift key behavior", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["Shift", "A"]],
        });
        return <div data-testid="target">Hello</div>;
      };
      const { getByTestId } = render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      const target = getByTestId("target");
      fireEvent.mouseMove(target, { clientX: 10, clientY: 10 });

      // Press Shift first
      fireEvent.keyDown(target, { code: "ShiftLeft", shiftKey: true });
      expect(callback).not.toHaveBeenCalled();

      // Hold shift and press A - this starts our trigger
      fireEvent.keyDown(target, { code: "KeyA", shiftKey: true });
      expect(callback).toHaveBeenCalledWith({
        target,
        triggers: [["Shift", "A"]],
        prevTriggers: [["Shift"]],
        cursor: { x: 10, y: 10 },
        stage: "start",
      } satisfies Triggers.UseEvent);

      // Keep holding both keys and verify the state
      expect(callback).toHaveBeenCalledTimes(1);

      // Now simulate Safari's behavior:
      // Without releasing Shift, we get a new key press with shiftKey: false
      // This should detect that Shift is no longer held
      fireEvent.keyDown(target, { code: "KeyB", shiftKey: false });

      // At this point, the trigger should detect that Shift is no longer pressed
      // and end the trigger
      expect(callback).toHaveBeenCalledTimes(2);
      expect(callback).toHaveBeenLastCalledWith({
        target,
        triggers: [["Shift", "A"]],
        prevTriggers: [["Shift", "A"]],
        cursor: { x: 10, y: 10 },
        stage: "end",
      } satisfies Triggers.UseEvent);

      // Try pressing another key to verify the trigger is truly ended
      fireEvent.keyDown(target, { code: "KeyC", shiftKey: false });
      expect(callback).toHaveBeenCalledTimes(2);

      // Cleanup
      fireEvent.keyUp(target, { code: "KeyA", shiftKey: false });
      fireEvent.keyUp(target, { code: "KeyB", shiftKey: false });
      fireEvent.keyUp(target, { code: "KeyC", shiftKey: false });
    });
  });
});
