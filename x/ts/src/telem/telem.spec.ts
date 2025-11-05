// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { binary } from "@/binary";
import {
  addSamples,
  type CrudeDataType,
  DataType,
  Density,
  Rate,
  Size,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type TimeStampStringFormat,
} from "@/telem";

describe("TimeStamp", () => {
  test("construct", () => {
    const ts = new TimeStamp(1000);
    expect(ts.equals(TimeSpan.MICROSECOND)).toBe(true);
  });

  test("construct from NaN", () => {
    const ts = new TimeStamp(NaN);
    expect(ts.isZero).toBe(true);
  });

  test("construct from infinity", () => {
    const ts = new TimeStamp(Infinity);
    expect(ts.equals(TimeStamp.MAX)).toBe(true);
  });

  test("construct from negative infinity", () => {
    const ts = new TimeStamp(-Infinity);
    expect(ts.equals(TimeStamp.MIN)).toBe(true);
  });

  test("construct from CrudeValueExtension", () => {
    const ts = new TimeStamp({ value: 1000n });
    expect(ts.equals(1000)).toBe(true);
  });

  test("toString", () => {
    const ts = new TimeStamp(TimeSpan.days(90))
      .add(TimeSpan.minutes(20))
      .add(TimeSpan.milliseconds(283))
      .add(TimeSpan.microseconds(900));
    const tsString = ts.toString();
    expect(tsString).toEqual("1970-04-01T00:20:00.283Z");
  });

  test("encode", () => {
    const ts = TimeStamp.now();
    new binary.JSONCodec().encode(ts);
  });

  test("construct from TimeStamp", () => {
    const ts = new TimeStamp(TimeSpan.microseconds(10));
    expect(ts.equals(TimeSpan.microseconds(10))).toBe(true);
  });

  test("construct from local TimeZone", () => {
    const ts = new TimeStamp(TimeSpan.microseconds(10), "local");
    expect(ts.equals(TimeSpan.microseconds(10).add(TimeStamp.utcOffset))).toBe(true);
  });

  test("constructing from MIN and MAX as numbers", () => {
    expect(new TimeStamp(TimeStamp.MIN.nanoseconds).equals(TimeStamp.MIN)).toBe(true);
    expect(new TimeStamp(TimeStamp.MAX.nanoseconds).equals(TimeStamp.MAX)).toBe(true);
  });

  test("construct from time string", () => {
    const ts = new TimeStamp("12:30", "UTC");
    expect(ts.date().getUTCHours()).toEqual(12);
    expect(ts.equals(TimeSpan.hours(12).add(TimeSpan.minutes(30)))).toBe(true);
    const ts2 = new TimeStamp("12:30:00.22");
    expect(
      ts2.equals(
        TimeSpan.hours(12).add(TimeSpan.minutes(30).add(TimeSpan.milliseconds(22))),
      ),
    ).toBe(true);
    const ts3 = new TimeStamp("12:30:00.22", "local");
    expect(
      ts3.equals(
        TimeSpan.hours(12)
          .add(TimeSpan.minutes(30).add(TimeSpan.milliseconds(22)))
          .add(TimeStamp.utcOffset),
      ),
    ).toBe(true);
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
      Number(TimeStamp.utcOffset.valueOf() / TimeStamp.HOUR.valueOf()),
    );
    expect(ts2.date().getUTCMinutes()).toEqual(0);
  });

  describe("equals", () => {
    it("should return true when comparing two equal TimeStamps", () => {
      const ts1 = new TimeStamp(1000);
      const ts2 = new TimeStamp(1000);
      expect(ts1.equals(ts2)).toBe(true);
    });

    it("should handle an object with a 'value' property", () => {
      const ts1 = new TimeStamp(1000);
      const ts2 = { value: 1000n };
      expect(ts1.equals(ts2)).toBe(true);
    });

    it("should return false when comparing two different TimeStamps", () => {
      const ts1 = new TimeStamp(1000);
      const ts2 = new TimeStamp(2000);
      expect(ts1.equals(ts2)).toBe(false);
    });

    it("should handle comparison with bigint values", () => {
      const ts = new TimeStamp(1000n);
      expect(ts.equals(1000n)).toBe(true);
      expect(ts.equals(2000n)).toBe(false);
    });

    it("should handle comparison with number values", () => {
      const ts = new TimeStamp(1000);
      expect(ts.equals(1000)).toBe(true);
      expect(ts.equals(2000)).toBe(false);
    });

    it("should handle comparison with TimeSpan values", () => {
      const ts = new TimeStamp(TimeSpan.microseconds(500));
      expect(ts.equals(TimeSpan.microseconds(500))).toBe(true);
      expect(ts.equals(TimeSpan.microseconds(600))).toBe(false);
    });

    it("should handle comparison with another TimeStamp instance", () => {
      const ts1 = new TimeStamp(TimeSpan.seconds(10));
      const ts2 = new TimeStamp(TimeSpan.seconds(10));
      const ts3 = new TimeStamp(TimeSpan.seconds(20));
      expect(ts1.equals(ts2)).toBe(true);
      expect(ts1.equals(ts3)).toBe(false);
    });

    it("should handle comparison with Date objects", () => {
      const date = new Date("2024-01-15T10:30:00.000Z");
      const ts = new TimeStamp(date);
      expect(ts.equals(date)).toBe(true);

      const differentDate = new Date("2024-01-16T10:30:00.000Z");
      expect(ts.equals(differentDate)).toBe(false);
    });

    it("should handle comparison with DateComponents arrays", () => {
      const ts = new TimeStamp([2024, 3, 15], "UTC");
      const sameComponents: [number, number, number] = [2024, 3, 15];
      const differentComponents: [number, number, number] = [2024, 3, 16];

      expect(ts.equals(new TimeStamp(sameComponents, "UTC"))).toBe(true);
      expect(ts.equals(new TimeStamp(differentComponents, "UTC"))).toBe(false);
    });

    it("should handle comparison with string representations", () => {
      const ts = new TimeStamp("2021-01-01T00:00:00.000Z", "UTC");
      const sameString = "2021-01-01T00:00:00.000Z";
      const differentString = "2021-01-02T00:00:00.000Z";

      expect(ts.equals(new TimeStamp(sameString, "UTC"))).toBe(true);
      expect(ts.equals(new TimeStamp(differentString, "UTC"))).toBe(false);
    });

    it("should handle edge case comparisons", () => {
      // Zero values
      const zeroTs = new TimeStamp(0);
      expect(zeroTs.equals(0)).toBe(true);
      expect(zeroTs.equals(TimeStamp.ZERO)).toBe(true);
      expect(zeroTs.equals(0n)).toBe(true);

      // MAX values
      const maxTs = TimeStamp.MAX;
      expect(maxTs.equals(TimeStamp.MAX)).toBe(true);
      expect(maxTs.equals(new TimeStamp(Infinity))).toBe(true);

      // MIN values
      const minTs = TimeStamp.MIN;
      expect(minTs.equals(TimeStamp.MIN)).toBe(true);
      expect(minTs.equals(new TimeStamp(-Infinity))).toBe(true);
    });

    it("should handle NaN values", () => {
      const nanTs = new TimeStamp(NaN);
      expect(nanTs.equals(0)).toBe(true); // NaN is converted to zero
      expect(nanTs.equals(TimeStamp.ZERO)).toBe(true);
    });

    it("should handle negative values", () => {
      const negativeTs = new TimeStamp(-1000);
      expect(negativeTs.equals(-1000)).toBe(true);
      expect(negativeTs.equals(-1000n)).toBe(true);
      expect(negativeTs.equals(new TimeStamp(-1000))).toBe(true);
      expect(negativeTs.equals(1000)).toBe(false);
    });

    it("should handle large bigint values", () => {
      const largeValue = 9007199254740992n; // Larger than MAX_SAFE_INTEGER
      const ts = new TimeStamp(largeValue);
      expect(ts.equals(largeValue)).toBe(true);
      expect(ts.equals(largeValue + 1n)).toBe(false);
    });

    it("should handle mixed precision comparisons", () => {
      // TimeStamp created from milliseconds vs microseconds
      const msTs = new TimeStamp(TimeSpan.milliseconds(1));
      const usTs = new TimeStamp(TimeSpan.microseconds(1000));
      expect(msTs.equals(usTs)).toBe(true);

      const differentUsTs = new TimeStamp(TimeSpan.microseconds(999));
      expect(msTs.equals(differentUsTs)).toBe(false);
    });

    it("should maintain transitivity", () => {
      // If a.equals(b) and b.equals(c), then a.equals(c)
      const a = new TimeStamp(1000);
      const b = new TimeStamp(TimeSpan.nanoseconds(1000));
      const c = new TimeStamp(1000n);

      expect(a.equals(b)).toBe(true);
      expect(b.equals(c)).toBe(true);
      expect(a.equals(c)).toBe(true);
    });

    it("should maintain reflexivity", () => {
      // a.equals(a) should always be true
      const ts = new TimeStamp(TimeSpan.seconds(42));
      expect(ts.equals(ts)).toBe(true);
    });

    it("should maintain symmetry", () => {
      // If a.equals(b), then b.equals(a)
      const ts1 = new TimeStamp(1000);
      const ts2 = new TimeStamp(1000);

      expect(ts1.equals(ts2)).toBe(true);
      expect(ts2.equals(ts1)).toBe(true);

      const ts3 = new TimeStamp(2000);
      expect(ts1.equals(ts3)).toBe(false);
      expect(ts3.equals(ts1)).toBe(false);
    });

    it("should handle comparison with complex TimeSpan arithmetic", () => {
      const ts = new TimeStamp(
        TimeSpan.hours(1).add(TimeSpan.minutes(30)).add(TimeSpan.seconds(45)),
      );
      const equivalentSpan = TimeSpan.seconds(5445); // 1h 30m 45s = 5445s
      expect(ts.equals(equivalentSpan)).toBe(true);
    });
  });

  describe("schema", () => {
    it("should parse bigint", () => {
      const ts = TimeStamp.z.parse(1000000000n);
      expect(ts).toBeInstanceOf(TimeStamp);
      expect(ts.valueOf()).toBe(1000000000n);
    });

    it("should parse Date object", () => {
      const date = new Date("2024-01-15T10:30:00.000Z");
      const ts = TimeStamp.z.parse(date);
      expect(ts).toBeInstanceOf(TimeStamp);
      expect(ts.valueOf()).toBe(BigInt(date.getTime()) * 1000000n);
    });

    it("should parse TimeSpan", () => {
      const span = new TimeSpan(5000000000n);
      const ts = TimeStamp.z.parse(span);
      expect(ts).toBeInstanceOf(TimeStamp);
      expect(ts.valueOf()).toBe(5000000000n);
    });

    it("should parse DateComponents array", () => {
      const ts = TimeStamp.z.parse([2024, 3, 15]);
      expect(ts).toBeInstanceOf(TimeStamp);
      const expected = new TimeStamp([2024, 3, 15]);
      expect(ts.valueOf()).toBe(expected.valueOf());
    });

    it("should parse DateComponents with missing elements", () => {
      const ts1 = TimeStamp.z.parse([2024]);
      expect(ts1).toBeInstanceOf(TimeStamp);
      expect(ts1.valueOf()).toBe(new TimeStamp([2024, 1, 1]).valueOf());

      const ts2 = TimeStamp.z.parse([2024, 6]);
      expect(ts2).toBeInstanceOf(TimeStamp);
      expect(ts2.valueOf()).toBe(new TimeStamp([2024, 6, 1]).valueOf());
    });

    it("should parse string representation of bigint", () => {
      const ts = TimeStamp.z.parse("123456789000");
      expect(ts).toBeInstanceOf(TimeStamp);
      expect(ts.valueOf()).toBe(123456789000n);
    });

    it("should parse number", () => {
      const ts = TimeStamp.z.parse(987654321);
      expect(ts).toBeInstanceOf(TimeStamp);
      expect(ts.valueOf()).toBe(987654321n);
    });

    it("should parse object with value property", () => {
      const ts = TimeStamp.z.parse({ value: 555555555n });
      expect(ts).toBeInstanceOf(TimeStamp);
      expect(ts.valueOf()).toBe(555555555n);
    });

    it("should pass through existing TimeStamp instance", () => {
      const original = new TimeStamp(777777777n);
      const ts = TimeStamp.z.parse(original);
      expect(ts).toBe(original);
      expect(ts.valueOf()).toBe(777777777n);
    });

    it("should handle edge cases", () => {
      const ts1 = TimeStamp.z.parse(0);
      expect(ts1.valueOf()).toBe(0n);

      const ts2 = TimeStamp.z.parse(Number.MAX_SAFE_INTEGER);
      expect(ts2.valueOf()).toBe(BigInt(Number.MAX_SAFE_INTEGER));
    });
  });

  test("span", () => {
    const ts = new TimeStamp(0);
    expect(ts.span(new TimeStamp(1000)).equals(TimeSpan.microseconds())).toBe(true);
  });

  test("range", () => {
    const ts = new TimeStamp(0);
    expect(
      ts.range(new TimeStamp(1000)).equals(new TimeRange(ts, TimeSpan.microseconds())),
    ).toBe(true);
  });

  test("spanRange", () => {
    const ts = new TimeStamp(0);
    expect(
      ts
        .spanRange(TimeSpan.microseconds())
        .equals(new TimeRange(ts, ts.add(TimeSpan.microseconds()))),
    ).toBe(true);
  });

  test("isZero", () => {
    const ts = new TimeStamp(0);
    expect(ts.isZero).toBe(true);
  });

  test("after", () => {
    const ts = new TimeStamp(0);
    expect(ts.after(new TimeStamp(-1))).toBe(true);
    const ts2 = new TimeStamp(1);
    expect(ts2.after(new TimeStamp(1))).toBe(false);
  });

  test("before", () => {
    const ts = new TimeStamp(0);
    expect(ts.before(new TimeStamp(1))).toBe(true);
    const ts2 = new TimeStamp(1);
    expect(ts2.before(new TimeStamp(1))).toBe(false);
  });

  test("beforeEq", () => {
    const ts = new TimeStamp(0);
    expect(ts.beforeEq(new TimeStamp(1))).toBe(true);
    const ts2 = new TimeStamp(1);
    expect(ts2.beforeEq(new TimeStamp(1))).toBe(true);
    const ts3 = new TimeStamp(2);
    expect(ts3.beforeEq(new TimeStamp(1))).toBe(false);
  });

  test("afterEq", () => {
    const ts = new TimeStamp(0);
    expect(ts.afterEq(new TimeStamp(-1))).toBe(true);
    const ts2 = new TimeStamp(1);
    expect(ts2.afterEq(new TimeStamp(1))).toBe(true);
    const ts3 = new TimeStamp(0);
    expect(ts3.afterEq(new TimeStamp(1))).toBe(false);
  });

  test("add", () => {
    const ts = new TimeStamp(0);
    expect(
      ts.add(TimeSpan.microseconds()).equals(new TimeStamp(TimeSpan.microseconds(1))),
    ).toBe(true);
  });

  test("sub", () => {
    const ts = new TimeStamp(TimeSpan.microseconds());
    expect(ts.sub(TimeSpan.microseconds()).equals(new TimeStamp(0))).toBe(true);
  });

  describe("arithmetic operations", () => {
    test("add with TimeSpan", () => {
      const ts = new TimeStamp(1000);
      const result = ts.add(TimeSpan.microseconds(500));
      expect(result).toBeInstanceOf(TimeStamp);
      expect(result.valueOf()).toBe(501000n);
    });

    test("add with number", () => {
      const ts = new TimeStamp(1000);
      const result = ts.add(500);
      expect(result).toBeInstanceOf(TimeStamp);
      expect(result.valueOf()).toBe(1500n);
    });

    test("add with bigint", () => {
      const ts = new TimeStamp(1000n);
      const result = ts.add(500n);
      expect(result).toBeInstanceOf(TimeStamp);
      expect(result.valueOf()).toBe(1500n);
    });

    test("sub with TimeSpan", () => {
      const ts = new TimeStamp(1000);
      const result = ts.sub(TimeSpan.nanoseconds(300));
      expect(result).toBeInstanceOf(TimeStamp);
      expect(result.valueOf()).toBe(700n);
    });

    test("sub with number", () => {
      const ts = new TimeStamp(1000);
      const result = ts.sub(300);
      expect(result).toBeInstanceOf(TimeStamp);
      expect(result.valueOf()).toBe(700n);
    });

    test("sub with bigint", () => {
      const ts = new TimeStamp(1000n);
      const result = ts.sub(300n);
      expect(result).toBeInstanceOf(TimeStamp);
      expect(result.valueOf()).toBe(700n);
    });
  });

  describe("toString with formats", () => {
    const ts = new TimeStamp([2022, 12, 15], "UTC")
      .add(TimeSpan.hours(12))
      .add(TimeSpan.minutes(20))
      .add(TimeSpan.milliseconds(12));

    const FORMAT_TESTS: [TimeStampStringFormat, string][] = [
      ["ISO", "2022-12-15T12:20:00.012Z"],
      ["ISODate", "2022-12-15"],
      ["ISOTime", "12:20:00.012"],
      ["time", "12:20:00"],
      ["preciseTime", "12:20:00.012"],
      ["date", "Dec 15"],
      ["preciseDate", "Dec 15 12:20:00.012"],
      ["dateTime", "Dec 15 12:20:00"],
    ];

    FORMAT_TESTS.forEach(([format, expected]) => {
      test(`should format timestamp as ${format}`, () => {
        expect(ts.toString(format, "UTC")).toEqual(expected);
      });
    });
  });

  describe("unit getters", () => {
    test("hour", () => {
      expect(TimeStamp.hours(1).add(TimeSpan.minutes(30)).hour).toEqual(1);
    });
    test("hours", () => {
      expect(TimeStamp.hours(1).add(TimeSpan.minutes(30)).hours).toEqual(1.5);
    });
    test("minute", () => {
      expect(TimeStamp.minutes(1).add(TimeStamp.seconds(20)).minute).toEqual(1);
    });
    test("minutes", () => {
      expect(TimeStamp.minutes(1).add(TimeStamp.seconds(30)).minutes).toEqual(1.5);
    });
    test("second", () => {
      expect(TimeStamp.seconds(1).add(TimeStamp.milliseconds(20)).second).toEqual(1);
    });
    test("seconds", () => {
      expect(TimeStamp.seconds(1).add(TimeStamp.milliseconds(500)).seconds).toEqual(
        1.5,
      );
    });
    test("millisecond", () => {
      expect(
        TimeStamp.milliseconds(1).add(TimeStamp.microseconds(20)).millisecond,
      ).toEqual(1);
    });
    test("milliseconds", () => {
      expect(
        TimeStamp.milliseconds(1).add(TimeStamp.microseconds(500)).milliseconds,
      ).toEqual(1.5);
    });
    test("microseconds", () => {
      expect(
        TimeStamp.microseconds(500).add(TimeSpan.nanoseconds(20)).microseconds,
      ).toEqual(500.02);
    });
    test("nanoseconds", () => {
      expect(
        TimeStamp.microseconds(1).add(TimeSpan.nanoseconds(30)).nanoseconds,
      ).toEqual(1030);
    });
    test("year", () => {
      expect(new TimeStamp([2022, 12, 15]).year).toEqual(2022);
    });
    test("month", () => {
      expect(new TimeStamp([2022, 12, 15]).month).toEqual(11);
    });
    test("day", () => {
      expect(new TimeStamp([2022, 12, 15], "UTC").day).toEqual(15);
    });
  });

  describe("unit setters", () => {
    test("setYear", () => {
      const ts = new TimeStamp([2022, 12, 15]);
      const updated = ts.setYear(2023);
      expect(updated.year).toEqual(2023);
      expect(updated.month).toEqual(ts.month); // Other components should remain unchanged
      expect(updated.day).toEqual(ts.day);
    });

    test("setMonth", () => {
      const ts = new TimeStamp([2022, 12, 15]);
      const updated = ts.setMonth(5); // June (0-indexed)
      expect(updated.month).toEqual(5);
      expect(updated.year).toEqual(ts.year); // Other components should remain unchanged
      expect(updated.day).toEqual(ts.day);
    });

    test("setDay", () => {
      const ts = new TimeStamp([2022, 12, 15]);
      const updated = ts.setDay(20);
      expect(updated.day).toEqual(20);
      expect(updated.year).toEqual(ts.year);
      expect(updated.month).toEqual(ts.month);
    });

    test("setHour", () => {
      const ts = new TimeStamp([2022, 12, 15]).add(TimeSpan.hours(10));
      const updated = ts.setHour(15);
      expect(updated.hour).toEqual(15);
      expect(updated.year).toEqual(ts.year);
      expect(updated.month).toEqual(ts.month);
      expect(updated.day).toEqual(ts.day);
    });

    test("setMinute", () => {
      const ts = new TimeStamp([2022, 12, 15])
        .add(TimeSpan.hours(10))
        .add(TimeSpan.minutes(30));
      const updated = ts.setMinute(45);
      expect(updated.minute).toEqual(45);
      expect(updated.hour).toEqual(ts.hour);
      expect(updated.year).toEqual(ts.year);
      expect(updated.month).toEqual(ts.month);
      expect(updated.day).toEqual(ts.day);
    });

    test("setSecond", () => {
      const ts = new TimeStamp([2022, 12, 15])
        .add(TimeSpan.hours(10))
        .add(TimeSpan.minutes(30))
        .add(TimeSpan.seconds(20));
      const updated = ts.setSecond(45);
      expect(updated.second).toEqual(45);
      expect(updated.minute).toEqual(ts.minute); // Other components should remain unchanged
      expect(updated.hour).toEqual(ts.hour);
      expect(updated.year).toEqual(ts.year);
      expect(updated.month).toEqual(ts.month);
      expect(updated.day).toEqual(ts.day);
    });

    test("setMillisecond", () => {
      const ts = new TimeStamp([2022, 12, 15])
        .add(TimeSpan.hours(10))
        .add(TimeSpan.minutes(30))
        .add(TimeSpan.seconds(20))
        .add(TimeSpan.milliseconds(100));
      const updated = ts.setMillisecond(500);
      expect(updated.millisecond).toEqual(500);
      expect(updated.second).toEqual(ts.second); // Other components should remain unchanged
      expect(updated.minute).toEqual(ts.minute);
      expect(updated.hour).toEqual(ts.hour);
      expect(updated.year).toEqual(ts.year);
      expect(updated.month).toEqual(ts.month);
      expect(updated.day).toEqual(ts.day);
    });

    test("localHour", () => {
      const ts = new TimeStamp([2022, 12, 15]).add(TimeSpan.hours(10));
      const localHour = ts.localHour;
      const expectedLocalHour = ts.date().getHours();
      expect(localHour).toEqual(expectedLocalHour);
      const utcHour = ts.hour;
      const tzOffsetHours = new Date().getTimezoneOffset() / 60;
      if (tzOffsetHours !== 0) expect(localHour).not.toEqual(utcHour);
    });

    test("setLocalHour", () => {
      const ts = new TimeStamp([2022, 12, 15])
        .add(TimeSpan.hours(10))
        .add(TimeSpan.minutes(30))
        .add(TimeSpan.seconds(20));
      const targetLocalHour = 15;
      const updated = ts.setLocalHour(targetLocalHour);
      expect(updated.localHour).toEqual(targetLocalHour);
      expect(updated.date().getMinutes()).toEqual(ts.date().getMinutes());
      expect(updated.date().getSeconds()).toEqual(ts.date().getSeconds());
      expect(updated.year).toEqual(ts.year);
      expect(updated.month).toEqual(ts.month);
      expect(updated.day).toEqual(ts.day);
    });

    test("setLocalHour edge cases", () => {
      // Test setting hour to 0 (midnight)
      const ts1 = new TimeStamp([2022, 12, 15]).add(TimeSpan.hours(12));
      const midnight = ts1.setLocalHour(0);
      expect(midnight.localHour).toEqual(0);

      // Test setting hour to 23 (11 PM)
      const ts2 = new TimeStamp([2022, 12, 15]).add(TimeSpan.hours(12));
      const elevenPM = ts2.setLocalHour(23);
      expect(elevenPM.localHour).toEqual(23);

      // Test that setting local hour might change the day in UTC
      const ts3 = new TimeStamp([2022, 12, 15]);
      const updated = ts3.setLocalHour(23);
      // Depending on timezone, this might be a different day in UTC
      expect(updated.localHour).toEqual(23);
    });

    test("localHour and setLocalHour round trip", () => {
      // Test that getting and setting local hour is consistent
      const ts = new TimeStamp([2022, 12, 15])
        .add(TimeSpan.hours(10))
        .add(TimeSpan.minutes(30));

      const originalLocalHour = ts.localHour;
      const roundTrip = ts.setLocalHour(originalLocalHour);

      // The timestamp should remain effectively the same
      expect(roundTrip.localHour).toEqual(originalLocalHour);
      expect(roundTrip.date().getMinutes()).toEqual(ts.date().getMinutes());
      expect(roundTrip.date().getSeconds()).toEqual(ts.date().getSeconds());

      // The valueOf() might differ slightly due to milliseconds precision
      // but should be within 1 second
      const diff = Math.abs(Number(roundTrip.valueOf() - ts.valueOf()));
      expect(diff).toBeLessThan(1000000000); // Less than 1 second in nanoseconds
    });
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
          remainder,
        ).toString()}`,
      ).toBe(true);
    });
    test("second", () => {
      const expectedRemainder = TimeSpan.milliseconds(12);
      const ts = new TimeStamp([2022, 12, 15])
        .add(TimeStamp.hours(12))
        .add(TimeSpan.minutes(20))
        .add(TimeSpan.milliseconds(12));
      const remainder = ts.remainder(TimeSpan.seconds());
      expect(remainder.equals(expectedRemainder)).toBe(true);
    });
  });

  describe("sort", () => {
    interface Spec {
      a: TimeStamp;
      b: TimeStamp;
      expected: number;
    }
    const TESTS: Spec[] = [
      {
        a: TimeStamp.seconds(3),
        b: TimeStamp.seconds(2),
        expected: TimeSpan.seconds(1).nanoseconds,
      },
      {
        a: TimeStamp.seconds(2),
        b: TimeStamp.seconds(3),
        expected: TimeSpan.seconds(-1).nanoseconds,
      },
      { a: TimeStamp.seconds(2), b: TimeStamp.seconds(2), expected: 0 },
    ];
    TESTS.forEach(({ a, b, expected }) => {
      test(`TimeStamp.sort(${a.toString()}, ${b.toString()}) = ${expected}`, () => {
        expect(TimeStamp.sort(a, b)).toEqual(expected);
      });
    });
  });

  describe("formatBySpan", () => {
    test("should return 'shortDate' for spans >= 30 days", () => {
      const ts = new TimeStamp([2022, 12, 15], "UTC");
      const span = TimeSpan.days(30);
      expect(ts.formatBySpan(span)).toBe("shortDate");
    });

    test("should return 'dateTime' for spans >= 1 day", () => {
      const ts = new TimeStamp([2022, 12, 15], "UTC");
      const span = TimeSpan.days(1);
      expect(ts.formatBySpan(span)).toBe("dateTime");
    });

    test("should return 'time' for spans >= 1 hour", () => {
      const ts = new TimeStamp([2022, 12, 15], "UTC");
      const span = TimeSpan.hours(1);
      expect(ts.formatBySpan(span)).toBe("time");
    });

    test("should return 'preciseTime' for spans >= 1 second", () => {
      const ts = new TimeStamp([2022, 12, 15], "UTC");
      const span = TimeSpan.seconds(1);
      expect(ts.formatBySpan(span)).toBe("preciseTime");
    });

    test("should return 'ISOTime' for spans < 1 second", () => {
      const ts = new TimeStamp([2022, 12, 15], "UTC");
      const span = TimeSpan.milliseconds(500);
      expect(ts.formatBySpan(span)).toBe("ISOTime");
    });

    test("should work with very small spans", () => {
      const ts = new TimeStamp([2022, 12, 15], "UTC");
      const span = TimeSpan.microseconds(100);
      expect(ts.formatBySpan(span)).toBe("ISOTime");
    });
  });
});

describe("TimeSpan", () => {
  test("construct from static", () => {
    expect(TimeSpan.nanoseconds(1).equals(1)).toBe(true);
    expect(TimeSpan.microseconds(1).equals(1000)).toBe(true);
    expect(TimeSpan.milliseconds(1).equals(1000000)).toBe(true);
    expect(TimeSpan.seconds(1).equals(1e9)).toBe(true);
    expect(TimeSpan.minutes(1).equals(6e10)).toBe(true);
    expect(TimeSpan.hours(1).equals(36e11)).toBe(true);
  });

  test("construct from CrudeValueExtension", () => {
    const ts = new TimeSpan({ value: 1000n });
    expect(ts.equals(1000)).toBe(true);
  });

  describe("fromMilliseconds", () => {
    it("should interpret a pure number or bigint as milliseconds", () => {
      const ts = TimeSpan.fromMilliseconds(1000);
      expect(ts.equals(TimeSpan.seconds())).toBe(true);
    });
    it("should interpret a TimeSpan as a normal TimeSpan", () => {
      const ts = TimeSpan.fromMilliseconds(TimeSpan.milliseconds(30));
      expect(ts.equals(TimeSpan.milliseconds(30))).toBe(true);
    });
  });

  describe("fromSeconds", () => {
    it("should interpret a pure number or bigint as seconds", () => {
      const ts = TimeSpan.fromSeconds(1);
      expect(ts.equals(TimeSpan.SECOND)).toBe(true);
    });
    it("should interpret a TimeSpan as a normal TimeSpan", () => {
      const ts = TimeSpan.fromSeconds(TimeSpan.milliseconds(30));
      expect(ts.equals(TimeSpan.milliseconds(30))).toBe(true);
    });
  });

  test("seconds", () => {
    expect(TimeSpan.seconds(1).seconds).toEqual(1);
  });

  test("isZero", () => {
    expect(TimeSpan.ZERO.isZero).toBe(true);
    expect(TimeSpan.seconds(1).isZero).toBe(false);
  });

  test("add", () => {
    expect(TimeSpan.seconds(1).add(TimeSpan.SECOND).equals(2e9)).toBe(true);
  });

  test("sub", () => {
    expect(TimeSpan.seconds(1).sub(TimeSpan.SECOND).isZero).toBe(true);
  });

  describe("arithmetic operations", () => {
    test("add with TimeSpan", () => {
      const ts1 = new TimeSpan(1000);
      const ts2 = new TimeSpan(500);
      const result = ts1.add(ts2);
      expect(result).toBeInstanceOf(TimeSpan);
      expect(result.valueOf()).toBe(1500n);
    });

    test("add with number", () => {
      const ts = new TimeSpan(1000);
      const result = ts.add(500);
      expect(result).toBeInstanceOf(TimeSpan);
      expect(result.valueOf()).toBe(1500n);
    });

    test("add with bigint", () => {
      const ts = new TimeSpan(1000n);
      const result = ts.add(500n);
      expect(result).toBeInstanceOf(TimeSpan);
      expect(result.valueOf()).toBe(1500n);
    });

    test("sub with TimeSpan", () => {
      const ts1 = new TimeSpan(1000);
      const ts2 = new TimeSpan(300);
      const result = ts1.sub(ts2);
      expect(result).toBeInstanceOf(TimeSpan);
      expect(result.valueOf()).toBe(700n);
    });

    test("sub with number", () => {
      const ts = new TimeSpan(1000);
      const result = ts.sub(300);
      expect(result).toBeInstanceOf(TimeSpan);
      expect(result.valueOf()).toBe(700n);
    });

    test("sub with bigint", () => {
      const ts = new TimeSpan(1000n);
      const result = ts.sub(300n);
      expect(result).toBeInstanceOf(TimeSpan);
      expect(result.valueOf()).toBe(700n);
    });

    test("mult", () => {
      const ts = new TimeSpan(1000);
      const result = ts.mult(2);
      expect(result).toBeInstanceOf(TimeSpan);
      expect(result.valueOf()).toBe(2000n);
    });

    test("mult with decimal", () => {
      const ts = new TimeSpan(1000);
      const result = ts.mult(0.5);
      expect(result).toBeInstanceOf(TimeSpan);
      expect(result.valueOf()).toBe(500n);
    });

    test("mult with negative", () => {
      const ts = new TimeSpan(1000);
      const result = ts.mult(-2);
      expect(result).toBeInstanceOf(TimeSpan);
      expect(result.valueOf()).toBe(-2000n);
    });

    test("div", () => {
      const ts = new TimeSpan(1000);
      const result = ts.div(2);
      expect(result).toBeInstanceOf(TimeSpan);
      expect(result.valueOf()).toBe(500n);
    });

    test("div with decimal", () => {
      const ts = new TimeSpan(1000);
      const result = ts.div(0.5);
      expect(result).toBeInstanceOf(TimeSpan);
      expect(result.valueOf()).toBe(2000n);
    });

    test("div resulting in fractional nanoseconds truncates", () => {
      const ts = new TimeSpan(1001);
      const result = ts.div(2);
      expect(result).toBeInstanceOf(TimeSpan);
      expect(result.valueOf()).toBe(500n); // 1001/2 = 500.5, truncated to 500
    });
  });

  const TRUNCATE_TESTS = [
    [TimeSpan.days(1).add(TimeSpan.nanoseconds(50)), TimeSpan.DAY, TimeSpan.days(1)],
    [TimeSpan.hours(1).add(TimeSpan.minutes(30)), TimeSpan.HOUR, TimeSpan.hours(1)],
  ];

  test("truncate", () => {
    TRUNCATE_TESTS.forEach(([ts, unit, expected]) => {
      expect(
        ts.truncate(unit).equals(expected),
        `expected ${expected.toString()} got ${ts.truncate(unit).toString()}`,
      ).toBe(true);
    });
  });

  const REMAINDER_TESTS = [
    [
      TimeSpan.days(1).add(TimeSpan.nanoseconds(50)),
      TimeSpan.DAY,
      TimeSpan.nanoseconds(50),
    ],
    [TimeSpan.hours(1).add(TimeSpan.minutes(30)), TimeSpan.HOUR, TimeSpan.minutes(30)],
  ];

  test("remainder", () => {
    REMAINDER_TESTS.forEach(([ts, unit, expected]) => {
      expect(ts.remainder(unit).equals(expected)).toBe(true);
    });
  });

  const TO_STRING_TESTS = [
    [TimeSpan.nanoseconds(1), "1ns"],
    [TimeSpan.microseconds(1), "1µs"],
    [TimeSpan.milliseconds(1), "1ms"],
    [TimeSpan.seconds(1), "1s"],
    [TimeSpan.minutes(1), "1m"],
    [TimeSpan.hours(1), "1h"],
    [TimeSpan.days(1), "1d"],
    [
      TimeSpan.milliseconds(1)
        .add(TimeSpan.microseconds(500))
        .add(TimeSpan.nanoseconds(50)),
      "1ms 500µs 50ns",
    ],
    [TimeSpan.seconds(1).add(TimeSpan.microseconds(500)), "1s 500µs"],
  ];

  test("toString", () => {
    TO_STRING_TESTS.forEach(([ts, expected]) => {
      expect(ts.toString()).toEqual(expected);
    });
  });

  describe("toString with semantic format", () => {
    const TESTS: [TimeSpan, string][] = [
      // Sub-second durations
      [TimeSpan.ZERO, "0s"],
      [TimeSpan.nanoseconds(50), "< 1s"],
      [TimeSpan.microseconds(50), "< 1s"],
      [TimeSpan.milliseconds(50), "< 1s"],
      [TimeSpan.milliseconds(999), "< 1s"],

      // Seconds
      [TimeSpan.seconds(1), "1s"],
      [TimeSpan.seconds(30), "30s"],
      [TimeSpan.seconds(59), "59s"],

      // Minutes with seconds (< 5 minutes)
      [TimeSpan.seconds(60), "1m"],
      [TimeSpan.seconds(90), "1m 30s"],
      [TimeSpan.seconds(119), "1m 59s"],
      [TimeSpan.seconds(120), "2m"],
      [TimeSpan.seconds(150), "2m 30s"],
      [TimeSpan.seconds(240), "4m"],
      [TimeSpan.seconds(270), "4m 30s"],

      // Minutes without seconds (>= 5 minutes)
      [TimeSpan.seconds(300), "5m"],
      [TimeSpan.seconds(330), "5m"], // seconds dropped
      [TimeSpan.minutes(30), "30m"],
      [TimeSpan.minutes(59), "59m"],

      // Hours with minutes (< 3 hours)
      [TimeSpan.minutes(60), "1h"],
      [TimeSpan.minutes(90), "1h 30m"],
      [TimeSpan.minutes(119), "1h 59m"],
      [TimeSpan.minutes(120), "2h"],
      [TimeSpan.minutes(150), "2h 30m"],
      [TimeSpan.minutes(179), "2h 59m"],

      // Hours without minutes (>= 3 hours)
      [TimeSpan.minutes(180), "3h"],
      [TimeSpan.minutes(195), "3h"], // minutes dropped
      [TimeSpan.hours(12), "12h"],
      [TimeSpan.hours(23), "23h"],

      // Days with hours (< 2 days)
      [TimeSpan.hours(24), "1d"],
      [TimeSpan.hours(25), "1d 1h"],
      [TimeSpan.hours(36), "1d 12h"],
      [TimeSpan.hours(47), "1d 23h"],

      // Days without hours (>= 2 days)
      [TimeSpan.hours(48), "2d"],
      [TimeSpan.hours(50), "2d"], // hours dropped
      [TimeSpan.days(3), "3d"],
      [TimeSpan.days(6), "6d"],

      // Weeks with days (< 2 weeks)
      [TimeSpan.days(7), "1w"],
      [TimeSpan.days(8), "1w 1d"],
      [TimeSpan.days(10), "1w 3d"],
      [TimeSpan.days(13), "1w 6d"],

      // Weeks without days (>= 2 weeks)
      [TimeSpan.days(14), "2w"],
      [TimeSpan.days(15), "2w"], // days dropped
      [TimeSpan.days(21), "3w"],
      [TimeSpan.days(28), "4w"],

      // Months with days (< 3 months)
      [TimeSpan.days(30), "1mo"],
      [TimeSpan.days(35), "1mo 5d"],
      [TimeSpan.days(45), "1mo 15d"],
      [TimeSpan.days(60), "2mo"],
      [TimeSpan.days(75), "2mo 15d"],

      // Months without days (>= 3 months)
      [TimeSpan.days(90), "3mo"],
      [TimeSpan.days(95), "3mo"], // days dropped
      [TimeSpan.days(180), "6mo"],
      [TimeSpan.days(330), "11mo"],

      // Years with months (< 2 years)
      [TimeSpan.days(364), "12mo"],
      [TimeSpan.days(365), "1y"],
      [TimeSpan.days(395), "1y 1mo"],
      [TimeSpan.days(500), "1y 4mo"],
      [TimeSpan.days(700), "1y 11mo"],

      // Years without months (>= 2 years)
      [TimeSpan.days(730), "2y"],
      [TimeSpan.days(750), "2y"], // months dropped
      [TimeSpan.days(1095), "3y"],
      [TimeSpan.days(3650), "10y"],

      // Complex durations
      [TimeSpan.seconds(3661), "1h 1m"],
      [TimeSpan.minutes(1441), "1d"], // 24h 1m, but minutes are dropped at day level
      [TimeSpan.minutes(1500), "1d 1h"], // 25h exactly
      [TimeSpan.hours(169), "1w 1h"],

      // Negative durations
      [TimeSpan.seconds(-30), "-30s"],
      [TimeSpan.minutes(-90), "-1h 30m"],
      [TimeSpan.hours(-25), "-1d 1h"],
      [TimeSpan.days(-8), "-1w 1d"],
      [TimeSpan.days(-400), "-1y 1mo"],
    ];

    TESTS.forEach(([ts, expected]) => {
      test(`${ts.valueOf()} => ${expected}`, () => {
        expect(ts.toString("semantic")).toEqual(expected);
      });
    });
  });

  describe("toString with format", () => {
    test("toString with semantic format", () => {
      const ts = TimeSpan.hours(25);
      expect(ts.toString("semantic")).toEqual("1d 1h");
    });

    test("toString with default format", () => {
      const ts = TimeSpan.hours(25);
      expect(ts.toString()).toEqual("1d 1h");
    });

    test("toString with full format", () => {
      const ts = TimeSpan.hours(25).add(TimeSpan.minutes(30)).add(TimeSpan.seconds(15));
      expect(ts.toString("full")).toEqual("1d 1h 30m 15s");
    });
  });

  describe("schema", () => {
    it("should parse bigint", () => {
      const ts = TimeSpan.z.parse(1000000000n);
      expect(ts).toBeInstanceOf(TimeSpan);
      expect(ts.valueOf()).toBe(1000000000n);
    });

    it("should parse number", () => {
      const ts = TimeSpan.z.parse(987654321);
      expect(ts).toBeInstanceOf(TimeSpan);
      expect(ts.valueOf()).toBe(987654321n);
    });

    it("should parse string representation of bigint", () => {
      const ts = TimeSpan.z.parse("123456789000");
      expect(ts).toBeInstanceOf(TimeSpan);
      expect(ts.valueOf()).toBe(123456789000n);
    });

    it("should parse object with value property", () => {
      const ts = TimeSpan.z.parse({ value: 555555555n });
      expect(ts).toBeInstanceOf(TimeSpan);
      expect(ts.valueOf()).toBe(555555555n);
    });

    it("should pass through existing TimeSpan instance", () => {
      const original = new TimeSpan(777777777n);
      const ts = TimeSpan.z.parse(original);
      expect(ts).toStrictEqual(original);
      expect(ts.valueOf()).toBe(777777777n);
    });

    it("should parse TimeStamp", () => {
      const timestamp = new TimeStamp(999999999n);
      const ts = TimeSpan.z.parse(timestamp);
      expect(ts).toBeInstanceOf(TimeSpan);
      expect(ts.valueOf()).toBe(999999999n);
    });

    it("should parse Rate", () => {
      const rate = new Rate(100);
      const ts = TimeSpan.z.parse(rate);
      expect(ts).toBeInstanceOf(TimeSpan);
      // The schema transforms Rate to TimeSpan using new TimeSpan(rate)
      // which uses rate.valueOf() directly (100) as nanoseconds
      expect(ts.valueOf()).toBe(100n);
    });

    it("should handle edge cases", () => {
      const ts1 = TimeSpan.z.parse(0);
      expect(ts1.valueOf()).toBe(0n);

      const ts2 = TimeSpan.z.parse(Number.MAX_SAFE_INTEGER);
      expect(ts2.valueOf()).toBe(BigInt(Number.MAX_SAFE_INTEGER));
    });
  });
});

describe("Rate", () => {
  test("construct", () => expect(new Rate(1).equals(1)).toBe(true));

  test("period", () => expect(new Rate(1).period.equals(TimeSpan.SECOND)).toBe(true));

  test("period", () =>
    expect(new Rate(2).period.equals(TimeSpan.milliseconds(500))).toBe(true));

  test("sampleCount", () =>
    expect(new Rate(1).sampleCount(TimeSpan.SECOND)).toEqual(1));

  test("byteCount", () =>
    expect(new Rate(1).byteCount(TimeSpan.SECOND, Density.BIT64)).toEqual(8));

  test("span", () =>
    expect(new Rate(1).span(4).equals(TimeSpan.seconds(4))).toBe(true));

  test("byteSpan", () =>
    expect(
      new Rate(1).byteSpan(new Size(32), Density.BIT64).equals(TimeSpan.seconds(4)),
    ).toBe(true));

  test("Hz", () => expect(Rate.hz(1).equals(1)).toBe(true));
  test("KHz", () => expect(Rate.khz(1).equals(1e3)).toBe(true));

  describe("schema", () => {
    it("should parse number", () => {
      const rate = Rate.z.parse(50);
      expect(rate).toBeInstanceOf(Rate);
      expect(rate.valueOf()).toBe(50);
    });

    it("should pass through existing Rate instance", () => {
      const original = new Rate(100);
      const rate = Rate.z.parse(original);
      expect(rate).toBe(original);
      expect(rate.valueOf()).toBe(100);
    });

    const testCases = [
      { input: 1, expected: 1 },
      { input: 0.5, expected: 0.5 },
      { input: 1000, expected: 1000 },
      { input: 0, expected: 0 },
    ];

    testCases.forEach(({ input, expected }) => {
      it(`should parse ${input} to Rate with value ${expected}`, () => {
        const rate = Rate.z.parse(input);
        expect(rate).toBeInstanceOf(Rate);
        expect(rate.valueOf()).toBe(expected);
      });
    });
  });

  describe("arithmetic operations", () => {
    test("add", () => {
      const r1 = new Rate(100);
      const r2 = new Rate(50);
      const result = r1.add(r2);
      expect(result).toBeInstanceOf(Rate);
      expect(result.valueOf()).toBe(150);
    });

    test("add with number", () => {
      const r1 = new Rate(100);
      const result = r1.add(50);
      expect(result).toBeInstanceOf(Rate);
      expect(result.valueOf()).toBe(150);
    });

    test("sub", () => {
      const r1 = new Rate(100);
      const r2 = new Rate(30);
      const result = r1.sub(r2);
      expect(result).toBeInstanceOf(Rate);
      expect(result.valueOf()).toBe(70);
    });

    test("sub with number", () => {
      const r1 = new Rate(100);
      const result = r1.sub(30);
      expect(result).toBeInstanceOf(Rate);
      expect(result.valueOf()).toBe(70);
    });

    test("mult", () => {
      const r1 = new Rate(100);
      const result = r1.mult(2);
      expect(result).toBeInstanceOf(Rate);
      expect(result.valueOf()).toBe(200);
    });

    test("mult with decimal", () => {
      const r1 = new Rate(100);
      const result = r1.mult(0.5);
      expect(result).toBeInstanceOf(Rate);
      expect(result.valueOf()).toBe(50);
    });

    test("div", () => {
      const r1 = new Rate(100);
      const result = r1.div(2);
      expect(result).toBeInstanceOf(Rate);
      expect(result.valueOf()).toBe(50);
    });

    test("div with decimal", () => {
      const r1 = new Rate(100);
      const result = r1.div(0.5);
      expect(result).toBeInstanceOf(Rate);
      expect(result.valueOf()).toBe(200);
    });
  });
});

describe("TimeRange", () => {
  test("construct", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr.start.equals(new TimeStamp(0))).toBe(true);
    expect(tr.end.equals(new TimeStamp(1000))).toBe(true);
  });

  test("construct from object", () => {
    const tr = new TimeRange({
      start: new TimeStamp(1000),
      end: new TimeStamp(100000),
    });
    expect(tr.start.equals(new TimeStamp(1000))).toBe(true);
    expect(tr.end.equals(new TimeStamp(100000))).toBe(true);
  });

  test("span", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr.span.equals(TimeSpan.MICROSECOND)).toBe(true);
  });

  test("isValid", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr.isValid).toBe(true);
    const tr2 = new TimeRange(new TimeStamp(1000), new TimeStamp(0));
    expect(tr2.isValid).toBe(false);
  });

  test("isZero", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(0));
    expect(tr.isZero).toBe(true);
    const tr2 = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr2.isZero).toBe(false);
  });

  test("swap", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr.swap().equals(new TimeRange(new TimeStamp(1000), new TimeStamp(0)))).toBe(
      true,
    );
  });
  describe("contains", () => {
    test("TimeStamp", () => {
      const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
      expect(tr.contains(new TimeStamp(500))).toBe(true);
      expect(tr.contains(new TimeStamp(1001))).toBe(false);
    });
    test("TimeRange", () => {
      const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
      expect(tr.contains(new TimeRange(new TimeStamp(500), new TimeStamp(600)))).toBe(
        true,
      );
      expect(tr.contains(new TimeRange(new TimeStamp(500), new TimeStamp(1001)))).toBe(
        false,
      );
    });
  });
  describe("overlapsWith", () => {
    it("should return true if the end of one time range is after the start of the next time range", () => {
      const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
      const one = new TimeRange(new TimeStamp(500), new TimeStamp(600));
      expect(tr.overlapsWith(one)).toBe(true);
      expect(one.overlapsWith(tr)).toBe(true);
    });
    it("should return false if two time ranges are clearly separate", () => {
      const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
      const one = new TimeRange(new TimeStamp(1001), new TimeStamp(2000));
      expect(tr.overlapsWith(one)).toBe(false);
      expect(one.overlapsWith(tr)).toBe(false);
    });
    it("should return false if the end of the first time range is the start of the next time range", () => {
      const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
      const one = new TimeRange(new TimeStamp(1000), new TimeStamp(2000));
      expect(tr.overlapsWith(one)).toBe(false);
      expect(one.overlapsWith(tr)).toBe(false);
    });
    it("should return true only if the overlap is within a threshold", () => {
      const tr = new TimeRange(TimeStamp.milliseconds(0), TimeStamp.milliseconds(1000));
      const one = new TimeRange(
        TimeStamp.milliseconds(998),
        TimeStamp.milliseconds(2000),
      );
      expect(tr.overlapsWith(one, TimeSpan.milliseconds(2))).toBe(true);
      expect(one.overlapsWith(tr, TimeSpan.milliseconds(3))).toBe(false);
    });
    it("should return two for two ZERO time ranges", () => {
      const tr = new TimeRange(TimeStamp.ZERO, TimeStamp.ZERO);
      const one = new TimeRange(TimeStamp.ZERO, TimeStamp.ZERO);
      expect(tr.overlapsWith(one)).toBe(true);
    });
  });

  describe("boundBy", () => {
    it("should bound the time range to the provided constraints", () => {
      const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(4));
      const bound = new TimeRange(TimeSpan.seconds(2), TimeSpan.seconds(3));
      const bounded = tr.boundBy(bound);
      const expected = new TimeRange(TimeSpan.seconds(2), TimeSpan.seconds(3));
      expect(bounded.equals(expected)).toBe(true);
    });
    it("should bound the time range even if the start is after the end", () => {
      const tr = new TimeRange(TimeSpan.seconds(4), TimeSpan.seconds(1));
      const bound = new TimeRange(TimeSpan.seconds(2), TimeSpan.seconds(3));
      const bounded = tr.boundBy(bound);
      const expected = new TimeRange(TimeSpan.seconds(3), TimeSpan.seconds(2));
      expect(bounded.equals(expected)).toBe(true);
    });
  });

  describe("roughlyEquals", () => {
    it("should return true if the two time ranges are within the provided threshold", () => {
      const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(4));
      const one = new TimeRange(
        TimeSpan.seconds(1),
        TimeSpan.seconds(4).add(TimeSpan.milliseconds(500)),
      );
      expect(tr.equals(one, TimeSpan.seconds(1))).toBe(true);
      expect(tr.equals(one, TimeSpan.seconds(0))).toBe(false);
    });
  });

  test("toString", () => {
    const ts = new TimeStamp(TimeSpan.days(2))
      .add(TimeSpan.minutes(20))
      .add(TimeSpan.milliseconds(283))
      .add(TimeSpan.microseconds(900));
    const ts2 = new TimeStamp(TimeSpan.days(4))
      .add(TimeSpan.minutes(20))
      .add(TimeSpan.milliseconds(283))
      .add(TimeSpan.microseconds(900));
    const tr = ts.range(ts2);
    const trString = tr.toString();
    expect(trString).toEqual("1970-01-03T00:20:00.283Z - 1970-01-05T00:20:00.283Z");
  });

  describe("sort", () => {
    interface Spec {
      a: TimeRange;
      b: TimeRange;
      expected: number;
    }
    const TESTS: Spec[] = [
      { a: new TimeRange(1, 2), b: new TimeRange(2, 3), expected: -1 },
      { a: new TimeRange(2, 3), b: new TimeRange(1, 2), expected: 1 },
      { a: new TimeRange(1, 2), b: new TimeRange(1, 2), expected: 0 },
      { a: new TimeRange(2, 0), b: new TimeRange(1, 1), expected: 1 },
      { a: new TimeRange(2, 2), b: new TimeRange(3, 0), expected: -1 },
      { a: new TimeRange(2, 8), b: new TimeRange(2, 9), expected: -1 },
      { a: new TimeRange(2, 9), b: new TimeRange(2, 8), expected: 1 },
    ];
    TESTS.forEach(({ a, b, expected }) => {
      test(`TimeRange.sort(${a.toString()}, ${b.toString()}) = ${expected}`, () => {
        expect(TimeRange.sort(a, b)).toEqual(expected);
      });
    });
  });

  describe("simplify", () => {
    it("should merge overlapping time ranges", () => {
      const trs = [
        new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(4)),
        new TimeRange(TimeSpan.seconds(2), TimeSpan.seconds(3)),
        new TimeRange(TimeSpan.seconds(3), TimeSpan.seconds(5)),
        new TimeRange(TimeSpan.seconds(6), TimeSpan.seconds(7)),
      ];
      const expected = [
        new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(5)),
        new TimeRange(TimeSpan.seconds(6), TimeSpan.seconds(7)),
      ];
      const simplified = TimeRange.simplify(trs);
      expect(simplified).toEqual(expected);
    });
    it("should merge time ranges that are adjacent", () => {
      const trs = [
        new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(4)),
        new TimeRange(TimeSpan.seconds(4), TimeSpan.seconds(5)),
      ];
      const expected = [new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(5))];
      const simplified = TimeRange.simplify(trs);
      expect(simplified).toEqual(expected);
    });
    it("should remove zero length time ranges", () => {
      const trs = [
        new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(4)),
        new TimeRange(TimeSpan.seconds(5), TimeSpan.seconds(5)),
      ];
      const expected = [new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(4))];
      const simplified = TimeRange.simplify(trs);
      expect(simplified).toEqual(expected);
    });
    it("should make ranges valid by swapping start and end", () => {
      const trs = [
        new TimeRange(TimeSpan.seconds(4), TimeSpan.seconds(1)),
        new TimeRange(TimeSpan.seconds(2), TimeSpan.seconds(3)),
      ];
      const expected = [new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(4))];
      const simplified = TimeRange.simplify(trs);
      expect(simplified).toEqual(expected);
    });
    it("should work with zero ranges", () => {
      const trs: TimeRange[] = [];
      const expected: TimeRange[] = [];
      const simplified = TimeRange.simplify(trs);
      expect(simplified).toEqual(expected);
    });
  });
  describe("numericBounds", () => {
    it("should return correct numeric bounds for a valid time range", () => {
      const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(4));
      const bounds = tr.numericBounds;
      expect(bounds.lower).toBe(tr.start.nanoseconds);
      expect(bounds.upper).toBe(tr.end.nanoseconds);
    });

    it("should return correct numeric bounds for an invalid time range", () => {
      const tr = new TimeRange(TimeSpan.seconds(4), TimeSpan.seconds(1));
      const bounds = tr.numericBounds;
      expect(bounds.lower).toBe(tr.start.nanoseconds);
      expect(bounds.upper).toBe(tr.end.nanoseconds);
    });

    it("should handle zero time range", () => {
      const tr = new TimeRange(TimeStamp.ZERO, TimeStamp.ZERO);
      const bounds = tr.numericBounds;
      expect(bounds.lower).toBe(0);
      expect(bounds.upper).toBe(0);
    });

    it("should handle large time values", () => {
      const tr = new TimeRange(TimeSpan.days(365), TimeSpan.days(730));
      const bounds = tr.numericBounds;
      expect(bounds.lower).toBe(tr.start.nanoseconds);
      expect(bounds.upper).toBe(tr.end.nanoseconds);
    });
  });

  describe("schema", () => {
    it("should parse object with start and end TimeStamps", () => {
      const tr = TimeRange.z.parse({
        start: new TimeStamp(1000),
        end: new TimeStamp(2000),
      });
      expect(tr).toBeInstanceOf(TimeRange);
      expect(tr.start.valueOf()).toBe(1000n);
      expect(tr.end.valueOf()).toBe(2000n);
    });

    it("should pass through existing TimeRange instance", () => {
      const original = new TimeRange(new TimeStamp(1000), new TimeStamp(2000));
      const tr = TimeRange.z.parse(original);
      expect(tr).toStrictEqual(original);
      expect(tr.start.valueOf()).toBe(1000n);
      expect(tr.end.valueOf()).toBe(2000n);
    });

    it("should parse object with various timestamp formats", () => {
      const tr = TimeRange.z.parse({
        start: 1000,
        end: "2000",
      });
      expect(tr).toBeInstanceOf(TimeRange);
      expect(tr.start.valueOf()).toBe(1000n);
      expect(tr.end.valueOf()).toBe(2000n);
    });

    const testCases = [
      {
        input: { start: 0, end: 1000 },
        expectedStart: 0n,
        expectedEnd: 1000n,
      },
      {
        input: { start: new Date("2024-01-01"), end: new Date("2024-01-02") },
        expectedStart: BigInt(new Date("2024-01-01").getTime()) * 1000000n,
        expectedEnd: BigInt(new Date("2024-01-02").getTime()) * 1000000n,
      },
    ];

    testCases.forEach(({ input, expectedStart, expectedEnd }, i) => {
      it(`should parse test case ${i + 1}`, () => {
        const tr = TimeRange.z.parse(input);
        expect(tr).toBeInstanceOf(TimeRange);
        expect(tr.start.valueOf()).toBe(expectedStart);
        expect(tr.end.valueOf()).toBe(expectedEnd);
      });
    });
  });
});

describe("Density", () => {
  describe("construct", () => {
    test("construct from CrudeValueExtension", () => {
      const density = new Density({ value: 8 });
      expect(density.valueOf()).toBe(8);
    });

    test("construct from number", () => {
      const density = new Density(8);
      expect(density.valueOf()).toBe(8);
    });

    test("construct from Density", () => {
      const density = new Density(8);
      expect(density.valueOf()).toBe(8);
    });
  });

  describe("schema", () => {
    it("should parse number", () => {
      const density = Density.z.parse(8);
      expect(density).toBeInstanceOf(Density);
      expect(density.valueOf()).toBe(8);
    });

    it("should pass through existing Density instance", () => {
      const original = new Density(4);
      const density = Density.z.parse(original);
      expect(density).toBe(original);
      expect(density.valueOf()).toBe(4);
    });

    const testCases = [
      { input: 1, expected: 1 },
      { input: 2, expected: 2 },
      { input: 4, expected: 4 },
      { input: 8, expected: 8 },
      { input: 0, expected: 0 },
    ];

    testCases.forEach(({ input, expected }) => {
      it(`should parse ${input} to Density with value ${expected}`, () => {
        const density = Density.z.parse(input);
        expect(density).toBeInstanceOf(Density);
        expect(density.valueOf()).toBe(expected);
      });
    });
  });

  describe("arithmetic operations", () => {
    test("add", () => {
      const d1 = new Density(8);
      const d2 = new Density(4);
      const result = d1.add(d2);
      expect(result).toBeInstanceOf(Density);
      expect(result.valueOf()).toBe(12);
    });

    test("add with number", () => {
      const d1 = new Density(8);
      const result = d1.add(4);
      expect(result).toBeInstanceOf(Density);
      expect(result.valueOf()).toBe(12);
    });

    test("sub", () => {
      const d1 = new Density(8);
      const d2 = new Density(3);
      const result = d1.sub(d2);
      expect(result).toBeInstanceOf(Density);
      expect(result.valueOf()).toBe(5);
    });

    test("sub with number", () => {
      const d1 = new Density(8);
      const result = d1.sub(3);
      expect(result).toBeInstanceOf(Density);
      expect(result.valueOf()).toBe(5);
    });

    test("mult", () => {
      const d1 = new Density(4);
      const result = d1.mult(2);
      expect(result).toBeInstanceOf(Density);
      expect(result.valueOf()).toBe(8);
    });

    test("mult with decimal", () => {
      const d1 = new Density(8);
      const result = d1.mult(0.5);
      expect(result).toBeInstanceOf(Density);
      expect(result.valueOf()).toBe(4);
    });

    test("div", () => {
      const d1 = new Density(8);
      const result = d1.div(2);
      expect(result).toBeInstanceOf(Density);
      expect(result.valueOf()).toBe(4);
    });

    test("div with decimal", () => {
      const d1 = new Density(4);
      const result = d1.div(0.5);
      expect(result).toBeInstanceOf(Density);
      expect(result.valueOf()).toBe(8);
    });
  });
});

describe("DataType", () => {
  describe("isVariable", () => {
    it("should return true if the data type has a variable length", () => {
      expect(DataType.INT32.isVariable).toBe(false);
    });
    it("should return false if the data type does not have a variable length", () => {
      expect(DataType.STRING.isVariable).toBe(true);
    });
  });

  describe("construct", () => {
    test("construct from CrudeValueExtension", () => {
      const dt = new DataType({ value: "int32" });
      expect(dt.toString()).toBe("int32");
    });

    [
      [new Int32Array(), DataType.INT32],
      [new Int8Array(), DataType.INT8],
      [new Uint8Array(), DataType.UINT8],
      [new Uint16Array(), DataType.UINT16],
      [new Uint32Array(), DataType.UINT32],
      [new BigInt64Array(), DataType.INT64],
      [new BigUint64Array(), DataType.UINT64],
      [new Float32Array(), DataType.FLOAT32],
      [new Float64Array(), DataType.FLOAT64],
      [new BigInt64Array(), DataType.INT64],
      [new BigUint64Array(), DataType.UINT64],
    ].forEach(([array, expected]) => {
      test(`construct from ${array.constructor.name} to ${expected.toString()}`, () => {
        const dt = new DataType(array);
        expect(dt.toString()).toBe(expected.toString());
      });
    });

    test("from data type", () => {
      const dt = new DataType(DataType.INT32);
      expect(dt.toString()).toBe(DataType.INT32.toString());
    });

    test("from string", () => {
      const dt = new DataType("int32");
      expect(dt.toString()).toBe("int32");
    });
  });

  describe("canSafelyCastTo", () => {
    const TESTS: [DataType, DataType, boolean][] = [
      [DataType.INT32, DataType.INT32, true],
      [DataType.INT32, DataType.INT64, true],
      [DataType.INT32, DataType.FLOAT32, false],
      [DataType.INT32, DataType.FLOAT64, true],
      [DataType.INT32, DataType.STRING, false],
      [DataType.INT32, DataType.BOOLEAN, false],
      [DataType.INT32, DataType.INT8, false],
      [DataType.INT64, DataType.INT32, false],
      [DataType.INT64, DataType.INT64, true],
      [DataType.INT64, DataType.FLOAT32, false],
      [DataType.INT64, DataType.FLOAT64, false],
      [DataType.INT64, DataType.STRING, false],
      [DataType.FLOAT64, DataType.FLOAT32, false],
      [DataType.FLOAT64, DataType.FLOAT64, true],
      [DataType.FLOAT64, DataType.STRING, false],
      [DataType.FLOAT64, DataType.BOOLEAN, false],
      [DataType.FLOAT32, DataType.FLOAT64, true],
      [DataType.FLOAT32, DataType.FLOAT32, true],
      [DataType.FLOAT32, DataType.STRING, false],
      [DataType.FLOAT32, DataType.BOOLEAN, false],
      [DataType.STRING, DataType.STRING, true],
      [DataType.STRING, DataType.INT32, false],
      [DataType.STRING, DataType.INT64, false],
      [DataType.STRING, DataType.FLOAT32, false],
      [DataType.STRING, DataType.FLOAT64, false],
      [DataType.STRING, DataType.BOOLEAN, false],
      [DataType.STRING, DataType.INT8, false],
      [DataType.BOOLEAN, DataType.BOOLEAN, true],
      [DataType.BOOLEAN, DataType.INT32, false],
      [DataType.BOOLEAN, DataType.INT64, false],
      [DataType.INT8, DataType.FLOAT32, true],
    ];
    TESTS.forEach(([from, to, expected]) =>
      it(`should return ${expected} when casting from ${from.toString()} to ${to.toString()}`, () => {
        expect(from.canSafelyCastTo(to)).toBe(expected);
      }),
    );
  });

  describe("canCastTo", () => {
    it("should return true for any two numeric data types", () => {
      const numericTypes = [
        DataType.INT32,
        DataType.INT64,
        DataType.FLOAT32,
        DataType.FLOAT64,
      ];
      for (const from of numericTypes)
        for (const to of numericTypes) expect(from.canCastTo(to)).toBe(true);
    });
    it("should return true for non-numeric data types ONLY if they are equal", () => {
      const nonNumericTypes = [DataType.STRING, DataType.BOOLEAN];
      for (const from of nonNumericTypes)
        for (const to of nonNumericTypes) expect(from.canCastTo(to)).toBe(from === to);
    });
  });

  describe("equals", () => {
    const TESTS: [DataType, CrudeDataType, boolean][] = [
      [DataType.INT32, DataType.INT32, true],
      [DataType.INT32, DataType.INT64, false],
      [DataType.INT32, "int32", true],
      [DataType.INT32, "int64", false],
    ];
    TESTS.forEach(([dt, other, expected]) =>
      it(`should return ${expected} when comparing ${dt.toString()} to ${JSON.stringify(other)}`, () => {
        expect(dt.equals(other)).toBe(expected);
      }),
    );
  });

  describe("schema", () => {
    it("should parse string", () => {
      const dt = DataType.z.parse("int32");
      expect(dt).toBeInstanceOf(DataType);
      expect(dt.toString()).toBe("int32");
    });

    it("should pass through existing DataType instance", () => {
      const original = DataType.INT32;
      const dt = DataType.z.parse(original);
      expect(dt).toBe(original);
      expect(dt.toString()).toBe("int32");
    });

    const testCases = [
      { input: "int8", expected: "int8", short: "i8" },
      { input: "int16", expected: "int16", short: "i16" },
      { input: "int32", expected: "int32", short: "i32" },
      { input: "int64", expected: "int64", short: "i64" },
      { input: "uint8", expected: "uint8", short: "u8" },
      { input: "uint16", expected: "uint16", short: "u16" },
      { input: "uint32", expected: "uint32", short: "u32" },
      { input: "uint64", expected: "uint64", short: "u64" },
      { input: "float32", expected: "float32", short: "f32" },
      { input: "float64", expected: "float64", short: "f64" },
      { input: "string", expected: "string", short: "str" },
      { input: "boolean", expected: "boolean", short: "bool" },
      { input: "timestamp", expected: "timestamp", short: "ts" },
      { input: "uuid", expected: "uuid", short: "uuid" },
      { input: "json", expected: "json", short: "json" },
    ];

    testCases.forEach(({ input, expected, short }) => {
      it(`should parse "${input}" to DataType with value "${expected}"`, () => {
        const dt = DataType.z.parse(input);
        expect(dt).toBeInstanceOf(DataType);
        expect(dt.toString()).toBe(expected);
        expect(dt.toString(true)).toBe(short);
      });
    });
  });
});

describe("Size", () => {
  describe("construct", () => {
    test("construct from CrudeValueExtension", () => {
      const size = new Size({ value: 1024 });
      expect(size.valueOf()).toBe(1024);
    });

    test("construct from number", () => {
      const size = new Size(1024);
      expect(size.valueOf()).toBe(1024);
    });

    test("construct from Size", () => {
      const size = new Size(1024);
      expect(size.valueOf()).toBe(1024);
    });
  });
  const TO_STRING_TESTS = [
    [Size.bytes(1), "1B"],
    [Size.kilobytes(1), "1KB"],
    [Size.megabytes(1), "1MB"],
    [Size.gigabytes(1), "1GB"],
    [Size.terabytes(1), "1TB"],
    [Size.megabytes(4).add(Size.kilobytes(500)), "4MB 500KB"],
  ];

  test("toString", () => {
    TO_STRING_TESTS.forEach(([size, expected]) => {
      expect(size.toString()).toEqual(expected);
    });
  });

  const TRUNCATE_TESTS = [
    [Size.bytes(1).add(Size.kilobytes(1)), Size.KILOBYTE, Size.kilobytes(1)],
    [Size.megabytes(100).add(Size.kilobytes(500)), Size.MEGABYTE, Size.megabytes(100)],
    [
      Size.gigabytes(1).add(Size.megabytes(500)).add(Size.kilobytes(500)),
      Size.MEGABYTE,
      Size.gigabytes(1).add(Size.megabytes(500)),
    ],
  ];

  test("truncate", () => {
    TRUNCATE_TESTS.forEach(([size, unit, expected]) => {
      expect(size.truncate(unit).valueOf()).toEqual(expected.valueOf());
    });
  });

  describe("schema", () => {
    it("should parse number", () => {
      const size = Size.z.parse(1024);
      expect(size).toBeInstanceOf(Size);
      expect(size.valueOf()).toBe(1024);
    });

    it("should pass through existing Size instance", () => {
      const original = new Size(2048);
      const size = Size.z.parse(original);
      expect(size).toBe(original);
      expect(size.valueOf()).toBe(2048);
    });

    const testCases = [
      { input: 0, expected: 0 },
      { input: 1, expected: 1 },
      { input: 1024, expected: 1024 },
      { input: 1048576, expected: 1048576 },
      { input: 1073741824, expected: 1073741824 },
    ];

    testCases.forEach(({ input, expected }) => {
      it(`should parse ${input} to Size with value ${expected}`, () => {
        const size = Size.z.parse(input);
        expect(size).toBeInstanceOf(Size);
        expect(size.valueOf()).toBe(expected);
      });
    });
  });

  describe("arithmetic operations", () => {
    test("add", () => {
      const s1 = Size.kilobytes(10);
      const s2 = Size.kilobytes(5);
      const result = s1.add(s2);
      expect(result).toBeInstanceOf(Size);
      expect(result.valueOf()).toBe(15000);
    });

    test("add with number", () => {
      const s1 = Size.bytes(100);
      const result = s1.add(50);
      expect(result).toBeInstanceOf(Size);
      expect(result.valueOf()).toBe(150);
    });

    test("sub", () => {
      const s1 = Size.kilobytes(10);
      const s2 = Size.kilobytes(3);
      const result = s1.sub(s2);
      expect(result).toBeInstanceOf(Size);
      expect(result.valueOf()).toBe(7000);
    });

    test("sub with number", () => {
      const s1 = Size.bytes(100);
      const result = s1.sub(30);
      expect(result).toBeInstanceOf(Size);
      expect(result.valueOf()).toBe(70);
    });

    test("mult", () => {
      const s1 = Size.kilobytes(5);
      const result = s1.mult(2);
      expect(result).toBeInstanceOf(Size);
      expect(result.valueOf()).toBe(10000);
    });

    test("mult with decimal", () => {
      const s1 = Size.kilobytes(10);
      const result = s1.mult(0.5);
      expect(result).toBeInstanceOf(Size);
      expect(result.valueOf()).toBe(5000);
    });

    test("div", () => {
      const s1 = Size.kilobytes(10);
      const result = s1.div(2);
      expect(result).toBeInstanceOf(Size);
      expect(result.valueOf()).toBe(5000);
    });

    test("div with decimal", () => {
      const s1 = Size.kilobytes(5);
      const result = s1.div(0.5);
      expect(result).toBeInstanceOf(Size);
      expect(result.valueOf()).toBe(10000);
    });
  });
});

describe("addSamples", () => {
  test("adds two numbers", () => {
    expect(addSamples(1, 2)).toBe(3);
    expect(addSamples(1.5, 2.5)).toBe(4);
    expect(addSamples(-1, 1)).toBe(0);
  });

  test("adds two bigints", () => {
    expect(addSamples(1n, 2n)).toBe(3n);
    expect(addSamples(-1n, 1n)).toBe(0n);
    expect(addSamples(9007199254740991n, 1n)).toBe(9007199254740992n);
  });

  test("handles mixed numeric types", () => {
    expect(addSamples(1, 2n)).toBe(3);
    expect(addSamples(2n, 1)).toBe(3);
    expect(addSamples(1.5, 2n)).toBe(3.5);
    expect(addSamples(2n, 1.5)).toBe(3.5);
  });

  test("handles edge cases", () => {
    expect(addSamples(0, 0)).toBe(0);
    expect(addSamples(Number.MAX_SAFE_INTEGER, 1)).toBe(Number.MAX_SAFE_INTEGER + 1);
    expect(addSamples(Number.MIN_SAFE_INTEGER, -1)).toBe(Number.MIN_SAFE_INTEGER - 1);
  });
});
