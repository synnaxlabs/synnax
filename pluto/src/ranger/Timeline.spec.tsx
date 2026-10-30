// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mockBoundingClientRect } from "@synnaxlabs/lyra/testutil";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Ranger } from "@/ranger";

// A float64 holds the nanoseconds of this instant exactly.
const EXACT = new Date(Date.UTC(2026, 0, 1));
// A float64 rounds the nanoseconds of this instant up by 64 ns.
const ROUNDS_UP = new Date(EXACT.getTime() + 3);

const DAY = Number(TimeSpan.DAY.valueOf());
const HOUR = Number(TimeSpan.HOUR.valueOf());
const MILLISECOND = Number(TimeSpan.MILLISECOND.valueOf());
const SECOND = Number(TimeSpan.SECOND.valueOf());

const triggers = (container: HTMLElement): NodeListOf<HTMLElement> =>
  container.querySelectorAll(".lyra-time-editor__trigger");

const startLabel = (): string =>
  screen.getByLabelText(/^Start, /).getAttribute("aria-label") ?? "";

describe("label resolution", () => {
  const fraction = 123 * MILLISECOND + 456_000;

  it("should read to the minute across days", () => {
    const now = TimeStamp.now().nanoseconds;
    render(
      <Ranger.Timeline
        value={{ start: now - 2 * DAY + fraction, end: now }}
        onChange={() => {}}
      />,
    );
    expect(startLabel()).not.toMatch(/\./);
  });

  it("should read to the millisecond across seconds", () => {
    const now = TimeStamp.now().nanoseconds;
    render(
      <Ranger.Timeline
        value={{ start: now - 15 * SECOND + fraction, end: now }}
        onChange={() => {}}
      />,
    );
    expect(startLabel()).toMatch(/\.\d{3}$/);
  });

  it("should read to the microsecond across milliseconds", () => {
    const now = TimeStamp.now().nanoseconds;
    render(
      <Ranger.Timeline
        value={{ start: now - 3 * MILLISECOND + 456_000, end: now }}
        onChange={() => {}}
      />,
    );
    expect(startLabel()).toMatch(/\.\d{3} \d{3}$/);
  });
});

describe("Ranger.Timeline", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  describe("preview", () => {
    it("should not open a timestamp editor", () => {
      const now = TimeStamp.now().nanoseconds;
      const { container } = render(
        <Ranger.Timeline
          value={{ start: now - 2 * HOUR, end: now - HOUR }}
          onChange={() => {}}
          preview
        />,
      );
      fireEvent.click(triggers(container)[0]);
      expect(screen.queryByRole("textbox")).toBeNull();
    });

    it("should not open the stage menu", () => {
      const now = TimeStamp.now().nanoseconds;
      render(
        <Ranger.Timeline
          value={{ start: now - 2 * HOUR, end: now - HOUR }}
          onChange={() => {}}
          preview
        />,
      );
      fireEvent.click(screen.getByText("Completed"));
      expect(screen.queryByText("Reopen")).toBeNull();
    });
  });

  it("should move to in progress when the planned start passes", () => {
    vi.useFakeTimers({ now: EXACT });
    try {
      const now = TimeStamp.now().nanoseconds;
      render(
        <Ranger.Timeline
          value={{ start: now + 50 * MILLISECOND, end: now + HOUR }}
          onChange={() => {}}
        />,
      );
      expect(screen.getByText("To do")).toBeTruthy();
      act(() => {
        vi.advanceTimersByTime(50);
      });
      expect(screen.getByText("In progress")).toBeTruthy();
    } finally {
      vi.useRealTimers();
    }
  });

  it("should move to completed when the planned end passes", () => {
    vi.useFakeTimers({ now: EXACT });
    try {
      const now = TimeStamp.now().nanoseconds;
      render(
        <Ranger.Timeline
          value={{ start: now - HOUR, end: now + 50 * MILLISECOND }}
          onChange={() => {}}
        />,
      );
      expect(screen.getByText("In progress")).toBeTruthy();
      act(() => {
        vi.advanceTimersByTime(50);
      });
      expect(screen.getByText("Completed")).toBeTruthy();
    } finally {
      vi.useRealTimers();
    }
  });

  it("should read in progress at the planned start", () => {
    vi.useFakeTimers({ now: EXACT });
    try {
      const now = TimeStamp.now().nanoseconds;
      render(
        <Ranger.Timeline value={{ start: now, end: now + HOUR }} onChange={() => {}} />,
      );
      expect(screen.getByText("In progress")).toBeTruthy();
    } finally {
      vi.useRealTimers();
    }
  });

  it("should read completed at the planned end", () => {
    vi.useFakeTimers({ now: EXACT });
    try {
      const now = TimeStamp.now().nanoseconds;
      render(
        <Ranger.Timeline value={{ start: now - HOUR, end: now }} onChange={() => {}} />,
      );
      expect(screen.getByText("Completed")).toBeTruthy();
    } finally {
      vi.useRealTimers();
    }
  });

  it("should move to in progress at a planned start between clock readings", () => {
    vi.useFakeTimers({ now: ROUNDS_UP });
    try {
      const start = new TimeStamp(ROUNDS_UP).nanoseconds;
      render(
        <Ranger.Timeline value={{ start, end: start + HOUR }} onChange={() => {}} />,
      );
      expect(screen.getByText("To do")).toBeTruthy();
      act(() => {
        vi.advanceTimersByTime(1);
      });
      expect(screen.getByText("In progress")).toBeTruthy();
    } finally {
      vi.useRealTimers();
    }
  });

  it("should move to completed at a planned end between clock readings", () => {
    vi.useFakeTimers({ now: ROUNDS_UP });
    try {
      const end = new TimeStamp(ROUNDS_UP).nanoseconds;
      render(
        <Ranger.Timeline value={{ start: end - HOUR, end }} onChange={() => {}} />,
      );
      expect(screen.getByText("In progress")).toBeTruthy();
      act(() => {
        vi.advanceTimersByTime(1);
      });
      expect(screen.getByText("Completed")).toBeTruthy();
    } finally {
      vi.useRealTimers();
    }
  });

  it("should never read an open range finer than the second", () => {
    const start = TimeStamp.now().nanoseconds - 5 * SECOND + 123 * MILLISECOND;
    const { container } = render(
      <Ranger.Timeline
        value={{ start, end: TimeStamp.MAX.nanoseconds }}
        onChange={() => {}}
      />,
    );
    expect(triggers(container)[0].textContent).toMatch(/\d{2}:\d{2}(:\d{2})?$/);
  });

  it("should tick the elapsed clock by the second under a far end", () => {
    const now = TimeStamp.now().nanoseconds;
    const { container } = render(
      <Ranger.Timeline
        value={{ start: now - 48 * SECOND, end: now + 2 * DAY }}
        onChange={() => {}}
      />,
    );
    expect(container.querySelector(".lyra-timespan__label")?.textContent).toBe("48s");
  });

  it("should offer a planned end once a range is scheduled", () => {
    const now = TimeStamp.now().nanoseconds;
    const onChange = vi.fn();
    render(
      <Ranger.Timeline
        value={{ start: now + HOUR, end: TimeStamp.MAX.nanoseconds }}
        onChange={onChange}
      />,
    );
    fireEvent.click(screen.getByText("Set an end time"));
    const field = screen.getByRole("textbox");
    fireEvent.change(field, { target: { value: "start + 2h" } });
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith({ start: now + HOUR, end: now + 3 * HOUR });
  });

  it("should name each time cell by the end it holds", () => {
    const now = TimeStamp.now().nanoseconds;
    render(
      <Ranger.Timeline
        value={{ start: now - HOUR, end: TimeStamp.MAX.nanoseconds }}
        onChange={() => {}}
      />,
    );
    // Within an hour of midnight the start reads "Yesterday" instead of "Today".
    expect(
      screen.getByRole("button", { name: /^Start, (?:Today|Yesterday) \d{2}:\d{2}/ }),
    ).toBeTruthy();
    expect(screen.getByRole("button", { name: "End, not set" })).toBeTruthy();
  });

  it("should name a cell by the new day once local midnight passes", () => {
    vi.useFakeTimers({ now: new Date(2026, 0, 1, 23, 59, 30) });
    try {
      const now = TimeStamp.now().nanoseconds;
      render(
        <Ranger.Timeline
          value={{ start: now - HOUR, end: now + 2 * DAY }}
          onChange={() => {}}
        />,
      );
      expect(startLabel()).toMatch(/^Start, Today/);
      act(() => {
        vi.advanceTimersByTime(60_000);
      });
      expect(startLabel()).toMatch(/^Start, Yesterday/);
    } finally {
      vi.useRealTimers();
    }
  });

  it("should read an open range to the minute once it runs for a day", () => {
    vi.useFakeTimers({ now: new Date(2026, 0, 1, 12, 0, 0) });
    try {
      const start = TimeStamp.now().nanoseconds - DAY + 30 * SECOND;
      render(
        <Ranger.Timeline
          value={{ start, end: TimeStamp.MAX.nanoseconds }}
          onChange={() => {}}
        />,
      );
      expect(startLabel()).toMatch(/\d{2}:\d{2}:\d{2}$/);
      act(() => {
        vi.advanceTimersByTime(60_000);
      });
      expect(startLabel()).toMatch(/ \d{2}:\d{2}$/);
    } finally {
      vi.useRealTimers();
    }
  });

  it("should not offer an end before a range is scheduled", () => {
    const unset = TimeStamp.MAX.nanoseconds;
    render(
      <Ranger.Timeline value={{ start: unset, end: unset }} onChange={() => {}} />,
    );
    expect(screen.queryByText("Set an end time")).toBeNull();
  });

  it("should end a running range at its start on a zero elapsed time", () => {
    const now = TimeStamp.now().nanoseconds;
    const start = now - HOUR;
    const onChange = vi.fn();
    const { container } = render(
      <Ranger.Timeline
        value={{ start, end: TimeStamp.MAX.nanoseconds }}
        onChange={onChange}
      />,
    );
    fireEvent.click(triggers(container)[1]);
    const field = screen.getByRole("textbox");
    fireEvent.change(field, { target: { value: "0s" } });
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith({ start, end: start });
  });

  describe("row order", () => {
    const precedes = (a: Element, b: Element): boolean =>
      (a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0;
    const span = (container: HTMLElement): Element =>
      container.querySelector(".lyra-timespan") as Element;
    const dot = (container: HTMLElement): Element =>
      container.querySelector(".pluto-range-timeline__dot") as Element;

    it("should put the elapsed time before the end placeholder", () => {
      const now = TimeStamp.now().nanoseconds;
      const { container } = render(
        <Ranger.Timeline
          value={{ start: now - HOUR, end: TimeStamp.MAX.nanoseconds }}
          onChange={() => {}}
        />,
      );
      expect(precedes(dot(container), span(container))).toBe(true);
      expect(precedes(span(container), screen.getByText("Set an end time"))).toBe(true);
    });

    it("should put the elapsed time after a set end", () => {
      const now = TimeStamp.now().nanoseconds;
      const { container } = render(
        <Ranger.Timeline
          value={{ start: now - HOUR, end: now + HOUR }}
          onChange={() => {}}
        />,
      );
      const end = screen.getByRole("button", { name: /^End, / });
      expect(precedes(end, dot(container))).toBe(true);
      expect(precedes(dot(container), span(container))).toBe(true);
    });

    it("should read started, ended, then the duration once completed", () => {
      const now = TimeStamp.now().nanoseconds;
      const { container } = render(
        <Ranger.Timeline
          value={{ start: now - 2 * HOUR, end: now - HOUR }}
          onChange={() => {}}
        />,
      );
      const started = screen.getByText("started");
      const ended = screen.getByText("ended");
      expect(precedes(started, ended)).toBe(true);
      expect(precedes(ended, dot(container))).toBe(true);
      expect(precedes(dot(container), span(container))).toBe(true);
    });
  });
});
