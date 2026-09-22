// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Switch } from "@/schematic/node/general/switch/Primitive";

const getInput = (container: HTMLElement): HTMLInputElement => {
  const el = container.querySelector<HTMLInputElement>("input");
  if (el == null) throw new Error("expected a switch input");
  return el;
};

describe("switch symbol", () => {
  describe("enabled", () => {
    it("should reflect the enabled state on the input", () => {
      const { container } = render(<Switch enabled />);
      expect(getInput(container).checked).toBe(true);
    });

    it("should default to off", () => {
      const { container } = render(<Switch />);
      expect(getInput(container).checked).toBe(false);
    });

    it("should call onClick when clicked", () => {
      const onClick = vi.fn();
      const { container } = render(<Switch onClick={onClick} />);
      fireEvent.click(getInput(container));
      expect(onClick).toHaveBeenCalledOnce();
    });
  });

  describe("activation delay", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("should swallow a plain click when a delay is set", () => {
      const onClick = vi.fn();
      const { container } = render(
        <Switch onClick={onClick} onClickDelay={TimeSpan.milliseconds(500)} />,
      );
      fireEvent.click(getInput(container));
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should actuate after the delay while the switch stays held", () => {
      const onClick = vi.fn();
      const { container } = render(
        <Switch onClick={onClick} onClickDelay={TimeSpan.milliseconds(500)} />,
      );
      fireEvent.mouseDown(getInput(container));
      vi.advanceTimersByTime(499);
      expect(onClick).not.toHaveBeenCalled();
      vi.advanceTimersByTime(1);
      expect(onClick).toHaveBeenCalledOnce();
    });

    it("should cancel the hold on an early release", () => {
      const onClick = vi.fn();
      const { container } = render(
        <Switch onClick={onClick} onClickDelay={TimeSpan.milliseconds(500)} />,
      );
      fireEvent.mouseDown(getInput(container));
      fireEvent.mouseUp(document);
      vi.advanceTimersByTime(1000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should not actuate after the switch is disabled mid-hold", () => {
      const onClick = vi.fn();
      const c = render(
        <Switch onClick={onClick} onClickDelay={TimeSpan.milliseconds(500)} />,
      );
      fireEvent.mouseDown(getInput(c.container));
      c.rerender(
        <Switch onClick={onClick} onClickDelay={TimeSpan.milliseconds(500)} disabled />,
      );
      vi.advanceTimersByTime(1000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should ignore a secondary-button hold", () => {
      const onClick = vi.fn();
      const { container } = render(
        <Switch onClick={onClick} onClickDelay={TimeSpan.milliseconds(500)} />,
      );
      fireEvent.mouseDown(getInput(container), { button: 2 });
      vi.advanceTimersByTime(1000);
      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe("disabled", () => {
    it("should disable the input", () => {
      const { container } = render(<Switch disabled />);
      expect(getInput(container).disabled).toBe(true);
    });

    it("should leave the input enabled by default", () => {
      const { container } = render(<Switch />);
      expect(getInput(container).disabled).toBe(false);
    });
  });

  describe("keyboard activation", () => {
    it.each([" ", "Enter"])(
      "should prevent the default keydown and keyup for %j",
      (key) => {
        const { container } = render(<Switch />);
        const target = getInput(container);
        expect(fireEvent.keyDown(target, { key })).toBe(false);
        expect(fireEvent.keyUp(target, { key })).toBe(false);
      },
    );
  });
});
