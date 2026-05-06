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

  describe("classes and modifiers", () => {
    it("should add the delayed modifier when delay is non-zero", () => {
      const { container } = render(<Toggle.Button onClickDelay={250} />);
      expect(getButton(container).className).toContain(
        "pluto-symbol-primitive-toggle--delayed",
      );
    });

    it("should not add the delayed modifier when delay is zero", () => {
      const { container } = render(<Toggle.Button />);
      expect(getButton(container).className).not.toContain(
        "pluto-symbol-primitive-toggle--delayed",
      );
    });

    it("should reflect the enabled flag via the enabled modifier", () => {
      const enabled = render(<Toggle.Button enabled />);
      expect(getButton(enabled.container).className).toContain("pluto--enabled");
      const disabled = render(<Toggle.Button enabled={false} />);
      expect(getButton(disabled.container).className).not.toContain("pluto--enabled");
    });

    it("should reflect the triggered flag via the triggered modifier", () => {
      const { container } = render(<Toggle.Button triggered />);
      expect(getButton(container).className).toContain("pluto--triggered");
    });

    it("should encode the orientation as a location class", () => {
      const { container } = render(<Toggle.Button orientation="top" />);
      expect(getButton(container).className).toContain("pluto--location-top");
    });

    it("should preserve user-supplied className", () => {
      const { container } = render(<Toggle.Button className="custom-cls" />);
      expect(getButton(container).className).toContain("custom-cls");
    });
  });
});
