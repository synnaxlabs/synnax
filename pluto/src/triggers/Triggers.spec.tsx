// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { mockBoundingClientRect } from "@/testutil/dom";
import { Triggers } from "@/triggers";

const TriggersWrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>{children}</Triggers.Provider>
);

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
            modes: { a: [["Shift"]], b: [["Shift", "Control"]] },
          };
          expect(Triggers.determineMode(config, [["Shift", "Control"]])).toEqual("b");
          expect(Triggers.determineMode(config, [["Shift"]])).toEqual("a");
        });
        it("should correctly match loose trigers", () => {
          const config: Triggers.ModeConfig<"a" | "b"> = {
            defaultMode: "a",
            modes: { a: [["Shift"]], b: [["Shift", "Control"]] },
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
      renderHook(() => Triggers.use({ callback, triggers: [["A"]] }), {
        wrapper: TriggersWrapper,
      });
      fireEvent.keyDown(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenCalledOnce();
      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["A"]],
        prevTriggers: [],
        cursor: { x: 0, y: 0 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });
      fireEvent.keyUp(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenCalledTimes(2);
      expect(callback).toHaveBeenLastCalledWith({
        target: document.body,
        triggers: [["A"]],
        prevTriggers: [["A"]],
        cursor: { x: 0, y: 0 },
        stage: "end",
        stopPropagation: expect.any(Function),
      });
    });

    it("should handle multi-key combinations", async () => {
      const callback = vi.fn();
      renderHook(() => Triggers.use({ callback, triggers: [["Control", "A"]] }), {
        wrapper: TriggersWrapper,
      });

      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      expect(callback).not.toHaveBeenCalled();

      fireEvent.keyDown(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["Control", "A"]],
        prevTriggers: [["Control"]],
        cursor: { x: 0, y: 0 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });

      fireEvent.keyUp(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenLastCalledWith({
        target: document.body,
        triggers: [["Control", "A"]],
        prevTriggers: [["Control", "A"]],
        cursor: { x: 0, y: 0 },
        stage: "end",
        stopPropagation: expect.any(Function),
      });

      fireEvent.keyUp(document.body, { code: "ControlLeft" });
    });

    it("should handle mouse triggers", async () => {
      const callback = vi.fn();
      renderHook(() => Triggers.use({ callback, triggers: [["MouseLeft"]] }), {
        wrapper: TriggersWrapper,
      });

      fireEvent.mouseDown(document.body, { button: 0 });
      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["MouseLeft"]],
        prevTriggers: [],
        cursor: { x: 0, y: 0 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });

      fireEvent.mouseUp(document.body, { button: 0 });
      expect(callback).toHaveBeenLastCalledWith({
        target: document.body,
        triggers: [["MouseLeft"]],
        prevTriggers: [["MouseLeft"]],
        cursor: { x: 0, y: 0 },
        stage: "end",
        stopPropagation: expect.any(Function),
      });
    });

    it("should handle double key presses", async () => {
      const callback = vi.fn();
      renderHook(
        () => Triggers.use({ callback, triggers: [["A", "A"]], double: true }),
        { wrapper: TriggersWrapper },
      );

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
        stopPropagation: expect.any(Function),
      });

      fireEvent.keyUp(document.body, { code: "KeyA" });
      expect(callback).toHaveBeenLastCalledWith({
        target: document.body,
        triggers: [["A", "A"]],
        prevTriggers: [["A", "A"]],
        cursor: { x: 0, y: 0 },
        stage: "end",
        stopPropagation: expect.any(Function),
      });
    });

    it("should handle loose matching", async () => {
      const callback = vi.fn();
      renderHook(
        () => Triggers.use({ callback, triggers: [["Control"]], loose: true }),
        { wrapper: TriggersWrapper },
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
        stopPropagation: expect.any(Function),
      });

      fireEvent.keyUp(document.body, { code: "KeyA" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });

      expect(callback).toHaveBeenLastCalledWith({
        target: document.body,
        triggers: [["Control"]],
        prevTriggers: [["Control"]],
        cursor: { x: 0, y: 0 },
        stage: "end",
        stopPropagation: expect.any(Function),
      });
    });

    it("should handle multiple simultaneous triggers", async () => {
      const callback = vi.fn();
      renderHook(
        () =>
          Triggers.use({
            callback,
            triggers: [
              ["Control", "A"],
              ["Control", "B"],
            ],
          }),
        { wrapper: TriggersWrapper },
      );

      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyA" });

      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["Control", "A"]],
        prevTriggers: [["Control"]],
        cursor: { x: 0, y: 0 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });

      fireEvent.keyUp(document.body, { code: "KeyA" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });

      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyB" });

      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["Control", "A"]],
        prevTriggers: [["Control", "A"]],
        cursor: { x: 0, y: 0 },
        stage: "end",
        stopPropagation: expect.any(Function),
      });

      fireEvent.keyUp(document.body, { code: "KeyB" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });
    });
  });

  describe("scope", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    interface RenderScopedProps {
      active: Triggers.Condition;
      inner?: Triggers.Condition;
      enabled?: Triggers.Condition;
    }

    const renderScoped = ({ active, inner, enabled }: RenderScopedProps) => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({ callback, triggers: [["A"]], enabled });
        return <div>Hello</div>;
      };
      const subject =
        inner == null ? (
          <C />
        ) : (
          <Triggers.Scope active={inner}>
            <C />
          </Triggers.Scope>
        );
      render(
        <Triggers.Provider>
          <Triggers.Scope active={active}>{subject}</Triggers.Scope>
        </Triggers.Provider>,
      );
      const stages = () => callback.mock.calls.map(([e]) => e.stage);
      return { stages };
    };

    const pressA = () => {
      fireEvent.keyDown(document.body, { code: "KeyA" });
      fireEvent.keyUp(document.body, { code: "KeyA" });
      // Clears the provider's double-press window so the next press is a fresh one.
      vi.advanceTimersByTime(500);
    };

    it("should deliver triggers while active", () => {
      const { stages } = renderScoped({ active: true });
      pressA();
      expect(stages()).toEqual(["start", "end"]);
    });

    it("should withhold the press while inactive", () => {
      const { stages } = renderScoped({ active: false });
      pressA();
      expect(stages()).not.toContain("start");
    });

    it("should withhold the release when the press was withheld", () => {
      const { stages } = renderScoped({ active: false });
      pressA();
      expect(stages()).toEqual([]);
    });

    it("should deliver the release after deactivation so a held key cannot stick", () => {
      let active = true;
      const { stages } = renderScoped({ active: () => active });
      fireEvent.keyDown(document.body, { code: "KeyA" });
      active = false;
      fireEvent.keyUp(document.body, { code: "KeyA" });
      vi.advanceTimersByTime(500);
      expect(stages()).toEqual(["start", "end"]);
    });

    it("should read a getter at fire time rather than at render", () => {
      let active = false;
      const { stages } = renderScoped({ active: () => active });
      pressA();
      expect(stages()).not.toContain("start");
      active = true;
      pressA();
      expect(stages()).toContain("start");
    });

    it("should not let an inner scope re-enable an outer one", () => {
      const { stages } = renderScoped({ active: false, inner: true });
      pressA();
      expect(stages()).not.toContain("start");
    });

    it("should withhold the press when enabled is false inside an active scope", () => {
      const { stages } = renderScoped({ active: true, enabled: false });
      pressA();
      expect(stages()).not.toContain("start");
    });
  });

  describe("useUndoRedo", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    interface RenderUndoRedoProps {
      enabled?: Triggers.Condition;
      scope?: Triggers.Condition;
    }

    const renderUndoRedo = ({ enabled, scope }: RenderUndoRedoProps = {}) => {
      const undo = vi.fn();
      const redo = vi.fn();
      const wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Triggers.Provider>
          {scope == null ? (
            children
          ) : (
            <Triggers.Scope active={scope}>{children}</Triggers.Scope>
          )}
        </Triggers.Provider>
      );
      renderHook(() => Triggers.useUndoRedo({ undo, redo, enabled }), { wrapper });
      return { undo, redo };
    };

    const press = (...codes: string[]) => {
      codes.forEach((code) => fireEvent.keyDown(document.body, { code }));
      [...codes].reverse().forEach((code) => fireEvent.keyUp(document.body, { code }));
      vi.advanceTimersByTime(500);
    };

    it("should call undo on control+z", () => {
      const { undo, redo } = renderUndoRedo();
      press("ControlLeft", "KeyZ");
      expect(undo).toHaveBeenCalledTimes(1);
      expect(redo).not.toHaveBeenCalled();
    });

    it("should call redo on control+shift+z", () => {
      const { undo, redo } = renderUndoRedo();
      press("ControlLeft", "ShiftLeft", "KeyZ");
      expect(redo).toHaveBeenCalledTimes(1);
      expect(undo).not.toHaveBeenCalled();
    });

    it("should treat meta+z as undo", () => {
      const { undo } = renderUndoRedo();
      press("MetaLeft", "KeyZ");
      expect(undo).toHaveBeenCalledTimes(1);
    });

    it("should not fire on the release", () => {
      const { undo } = renderUndoRedo();
      press("ControlLeft", "KeyZ");
      expect(undo).toHaveBeenCalledTimes(1);
    });

    it("should ignore z without a modifier", () => {
      const { undo, redo } = renderUndoRedo();
      press("KeyZ");
      expect(undo).not.toHaveBeenCalled();
      expect(redo).not.toHaveBeenCalled();
    });

    it("should withhold both handlers while disabled", () => {
      const { undo, redo } = renderUndoRedo({ enabled: false });
      press("ControlLeft", "KeyZ");
      press("ControlLeft", "ShiftLeft", "KeyZ");
      expect(undo).not.toHaveBeenCalled();
      expect(redo).not.toHaveBeenCalled();
    });

    it("should withhold both handlers inside an inactive scope", () => {
      const { undo, redo } = renderUndoRedo({ scope: false });
      press("ControlLeft", "KeyZ");
      press("ControlLeft", "ShiftLeft", "KeyZ");
      expect(undo).not.toHaveBeenCalled();
      expect(redo).not.toHaveBeenCalled();
    });
  });

  describe("useHeld", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    const renderHeld = (triggers: Triggers.Trigger[], loose?: boolean) => {
      const { result } = renderHook(() => Triggers.useHeld({ triggers, loose }), {
        wrapper: ({ children }) => <Triggers.Provider>{children}</Triggers.Provider>,
      });
      return result;
    };

    it("should start unheld", () => {
      const result = renderHeld([["Control"]]);
      expect(result.current).toEqual({ triggers: [], held: false });
    });

    it("should report held while the trigger is down", () => {
      const result = renderHeld([["Control"]]);
      act(() => {
        fireEvent.keyDown(document.body, { code: "ControlLeft" });
      });
      expect(result.current.held).toBe(true);
      expect(result.current.triggers).toEqual([["Control"]]);
    });

    it("should report unheld once the trigger is released", () => {
      const result = renderHeld([["Control"]]);
      act(() => {
        fireEvent.keyDown(document.body, { code: "ControlLeft" });
      });
      act(() => {
        fireEvent.keyUp(document.body, { code: "ControlLeft" });
      });
      expect(result.current).toEqual({ triggers: [], held: false });
    });

    it("should stay held under loose matching until the last trigger lifts", () => {
      const result = renderHeld([["Control"], ["Alt"]], true);
      act(() => {
        fireEvent.keyDown(document.body, { code: "ControlLeft" });
      });
      act(() => {
        fireEvent.keyDown(document.body, { code: "AltLeft" });
      });
      act(() => {
        fireEvent.keyUp(document.body, { code: "ControlLeft" });
      });
      expect(result.current).toEqual({ triggers: [["Alt"]], held: true });
      act(() => {
        fireEvent.keyUp(document.body, { code: "AltLeft" });
      });
      expect(result.current.held).toBe(false);
    });

    it("should drop an exact match once a second key joins it", () => {
      const result = renderHeld([["Control"], ["Alt"]]);
      act(() => {
        fireEvent.keyDown(document.body, { code: "ControlLeft" });
      });
      act(() => {
        fireEvent.keyDown(document.body, { code: "AltLeft" });
      });
      expect(result.current.held).toBe(false);
    });

    it("should ignore a key that is not one of its triggers", () => {
      const result = renderHeld([["Control"]]);
      act(() => {
        fireEvent.keyDown(document.body, { code: "KeyA" });
      });
      expect(result.current.held).toBe(false);
    });
  });

  describe("useHeldRef", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    const renderHeldRef = (triggers: Triggers.Trigger[]) => {
      const { result } = renderHook(() => Triggers.useHeldRef({ triggers }), {
        wrapper: ({ children }) => <Triggers.Provider>{children}</Triggers.Provider>,
      });
      return result;
    };

    it("should track the held state without re-rendering", () => {
      const renders = vi.fn();
      const { result } = renderHook(
        () => {
          renders();
          return Triggers.useHeldRef({ triggers: [["Control"]] });
        },
        {
          wrapper: ({ children }) => <Triggers.Provider>{children}</Triggers.Provider>,
        },
      );
      const before = renders.mock.calls.length;
      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      expect(result.current.current.held).toBe(true);
      expect(renders.mock.calls.length).toEqual(before);
    });

    it("should clear the ref once the trigger is released", () => {
      const result = renderHeldRef([["Control"]]);
      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });
      expect(result.current.current).toEqual({ triggers: [], held: false });
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
        stopPropagation: expect.any(Function),
      });

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
        stopPropagation: expect.any(Function),
      });

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

      fireEvent.mouseMove(regionRef.current, { clientX: 10, clientY: 10 });
      fireEvent.mouseDown(regionRef.current, { button: 0 });
      expect(callback).toHaveBeenCalledWith({
        target: regionRef.current,
        triggers: [["MouseLeft"]],
        prevTriggers: [],
        cursor: { x: 10, y: 10 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });

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

    it("should handle non-text-editing ctrl+key combinations in input elements", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({
          callback,
          triggers: [["Control", "B"]],
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

      fireEvent.keyDown(input, { code: "ControlLeft" });
      expect(callback).not.toHaveBeenCalled();

      // Then press B with Control held
      fireEvent.keyDown(input, { code: "KeyB", ctrlKey: true });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target: input,
        triggers: [["Control", "B"]],
        prevTriggers: [["Control"]],
        cursor: { x: 10, y: 10 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });

      fireEvent.keyUp(input, { code: "KeyB", ctrlKey: true });
      fireEvent.keyUp(input, { code: "ControlLeft" });
    });

    it("should suppress native text-editing shortcuts in input elements", async () => {
      const C = ({ callback }: { callback: () => void }) => {
        Triggers.use({
          callback,
          triggers: [
            ["Control", "A"],
            ["Control", "C"],
            ["Control", "V"],
            ["Control", "X"],
          ],
        });
        return <input type="text" data-testid="input" />;
      };
      const press = (code: string) => {
        const callback = vi.fn();
        const { getByTestId, unmount } = render(
          <Triggers.Provider>
            <C callback={callback} />
          </Triggers.Provider>,
        );
        const input = getByTestId("input");
        fireEvent.mouseMove(input, { clientX: 10, clientY: 10 });
        fireEvent.keyDown(input, { code: "ControlLeft" });
        fireEvent.keyDown(input, { code, ctrlKey: true });
        vi.advanceTimersByTime(500);
        fireEvent.keyUp(input, { code, ctrlKey: true });
        fireEvent.keyUp(input, { code: "ControlLeft" });
        unmount();
        return callback;
      };

      expect(press("KeyA")).not.toHaveBeenCalled();
      expect(press("KeyC")).not.toHaveBeenCalled();
      expect(press("KeyV")).not.toHaveBeenCalled();
      expect(press("KeyX")).not.toHaveBeenCalled();
    });

    it("should still trigger undo/redo in input elements", async () => {
      const callback = vi.fn();
      const C = () => {
        Triggers.use({ callback, triggers: [Triggers.UNDO] });
        return <input type="text" data-testid="input" />;
      };
      const { getByTestId } = render(
        <Triggers.Provider>
          <C />
        </Triggers.Provider>,
      );

      const input = getByTestId("input");
      fireEvent.mouseMove(input, { clientX: 10, clientY: 10 });
      fireEvent.keyDown(input, { code: "ControlLeft" });
      fireEvent.keyDown(input, { code: "KeyZ", ctrlKey: true });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target: input,
        triggers: [Triggers.UNDO],
        prevTriggers: [["Control"]],
        cursor: { x: 10, y: 10 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });

      fireEvent.keyUp(input, { code: "KeyZ", ctrlKey: true });
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

      fireEvent.keyDown(editable, { code: "ControlLeft" });
      fireEvent.keyDown(editable, { code: "KeyB", ctrlKey: true });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target: editable,
        triggers: [["Control", "B"]],
        prevTriggers: [["Control"]],
        cursor: { x: 10, y: 10 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });

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

      fireEvent.keyDown(input, { code: "Escape" });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target: input,
        triggers: [["Escape"]],
        prevTriggers: [],
        cursor: { x: 10, y: 10 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });

      fireEvent.keyUp(input, { code: "Escape" });
      vi.advanceTimersByTime(500);

      fireEvent.keyDown(input, { code: "ArrowUp" });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target: input,
        triggers: [["ArrowUp"]],
        prevTriggers: [],
        cursor: { x: 10, y: 10 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });

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
        stopPropagation: expect.any(Function),
      });

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

      fireEvent.keyDown(target, { code: "ControlLeft" });
      fireEvent.keyDown(target, { code: "KeyS", ctrlKey: true });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target,
        triggers: [["Control", "S"]],
        prevTriggers: [["Control"]],
        cursor: { x: 10, y: 10 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });

      fireEvent.keyUp(target, { code: "KeyS", ctrlKey: true });
      fireEvent.keyUp(target, { code: "ControlLeft" });
      vi.advanceTimersByTime(500);

      fireEvent.keyDown(target, { code: "MetaLeft" });
      fireEvent.keyDown(target, { code: "KeyS", metaKey: true });
      vi.advanceTimersByTime(500);

      expect(callback).toHaveBeenCalledWith({
        target,
        triggers: [["Control", "S"]],
        prevTriggers: [["Control"]],
        cursor: { x: 10, y: 10 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });

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
        stopPropagation: expect.any(Function),
      });

      fireEvent.keyUp(target, { code: "KeyX", metaKey: true });
      fireEvent.keyUp(target, { code: "MetaRight" });
    });

    it("should clear stuck non-modifier keys when Meta (Cmd) is released", async () => {
      const callback = vi.fn();
      renderHook(
        () =>
          Triggers.use({
            callback,
            triggers: [
              ["Control", "Shift", "P"],
              ["Control", "P"],
            ],
          }),
        { wrapper: TriggersWrapper },
      );

      // Cmd + Shift + P fires and opens whatever Control+Shift+P controls.
      fireEvent.keyDown(document.body, { code: "MetaLeft", metaKey: true });
      fireEvent.keyDown(document.body, {
        code: "ShiftLeft",
        metaKey: true,
        shiftKey: true,
      });
      fireEvent.keyDown(document.body, { code: "KeyP", metaKey: true, shiftKey: true });
      expect(callback).toHaveBeenCalledWith({
        target: document.body,
        triggers: [["Control", "Shift", "P"]],
        prevTriggers: [["Control", "Shift"]],
        cursor: { x: 0, y: 0 },
        stage: "start",
        stopPropagation: expect.any(Function),
      });

      // macOS suppresses the key up event for P while Cmd is held — only Shift and Cmd
      // key up events arrive. Without the fix, "P" stays stuck in the state.
      fireEvent.keyUp(document.body, {
        code: "ShiftLeft",
        metaKey: true,
        shiftKey: false,
      });
      fireEvent.keyUp(document.body, {
        code: "MetaLeft",
        metaKey: false,
        shiftKey: false,
      });

      const callsAfterRelease = callback.mock.calls.length;

      // Now press Cmd alone. Without the fix, the stuck "P" + new "Control" would match
      // the Control+P trigger and fire start. With the fix, the state was cleared on
      // Cmd release, so this should not match.
      fireEvent.keyDown(document.body, { code: "MetaLeft", metaKey: true });
      vi.advanceTimersByTime(500);
      expect(callback.mock.calls.length).toBe(callsAfterRelease);

      fireEvent.keyUp(document.body, { code: "MetaLeft", metaKey: false });
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
        stopPropagation: expect.any(Function),
      });

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
        stopPropagation: expect.any(Function),
      });

      // Try pressing another key to verify the trigger is truly ended
      fireEvent.keyDown(target, { code: "KeyC", shiftKey: false });
      expect(callback).toHaveBeenCalledTimes(2);

      fireEvent.keyUp(target, { code: "KeyA", shiftKey: false });
      fireEvent.keyUp(target, { code: "KeyB", shiftKey: false });
      fireEvent.keyUp(target, { code: "KeyC", shiftKey: false });
    });

    describe("priority and stopPropagation", () => {
      it("should invoke higher-priority subscribers before lower-priority ones", () => {
        const order: string[] = [];
        const high = vi.fn(() => order.push("high"));
        const low = vi.fn(() => order.push("low"));
        renderHook(
          () => {
            Triggers.use({ callback: high, triggers: [["A"]], priority: 100 });
            Triggers.use({ callback: low, triggers: [["A"]], priority: 0 });
          },
          { wrapper: TriggersWrapper },
        );
        fireEvent.keyDown(document.body, { code: "KeyA" });
        expect(order).toEqual(["high", "low"]);
      });

      it("should not invoke lower-priority subscribers when a higher-priority one stops propagation", () => {
        const high = vi.fn((e: Triggers.UseEvent) => e.stopPropagation());
        const low = vi.fn();
        renderHook(
          () => {
            Triggers.use({ callback: high, triggers: [["A"]], priority: 100 });
            Triggers.use({ callback: low, triggers: [["A"]], priority: 0 });
          },
          { wrapper: TriggersWrapper },
        );
        fireEvent.keyDown(document.body, { code: "KeyA" });
        expect(high).toHaveBeenCalledOnce();
        expect(low).not.toHaveBeenCalled();
      });

      it("should still invoke same-priority peers when one of them stops propagation", () => {
        const stopper = vi.fn((e: Triggers.UseEvent) => e.stopPropagation());
        const peer = vi.fn();
        const lower = vi.fn();
        renderHook(
          () => {
            Triggers.use({ callback: stopper, triggers: [["A"]], priority: 100 });
            Triggers.use({ callback: peer, triggers: [["A"]], priority: 100 });
            Triggers.use({ callback: lower, triggers: [["A"]], priority: 0 });
          },
          { wrapper: TriggersWrapper },
        );
        fireEvent.keyDown(document.body, { code: "KeyA" });
        expect(stopper).toHaveBeenCalledOnce();
        expect(peer).toHaveBeenCalledOnce();
        expect(lower).not.toHaveBeenCalled();
      });

      it("should invoke all subscribers when none stops propagation", () => {
        const high = vi.fn();
        const low = vi.fn();
        renderHook(
          () => {
            Triggers.use({ callback: high, triggers: [["A"]], priority: 100 });
            Triggers.use({ callback: low, triggers: [["A"]], priority: 0 });
          },
          { wrapper: TriggersWrapper },
        );
        fireEvent.keyDown(document.body, { code: "KeyA" });
        expect(high).toHaveBeenCalledOnce();
        expect(low).toHaveBeenCalledOnce();
      });

      it("should default priority to 0 so unconfigured subscribers are blocked by a stopping priority>0 peer", () => {
        const high = vi.fn((e: Triggers.UseEvent) => e.stopPropagation());
        const defaultPriority = vi.fn();
        renderHook(
          () => {
            Triggers.use({ callback: high, triggers: [["A"]], priority: 1 });
            Triggers.use({ callback: defaultPriority, triggers: [["A"]] });
          },
          { wrapper: TriggersWrapper },
        );
        fireEvent.keyDown(document.body, { code: "KeyA" });
        expect(high).toHaveBeenCalledOnce();
        expect(defaultPriority).not.toHaveBeenCalled();
      });

      it("should isolate stopPropagation across separate dispatches", () => {
        const high = vi.fn((e: Triggers.UseEvent) => {
          if (e.stage === "start") e.stopPropagation();
        });
        const low = vi.fn();
        renderHook(
          () => {
            Triggers.use({ callback: high, triggers: [["A"]], priority: 100 });
            Triggers.use({ callback: low, triggers: [["B"]], priority: 0 });
          },
          { wrapper: TriggersWrapper },
        );
        fireEvent.keyDown(document.body, { code: "KeyA" });
        expect(high).toHaveBeenCalledOnce();
        expect(low).not.toHaveBeenCalled();
        fireEvent.keyUp(document.body, { code: "KeyA" });
        fireEvent.keyDown(document.body, { code: "KeyB" });
        expect(low).toHaveBeenCalledOnce();
        expect(low.mock.calls[0][0].stage).toBe("start");
      });

      it("should withhold the release from a subscriber whose press was stopped", () => {
        const high = vi.fn((e: Triggers.UseEvent) => {
          if (e.stage === "start") e.stopPropagation();
        });
        const low = vi.fn();
        renderHook(
          () => {
            Triggers.use({ callback: high, triggers: [["A"]], priority: 100 });
            Triggers.use({ callback: low, triggers: [["A"]], priority: 0 });
          },
          { wrapper: TriggersWrapper },
        );
        fireEvent.keyDown(document.body, { code: "KeyA" });
        fireEvent.keyUp(document.body, { code: "KeyA" });
        expect(low).not.toHaveBeenCalled();
      });
    });
  });

  describe("Text", () => {
    it("should print a word key the way the physical key prints it", () => {
      const c = render(<Triggers.Text trigger={["Escape"]} />);
      // The bare enum name ("Escape") leaking through is the regression: the cap has
      // to read like the key, and it is a square that a full word overflows.
      expect(c.getByText(/^esc$/i)).toBeTruthy();
      expect(c.queryByText("Escape")).toBeNull();
    });

    it("should print the delete key as its own label", () => {
      const c = render(<Triggers.Text trigger={["Delete"]} />);
      expect(c.getByText(/^delete$/i)).toBeTruthy();
    });

    it("should print punctuation keys as the symbol on the key", () => {
      const c = render(<Triggers.Text trigger={["Control", "Equal"]} />);
      expect(c.getByText("=")).toBeTruthy();
      expect(c.queryByText("Equal")).toBeNull();
    });

    it("should print the minus key as its symbol", () => {
      const c = render(<Triggers.Text trigger={["Control", "Minus"]} />);
      expect(c.getByText("-")).toBeTruthy();
      expect(c.queryByText("Minus")).toBeNull();
    });

    it("should render a label passed as children beside the keycaps", () => {
      const c = render(<Triggers.Text trigger={["Escape"]}>Close</Triggers.Text>);
      expect(c.getByText("Close")).toBeTruthy();
      expect(c.getByText(/^esc$/i)).toBeTruthy();
    });
  });
});
