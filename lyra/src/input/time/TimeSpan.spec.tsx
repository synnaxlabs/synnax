// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Input } from "@/input";
import { preloadTimeLanguage } from "@/input/testutil";
import { mockBoundingClientRect } from "@/testutil";

const HOUR = Number(TimeSpan.HOUR.valueOf());

const open = (container: HTMLElement): HTMLInputElement => {
  const trigger = container.querySelector(".lyra-time-editor__trigger");
  if (trigger == null) throw new Error("no time editor trigger");
  fireEvent.click(trigger);
  return screen.getByRole<HTMLInputElement>("textbox");
};

describe("Input.TimeSpan", () => {
  preloadTimeLanguage();
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  it("should not commit an unedited value finer than a microsecond", () => {
    const onChange = vi.fn();
    const { container } = render(
      <Input.TimeSpan value={2 * HOUR + 512} onChange={onChange} />,
    );
    fireEvent.keyDown(open(container), { key: "Enter" });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("should advance an elapsed readout every second", () => {
    // A float64 holds the nanoseconds of a whole second exactly.
    vi.useFakeTimers({ now: new Date(2026, 0, 1, 12) });
    try {
      const since = Number(TimeStamp.now().sub(TimeSpan.seconds(5)).valueOf());
      const { container } = render(
        <Input.TimeSpan value={0} onChange={vi.fn()} elapsedSince={since} />,
      );
      const label = () => container.querySelector(".lyra-timespan__label")?.textContent;
      expect(label()).toBe("5s");
      act(() => {
        vi.advanceTimersByTime(TimeSpan.seconds(3).milliseconds);
      });
      expect(label()).toBe("8s");
    } finally {
      vi.useRealTimers();
    }
  });

  it("should stay open on Enter when the text is not a duration", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.TimeSpan value={HOUR} onChange={onChange} />);
    const field = open(container);
    fireEvent.change(field, { target: { value: "5 parsecs" } });
    fireEvent.keyDown(field, { key: "Enter" });
    expect(screen.getByRole("textbox")).toBe(field);
    expect(onChange).not.toHaveBeenCalled();
  });
});
