// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Telem } from "@synnaxlabs/lyra/telem";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { formatAgo, formatSpan, formatStart } from "@/feature/range/toolbar/time";

const NOW = new TimeStamp(new Date(2026, 10, 17, 15, 30));

describe("formatSpan", () => {
  it.each([
    [TimeSpan.seconds(32), "32s"],
    [TimeSpan.seconds(4 * 60 + 5), "4m 05s"],
    [TimeSpan.minutes(6 * 60 + 2), "6h 02m"],
    [TimeSpan.hours(23 * 24 + 6), "23d 6h"],
  ])("should format %s as %s", (span, text) => {
    expect(formatSpan(NOW.sub(span), NOW)).toBe(text);
  });

  it("should show no negative span", () => {
    expect(formatSpan(NOW.add(TimeSpan.seconds(3)), NOW)).toBe("0s");
  });
});

describe("formatAgo", () => {
  it.each([
    [TimeSpan.seconds(59), "Just now"],
    [TimeSpan.minutes(12), "12m ago"],
    [TimeSpan.hours(3), "3h ago"],
    [TimeSpan.days(2), "2d ago"],
  ])("should format an end %s back as %s", (span, text) => {
    expect(formatAgo(NOW.sub(span), NOW)).toBe(text);
  });

  it("should show the date of an end a week or more back", () => {
    const end = NOW.sub(TimeSpan.days(9));
    expect(formatAgo(end, NOW)).toBe(formatStart(end, NOW));
  });
});

describe("formatStart", () => {
  it("should show the time of day for a start today", () => {
    const start = NOW.sub(TimeSpan.hours(2));
    expect(formatStart(start, NOW)).toBe(Telem.Text.formatTime(start, TimeSpan.MINUTE));
  });

  it("should show the year only for a start in an earlier year", () => {
    const thisYear = formatStart(new TimeStamp(new Date(2026, 0, 5)), NOW);
    const lastYear = formatStart(new TimeStamp(new Date(2025, 0, 5)), NOW);
    expect(thisYear).not.toContain("2026");
    expect(lastYear).toContain("2025");
  });
});
