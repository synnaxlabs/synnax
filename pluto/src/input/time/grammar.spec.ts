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
  nudge,
  parseTimeSpan,
  parseTimeStamp,
  roundNumeric,
  unitAt,
} from "@/input/time/grammar";

const NOW = new TimeStamp(new Date(2026, 7, 23, 14, 5, 0, 0));

describe("parseTimeSpan", () => {
  it("should parse unit runs", () => {
    expect(parseTimeSpan("30s")?.value.equals(TimeSpan.seconds(30))).toBe(true);
    expect(parseTimeSpan("2h 30m")?.value.equals(TimeSpan.minutes(150))).toBe(true);
    expect(parseTimeSpan("2h30m")?.value.equals(TimeSpan.minutes(150))).toBe(true);
    expect(parseTimeSpan("1.5h")?.value.equals(TimeSpan.minutes(90))).toBe(true);
    expect(parseTimeSpan("500ms")?.value.equals(TimeSpan.milliseconds(500))).toBe(true);
    expect(parseTimeSpan("250us")?.value.equals(TimeSpan.microseconds(250))).toBe(true);
    expect(parseTimeSpan("250µs")?.value.equals(TimeSpan.microseconds(250))).toBe(true);
    expect(parseTimeSpan("10ns")?.value.equals(TimeSpan.nanoseconds(10))).toBe(true);
    expect(parseTimeSpan("2d")?.value.equals(TimeSpan.days(2))).toBe(true);
  });

  it("should parse clock form", () => {
    expect(parseTimeSpan("1:30:00")?.value.equals(TimeSpan.minutes(90))).toBe(true);
    expect(parseTimeSpan("0:30")?.value.equals(TimeSpan.minutes(30))).toBe(true);
    expect(
      parseTimeSpan("00:00:30.250")?.value.equals(TimeSpan.milliseconds(30250)),
    ).toBe(true);
  });

  it("should treat a bare number as seconds", () => {
    expect(parseTimeSpan("45")?.value.equals(TimeSpan.seconds(45))).toBe(true);
    expect(parseTimeSpan("0.5")?.value.equals(TimeSpan.milliseconds(500))).toBe(true);
  });

  it("should say which form the duration took", () => {
    expect(parseTimeSpan("2h 30m")?.kind).toBe("units");
    expect(parseTimeSpan("1:30:00")?.kind).toBe("clock");
    expect(parseTimeSpan("45")?.kind).toBe("seconds");
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
    expect(t).toMatchObject({
      kind: "anchor",
      anchor: "parent",
      offset: TimeSpan.milliseconds(3200),
    });
    expect(res).toMatchObject({ kind: "anchor", offset: TimeSpan.minutes(-5) });
  });

  it("should say how the expression was read", () => {
    const kinds: [string, object][] = [
      ["now", { kind: "anchor", anchor: "now", offset: TimeSpan.ZERO }],
      ["1756000000", { kind: "epoch", unit: "seconds" }],
      ["1756000000000", { kind: "epoch", unit: "milliseconds" }],
      ["1756000000000000000", { kind: "epoch", unit: "nanoseconds" }],
      ["2026-08-23T14:05:00Z", { kind: "iso" }],
      ["14:05", { kind: "clock" }],
      ["14:05:00.250 137", { kind: "clock" }],
      ["2026-08-23", { kind: "local" }],
      ["2026-08-23 14:05", { kind: "local" }],
    ];
    for (const [text, reading] of kinds)
      expect(parseTimeStamp(text, anchors), text).toMatchObject(reading);
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

  it("should read an epoch in the one unit that lands near the present", () => {
    const ns = NOW.valueOf();
    const units: [string, bigint][] = [
      ["seconds", 1000000000n],
      ["milliseconds", 1000000n],
      ["microseconds", 1000n],
      ["nanoseconds", 1n],
    ];
    for (const [unit, size] of units) {
      const res = parseTimeStamp((ns / size).toString(), anchors);
      expect(res, unit).toMatchObject({ kind: "epoch", unit });
      expect(res.ok && res.value.equals(NOW), unit).toBe(true);
    }
  });

  it("should read a 9-digit epoch as seconds", () => {
    const res = parseTimeStamp("999999999", anchors);
    expect(res).toMatchObject({ kind: "epoch", unit: "seconds" });
  });

  it("should read an epoch that no unit places near the present as nanoseconds", () => {
    const res = parseTimeStamp("86400000000000", anchors);
    expect(res).toMatchObject({ kind: "epoch", unit: "nanoseconds" });
    expect(res.ok && res.value.equals(new TimeStamp(TimeSpan.DAY))).toBe(true);
  });

  it("should reject nonsense", () => {
    expect(parseTimeStamp("yesterday-ish", anchors)).toEqual({
      ok: false,
      reason: "invalid",
    });
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
    for (const text of missing)
      expect(parseTimeStamp(text, anchors).ok, text).toBe(false);
  });
});

describe("roundNumeric", () => {
  it("should round to the nearest microsecond on either side of zero", () => {
    expect(roundNumeric(1600)).toBe(2000n);
    expect(roundNumeric(-1000)).toBe(-1000n);
    expect(roundNumeric(-400)).toBe(0n);
    expect(roundNumeric(-1600)).toBe(-2000n);
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

  it("should step months and years on the local calendar", () => {
    const ts = NOW.add(TimeSpan.microseconds(137));
    const next = nudge(ts, "month", 1);
    expect(next.toPreciseString("local")).toBe("2026-09-23 14:05:00.000 137");
    expect(nudge(ts, "year", -1).toPreciseString("local")).toBe(
      "2025-08-23 14:05:00.000 137",
    );
  });
});
