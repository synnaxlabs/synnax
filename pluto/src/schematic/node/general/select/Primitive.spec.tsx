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
        <Select
          value="a"
          options={OPTIONS}
          onChange={() => {}}
          onSend={onSend}
          onClickDelay={delay}
        />,
      );

    it("should send on a plain click without a delay", () => {
      const onSend = vi.fn();
      const { getByText } = renderSelect(onSend);
      fireEvent.click(getByText("Send"));
      expect(onSend).toHaveBeenCalledWith(7);
    });

    it("should swallow a click shorter than the delay", () => {
      const onSend = vi.fn();
      const { getByText } = renderSelect(onSend, 500);
      const btn = getByText("Send");
      fireEvent.mouseDown(btn);
      fireEvent.mouseUp(document);
      fireEvent.click(btn);
      vi.advanceTimersByTime(1000);
      expect(onSend).not.toHaveBeenCalled();
    });

    it("should send after the delay while the button stays held", () => {
      const onSend = vi.fn();
      const { getByText } = renderSelect(onSend, 500);
      fireEvent.mouseDown(getByText("Send"));
      vi.advanceTimersByTime(500);
      expect(onSend).toHaveBeenCalledWith(7);
    });
  });

  describe("keyboard activation", () => {
    it.each([" ", "Enter"])(
      "should prevent the default keydown and keyup for %j",
      (key) => {
        const { getByText } = render(
          <Select value="a" options={OPTIONS} onChange={() => {}} onSend={vi.fn()} />,
        );
        const target = getByText("Send");
        expect(fireEvent.keyDown(target, { key })).toBe(false);
        expect(fireEvent.keyUp(target, { key })).toBe(false);
      },
    );
  });
});
