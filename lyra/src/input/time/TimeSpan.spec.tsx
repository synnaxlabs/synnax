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
import { loadLanguage } from "@/input/time/suggest";
import { mockBoundingClientRect } from "@/testutil/dom";
import { Triggers } from "@/triggers";

const HOUR = Number(TimeSpan.HOUR.valueOf());

const open = (container: HTMLElement): HTMLInputElement => {
  const trigger = container.querySelector(".lyra-time-editor__trigger");
  if (trigger == null) throw new Error("no time editor trigger");
  fireEvent.click(trigger);
  return screen.getByRole<HTMLInputElement>("textbox");
};

describe("Input.TimeSpan", () => {
  beforeAll(loadLanguage);
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

  describe("precision", () => {
    const field = (): HTMLInputElement => screen.getByRole<HTMLInputElement>("textbox");
    const CYCLE = 62_500;

    it("should show and commit a value to the nanosecond", () => {
      const onChange = vi.fn();
      render(
        <Input.TimeSpan
          inline
          precision={TimeSpan.NANOSECOND}
          value={CYCLE}
          onChange={onChange}
        />,
      );
      expect(field().value).toBe("62μs 500ns");
      fireEvent.focus(field());
      fireEvent.change(field(), { target: { value: "62.6us" } });
      fireEvent.blur(field());
      expect(onChange).toHaveBeenCalledExactlyOnceWith(62_600);
    });

    it("should round a value to the microsecond by default", () => {
      const onChange = vi.fn();
      render(<Input.TimeSpan inline value={CYCLE} onChange={onChange} />);
      expect(field().value).toBe("63μs");
      fireEvent.focus(field());
      fireEvent.change(field(), { target: { value: "62.6us" } });
      fireEvent.blur(field());
      expect(onChange).not.toHaveBeenCalled();
    });
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

  describe("inline", () => {
    const field = (): HTMLInputElement => screen.getByRole<HTMLInputElement>("textbox");

    it("should show the hint under the readings of the value", () => {
      render(<Input.TimeSpan inline value={HOUR} onChange={vi.fn()} />);
      fireEvent.focus(field());
      expect(screen.getByText("1h")).toBeTruthy();
      expect(
        screen.getByText("Try 2h 30min, 1:30:00, 90, or an hour and a half"),
      ).toBeTruthy();
    });

    it("should read a bare number in its unit and commit on blur", () => {
      const onChange = vi.fn();
      render(
        <Input.TimeSpan
          inline
          unit={TimeSpan.MILLISECOND}
          value={HOUR}
          onChange={onChange}
        />,
      );
      fireEvent.focus(field());
      fireEvent.change(field(), { target: { value: "1000" } });
      expect(screen.getByText("milliseconds")).toBeTruthy();
      fireEvent.blur(field());
      expect(onChange).toHaveBeenCalledWith(Number(TimeSpan.SECOND.valueOf()));
    });

    it("should discard the edit on Escape", () => {
      const onChange = vi.fn();
      render(
        <Triggers.Provider>
          <Input.TimeSpan inline value={HOUR} onChange={onChange} />
        </Triggers.Provider>,
      );
      // A real focus, so the blur that Escape causes reaches the editor.
      act(() => field().focus());
      fireEvent.change(field(), { target: { value: "30s" } });
      fireEvent.keyDown(field(), { key: "Escape", code: "Escape" });
      expect(field().value).toBe("1h");
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should keep the committed span when a click outside also blurs the field", () => {
      const onChange = vi.fn();
      const onClear = vi.fn();
      render(
        <Input.TimeSpan inline value={HOUR} onChange={onChange} onClear={onClear} />,
      );
      fireEvent.focus(field());
      fireEvent.change(field(), { target: { value: "5min" } });
      fireEvent.pointerDown(document.body);
      fireEvent.blur(field());
      expect(onChange).toHaveBeenCalledExactlyOnceWith(
        Number(TimeSpan.minutes(5).valueOf()),
      );
      expect(onClear).not.toHaveBeenCalled();
    });

    it("should show the span as text in a preview", () => {
      render(<Input.TimeSpan inline preview value={HOUR} onChange={vi.fn()} />);
      expect(screen.queryByRole("textbox")).toBeNull();
      expect(screen.getByText("1h")).toBeTruthy();
    });

    it("should name the field by its aria-label", () => {
      render(
        <Input.TimeSpan
          inline
          value={HOUR}
          onChange={vi.fn()}
          aria-label="SYNC0 cycle"
        />,
      );
      expect(screen.getByRole("textbox", { name: "SYNC0 cycle" })).toBeTruthy();
    });

    it("should show the end content inside the field", () => {
      const { container } = render(
        <Input.TimeSpan
          inline
          value={undefined}
          onChange={vi.fn()}
          endContent={<Input.AutoBadge />}
        />,
      );
      const field = container.querySelector(".lyra-time-editor__field");
      expect(field?.contains(screen.getByText("Auto"))).toBe(true);
    });

    it("should show a button after the field that a click reaches", () => {
      const onClick = vi.fn();
      render(
        <Input.TimeSpan
          inline
          value={HOUR}
          onChange={vi.fn()}
          after={<Input.AutoButton onClick={onClick} />}
        />,
      );
      fireEvent.click(screen.getByRole("button"));
      expect(onClick).toHaveBeenCalledOnce();
    });

    it("should show why a span fails its check and not commit it", () => {
      const onChange = vi.fn();
      const check = (span: TimeSpan) =>
        span.greaterThan(TimeSpan.MINUTE) ? "Timeout must be 1min or less" : null;
      render(<Input.TimeSpan inline value={HOUR} onChange={onChange} check={check} />);
      fireEvent.focus(field());
      fireEvent.change(field(), { target: { value: "5min" } });
      fireEvent.keyDown(field(), { key: "Enter" });
      expect(screen.getByText("Timeout must be 1min or less")).toBeTruthy();
      expect(onChange).not.toHaveBeenCalled();
    });
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
