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

import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";

const getButton = (container: HTMLElement): HTMLButtonElement =>
  container.querySelector("button") as HTMLButtonElement;

describe("Toggle.Button", () => {
  describe("zero-delay (immediate) behavior", () => {
    it("should call onClick on click when no delay is configured", () => {
      const onClick = vi.fn();
      const { container } = render(<Toggle.Button onClick={onClick} />);
      fireEvent.click(getButton(container));
      expect(onClick).toHaveBeenCalledTimes(1);
    });

    it("should call onClick on click when delay is explicitly zero", () => {
      const onClick = vi.fn();
      const { container } = render(
        <Toggle.Button onClick={onClick} onClickDelay={0} />,
      );
      fireEvent.click(getButton(container));
      expect(onClick).toHaveBeenCalledTimes(1);
    });

    it("should still call onMouseDown for the zero-delay path", () => {
      const onMouseDown = vi.fn();
      const { container } = render(<Toggle.Button onMouseDown={onMouseDown} />);
      fireEvent.mouseDown(getButton(container));
      expect(onMouseDown).toHaveBeenCalledTimes(1);
    });

    it("should not set the toggle-delay CSS variable when delay is zero", () => {
      const { container } = render(<Toggle.Button style={{ width: 10 }} />);
      const btn = getButton(container);
      expect(btn.style.getPropertyValue("--pluto-toggle-delay")).toBe("");
      expect(btn.style.width).toBe("10px");
    });
  });

  describe("non-zero delay behavior", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("should not call onClick on a plain click event", () => {
      const onClick = vi.fn();
      const { container } = render(
        <Toggle.Button onClick={onClick} onClickDelay={500} />,
      );
      fireEvent.click(getButton(container));
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should defer onClick by the configured delay after mousedown", () => {
      const onClick = vi.fn();
      const { container } = render(
        <Toggle.Button onClick={onClick} onClickDelay={500} />,
      );
      fireEvent.mouseDown(getButton(container));
      expect(onClick).not.toHaveBeenCalled();
      vi.advanceTimersByTime(499);
      expect(onClick).not.toHaveBeenCalled();
      vi.advanceTimersByTime(1);
      expect(onClick).toHaveBeenCalledTimes(1);
    });

    it("should cancel the deferred onClick when mouseup arrives before the delay", () => {
      const onClick = vi.fn();
      const { container } = render(
        <Toggle.Button onClick={onClick} onClickDelay={500} />,
      );
      fireEvent.mouseDown(getButton(container));
      vi.advanceTimersByTime(100);
      fireEvent.mouseUp(document);
      vi.advanceTimersByTime(1000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should not fire after the toggle unmounts mid-hold", () => {
      const onClick = vi.fn();
      const c = render(<Toggle.Button onClick={onClick} onClickDelay={500} />);
      fireEvent.mouseDown(getButton(c.container));
      c.unmount();
      vi.advanceTimersByTime(1000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should not fire after the toggle is disabled mid-hold", () => {
      const onClick = vi.fn();
      const c = render(<Toggle.Button onClick={onClick} onClickDelay={500} />);
      fireEvent.mouseDown(getButton(c.container));
      c.rerender(<Toggle.Button onClick={onClick} onClickDelay={500} disabled />);
      vi.advanceTimersByTime(1000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should ignore a secondary-button press", () => {
      const onClick = vi.fn();
      const { container } = render(
        <Toggle.Button onClick={onClick} onClickDelay={500} />,
      );
      fireEvent.mouseDown(getButton(container), { button: 2 });
      vi.advanceTimersByTime(1000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should still call onMouseDown immediately even though onClick is deferred", () => {
      const onClick = vi.fn();
      const onMouseDown = vi.fn();
      const { container } = render(
        <Toggle.Button
          onClick={onClick}
          onMouseDown={onMouseDown}
          onClickDelay={500}
        />,
      );
      fireEvent.mouseDown(getButton(container));
      expect(onMouseDown).toHaveBeenCalledTimes(1);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should expose the delay as seconds via the CSS custom property", () => {
      const { container } = render(<Toggle.Button onClickDelay={1500} />);
      const btn = getButton(container);
      expect(btn.style.getPropertyValue("--pluto-toggle-delay")).toBe("1.5s");
    });

    it("should fire onClick exactly once even on repeated mousedowns within the same press", () => {
      const onClick = vi.fn();
      const { container } = render(
        <Toggle.Button onClick={onClick} onClickDelay={500} />,
      );
      fireEvent.mouseDown(getButton(container));
      vi.advanceTimersByTime(600);
      expect(onClick).toHaveBeenCalledTimes(1);
      // A trailing mouseup after the timer fires should not produce another call.
      fireEvent.mouseUp(document);
      vi.advanceTimersByTime(1000);
      expect(onClick).toHaveBeenCalledTimes(1);
    });
  });

  describe("hold fill", () => {
    const renderWithSVG = (delay: number): HTMLElement =>
      render(
        <Toggle.Button onClickDelay={delay}>
          <Primitive.SVG dimensions={{ width: 10, height: 10 }}>
            <rect />
          </Primitive.SVG>
        </Toggle.Button>,
      ).container;

    it("should host the masked fill inside the SVG when delayed", () => {
      expect(renderWithSVG(500).querySelector("svg rect[mask]")).not.toBeNull();
    });

    it("should host no fill without a delay", () => {
      expect(renderWithSVG(0).querySelector("rect[mask]")).toBeNull();
    });
  });

  describe("className", () => {
    it("should preserve a user-supplied className", () => {
      const { container } = render(<Toggle.Button className="custom-cls" />);
      expect(container.querySelector("button.custom-cls")).not.toBeNull();
    });
  });
});
