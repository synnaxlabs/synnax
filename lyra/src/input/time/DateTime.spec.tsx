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
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { Input } from "@/input";
import { preloadTimeLanguage } from "@/input/testutil";
import { mockBoundingClientRect } from "@/testutil";
import { Triggers } from "@/triggers";

const local = (...args: [number, number, number, number, number]): number =>
  new TimeStamp(new Date(...args)).nanoseconds;

const VALUE = local(2026, 7, 23, 14, 5);
const HOUR = Number(TimeSpan.HOUR.valueOf());

const open = (container: HTMLElement): HTMLInputElement => {
  const trigger = container.querySelector(".lyra-time-editor__trigger");
  if (trigger == null) throw new Error("no time editor trigger");
  fireEvent.click(trigger);
  return screen.getByRole<HTMLInputElement>("textbox");
};

const type = (field: HTMLInputElement, text: string): void => {
  fireEvent.change(field, { target: { value: text } });
};

describe("Input.DateTime", () => {
  preloadTimeLanguage();
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  it("should open the editor on the value in a fixed local layout", () => {
    const { container } = render(<Input.DateTime value={VALUE} onChange={vi.fn()} />);
    expect(open(container).value).toBe("2026-08-23 14:05:00");
  });

  it("should forward unlisted props to the trigger", () => {
    const { container } = render(
      <Input.DateTime value={VALUE} onChange={vi.fn()} aria-label="Start" disabled />,
    );
    const trigger = container.querySelector(".lyra-time-editor__trigger");
    expect(trigger?.getAttribute("aria-label")).toBe("Start");
    expect(trigger?.getAttribute("aria-disabled")).toBe("true");
  });

  it("should commit a typed instant on Enter", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    const field = open(container);
    type(field, "2026-08-23 15:30");
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith(local(2026, 7, 23, 15, 30));
    expect(screen.queryByRole("textbox")).toBeNull();
  });

  it("should discard an edit on Escape", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    const field = open(container);
    type(field, "2026-08-23 15:30");
    fireEvent.keyDown(field, { key: "Escape" });
    expect(onChange).not.toHaveBeenCalled();
    expect(screen.queryByRole("textbox")).toBeNull();
  });

  it("should discard an edit on Escape when triggers are listening", () => {
    const onChange = vi.fn();
    const { container } = render(
      <Triggers.Provider>
        <Input.DateTime value={VALUE} onChange={onChange} />
      </Triggers.Provider>,
    );
    const field = open(container);
    type(field, "2026-08-23 15:30");
    fireEvent.keyDown(field, { key: "Escape", code: "Escape" });
    expect(onChange).not.toHaveBeenCalled();
    expect(screen.queryByRole("textbox")).toBeNull();
  });

  it("should commit an edit when the trigger closes the editor", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    const field = open(container);
    type(field, "2026-08-23 15:30");
    fireEvent.click(container.querySelector(".lyra-time-editor__trigger") as Element);
    expect(onChange).toHaveBeenCalledWith(local(2026, 7, 23, 15, 30));
  });

  it("should not commit an unedited value finer than a microsecond", () => {
    const onChange = vi.fn();
    const { container } = render(
      <Input.DateTime value={VALUE + 512} onChange={onChange} />,
    );
    fireEvent.keyDown(open(container), { key: "Enter" });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("should nudge the unit under the caret with the arrow keys", () => {
    const { container } = render(<Input.DateTime value={VALUE} onChange={vi.fn()} />);
    const field = open(container);
    field.setSelectionRange(12, 12);
    fireEvent.keyDown(field, { key: "ArrowUp" });
    expect(field.value).toBe("2026-08-23 15:05:00");
  });

  it("should walk the readings of a phrase with the arrow keys", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    const field = open(container);
    type(field, "2h");
    fireEvent.keyDown(field, { key: "ArrowDown" });
    const before = TimeStamp.now().nanoseconds;
    fireEvent.keyDown(field, { key: "Enter" });
    const committed = onChange.mock.calls[0][0] as number;
    // The second reading of a bare duration is that long ago.
    expect(Math.abs(before - 2 * HOUR - committed)).toBeLessThan(
      Number(TimeSpan.SECOND.valueOf()),
    );
  });

  it("should read a bare duration from the other end of the range first", () => {
    const onChange = vi.fn();
    const end = local(2026, 7, 23, 18, 0);
    const { container } = render(
      <Input.DateTime
        value={VALUE}
        onChange={onChange}
        bound="start"
        anchors={{ end }}
      />,
    );
    const field = open(container);
    type(field, "2h");
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith(end - 2 * HOUR);
  });

  it("should say when a typed date does not exist", () => {
    const { container } = render(<Input.DateTime value={VALUE} onChange={vi.fn()} />);
    type(open(container), "2026-02-31");
    expect(screen.getByText("Not a time")).toBeTruthy();
  });

  it("should commit an edit made after a reading was clicked", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    type(open(container), "2026-08-23 15:30");
    fireEvent.click(screen.getByText(/^local time/));
    expect(onChange).toHaveBeenCalledTimes(1);
    const field = open(container);
    type(field, "2026-08-23 16:45");
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChange).toHaveBeenLastCalledWith(local(2026, 7, 23, 16, 45));
  });

  it("should commit an action and close", () => {
    const onChange = vi.fn();
    const parent = { start: local(2026, 7, 20, 9, 0), end: local(2026, 7, 21, 9, 0) };
    const { container } = render(
      <Input.DateTime value={VALUE} onChange={onChange} anchors={{ parent }} />,
    );
    open(container);
    fireEvent.click(screen.getByText("Parent start"));
    expect(onChange).toHaveBeenCalledWith(parent.start);
    expect(screen.queryByRole("textbox")).toBeNull();
  });

  it("should give the effect the value of the highlighted reading", () => {
    const { container } = render(
      <Input.DateTime
        value={VALUE}
        onChange={vi.fn()}
        effect={({ candidate }) => <span data-testid="effect">{candidate}</span>}
      />,
    );
    type(open(container), "2026-08-23 15:30");
    expect(Number(screen.getByTestId("effect").textContent)).toBe(
      local(2026, 7, 23, 15, 30),
    );
  });

  it("should give the effect the value of a hovered action", () => {
    const parent = { start: local(2026, 7, 20, 9, 0), end: local(2026, 7, 21, 9, 0) };
    const { container } = render(
      <Input.DateTime
        value={VALUE}
        onChange={vi.fn()}
        anchors={{ parent }}
        effect={({ candidate }) => <span data-testid="effect">{candidate}</span>}
      />,
    );
    open(container);
    fireEvent.mouseEnter(screen.getByRole("menuitem", { name: "Parent end" }));
    expect(Number(screen.getByTestId("effect").textContent)).toBe(parent.end);
  });

  it("should highlight a reading the pointer enters", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    const field = open(container);
    type(field, "2h");
    const readings = screen
      .getAllByRole("menuitem")
      .filter((item) => item.classList.contains("lyra-time-editor__suggestion"));
    fireEvent.mouseEnter(readings[1]);
    expect(readings[1].classList.contains("lyra--selected")).toBe(true);
    const before = TimeStamp.now().nanoseconds;
    fireEvent.keyDown(field, { key: "Enter" });
    const committed = onChange.mock.calls[0][0] as number;
    expect(Math.abs(before - 2 * HOUR - committed)).toBeLessThan(
      Number(TimeSpan.SECOND.valueOf()),
    );
  });

  it("should not show float noise in a reading anchored on a form value", () => {
    // Float64 cannot hold this instant to the nanosecond.
    const end = VALUE + 703_000_000;
    const { container } = render(
      <Input.DateTime
        value={VALUE}
        onChange={vi.fn()}
        bound="start"
        anchors={{ end }}
      />,
    );
    type(open(container), "2h");
    const [first] = screen.getAllByRole("menuitem");
    expect(first.textContent).toContain("12:05:00.703");
    expect(first.textContent).not.toMatch(/\.\d{3} \d{3}/);
  });

  it("should measure a reading's offset from the moment the editor opens", () => {
    vi.useFakeTimers({ toFake: ["Date"] });
    try {
      vi.setSystemTime(new Date(2026, 7, 23, 14, 5));
      const { container } = render(<Input.DateTime value={VALUE} onChange={vi.fn()} />);
      vi.setSystemTime(new Date(2026, 7, 23, 15, 5));
      open(container);
      expect(screen.getByText("local time · 1h ago")).toBeTruthy();
    } finally {
      vi.useRealTimers();
    }
  });

  it("should name the day again once midnight passes", () => {
    vi.useFakeTimers();
    try {
      vi.setSystemTime(new Date(2026, 7, 23, 23, 59, 30));
      const { container } = render(<Input.DateTime value={VALUE} onChange={vi.fn()} />);
      const day = () => container.querySelector(".lyra-datetime__day")?.textContent;
      expect(day()).toBe("Today");
      act(() => {
        vi.advanceTimersByTime(TimeSpan.seconds(30).milliseconds);
      });
      expect(day()).toBe("Yesterday");
    } finally {
      vi.useRealTimers();
    }
  });

  it("should leave out a hint that only repeats its action's label", () => {
    const { container } = render(<Input.DateTime value={VALUE} onChange={vi.fn()} />);
    open(container);
    expect(screen.getByRole("menuitem", { name: "Now" }).textContent).toBe("Now");
  });

  it("should stay open on Enter when the text is not a time", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.DateTime value={VALUE} onChange={onChange} />);
    const field = open(container);
    type(field, "tomorow 3pm-ish");
    fireEvent.keyDown(field, { key: "Enter" });
    expect(screen.getByRole("textbox")).toBe(field);
    expect(onChange).not.toHaveBeenCalled();
  });

  it("should commit the empty value when the field is cleared", () => {
    const onChange = vi.fn();
    const empty = TimeStamp.MAX.nanoseconds;
    const { container } = render(
      <Input.DateTime value={VALUE} onChange={onChange} emptyValue={empty} />,
    );
    const field = open(container);
    type(field, "");
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith(empty);
  });

  it("should show a value later than the empty value", () => {
    render(
      <Input.DateTime
        value={VALUE}
        onChange={vi.fn()}
        emptyValue={local(2000, 0, 1, 0, 0)}
        placeholder="Set a time"
      />,
    );
    expect(screen.queryByText("Set a time")).toBeNull();
  });
});

describe("Input.DateTime readings", () => {
  preloadTimeLanguage();
  const NOW = new TimeStamp(new Date(2026, 7, 23, 14, 5));
  const START = NOW.sub(TimeSpan.hours(1));
  const PARENT = NOW.sub(TimeSpan.days(1));
  const at = (ts: TimeStamp): number => Number(ts.valueOf());
  const anchors = {
    start: at(START),
    parent: { start: at(PARENT), end: at(NOW.add(TimeSpan.days(1))) },
  };

  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(NOW.date());
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  const readings = (): string[] =>
    screen
      .getAllByRole("menuitem")
      .filter((item) => item.classList.contains("lyra-time-editor__suggestion"))
      .map((item) => item.textContent ?? "");

  interface EditProps extends Partial<Input.DateTimeProps> {
    steps?: number;
  }

  // Types text into a fresh input and returns its readings and what Enter commits.
  const edit = (
    text: string,
    { steps = 0, ...props }: EditProps = {},
  ): { readings: string[]; committed?: number } => {
    const onChange = vi.fn();
    // The value differs from every reading, so Enter always commits one.
    const { container } = render(
      <Input.DateTime value={local(2000, 0, 1, 0, 0)} onChange={onChange} {...props} />,
    );
    const field = open(container);
    type(field, text);
    const shown = screen.queryAllByRole("menuitem").length > 0 ? readings() : [];
    for (let i = 0; i < steps; i++) fireEvent.keyDown(field, { key: "ArrowDown" });
    fireEvent.keyDown(field, { key: "Enter" });
    const committed = onChange.mock.calls[0]?.[0] as number | undefined;
    cleanup();
    return { readings: shown, committed };
  };

  // A float64 holds an instant near the present only to 256ns.
  const expectAt = (committed: number | undefined, expected: TimeStamp): void => {
    expect(committed).toBeDefined();
    expect(Math.abs((committed ?? 0) - at(expected))).toBeLessThan(1000);
  };

  it("should resolve anchor words", () => {
    expectAt(edit("now", { anchors }).committed, NOW);
    expectAt(edit("START", { anchors }).committed, START);
  });

  it("should offset an anchor by a signed duration", () => {
    const before = edit("now - 5m", { anchors });
    expectAt(before.committed, NOW.sub(TimeSpan.minutes(5)));
    expect(before.readings[0]).toContain("5min before now");
    const after = edit("T+3.2s", { anchors });
    expectAt(after.committed, PARENT.add(TimeSpan.milliseconds(3200)));
    expect(after.readings[0]).toContain("3s 200ms after the parent start");
    expect(edit("T", { anchors }).readings[0]).toContain("the parent start");
  });

  it("should say how the expression was read", () => {
    const kinds: [string, string][] = [
      ["1756000000", "Unix seconds"],
      ["1756000000000", "Unix milliseconds"],
      ["1756000000000000000", "Unix nanoseconds"],
      ["2026-08-23T14:05:00Z", "ISO 8601"],
      ["14:05", "today"],
      ["2026-08-23", "local time"],
      ["2026-08-23 14:05", "local time"],
    ];
    for (const [text, reading] of kinds)
      expect(edit(text).readings[0], text).toContain(reading);
  });

  it("should not read an anchor the input does not have", () => {
    expect(edit("T+1s").readings).toEqual([]);
    for (const reading of edit("end - 1s").readings)
      expect(reading).not.toContain("the end");
  });

  it("should parse a local date-time with sub-millisecond digits", () => {
    const expected = new TimeStamp(new Date(2026, 7, 23, 14, 5, 0, 250)).add(
      TimeSpan.microseconds(137),
    );
    expectAt(edit("2026-08-23 14:05:00.250137").committed, expected);
  });

  it("should parse a date alone as local midnight", () => {
    expectAt(edit("2026-08-23").committed, new TimeStamp(new Date(2026, 7, 23)));
  });

  it("should parse a time of day as today", () => {
    expectAt(edit("2:05 pm").committed, NOW);
    expectAt(
      edit("23:59:59").committed,
      new TimeStamp(new Date(2026, 7, 23, 23, 59, 59)),
    );
  });

  it("should parse an ISO string with a zone", () => {
    const expected = new TimeStamp(new Date(Date.UTC(2026, 7, 23, 14, 5, 0, 250)));
    expectAt(edit("2026-08-23T14:05:00.250Z").committed, expected);
  });

  it("should read an epoch in the one unit that lands near the present", () => {
    const ns = NOW.valueOf();
    const units: [string, bigint][] = [
      ["seconds", 1000000000n],
      ["milliseconds", 1000000n],
      ["microseconds", 1000n],
      ["nanoseconds", 1n],
    ];
    for (const [unit, size] of units) {
      const { readings: shown, committed } = edit((ns / size).toString());
      expect(shown, unit).toHaveLength(1);
      expect(shown[0], unit).toContain(`Unix ${unit}`);
      expectAt(committed, NOW);
    }
  });

  it("should read a 9-digit epoch as seconds", () => {
    expect(edit("999999999").readings[0]).toContain("Unix seconds");
  });

  it("should read an epoch that no unit places near the present as nanoseconds", () => {
    const { readings: shown, committed } = edit("86400000000000");
    expect(shown[0]).toContain("Unix nanoseconds");
    expect(committed).toBe(Number(TimeSpan.DAY.valueOf()));
  });

  it("should round a commit to the nearest microsecond", () => {
    expect(edit("86400000001600").committed).toBe(86400000002000);
  });

  it("should reject dates and times that do not exist", () => {
    const missing = [
      "2026-02-31",
      "2026-02-29 10:00",
      "2026-13-01",
      "25:00",
      "12:99",
      "12:30:60",
      "13:00 pm",
      "0:30 am",
    ];
    // Natural language may still offer its own reading, but never the grammar's.
    for (const text of missing)
      for (const reading of edit(text).readings) {
        expect(reading, text).not.toContain("local time");
        expect(reading, text).not.toContain("today");
      }
  });

  it("should place a bare time on the other end's day for an end cell", () => {
    const props = { anchors: { start: at(new TimeStamp(new Date(2026, 7, 20, 9))) } };
    const { readings: shown } = edit("11:30", { ...props, bound: "end" });
    expect(shown[0]).toContain("today");
    expect(shown[1]).toContain("on the day of the start");
    const { committed } = edit("11:30", { ...props, bound: "end", steps: 1 });
    expectAt(committed, new TimeStamp(new Date(2026, 7, 20, 11, 30)));
  });

  it("should read a bare duration relative to the range", () => {
    const start = new TimeStamp(new Date(2026, 7, 20, 9));
    const { readings: shown, committed } = edit("2h", {
      anchors: { start: at(start) },
      bound: "end",
    });
    expect(shown[0]).toContain("2h after the start");
    expectAt(committed, start.add(TimeSpan.hours(2)));
    expect(shown.some((r) => r.includes("2h from now"))).toBe(true);
  });

  it("should shift the current value by a signed duration", () => {
    const { readings: shown, committed } = edit("+30s", { value: at(START) });
    expect(shown[0]).toContain("30s after the current value");
    expectAt(committed, START.add(TimeSpan.seconds(30)));
  });

  it("should read natural language", () => {
    expectAt(
      edit("tomorrow at 3pm").committed,
      new TimeStamp(new Date(2026, 7, 24, 15, 0)),
    );
    expectAt(edit("5m ago").committed, NOW.sub(TimeSpan.minutes(5)));
  });

  it("should offer the end of a whole day to an end cell", () => {
    const { readings: shown } = edit("tomorrow", { bound: "end" });
    expect(shown[0]).toContain("start of tomorrow");
    expect(shown[1]).toContain("end of tomorrow");
  });

  it("should read nothing in nonsense", () => {
    expect(edit("purple").readings).toEqual([]);
  });

  it("should nudge each unit of the fixed layout under the caret", () => {
    const text = "2026-08-23 14:05:00.250 137 004";
    const nudged: [number, string][] = [
      [0, "2027-08-23 14:05:00.250 137 004"],
      [6, "2026-09-23 14:05:00.250 137 004"],
      [9, "2026-08-24 14:05:00.250 137 004"],
      [12, "2026-08-23 15:05:00.250 137 004"],
      [15, "2026-08-23 14:06:00.250 137 004"],
      [19, "2026-08-23 14:05:01.250 137 004"],
      [21, "2026-08-23 14:05:00.251 137 004"],
      [25, "2026-08-23 14:05:00.250 138 004"],
      [30, "2026-08-23 14:05:00.250 137 005"],
    ];
    for (const [caret, expected] of nudged) {
      const { container } = render(
        <Input.DateTime value={at(NOW)} onChange={vi.fn()} />,
      );
      const field = open(container);
      type(field, text);
      field.setSelectionRange(caret, caret);
      fireEvent.keyDown(field, { key: "ArrowUp" });
      expect(field.value, `caret ${caret}`).toBe(expected);
      cleanup();
    }
  });

  it("should step a month back on the local calendar", () => {
    const { container } = render(<Input.DateTime value={at(NOW)} onChange={vi.fn()} />);
    const field = open(container);
    type(field, "2026-08-23 14:05:00.000 137");
    field.setSelectionRange(6, 6);
    fireEvent.keyDown(field, { key: "ArrowDown" });
    expect(field.value).toBe("2026-07-23 14:05:00.000 137");
  });
});
