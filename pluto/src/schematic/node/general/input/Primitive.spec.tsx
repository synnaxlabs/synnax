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

import { Input } from "@/schematic/node/general/input/Primitive";

describe("input symbol", () => {
  describe("activation delay", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("should send on a plain click without a delay", () => {
      const onSend = vi.fn();
      const { getByText } = render(<Input value="hi" onSend={onSend} />);
      fireEvent.click(getByText("Set"));
      expect(onSend).toHaveBeenCalledWith("hi");
    });

    it("should swallow a click shorter than the delay", () => {
      const onSend = vi.fn();
      const { getByText } = render(
        <Input value="hi" onSend={onSend} onClickDelay={500} />,
      );
      const btn = getByText("Set");
      fireEvent.mouseDown(btn);
      fireEvent.mouseUp(document);
      fireEvent.click(btn);
      vi.advanceTimersByTime(1000);
      expect(onSend).not.toHaveBeenCalled();
    });

    it("should send after the delay while the button stays held", () => {
      const onSend = vi.fn();
      const { getByText } = render(
        <Input value="hi" onSend={onSend} onClickDelay={500} />,
      );
      fireEvent.mouseDown(getByText("Set"));
      act(() => {
        vi.advanceTimersByTime(500);
      });
      expect(onSend).toHaveBeenCalledWith("hi");
    });
  });

  describe("draft", () => {
    const isDirty = (container: HTMLElement): boolean =>
      container
        .querySelector(".pluto-input-symbol")
        ?.classList.contains("pluto--dirty") ?? false;

    it("should mark the field dirty while an edit differs from the target", () => {
      const { container } = render(<Input value="hi" onSend={vi.fn()} />);
      const input = container.querySelector("input");
      if (input == null) throw new Error("expected an input");
      fireEvent.change(input, { target: { value: "bye" } });
      expect(isDirty(container)).toBe(true);
      fireEvent.change(input, { target: { value: "hi" } });
      expect(isDirty(container)).toBe(false);
    });

    const sendBye = (onSend: (value: string) => void) => {
      const r = render(<Input value="hi" onSend={onSend} />);
      const input = r.container.querySelector("input");
      if (input == null) throw new Error("expected an input");
      fireEvent.change(input, { target: { value: "bye" } });
      fireEvent.click(r.getByText("Set"));
      return { ...r, input };
    };

    it("should keep the sent value until the target reports it", () => {
      const onSend = vi.fn();
      const { container, input, rerender } = sendBye(onSend);
      expect(onSend).toHaveBeenCalledExactlyOnceWith("bye");
      expect(input.value).toBe("bye");
      expect(isDirty(container)).toBe(true);
      rerender(<Input value="bye" onSend={onSend} />);
      expect(input.value).toBe("bye");
      expect(isDirty(container)).toBe(false);
    });

    it("should send the sent value again on a second Set during the wait", () => {
      const onSend = vi.fn();
      const { getByText } = sendBye(onSend);
      fireEvent.click(getByText("Set"));
      expect(onSend.mock.calls).toEqual([["bye"], ["bye"]]);
    });

    it("should settle an empty value sent over an empty target", () => {
      const onSend = vi.fn();
      const { container, getByText } = render(<Input value={null} onSend={onSend} />);
      const input = container.querySelector("input");
      if (input == null) throw new Error("expected an input");
      fireEvent.change(input, { target: { value: "x" } });
      fireEvent.change(input, { target: { value: "" } });
      fireEvent.click(getByText("Set"));
      expect(onSend).toHaveBeenCalledExactlyOnceWith("");
      expect(isDirty(container)).toBe(false);
    });

    it("should follow the target once it reports another value", () => {
      const onSend = vi.fn();
      const { container, input, rerender } = sendBye(onSend);
      rerender(<Input value="other" onSend={onSend} />);
      expect(input.value).toBe("other");
      expect(isDirty(container)).toBe(false);
    });
  });
});
