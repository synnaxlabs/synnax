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

describe("Input.TimeSpan readings", () => {
  preloadTimeLanguage();
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  // Types text into a fresh input and returns its readings and what Enter commits.
  const edit = (text: string): { readings: string[]; committed?: number } => {
    const onChange = vi.fn();
    const { container } = render(
      <Input.TimeSpan value={HOUR / 7} onChange={onChange} />,
    );
    const field = open(container);
    fireEvent.change(field, { target: { value: text } });
    const readings = screen
      .queryAllByRole("menuitem")
      .filter((item) => item.classList.contains("lyra-time-editor__suggestion"))
      .map((item) => item.textContent ?? "");
    fireEvent.keyDown(field, { key: "Enter" });
    const committed = onChange.mock.calls[0]?.[0] as number | undefined;
    cleanup();
    return { readings, committed };
  };

  const ns = (span: TimeSpan): number => Number(span.valueOf());

  it("should parse unit runs", () => {
    const runs: [string, TimeSpan][] = [
      ["30s", TimeSpan.seconds(30)],
      ["2h 30m", TimeSpan.minutes(150)],
      ["2h30m", TimeSpan.minutes(150)],
      ["1.5h", TimeSpan.minutes(90)],
      ["500ms", TimeSpan.milliseconds(500)],
      ["250us", TimeSpan.microseconds(250)],
      ["250µs", TimeSpan.microseconds(250)],
      ["2d", TimeSpan.days(2)],
    ];
    for (const [text, span] of runs) expect(edit(text).committed, text).toBe(ns(span));
    // A commit rounds to the microsecond, so the label shows a nanosecond run.
    expect(edit("10ns").readings[0]).toContain("10ns");
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

  it("should say which form the duration took", () => {
    expect(edit("45").readings[0]).toContain("seconds");
    expect(edit("2h 30m").readings[0]).toContain("duration");
    expect(edit("1:30:00").readings[0]).toContain("duration");
  });

  it("should reject text that is not a duration", () => {
    expect(edit("soon").committed).toBeUndefined();
    expect(edit("5 parsecs").committed).toBeUndefined();
  });

  it("should read Ms only as megaseconds", () => {
    const { readings, committed } = edit("5Ms");
    expect(readings.length).toBeGreaterThan(0);
    const label = TimeSpan.seconds(5e6).toString("full");
    for (const reading of readings) expect(reading).toContain(label);
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
