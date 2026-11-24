// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
import { Input } from "@/input";
import { renderThemed } from "@/styletest";
import { preloadTimeLanguage } from "@/testutil";
import { Triggers } from "@/triggers";

const HINT = "Try 2h 30min, 1:30:00, 90, or an hour and a half";

describe("Input.TimeSpan", () => {
  // Only a real key press lets React hide the editor before the modal sees the Escape.
  it("should close only its suggestions in a modal on one Escape", async () => {
    await renderThemed(
      <Triggers.Provider>
        <Dialog.Frame variant="modal">
          <Dialog.Trigger>Open</Dialog.Trigger>
          <Dialog.Dialog>
            <p>Modal content</p>
            <Input.TimeSpan
              inline
              value={Number(TimeSpan.HOUR.valueOf())}
              onChange={vi.fn()}
            />
          </Dialog.Dialog>
        </Dialog.Frame>
      </Triggers.Provider>,
    );
    await userEvent.click(page.getByText("Open"));
    await userEvent.click(page.getByRole("textbox"));
    await expect.element(page.getByText(HINT)).toBeVisible();
    await userEvent.keyboard("{Escape}");
    await expect.element(page.getByText(HINT)).not.toBeInTheDocument();
    await expect.element(page.getByText("Modal content")).toBeVisible();
    await userEvent.keyboard("{Escape}");
    await expect.element(page.getByText("Modal content")).not.toBeInTheDocument();
  });
});

describe("Input.TimeSpan", () => {
  const HOUR = Number(TimeSpan.HOUR.valueOf());

  const open = (container: HTMLElement): HTMLInputElement => {
    const trigger = container.querySelector(".lyra-time-editor__trigger");
    if (trigger == null) throw new Error("no time editor trigger");
    fireEvent.click(trigger);
    return screen.getByRole<HTMLInputElement>("textbox");
  };

  preloadTimeLanguage();

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

  describe("readings", () => {
    const ns = (span: TimeSpan): number => Number(span.valueOf());

    // Types text into a fresh input and returns its readings and what Enter commits.
    const edit = (
      text: string,
      unit?: TimeSpan,
    ): { values: string[]; readings: string[]; committed?: number } => {
      const onChange = vi.fn();
      const { container } = render(
        <Input.TimeSpan value={undefined} onChange={onChange} unit={unit} />,
      );
      const field = open(container);
      fireEvent.change(field, { target: { value: text } });
      const items = screen
        .queryAllByRole("menuitem")
        .filter((item) => item.classList.contains("lyra-time-editor__suggestion"));
      const values = items.map((item) => item.firstElementChild?.textContent ?? "");
      const readings = items.map((item) => item.lastElementChild?.textContent ?? "");
      fireEvent.keyDown(field, { key: "Enter" });
      const committed = onChange.mock.calls[0]?.[0] as number | undefined;
      cleanup();
      return { values, readings, committed };
    };

    it("should parse unit runs", () => {
      const runs: [string, TimeSpan][] = [
        ["30s", TimeSpan.seconds(30)],
        ["2h 30m", TimeSpan.minutes(150)],
        ["2h30m", TimeSpan.minutes(150)],
        ["1.5h", TimeSpan.minutes(90)],
        ["500ms", TimeSpan.milliseconds(500)],
        ["250us", TimeSpan.microseconds(250)],
        ["250µs", TimeSpan.microseconds(250)],
        ["10ns", TimeSpan.nanoseconds(10)],
        ["2d", TimeSpan.days(2)],
      ];
      for (const [text, span] of runs)
        expect(edit(text).committed, text).toBe(ns(span));
    });

    it("should parse clock form", () => {
      expect(edit("1:30:00").committed).toBe(ns(TimeSpan.minutes(90)));
      expect(edit("0:30").committed).toBe(ns(TimeSpan.minutes(30)));
      expect(edit("00:00:30.250").committed).toBe(ns(TimeSpan.milliseconds(30250)));
    });

    it("should treat a bare number as seconds", () => {
      expect(edit("45").committed).toBe(ns(TimeSpan.seconds(45)));
      expect(edit("0.5").committed).toBe(ns(TimeSpan.milliseconds(500)));
    });

    it("should read a bare number in the given unit", () => {
      expect(edit("250", TimeSpan.MILLISECOND).committed).toBe(
        ns(TimeSpan.milliseconds(250)),
      );
      expect(edit("2s", TimeSpan.MILLISECOND).committed).toBe(ns(TimeSpan.seconds(2)));
    });

    it("should name the unit of a bare number as its only reading", () => {
      const { readings, committed } = edit("1000", TimeSpan.MILLISECOND);
      expect(readings).toEqual(["milliseconds"]);
      expect(committed).toBe(ns(TimeSpan.SECOND));
    });

    it("should lead with the grammar's reading", () => {
      expect(edit("2h 30m").readings[0]).toBe("duration");
      expect(edit("1:30:00").readings[0]).toBe("duration");
      expect(edit("45").readings[0]).toBe("seconds");
    });

    it("should reject text that is not a duration", () => {
      expect(edit("").committed).toBeUndefined();
      for (const text of ["soon", "5 parsecs"]) {
        const { readings, committed } = edit(text);
        expect(readings, text).toEqual([]);
        expect(committed, text).toBeUndefined();
      }
    });

    it("should throw on a unit it cannot name", () => {
      const { container } = render(
        <Input.TimeSpan
          value={undefined}
          onChange={vi.fn()}
          unit={TimeSpan.seconds(2)}
        />,
      );
      const field = open(container);
      expect(() => fireEvent.change(field, { target: { value: "5" } })).toThrow(
        "2s is not a unit of time",
      );
    });

    it("should read Ms only as megaseconds", () => {
      const { values, committed } = edit("5Ms");
      expect(values.length).toBeGreaterThan(0);
      const label = TimeSpan.seconds(5e6).toString("full");
      for (const value of values) expect(value).toBe(label);
      expect(committed).toBe(ns(TimeSpan.seconds(5e6)));
    });

    it("should read natural language durations", () => {
      const phrases: [string, TimeSpan][] = [
        ["an hour and a half", TimeSpan.minutes(90)],
        ["half an hour", TimeSpan.minutes(30)],
        ["2 hours and 30 minutes", TimeSpan.minutes(150)],
        ["five minutes", TimeSpan.minutes(5)],
      ];
      for (const [text, span] of phrases)
        expect(edit(text).committed, text).toBe(ns(span));
    });
  });

  it("should round its value to the nearest step on either side of zero", () => {
    const rounded: [number, TimeSpan | undefined, string][] = [
      [1600, undefined, "2μs"],
      [-1000, undefined, "-1μs"],
      [-400, undefined, "0s"],
      [-1600, undefined, "-2μs"],
      [62_500, TimeSpan.NANOSECOND, "62μs 500ns"],
      [-62_500, TimeSpan.NANOSECOND, "-62μs -500ns"],
      [15_000, TimeSpan.microseconds(10), "20μs"],
      [-14_000, TimeSpan.microseconds(10), "-10μs"],
    ];
    for (const [value, precision, label] of rounded) {
      const { container } = render(
        <Input.TimeSpan value={value} onChange={vi.fn()} precision={precision} />,
      );
      const shown = container.querySelector(".lyra-timespan__label")?.textContent;
      expect(shown, String(value)).toBe(label);
      cleanup();
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
