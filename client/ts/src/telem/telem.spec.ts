// Copyright 2022 Synnax Labs, Inc.
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
    expect(ts.equals(TimeSpan.microseconds())).toBeTruthy();
  });

  test("construct from TimeStamp", () => {
    const ts = new TimeStamp(TimeSpan.microseconds(10));
    expect(ts.equals(TimeSpan.microseconds(10))).toBeTruthy();
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
});

describe("DataType", () => {
  test("json serialization", () => {
    const dt = DataType.INT32;
    const v = JSON.parse(JSON.stringify({ dt }));
    expect(v.dt === "int32").toBeTruthy();
  });
});
