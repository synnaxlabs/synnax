// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { DataType, Density, Rate, Size, TimeRange, TimeSpan, TimeStamp } from "@/telem";

describe("TimeStamp", () => {
  test("construct", () => {
    const ts = new TimeStamp(1000);
    expect(ts.equals(TimeSpan.MICROSECOND)).toBeTruthy();
  });

  test("construct from TimeStamp", () => {
    const ts = new TimeStamp(TimeSpan.microseconds(10));
    expect(ts.equals(TimeSpan.microseconds(10))).toBeTruthy();
  });

  test("construct from local TimeZone", () => {
    const ts = new TimeStamp(TimeSpan.microseconds(10), "local");
    expect(ts.equals(TimeSpan.microseconds(10).add(TimeStamp.utcOffset))).toBeTruthy();
  });

  test("construct from time string", () => {
    const ts = new TimeStamp("12:30", "UTC");
    expect(ts.date().getUTCHours()).toEqual(12);
    expect(ts.equals(TimeSpan.hours(12).add(TimeSpan.minutes(30)))).toBeTruthy();
    const ts2 = new TimeStamp("12:30:00.22");
    expect(
      ts2.equals(
        TimeSpan.hours(12).add(TimeSpan.minutes(30).add(TimeSpan.milliseconds(22)))
      )
    ).toBeTruthy();
    const ts3 = new TimeStamp("12:30:00.22", "local");
    expect(
      ts3.equals(
        TimeSpan.hours(12)
          .add(TimeSpan.minutes(30).add(TimeSpan.milliseconds(22)))
          .add(TimeStamp.utcOffset)
      )
    ).toBeTruthy();
  });

  test("construct from date", () => {
    const ts = new TimeStamp([2021, 1, 1], "UTC");
    expect(ts.date().getUTCFullYear()).toEqual(2021);
    expect(ts.date().getUTCMonth()).toEqual(0);
    expect(ts.date().getUTCDate()).toEqual(1);
    expect([0, 1]).toContain(ts.date().getUTCHours());
    expect(ts.date().getUTCMinutes()).toEqual(0);
  });

  test("construct from date time string", () => {
    const ts = new TimeStamp("2021-01-01T00:00:00.000Z", "UTC");
    expect(ts.date().getUTCFullYear()).toEqual(2021);
    expect(ts.date().getUTCHours()).toEqual(0);
    const ts2 = new TimeStamp("2021-01-01", "local");
    expect(ts2.date().getUTCFullYear()).toEqual(2021);
    expect(ts2.date().getUTCHours()).toEqual(
      TimeStamp.utcOffset.valueOf() / TimeStamp.HOUR.valueOf()
    );
    expect(ts2.date().getUTCMinutes()).toEqual(0);
  });

  test("span", () => {
    const ts = new TimeStamp(0);
    expect(ts.span(new TimeStamp(1000)).equals(TimeSpan.microseconds())).toBeTruthy();
  });

  test("range", () => {
    const ts = new TimeStamp(0);
    expect(
      ts.range(new TimeStamp(1000)).equals(new TimeRange(ts, TimeSpan.microseconds()))
    ).toBeTruthy();
  });

  test("spanRange", () => {
    const ts = new TimeStamp(0);
    expect(
      ts
        .spanRange(TimeSpan.microseconds())
        .equals(new TimeRange(ts, ts.add(TimeSpan.microseconds())))
    ).toBeTruthy();
  });

  test("isZero", () => {
    const ts = new TimeStamp(0);
    expect(ts.isZero).toBeTruthy();
  });

  test("after", () => {
    const ts = new TimeStamp(0);
    expect(ts.after(new TimeStamp(-1))).toBeTruthy();
    const ts2 = new TimeStamp(1);
    expect(ts2.after(new TimeStamp(1))).toBeFalsy();
  });

  test("before", () => {
    const ts = new TimeStamp(0);
    expect(ts.before(new TimeStamp(1))).toBeTruthy();
    const ts2 = new TimeStamp(1);
    expect(ts2.before(new TimeStamp(1))).toBeFalsy();
  });

  test("beforeEq", () => {
    const ts = new TimeStamp(0);
    expect(ts.beforeEq(new TimeStamp(1))).toBeTruthy();
    const ts2 = new TimeStamp(1);
    expect(ts2.beforeEq(new TimeStamp(1))).toBeTruthy();
    const ts3 = new TimeStamp(2);
    expect(ts3.beforeEq(new TimeStamp(1))).toBeFalsy();
  });

  test("afterEq", () => {
    const ts = new TimeStamp(0);
    expect(ts.afterEq(new TimeStamp(-1))).toBeTruthy();
    const ts2 = new TimeStamp(1);
    expect(ts2.afterEq(new TimeStamp(1))).toBeTruthy();
    const ts3 = new TimeStamp(0);
    expect(ts3.afterEq(new TimeStamp(1))).toBeFalsy();
  });

  test("add", () => {
    const ts = new TimeStamp(0);
    expect(
      ts.add(TimeSpan.microseconds()).equals(new TimeStamp(TimeSpan.microseconds(1)))
    ).toBeTruthy();
  });

  test("sub", () => {
    const ts = new TimeStamp(TimeSpan.microseconds());
    expect(ts.sub(TimeSpan.microseconds()).equals(new TimeStamp(0))).toBeTruthy();
  });

  test("stringification", () => {
    const ts = new TimeStamp([2022, 12, 15], "UTC")
      .add(TimeSpan.hours(12))
      .add(TimeSpan.minutes(20))
      .add(TimeSpan.milliseconds(12));
    expect(ts.fString("ISO", "UTC")).toEqual("2022-12-15T12:20:00.012Z");
    expect(ts.fString("time", "UTC")).toEqual("12:20:00");
    expect(ts.fString("date", "UTC")).toEqual("Dec 15");
    if (!TimeStamp.utcOffset.equals(0)) {
      expect(ts.fString("ISO", "local")).not.toEqual("2022-12-15T12:20:00.012Z");
      expect(ts.fString("time", "local")).not.toEqual("12:20:00");
    }
  });
  describe("remainder", () => {
    test("day", () => {
      const expectedRemainder = TimeStamp.hours(12)
        .add(TimeSpan.minutes(20))
        .add(TimeSpan.milliseconds(12));
      const ts = new TimeStamp([2022, 12, 15])
        .add(TimeStamp.hours(12))
        .add(TimeSpan.minutes(20))
        .add(TimeSpan.milliseconds(12));
      const remainder = ts.remainder(TimeStamp.DAY);
      expect(
        remainder.equals(expectedRemainder),
        `expected ${new TimeSpan(expectedRemainder).toString()} got ${new TimeSpan(
          remainder
        ).toString()}`
      ).toBeTruthy();
    });
    test("second", () => {
      const expectedRemainder = TimeSpan.milliseconds(12);
      const ts = new TimeStamp([2022, 12, 15])
        .add(TimeStamp.hours(12))
        .add(TimeSpan.minutes(20))
        .add(TimeSpan.milliseconds(12));
      const remainder = ts.remainder(TimeSpan.seconds());
      expect(remainder.equals(expectedRemainder)).toBeTruthy();
    });
  });
});

describe("TimeSpan", () => {
  test("construct from static", () => {
    expect(TimeSpan.nanoseconds(1).equals(1)).toBeTruthy();
    expect(TimeSpan.microseconds(1).equals(1000)).toBeTruthy();
    expect(TimeSpan.milliseconds(1).equals(1000000)).toBeTruthy();
    expect(TimeSpan.seconds(1).equals(1e9)).toBeTruthy();
    expect(TimeSpan.minutes(1).equals(6e10)).toBeTruthy();
    expect(TimeSpan.hours(1).equals(36e11)).toBeTruthy();
  });

  test("seconds", () => {
    expect(TimeSpan.seconds(1).seconds).toEqual(1);
  });

  test("isZero", () => {
    expect(TimeSpan.ZERO.isZero).toBeTruthy();
    expect(TimeSpan.seconds(1).isZero).toBeFalsy();
  });

  test("add", () => {
    expect(TimeSpan.seconds(1).add(TimeSpan.SECOND).equals(2e9)).toBeTruthy();
  });

  test("sub", () => {
    expect(TimeSpan.seconds(1).sub(TimeSpan.SECOND).isZero).toBeTruthy();
  });
});

describe("Rate", () => {
  test("construct", () => expect(new Rate(1).equals(1)).toBeTruthy());

  test("period", () => expect(new Rate(1).period.equals(TimeSpan.SECOND)).toBeTruthy());

  test("period", () =>
    expect(new Rate(2).period.equals(TimeSpan.milliseconds(500))).toBeTruthy());

  test("sampleCount", () =>
    expect(new Rate(1).sampleCount(TimeSpan.SECOND)).toEqual(1));

  test("byteCount", () =>
    expect(new Rate(1).byteCount(TimeSpan.SECOND, Density.BIT64)).toEqual(8));

  test("span", () =>
    expect(new Rate(1).span(4).equals(TimeSpan.seconds(4))).toBeTruthy());

  test("byteSpan", () =>
    expect(
      new Rate(1).byteSpan(new Size(32), Density.BIT64).equals(TimeSpan.seconds(4))
    ).toBeTruthy());

  test("Hz", () => expect(Rate.hz(1).equals(1)).toBeTruthy());
  test("KHz", () => expect(Rate.khz(1).equals(1e3)).toBeTruthy());
});

describe("TimeRange", () => {
  test("construct", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr.start.equals(new TimeStamp(0))).toBeTruthy();
    expect(tr.end.equals(new TimeStamp(1000))).toBeTruthy();
  });

  test("span", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr.span.equals(TimeSpan.MICROSECOND)).toBeTruthy();
  });

  test("isValid", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr.isValid).toBeTruthy();
    const tr2 = new TimeRange(new TimeStamp(1000), new TimeStamp(0));
    expect(tr2.isValid).toBeFalsy();
  });

  test("isZero", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(0));
    expect(tr.isZero).toBeTruthy();
    const tr2 = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr2.isZero).toBeFalsy();
  });

  test("swap", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(
      tr.swap().equals(new TimeRange(new TimeStamp(1000), new TimeStamp(0)))
    ).toBeTruthy();
  });
  describe("contains", () => {
    test("TimeStamp", () => {
      const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
      expect(tr.contains(new TimeStamp(500))).toBeTruthy();
      expect(tr.contains(new TimeStamp(1001))).toBeFalsy();
    });
    test("TimeRange", () => {
      const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
      expect(
        tr.contains(new TimeRange(new TimeStamp(500), new TimeStamp(600)))
      ).toBeTruthy();
      expect(
        tr.contains(new TimeRange(new TimeStamp(500), new TimeStamp(1001)))
      ).toBeFalsy();
    });
  });
  test("overlapsWith", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    const one = new TimeRange(new TimeStamp(500), new TimeStamp(600));
    expect(tr.overlapsWith(one)).toBeTruthy();
    expect(one.overlapsWith(tr)).toBeTruthy();
    const two = new TimeRange(new TimeStamp(1001), new TimeStamp(2000));
    expect(tr.overlapsWith(two)).toBeFalsy();
    expect(two.overlapsWith(tr)).toBeFalsy();
  });
});

describe("DataType", () => {
  test("json serialization", () => {
    const dt = DataType.INT32;
    const v = JSON.parse(JSON.stringify({ dt }));
    expect(v.dt === "int32").toBeTruthy();
  });
});
