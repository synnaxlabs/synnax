// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  describeDay,
  formatTime,
  formatTimeStamp,
  nudge,
  parseTimeSpan,
  parseTimeStamp,
  unitAt,
} from "@/input/time/grammar";

const NOW = new TimeStamp(new Date(2026, 7, 23, 14, 5, 0, 0));

describe("parseTimeSpan", () => {
  it("should parse unit runs", () => {
    expect(parseTimeSpan("30s")?.equals(TimeSpan.seconds(30))).toBe(true);
    expect(parseTimeSpan("2h 30m")?.equals(TimeSpan.minutes(150))).toBe(true);
    expect(parseTimeSpan("2h30m")?.equals(TimeSpan.minutes(150))).toBe(true);
    expect(parseTimeSpan("1.5h")?.equals(TimeSpan.minutes(90))).toBe(true);
    expect(parseTimeSpan("500ms")?.equals(TimeSpan.milliseconds(500))).toBe(true);
    expect(parseTimeSpan("250us")?.equals(TimeSpan.microseconds(250))).toBe(true);
    expect(parseTimeSpan("250µs")?.equals(TimeSpan.microseconds(250))).toBe(true);
    expect(parseTimeSpan("10ns")?.equals(TimeSpan.nanoseconds(10))).toBe(true);
    expect(parseTimeSpan("2d")?.equals(TimeSpan.days(2))).toBe(true);
  });

  it("should parse clock form", () => {
    expect(parseTimeSpan("1:30:00")?.equals(TimeSpan.minutes(90))).toBe(true);
    expect(parseTimeSpan("0:30")?.equals(TimeSpan.minutes(30))).toBe(true);
    expect(parseTimeSpan("00:00:30.250")?.equals(TimeSpan.milliseconds(30250))).toBe(
      true,
    );
  });

  it("should treat a bare number as seconds", () => {
    expect(parseTimeSpan("45")?.equals(TimeSpan.seconds(45))).toBe(true);
    expect(parseTimeSpan("0.5")?.equals(TimeSpan.milliseconds(500))).toBe(true);
  });

  it("should reject text that is not a duration", () => {
    expect(parseTimeSpan("")).toBeNull();
    expect(parseTimeSpan("soon")).toBeNull();
    expect(parseTimeSpan("5 parsecs")).toBeNull();
  });
});

describe("parseTimeStamp", () => {
  const anchors = {
    now: NOW,
    start: NOW.sub(TimeSpan.hours(1)),
    parent: NOW.sub(TimeSpan.days(1)),
  };

  it("should resolve anchor words", () => {
    const res = parseTimeStamp("now", anchors);
    expect(res.ok && res.value.equals(NOW)).toBe(true);
    const start = parseTimeStamp("START", anchors);
    expect(start.ok && start.value.equals(anchors.start)).toBe(true);
  });

  it("should offset an anchor by a signed duration", () => {
    const res = parseTimeStamp("now - 5m", anchors);
    expect(res.ok && res.value.equals(NOW.sub(TimeSpan.minutes(5)))).toBe(true);
    const t = parseTimeStamp("T+3.2s", anchors);
    expect(
      t.ok && t.value.equals(anchors.parent.add(TimeSpan.milliseconds(3200))),
    ).toBe(true);
    expect(t.ok && t.anchor).toBe("parent");
  });

  it("should report a missing anchor", () => {
    expect(parseTimeStamp("T+1s", { now: NOW })).toEqual({
      ok: false,
      reason: "no-parent",
    });
    expect(parseTimeStamp("end", { now: NOW })).toEqual({
      ok: false,
      reason: "no-anchor",
    });
  });

  it("should parse a local date-time with sub-millisecond digits", () => {
    const res = parseTimeStamp("2026-08-23 14:05:00.250137", anchors);
    const expected = new TimeStamp(new Date(2026, 7, 23, 14, 5, 0, 250)).add(
      TimeSpan.microseconds(137),
    );
    expect(res.ok && res.value.equals(expected)).toBe(true);
  });

  it("should parse a date alone as local midnight", () => {
    const res = parseTimeStamp("2026-08-23", anchors);
    expect(res.ok && res.value.equals(new TimeStamp(new Date(2026, 7, 23)))).toBe(true);
  });

  it("should parse a time of day as today", () => {
    const res = parseTimeStamp("2:05 pm", anchors);
    expect(res.ok && res.value.equals(NOW)).toBe(true);
    const late = parseTimeStamp("23:59:59", anchors);
    expect(
      late.ok && late.value.equals(new TimeStamp(new Date(2026, 7, 23, 23, 59, 59))),
    ).toBe(true);
  });

  it("should parse an ISO string with a zone", () => {
    const res = parseTimeStamp("2026-08-23T14:05:00.250Z", anchors);
    const expected = new TimeStamp(new Date(Date.UTC(2026, 7, 23, 14, 5, 0, 250)));
    expect(res.ok && res.value.equals(expected)).toBe(true);
  });

  it("should parse an epoch in nanoseconds", () => {
    const res = parseTimeStamp(NOW.valueOf().toString(), anchors);
    expect(res.ok && res.value.equals(NOW)).toBe(true);
  });

  it("should reject nonsense", () => {
    expect(parseTimeStamp("yesterday-ish", anchors)).toEqual({
      ok: false,
      reason: "invalid",
    });
  });
});

describe("formatTimeStamp", () => {
  it("should drop trailing zero groups", () => {
    expect(formatTimeStamp(NOW)).toBe("2026-08-23 14:05:00");
    expect(formatTimeStamp(NOW.add(TimeSpan.milliseconds(250)))).toBe(
      "2026-08-23 14:05:00.250",
    );
    expect(formatTimeStamp(NOW.add(TimeSpan.microseconds(250137)))).toBe(
      "2026-08-23 14:05:00.250 137",
    );
    expect(formatTimeStamp(NOW.add(TimeSpan.nanoseconds(250137004)))).toBe(
      "2026-08-23 14:05:00.250 137 004",
    );
    expect(formatTimeStamp(NOW.add(TimeSpan.microseconds(4)))).toBe(
      "2026-08-23 14:05:00.000 004",
    );
  });

  it("should round trip through parseTimeStamp", () => {
    const ts = NOW.add(TimeSpan.nanoseconds(123456789));
    const res = parseTimeStamp(formatTimeStamp(ts), { now: NOW });
    expect(res.ok && res.value.equals(ts)).toBe(true);
  });
});

describe("unitAt", () => {
  it("should name the unit under the caret", () => {
    // 2026-08-23 14:05:00.250 137 004
    expect(unitAt(0)).toBe("year");
    expect(unitAt(4)).toBe("year");
    expect(unitAt(6)).toBe("month");
    expect(unitAt(9)).toBe("day");
    expect(unitAt(12)).toBe("hour");
    expect(unitAt(15)).toBe("minute");
    expect(unitAt(19)).toBe("second");
    expect(unitAt(21)).toBe("millisecond");
    expect(unitAt(25)).toBe("microsecond");
    expect(unitAt(30)).toBe("nanosecond");
  });
});

describe("nudge", () => {
  it("should step fixed units", () => {
    expect(nudge(NOW, "second", 1).equals(NOW.add(TimeSpan.SECOND))).toBe(true);
    expect(
      nudge(NOW, "millisecond", -10).equals(NOW.sub(TimeSpan.milliseconds(10))),
    ).toBe(true);
  });

  it("should step months on the calendar and keep sub-millisecond digits", () => {
    const ts = NOW.add(TimeSpan.microseconds(137));
    const next = nudge(ts, "month", 1);
    expect(formatTimeStamp(next)).toBe("2026-09-23 14:05:00.000 137");
    expect(formatTimeStamp(nudge(ts, "year", -1))).toBe("2025-08-23 14:05:00.000 137");
  });
});

describe("describeDay", () => {
  it("should name nearby days and fall back to month and day", () => {
    expect(describeDay(NOW, NOW)).toBe("Today");
    expect(describeDay(NOW.sub(TimeSpan.DAY), NOW)).toBe("Yesterday");
    expect(describeDay(NOW.add(TimeSpan.DAY), NOW)).toBe("Tomorrow");
    expect(describeDay(NOW.sub(TimeSpan.days(3)), NOW)).toBe("Thursday");
    expect(describeDay(NOW.sub(TimeSpan.days(30)), NOW)).toBe("Jul 24");
    expect(describeDay(NOW.sub(TimeSpan.days(400)), NOW)).toBe("Jul 19, 2025");
  });
});

describe("formatTime", () => {
  it("should drop zero seconds and keep set precision", () => {
    expect(formatTime(NOW)).toBe("14:05");
    expect(formatTime(NOW.add(TimeSpan.seconds(32)))).toBe("14:05:32");
    expect(formatTime(NOW.add(TimeSpan.milliseconds(250)))).toBe("14:05:00.250");
  });
});

describe("formatTime", () => {
  it("should drop label digits below the resolution", () => {
    const ts = NOW.add(TimeSpan.seconds(32)).add(TimeSpan.milliseconds(250));
    expect(formatTime(ts, TimeSpan.MINUTE)).toBe("14:05");
    expect(formatTime(ts, TimeSpan.SECOND)).toBe("14:05:32");
    expect(formatTime(ts)).toBe("14:05:32.250");
  });
});
