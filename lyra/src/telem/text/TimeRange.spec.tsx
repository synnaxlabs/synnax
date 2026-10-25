// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange as XTimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Telem } from "@/telem";

describe("TimeRange", () => {
  it("should render time range with start and end", () => {
    const start = TimeSpan.hours(10);
    const end = TimeSpan.hours(14);
    const range = new XTimeRange(start, end);
    const c = render(
      <Telem.Text.TimeRange displayTimeZone="UTC">{range}</Telem.Text.TimeRange>,
    );
    expect(c.container.textContent).toContain("Jan 1 10:00:00");
    expect(c.container.textContent).toContain("14:00:00");
  });
  it("should render open time range with Started", () => {
    const start = TimeSpan.hours(10);
    const range = new XTimeRange(start, TimeStamp.MAX);
    const c = render(
      <Telem.Text.TimeRange displayTimeZone="UTC">{range}</Telem.Text.TimeRange>,
    );
    expect(c.getByText("Started Jan 1 10:00:00")).toBeTruthy();
  });
  it("should show Today for current date", () => {
    const now = TimeStamp.now();
    const range = new XTimeRange(now, now.add(TimeSpan.hours(4)));
    const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
    expect(c.container.textContent).toContain("Today");
  });
  describe("at local noon", () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date(2026, 0, 15, 12));
    });
    afterEach(() => {
      vi.useRealTimers();
    });
    it("should lowercase today after Started", () => {
      const earlier = TimeStamp.now().sub(TimeSpan.minutes(5));
      const range = new XTimeRange(earlier, TimeStamp.MAX);
      const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
      expect(c.container.textContent).toMatch(/^Started today /);
    });
    it("should lowercase today after Starts", () => {
      const later = TimeStamp.now().add(TimeSpan.minutes(5));
      const range = new XTimeRange(later, TimeStamp.MAX);
      const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
      expect(c.container.textContent).toMatch(/^Starts today /);
    });
    it("should turn Starts into Started when the start passes", () => {
      const later = TimeStamp.now().add(TimeSpan.seconds(30));
      const range = new XTimeRange(later, TimeStamp.MAX);
      const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
      act(() => {
        vi.advanceTimersByTime(TimeSpan.seconds(30).milliseconds);
      });
      expect(c.container.textContent).toMatch(/^Started today /);
    });
  });
  it("should drop today once local midnight passes", () => {
    vi.useFakeTimers({ now: new Date(2026, 0, 15, 23, 59, 30) });
    try {
      const range = new XTimeRange(TimeStamp.now().sub(TimeSpan.HOUR), TimeStamp.MAX);
      const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
      expect(c.container.textContent).toMatch(/^Started today /);
      act(() => {
        vi.advanceTimersByTime(TimeSpan.seconds(30).milliseconds);
      });
      expect(c.container.textContent).toMatch(/^Started Jan 15 /);
    } finally {
      vi.useRealTimers();
    }
  });
  it("should return null for fully-open time range", () => {
    const range = new XTimeRange(TimeStamp.MAX, TimeStamp.MAX);
    const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
    expect(c.container.textContent).toBe("");
  });
  it("should show date for multi-day ranges", () => {
    const start = new TimeStamp(1704108000000000);
    const end = start.add(TimeSpan.days(2).add(TimeSpan.hours(4)));
    const range = new XTimeRange(start, end);
    const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
    const text = c.container.textContent || "";
    expect(text.includes("Jan") || text.includes("Dec")).toBe(true);
  });
  it("should pass through text props", () => {
    const start = TimeSpan.hours(10);
    const end = TimeSpan.hours(14);
    const range = new XTimeRange(start, end);
    const c = render(
      <Telem.Text.TimeRange level="h3" color={7}>
        {range}
      </Telem.Text.TimeRange>,
    );
    expect(c.container.querySelector("h3")).toBeTruthy();
  });
  it("should handle zero span ranges", () => {
    const ts = TimeSpan.hours(10);
    const range = new XTimeRange(ts, ts);
    const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
    expect(c.container.textContent).toBeTruthy();
  });
});
