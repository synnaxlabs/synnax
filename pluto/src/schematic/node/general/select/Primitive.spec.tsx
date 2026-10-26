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

import { Select } from "@/schematic/node/general/select/Primitive";

const OPTIONS = [{ key: "a", name: "A", value: 7 }];

describe("select symbol", () => {
  describe("activation delay", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    const renderSelect = (onSend: (v: number) => void, delay?: number) =>
      render(
        <Select value="a" options={OPTIONS} onSend={onSend} onClickDelay={delay} />,
      );

    it("should send on a plain click without a delay", () => {
      const onSend = vi.fn();
      const { getByText } = renderSelect(onSend);
      fireEvent.click(getByText("Set"));
      expect(onSend).toHaveBeenCalledWith(7);
    });

    it("should swallow a click shorter than the delay", () => {
      const onSend = vi.fn();
      const { getByText } = renderSelect(onSend, 500);
      const btn = getByText("Set");
      fireEvent.mouseDown(btn);
      fireEvent.mouseUp(document);
      fireEvent.click(btn);
      vi.advanceTimersByTime(1000);
      expect(onSend).not.toHaveBeenCalled();
    });

    it("should send after the delay while the button stays held", () => {
      const onSend = vi.fn();
      const { getByText } = renderSelect(onSend, 500);
      fireEvent.mouseDown(getByText("Set"));
      act(() => {
        vi.advanceTimersByTime(500);
      });
      expect(onSend).toHaveBeenCalledWith(7);
    });
  });

  describe("draft", () => {
    const MORE_OPTIONS = [
      ...OPTIONS,
      { key: "b", name: "B", value: 8 },
      { key: "c", name: "C", value: 9 },
    ];

    const isDirty = (container: HTMLElement): boolean =>
      container
        .querySelector(".pluto-select-symbol")
        ?.classList.contains("pluto--dirty") ?? false;

    const sendB = (onSend: (value: number) => void) => {
      const r = render(<Select value="a" options={MORE_OPTIONS} onSend={onSend} />);
      fireEvent.click(r.getByText("A"));
      fireEvent.click(r.getByText("B"));
      fireEvent.click(r.getByText("Set"));
      return r;
    };

    it("should keep the sent option until the target reports it", () => {
      const onSend = vi.fn();
      const { container, queryByText, rerender } = sendB(onSend);
      expect(onSend).toHaveBeenCalledExactlyOnceWith(8);
      expect(queryByText("A")).toBeNull();
      expect(isDirty(container)).toBe(true);
      rerender(<Select value="b" options={MORE_OPTIONS} onSend={onSend} />);
      expect(isDirty(container)).toBe(false);
    });

    it("should follow the target once it reports another option", () => {
      const onSend = vi.fn();
      const { container, queryByText, rerender } = sendB(onSend);
      rerender(<Select value="c" options={MORE_OPTIONS} onSend={onSend} />);
      expect(queryByText("C")).not.toBeNull();
      expect(isDirty(container)).toBe(false);
    });
  });
});
