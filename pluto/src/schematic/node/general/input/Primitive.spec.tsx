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

    it("should send the draft and clear the dirty mark on Set", () => {
      const onSend = vi.fn();
      const { container, getByText } = render(<Input value="hi" onSend={onSend} />);
      const input = container.querySelector("input");
      if (input == null) throw new Error("expected an input");
      fireEvent.change(input, { target: { value: "bye" } });
      fireEvent.click(getByText("Set"));
      expect(onSend).toHaveBeenCalledExactlyOnceWith("bye");
      expect(isDirty(container)).toBe(false);
    });
  });
});
