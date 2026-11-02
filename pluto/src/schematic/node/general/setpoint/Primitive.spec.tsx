// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Setpoint } from "@/schematic/node/general/setpoint/Primitive";

describe("setpoint symbol", () => {
  describe("activation delay", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("should set the target on a plain click without a delay", () => {
      const onSend = vi.fn();
      const { getByText } = render(<Setpoint onSend={onSend} value={3} />);
      fireEvent.click(getByText("Set"));
      expect(onSend).toHaveBeenCalledWith(3);
    });

    it("should show a blank field and send nothing when no target is known", () => {
      const onSend = vi.fn();
      const { container, getByText } = render(<Setpoint onSend={onSend} />);
      expect(container.querySelector("input")?.value).toBe("");
      fireEvent.click(getByText("Set"));
      expect(onSend).not.toHaveBeenCalled();
    });

    it("should swallow a click shorter than the delay", () => {
      const onSend = vi.fn();
      const { getByText } = render(
        <Setpoint onSend={onSend} value={3} onClickDelay={500} />,
      );
      const btn = getByText("Set");
      fireEvent.mouseDown(btn);
      fireEvent.mouseUp(document);
      fireEvent.click(btn);
      vi.advanceTimersByTime(1000);
      expect(onSend).not.toHaveBeenCalled();
    });

    it("should send the value typed before a plain click", () => {
      const onSend = vi.fn();
      const { container, getByText } = render(<Setpoint onSend={onSend} />);
      const input = container.querySelector("input");
      if (input == null) throw new Error("expected an input");
      input.focus();
      fireEvent.change(input, { target: { value: "5" } });
      const btn = getByText("Set");
      fireEvent.mouseDown(btn);
      fireEvent.click(btn);
      expect(onSend).toHaveBeenCalledExactlyOnceWith(5);
    });

    it("should send the value typed before the hold", () => {
      const onSend = vi.fn();
      const { container, getByText } = render(
        <Setpoint onSend={onSend} onClickDelay={500} />,
      );
      const input = container.querySelector("input");
      if (input == null) throw new Error("expected an input");
      input.focus();
      fireEvent.change(input, { target: { value: "5" } });
      fireEvent.mouseDown(getByText("Set"));
      act(() => {
        vi.advanceTimersByTime(500);
      });
      expect(onSend).toHaveBeenCalledExactlyOnceWith(5);
    });

    it("should set after the delay while the button stays held", () => {
      const onSend = vi.fn();
      const { getByText } = render(
        <Setpoint onSend={onSend} value={3} onClickDelay={500} />,
      );
      fireEvent.mouseDown(getByText("Set"));
      act(() => {
        vi.advanceTimersByTime(500);
      });
      expect(onSend).toHaveBeenCalledWith(3);
    });
  });

  describe("draft", () => {
    const isDirty = (container: HTMLElement): boolean =>
      container.querySelector(".pluto-setpoint")?.classList.contains("pluto--dirty") ??
      false;

    it("should mark the field dirty on the first keystroke", () => {
      const { container } = render(<Setpoint onSend={vi.fn()} value={3} />);
      const input = container.querySelector("input");
      if (input == null) throw new Error("expected an input");
      input.focus();
      fireEvent.input(input, { target: { value: "5" } });
      expect(isDirty(container)).toBe(true);
    });

    it("should clear the dirty mark when an invalid entry is abandoned", async () => {
      const { container } = render(<Setpoint onSend={vi.fn()} value={3} />);
      const input = container.querySelector("input");
      if (input == null) throw new Error("expected an input");
      input.focus();
      fireEvent.input(input, { target: { value: "abc" } });
      await act(async () => fireEvent.blur(input));
      expect(isDirty(container)).toBe(false);
    });

    it("should stage a typed value on Enter without sending it", () => {
      const onSend = vi.fn();
      const { container } = render(<Setpoint onSend={onSend} value={3} />);
      const input = container.querySelector("input");
      if (input == null) throw new Error("expected an input");
      input.focus();
      fireEvent.change(input, { target: { value: "5" } });
      fireEvent.keyDown(input, { key: "Enter", code: "Enter" });
      expect(onSend).not.toHaveBeenCalled();
      expect(input.value).toBe("5");
      expect(isDirty(container)).toBe(true);
    });

    const sendFive = (onSend: (value: number) => void) => {
      const r = render(<Setpoint onSend={onSend} value={3} />);
      const input = r.container.querySelector("input");
      if (input == null) throw new Error("expected an input");
      input.focus();
      fireEvent.change(input, { target: { value: "5" } });
      const btn = r.getByText("Set");
      fireEvent.mouseDown(btn);
      fireEvent.click(btn);
      return { ...r, input, btn };
    };

    it("should keep the sent value until the target reports it", () => {
      const onSend = vi.fn();
      const { container, input, rerender } = sendFive(onSend);
      expect(onSend).toHaveBeenCalledExactlyOnceWith(5);
      expect(input.value).toBe("5");
      expect(isDirty(container)).toBe(true);
      rerender(<Setpoint onSend={onSend} value={5} />);
      expect(input.value).toBe("5");
      expect(isDirty(container)).toBe(false);
    });

    it("should send the sent value again on a second Set during the wait", () => {
      const onSend = vi.fn();
      const { btn } = sendFive(onSend);
      fireEvent.mouseDown(btn);
      fireEvent.click(btn);
      expect(onSend.mock.calls).toEqual([[5], [5]]);
    });

    it("should follow the target once it reports another value", () => {
      const onSend = vi.fn();
      const { container, input, rerender } = sendFive(onSend);
      rerender(<Setpoint onSend={onSend} value={4} />);
      expect(input.value).toBe("4");
      expect(isDirty(container)).toBe(false);
    });
  });

  describe("units", () => {
    it("should show the units at the end of the field", () => {
      const { container } = render(<Setpoint onSend={vi.fn()} units="psi" />);
      expect(container.querySelector(".lyra-input__end-content")?.textContent).toBe(
        "psi",
      );
    });

    it("should show no units when the units are empty", () => {
      const { container } = render(<Setpoint onSend={vi.fn()} units="" />);
      expect(container.querySelector(".lyra-input__end-content")).toBeNull();
    });
  });
});
