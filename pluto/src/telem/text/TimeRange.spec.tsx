// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange as XTimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Telem } from "@/telem";

describe("TimeRange", () => {
  it("should render time range with start and end", () => {
    const start = TimeSpan.hours(10);
    const end = TimeSpan.hours(14);
    const range = new XTimeRange(start, end);
    const c = render(
      <Telem.Text.TimeRange suppliedTZ="UTC" displayTZ="UTC">
        {range}
      </Telem.Text.TimeRange>,
    );
    expect(c.getByText("Jan 1 10:00:00")).toBeTruthy();
    expect(c.getByText("14:00:00")).toBeTruthy();
  });
  it("should render open time range with Now", () => {
    const start = TimeSpan.hours(10);
    const range = new XTimeRange(start, TimeStamp.MAX);
    const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
    expect(c.getByText("Now")).toBeTruthy();
  });
  it("should show Today for current date", () => {
    const now = TimeStamp.now();
    const range = new XTimeRange(now, now.add(TimeSpan.hours(4)));
    const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
    expect(c.container.textContent).toContain("Today");
  });
  it("should show span when showSpan is true", () => {
    const start = TimeSpan.hours(10);
    const end = TimeSpan.hours(14).add(TimeSpan.minutes(30));
    const range = new XTimeRange(start, end);
    const c = render(<Telem.Text.TimeRange showSpan>{range}</Telem.Text.TimeRange>);
    expect(c.container.textContent).toContain("4h 30m");
  });
  it("should not show span when showSpan is false", () => {
    const start = TimeSpan.hours(10);
    const end = TimeSpan.hours(14).add(TimeSpan.minutes(30));
    const range = new XTimeRange(start, end);
    const c = render(
      <Telem.Text.TimeRange showSpan={false}>{range}</Telem.Text.TimeRange>,
    );
    expect(c.container.textContent).not.toContain("4h 30m");
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
    const c = render(<Telem.Text.TimeRange showSpan>{range}</Telem.Text.TimeRange>);
    expect(c.container.textContent).toBeTruthy();
  });
});
