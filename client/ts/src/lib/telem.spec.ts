import { describe, expect, test } from "vitest";

import { DataType, Density, Rate, Size, TimeRange, TimeSpan, TimeStamp } from "./telem";

describe("TimeStamp", () => {
  test("construct", () => {
    const ts = new TimeStamp(1000);
    expect(ts.equals(TimeSpan.Microseconds())).toBeTruthy();
  });

  test("construct from TimeStamp", () => {
    const ts = new TimeStamp(TimeSpan.Microseconds(10));
    expect(ts.equals(TimeSpan.Microseconds(10))).toBeTruthy();
  });

  test("span", () => {
    const ts = new TimeStamp(0);
    expect(ts.span(new TimeStamp(1000)).equals(TimeSpan.Microseconds())).toBeTruthy();
  });

  test("range", () => {
    const ts = new TimeStamp(0);
    expect(
      ts.range(new TimeStamp(1000)).equals(new TimeRange(ts, TimeSpan.Microseconds()))
    ).toBeTruthy();
  });

  test("spanRange", () => {
    const ts = new TimeStamp(0);
    expect(
      ts
        .spanRange(TimeSpan.Microseconds())
        .equals(new TimeRange(ts, ts.add(TimeSpan.Microseconds())))
    ).toBeTruthy();
  });

  test("isZero", () => {
    const ts = new TimeStamp(0);
    expect(ts.isZero()).toBeTruthy();
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
      ts.add(TimeSpan.Microseconds()).equals(new TimeStamp(TimeSpan.Microseconds(1)))
    ).toBeTruthy();
  });

  test("sub", () => {
    const ts = new TimeStamp(TimeSpan.Microseconds());
    expect(ts.sub(TimeSpan.Microseconds()).equals(new TimeStamp(0))).toBeTruthy();
  });
});

describe("TimeSpan", () => {
  test("construct from static", () => {
    expect(TimeSpan.Nanoseconds(1).equals(1)).toBeTruthy();
    expect(TimeSpan.Microseconds(1).equals(1000)).toBeTruthy();
    expect(TimeSpan.Milliseconds(1).equals(1000000)).toBeTruthy();
    expect(TimeSpan.Seconds(1).equals(1e9)).toBeTruthy();
    expect(TimeSpan.Minutes(1).equals(6e10)).toBeTruthy();
    expect(TimeSpan.Hours(1).equals(36e11)).toBeTruthy();
  });

  test("seconds", () => {
    expect(TimeSpan.Seconds(1).seconds()).toEqual(1);
  });

  test("isZero", () => {
    expect(TimeSpan.Zero.isZero()).toBeTruthy();
    expect(TimeSpan.Seconds(1).isZero()).toBeFalsy();
  });

  test("add", () => {
    expect(TimeSpan.Seconds(1).add(TimeSpan.Second).equals(2e9)).toBeTruthy();
  });

  test("sub", () => {
    expect(TimeSpan.Seconds(1).sub(TimeSpan.Second).isZero()).toBeTruthy();
  });
});

describe("Rate", () => {
  test("construct", () => expect(new Rate(1).equals(1)).toBeTruthy());

  test("period", () =>
    expect(new Rate(1).period().equals(TimeSpan.Second)).toBeTruthy());

  test("sampleCount", () =>
    expect(new Rate(1).sampleCount(TimeSpan.Second)).toEqual(1));

  test("byteCount", () =>
    expect(new Rate(1).byteCount(TimeSpan.Second, Density.Bit64)).toEqual(8));

  test("span", () =>
    expect(new Rate(1).span(4).equals(TimeSpan.Seconds(4))).toBeTruthy());

  test("byteSpan", () =>
    expect(
      new Rate(1).byteSpan(new Size(32), Density.Bit64).equals(TimeSpan.Seconds(4))
    ).toBeTruthy());

  test("Hz", () => expect(Rate.Hz(1).equals(1)).toBeTruthy());
  test("KHz", () => expect(Rate.KHz(1).equals(1e3)).toBeTruthy());
});

describe("TimeRange", () => {
  test("construct", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr.start.equals(new TimeStamp(0))).toBeTruthy();
    expect(tr.end.equals(new TimeStamp(1000))).toBeTruthy();
  });

  test("span", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr.span().equals(TimeSpan.Microsecond)).toBeTruthy();
  });

  test("isValid", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr.isValid()).toBeTruthy();
    const tr2 = new TimeRange(new TimeStamp(1000), new TimeStamp(0));
    expect(tr2.isValid()).toBeFalsy();
  });

  test("isZero", () => {
    const tr = new TimeRange(new TimeStamp(0), new TimeStamp(0));
    expect(tr.isZero()).toBeTruthy();
    const tr2 = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
    expect(tr2.isZero()).toBeFalsy();
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
    const dt = DataType.Int32;
    const v = JSON.parse(JSON.stringify({ dt }));
    expect(v.dt === "int32").toBeTruthy();
  });
});
