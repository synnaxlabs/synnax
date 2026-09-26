// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, type RenderResult } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { Input } from "@/input";

// The drag button ignores movement inside its threshold, then scales what is left, so a
// drag to x carries a value of (x - THRESHOLD) at a unit scale.
const THRESHOLD = 15;
const UNIT_SCALE = { x: 1, y: 1 };

const dragButtonOf = (c: RenderResult): HTMLElement => {
  const el = c.container.querySelector<HTMLElement>(`.${CSS.BE("input", "drag-btn")}`);
  if (el == null) throw new Error("no drag button rendered");
  return el;
};

const pointer = (x: number) => ({
  pointerId: 1,
  button: 0,
  isPrimary: true,
  clientX: x,
  clientY: 0,
});

const dragTo = (el: HTMLElement, ...xs: number[]): void => {
  fireEvent.pointerDown(el, pointer(0));
  xs.forEach((x) => fireEvent.pointerMove(el, pointer(x)));
};

const release = (el: HTMLElement, x: number): void => {
  fireEvent.pointerUp(el, pointer(x));
};

describe("Input.Numeric", () => {
  describe("drag handle", () => {
    it("should emit a change per drag frame by default", () => {
      const onChange = vi.fn();
      const c = render(
        <Input.Numeric value={0} onChange={onChange} dragScale={UNIT_SCALE} />,
      );
      dragTo(dragButtonOf(c), 100, 200);
      expect(onChange.mock.calls.map(([v]) => v)).toEqual([
        100 - THRESHOLD,
        200 - THRESHOLD,
      ]);
    });

    describe("onlyChangeOnBlur", () => {
      it("should not emit while the drag is in progress", () => {
        const onChange = vi.fn();
        const c = render(
          <Input.Numeric
            value={0}
            onChange={onChange}
            dragScale={UNIT_SCALE}
            onlyChangeOnBlur
          />,
        );
        dragTo(dragButtonOf(c), 100, 200);
        expect(onChange).not.toHaveBeenCalled();
      });

      it("should show the dragged value in the input while the drag is in progress", () => {
        const c = render(
          <Input.Numeric
            value={0}
            onChange={vi.fn()}
            dragScale={UNIT_SCALE}
            onlyChangeOnBlur
          />,
        );
        dragTo(dragButtonOf(c), 100);
        expect((c.getByRole("textbox") as HTMLInputElement).value).toEqual(
          `${100 - THRESHOLD}`,
        );
      });

      it("should emit once with the final value when the drag is released", () => {
        const onChange = vi.fn();
        const c = render(
          <Input.Numeric
            value={0}
            onChange={onChange}
            dragScale={UNIT_SCALE}
            onlyChangeOnBlur
          />,
        );
        const btn = dragButtonOf(c);
        dragTo(btn, 100, 200);
        release(btn, 200);
        expect(onChange).toHaveBeenCalledOnce();
        expect(onChange).toHaveBeenCalledWith(200 - THRESHOLD);
      });

      it("should clamp the released value to the bounds", () => {
        const onChange = vi.fn();
        const c = render(
          <Input.Numeric
            value={0}
            onChange={onChange}
            bounds={{ lower: 0, upper: 50 }}
            dragScale={UNIT_SCALE}
            onlyChangeOnBlur
          />,
        );
        const btn = dragButtonOf(c);
        dragTo(btn, 100, 200);
        release(btn, 200);
        expect(onChange).toHaveBeenCalledOnce();
        expect(onChange).toHaveBeenCalledWith(50);
      });

      it("should emit the typed value on blur", () => {
        const onChange = vi.fn();
        const c = render(
          <Input.Numeric value={0} onChange={onChange} onlyChangeOnBlur />,
        );
        const input = c.getByRole("textbox");
        fireEvent.change(input, { target: { value: "42" } });
        fireEvent.blur(input);
        expect(onChange).toHaveBeenCalledOnce();
        expect(onChange).toHaveBeenCalledWith(42);
      });
    });
  });

  describe("bounds", () => {
    it("should clamp a typed value to bounds that moved after mount", () => {
      const onChange = vi.fn();
      const c = render(
        <Input.Numeric
          value={0}
          onChange={onChange}
          bounds={{ lower: 0, upper: 100 }}
        />,
      );
      c.rerender(
        <Input.Numeric
          value={0}
          onChange={onChange}
          bounds={{ lower: 0, upper: 50 }}
        />,
      );
      const input = c.getByRole("textbox");
      fireEvent.change(input, { target: { value: "80" } });
      fireEvent.blur(input);
      expect(onChange).toHaveBeenCalledWith(50);
    });

    it("should clamp a dragged value to bounds that moved after mount", () => {
      const onChange = vi.fn();
      const rest = { value: 0, onChange, dragScale: UNIT_SCALE };
      const c = render(<Input.Numeric {...rest} bounds={{ lower: 0, upper: 100 }} />);
      c.rerender(<Input.Numeric {...rest} bounds={{ lower: 0, upper: 50 }} />);
      dragTo(dragButtonOf(c), 100);
      expect(onChange).toHaveBeenLastCalledWith(50);
    });
  });
});
