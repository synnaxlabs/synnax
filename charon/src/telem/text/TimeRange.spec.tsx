// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { telem } from "@synnaxlabs/x/telem";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Telem } from "@/telem";

describe("TimeRange", () => {
  it("should render time range with start and end", () => {
    const start = telem.TimeSpan.hours(10);
    const end = telem.TimeSpan.hours(14);
    const range = new telem.TimeRange(start, end);
    const c = render(
      <Telem.Text.TimeRange displayTZ="UTC">{range}</Telem.Text.TimeRange>,
    );
    expect(c.container.textContent).toContain("Jan 1 10:00:00");
    expect(c.container.textContent).toContain("14:00:00");
  });
  it("should render open time range with Started", () => {
    const start = telem.TimeSpan.hours(10);
    const range = new telem.TimeRange(start, telem.TimeStamp.MAX);
    const c = render(
      <Telem.Text.TimeRange displayTZ="UTC">{range}</Telem.Text.TimeRange>,
    );
    expect(c.getByText("Started Jan 1 10:00:00")).toBeTruthy();
  });
  it("should show Today for current date", () => {
    const now = telem.TimeStamp.now();
    const range = new telem.TimeRange(now, now.add(telem.TimeSpan.hours(4)));
    const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
    expect(c.container.textContent).toContain("Today");
  });
  it("should return null for fully-open time range", () => {
    const range = new telem.TimeRange(telem.TimeStamp.MAX, telem.TimeStamp.MAX);
    const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
    expect(c.container.textContent).toBe("");
  });
  it("should show date for multi-day ranges", () => {
    const start = new telem.TimeStamp(1704108000000000);
    const end = start.add(telem.TimeSpan.days(2).add(telem.TimeSpan.hours(4)));
    const range = new telem.TimeRange(start, end);
    const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
    const text = c.container.textContent || "";
    expect(text.includes("Jan") || text.includes("Dec")).toBe(true);
  });
  it("should pass through text props", () => {
    const start = telem.TimeSpan.hours(10);
    const end = telem.TimeSpan.hours(14);
    const range = new telem.TimeRange(start, end);
    const c = render(
      <Telem.Text.TimeRange level="h3" color={7}>
        {range}
      </Telem.Text.TimeRange>,
    );
    expect(c.container.querySelector("h3")).toBeTruthy();
  });
  it("should handle zero span ranges", () => {
    const ts = telem.TimeSpan.hours(10);
    const range = new telem.TimeRange(ts, ts);
    const c = render(<Telem.Text.TimeRange>{range}</Telem.Text.TimeRange>);
    expect(c.container.textContent).toBeTruthy();
  });
});
