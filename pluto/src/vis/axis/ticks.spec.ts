// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, newIndexedPair } from "@synnaxlabs/client/testutil";
import { scale, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { newTickFactory, type Tick, type TickType } from "@/vis/axis/ticks";

const renderParams = (start: Date, end: Date, size: number) => ({
  decimalToDataScale: scale.Scale.scale<number>(0, 1).scale(
    Number(new TimeStamp(start).valueOf()),
    Number(new TimeStamp(end).valueOf()),
  ),
  size,
});

const create = (start: Date, end: Date, size: number): Tick[] =>
  newTickFactory({ type: "time" }).create(renderParams(start, end, size));

const labels = (start: Date, end: Date, size: number): string[] =>
  create(start, end, size).map((t) => t.label);

// Maps a tick position back to milliseconds within the domain.
const tickMS = (tick: Tick, start: Date, end: Date, size: number): number =>
  start.getTime() + (tick.position / size) * (end.getTime() - start.getTime());

const DAY_MS = TimeSpan.DAY.milliseconds;

describe("TimeTickFactory", () => {
  describe("second scale", () => {
    it("should label seconds with the minute at the rollover", () => {
      const start = new Date(2026, 7, 24, 12, 0, 0);
      const end = new Date(2026, 7, 24, 12, 0, 30);
      expect(labels(start, end, 450)).toEqual([
        "12:00",
        "5s",
        "10s",
        "15s",
        "20s",
        "25s",
        "30s",
      ]);
    });
  });

  describe("minute scale", () => {
    it("should label minutes in military time", () => {
      const start = new Date(2026, 7, 24, 12, 0);
      const end = new Date(2026, 7, 24, 12, 10);
      expect(labels(start, end, 800)).toEqual([
        "12:00",
        "12:01",
        "12:02",
        "12:03",
        "12:04",
        "12:05",
        "12:06",
        "12:07",
        "12:08",
        "12:09",
        "12:10",
      ]);
    });

    it("should produce more ticks on a wider plot", () => {
      const start = new Date(2026, 7, 24, 12, 0);
      const end = new Date(2026, 7, 24, 13, 0);
      const narrow = labels(start, end, 300);
      const wide = labels(start, end, 1200);
      expect(wide.length).toBeGreaterThan(narrow.length);
    });
  });

  describe("hour scale", () => {
    it("should label hours in military time", () => {
      const start = new Date(2026, 7, 24, 6, 0);
      const end = new Date(2026, 7, 24, 18, 0);
      expect(labels(start, end, 300)).toEqual([
        "6:00",
        "9:00",
        "12:00",
        "15:00",
        "18:00",
      ]);
    });

    it("should label a midnight tick with its date", () => {
      const start = new Date(2026, 7, 24, 18, 0);
      const end = new Date(2026, 7, 25, 6, 0);
      expect(labels(start, end, 300)).toContain("Aug 25");
    });
  });

  describe("day scale", () => {
    it("should keep two-day ticks uniform across a month boundary", () => {
      const start = new Date(2026, 7, 24);
      const end = new Date(2026, 8, 5);
      expect(labels(start, end, 500)).toEqual([
        "Aug 25",
        "Aug 27",
        "Aug 29",
        "Aug 31",
        "Sep 2",
        "Sep 4",
      ]);
    });

    it("should keep tick spacing uniform over a thirty day span", () => {
      const start = new Date(2026, 7, 24);
      const end = new Date(2026, 8, 23);
      const ticks = create(start, end, 800);
      expect(ticks.length).toBeGreaterThan(2);
      const gaps = ticks
        .slice(1)
        .map((t, i) => tickMS(t, start, end, 800) - tickMS(ticks[i], start, end, 800));
      gaps.forEach((gap) => expect(gap / DAY_MS).toBeCloseTo(gaps[0] / DAY_MS, 1));
    });
  });

  describe("week scale", () => {
    it("should space quarter-span ticks seven days apart", () => {
      const start = new Date(2026, 6, 1);
      const end = new Date(2026, 8, 30);
      const ticks = create(start, end, 800);
      expect(ticks.length).toBeGreaterThan(2);
      ticks.slice(1).forEach((t, i) => {
        const gap = tickMS(t, start, end, 800) - tickMS(ticks[i], start, end, 800);
        expect(gap / DAY_MS).toBeCloseTo(7, 1);
      });
    });
  });

  describe("month scale", () => {
    it("should label a year span with months and the year at January", () => {
      const start = new Date(2026, 0, 1);
      const end = new Date(2027, 0, 1);
      expect(labels(start, end, 900)).toEqual([
        "2026",
        "Feb",
        "Mar",
        "Apr",
        "May",
        "Jun",
        "Jul",
        "Aug",
        "Sep",
        "Oct",
        "Nov",
        "Dec",
        "2027",
      ]);
    });
  });

  describe("year scale", () => {
    it("should label a multi-year span with years", () => {
      const start = new Date(2026, 0, 1);
      const end = new Date(2030, 0, 1);
      expect(labels(start, end, 400)).toEqual(["2026", "2027", "2028", "2029", "2030"]);
    });
  });

  describe("sub-5ms domains", () => {
    const preciseTicks = (spanNS: number, size: number): Tick[] =>
      newTickFactory({ type: "time" }).create({
        decimalToDataScale: scale.Scale.scale<number>(0, 1).scale(0, spanNS),
        size,
      });

    it("should label microsecond ticks below a 50μs span", () => {
      const ticks = preciseTicks(Number(TimeSpan.microseconds(10).valueOf()), 450);
      expect(ticks.length).toBeGreaterThan(0);
      ticks.forEach((t) => expect(t.label).toMatch(/μs$/));
    });

    it("should label millisecond ticks below a 5ms span", () => {
      const ticks = preciseTicks(Number(TimeSpan.milliseconds(4).valueOf()), 450);
      expect(ticks.length).toBeGreaterThan(0);
      ticks.forEach((t) => expect(t.label).toMatch(/ms$/));
    });
  });

  describe("millisecond scale", () => {
    it("should label millisecond ticks with fractional seconds", () => {
      const start = new Date(2026, 7, 24, 12, 0, 0, 0);
      const end = new Date(2026, 7, 24, 12, 0, 0, 100);
      expect(labels(start, end, 300).slice(1)).toEqual(["0.05s", "0.1s"]);
    });
  });

  describe("daily ticks", () => {
    it("should keep consecutive daily ticks untouched", () => {
      const start = new Date(2026, 7, 24);
      const end = new Date(2026, 7, 28);
      expect(labels(start, end, 400)).toEqual([
        "Aug 24",
        "Aug 25",
        "Aug 26",
        "Aug 27",
        "Aug 28",
      ]);
    });
  });

  describe("caching", () => {
    it("should return the cached array for an unchanged domain and size", () => {
      const factory = newTickFactory({ type: "time" });
      const params = renderParams(new Date(2026, 7, 24), new Date(2026, 8, 5), 500);
      expect(factory.create(params)).toBe(factory.create(params));
    });

    it("should recompute when the size changes", () => {
      const factory = newTickFactory({ type: "time" });
      const start = new Date(2026, 7, 24, 12, 0);
      const end = new Date(2026, 7, 24, 13, 0);
      const narrow = factory.create(renderParams(start, end, 300));
      const wide = factory.create(renderParams(start, end, 1200));
      expect(wide).not.toBe(narrow);
      expect(wide.length).toBeGreaterThan(narrow.length);
    });
  });

  describe("degenerate inputs", () => {
    it("should not throw on a zero-span domain", () => {
      const start = new Date(2026, 7, 24, 12, 0);
      expect(() => create(start, start, 500)).not.toThrow();
    });

    it("should not throw on a zero-size axis", () => {
      const start = new Date(2026, 7, 24, 12, 0);
      const end = new Date(2026, 7, 24, 13, 0);
      expect(() => create(start, end, 0)).not.toThrow();
    });
  });

  describe("tick spacing", () => {
    it("should produce fewer ticks with a larger spacing", () => {
      const start = new Date(2026, 7, 24, 12, 0);
      const end = new Date(2026, 7, 24, 13, 0);
      const dense = newTickFactory({ type: "time", tickSpacing: 50 }).create(
        renderParams(start, end, 600),
      );
      const sparse = newTickFactory({ type: "time", tickSpacing: 200 }).create(
        renderParams(start, end, 600),
      );
      expect(sparse.length).toBeLessThan(dense.length);
    });
  });
});

describe("LinearTickFactory", () => {
  const linear = (lower: number, upper: number, size: number): Tick[] =>
    newTickFactory({}).create({
      decimalToDataScale: scale.Scale.scale<number>(0, 1).scale(lower, upper),
      size,
    });

  it("should be the default tick type", () => {
    expect(linear(0, 100, 750).map((t) => t.label)).toContain("50");
  });

  it("should label round steps across the domain", () => {
    expect(linear(0, 100, 750).map((t) => t.label)).toEqual([
      "0",
      "10",
      "20",
      "30",
      "40",
      "50",
      "60",
      "70",
      "80",
      "90",
      "100",
    ]);
  });

  it("should label fractional steps without float noise", () => {
    expect(linear(0, 1, 750).map((t) => t.label)).toEqual([
      "0",
      "0.1",
      "0.2",
      "0.3",
      "0.4",
      "0.5",
      "0.6",
      "0.7",
      "0.8",
      "0.9",
      "1",
    ]);
  });

  it("should label a domain crossing zero", () => {
    const result = linear(-50, 50, 750).map((t) => t.label);
    expect(result).toContain("-40");
    expect(result).toContain("0");
    expect(result).toContain("40");
  });

  it("should place positions within the axis size", () => {
    linear(0, 100, 750).forEach((t) => {
      expect(t.position).toBeGreaterThanOrEqual(0);
      expect(t.position).toBeLessThanOrEqual(750);
    });
  });

  it("should return the cached array for an unchanged domain and size", () => {
    const factory = newTickFactory({});
    const params = {
      decimalToDataScale: scale.Scale.scale<number>(0, 1).scale(0, 100),
      size: 750,
    };
    expect(factory.create(params)).toBe(factory.create(params));
  });

  it("should recompute when the domain changes", () => {
    const factory = newTickFactory({});
    const first = factory.create({
      decimalToDataScale: scale.Scale.scale<number>(0, 1).scale(0, 100),
      size: 750,
    });
    const second = factory.create({
      decimalToDataScale: scale.Scale.scale<number>(0, 1).scale(0, 200),
      size: 750,
    });
    expect(second.map((t) => t.label)).not.toEqual(first.map((t) => t.label));
  });
});

describe("newTickFactory", () => {
  it("should reject an unknown tick type", () => {
    expect(() => newTickFactory({ type: "bogus" as TickType })).toThrow();
  });
});

describe("against a live Core", () => {
  const client = createTestClient();

  it("should produce calendar ticks from timestamps read off the Core", async () => {
    const [index, data] = await newIndexedPair(client);
    const start = new TimeStamp(new Date(2026, 7, 24));
    const end = new TimeStamp(new Date(2026, 8, 5));
    const step = TimeSpan.hours(6).valueOf();
    const timestamps: bigint[] = [];
    for (let t = start.valueOf(); t < end.valueOf(); t += step) timestamps.push(t);
    const values = timestamps.map((_, i) => Math.sin(i / 8));
    await client.write(start, { [index.key]: timestamps, [data.key]: values });

    const frame = await client.read(new TimeRange(start, end), [index.key]);
    const series = frame.get(index.key);
    expect(series.length).toBe(timestamps.length);

    const ticks = newTickFactory({ type: "time" }).create({
      decimalToDataScale: scale.Scale.scale<number>(0, 1).scale(
        Number(series.at(0, true)),
        Number(series.at(-1, true)),
      ),
      size: 500,
    });
    expect(ticks.map((t) => t.label)).toEqual([
      "Aug 25",
      "Aug 27",
      "Aug 29",
      "Aug 31",
      "Sep 2",
      "Sep 4",
    ]);
  });
});

describe("RelativeTickFactory", () => {
  const S = TimeSpan.SECOND.nanoseconds;
  const relative = (lower: number, upper: number, size: number): Tick[] =>
    newTickFactory({ type: "relative" }).create({
      decimalToDataScale: scale.Scale.scale<number>(0, 1).scale(lower, upper),
      size,
    });

  it("should put T0 at the zero of the axis", () => {
    const ticks = relative(-5 * S, 5 * S, 500);
    expect(ticks.find((t) => t.label === "T0")?.position).toBeCloseTo(250);
    expect(ticks.some((t) => t.label.startsWith("T-"))).toBe(true);
    expect(ticks.some((t) => t.label.startsWith("T+"))).toBe(true);
  });

  it("should label every tick of an axis in one unit", () => {
    const labels = relative(-0.5 * S, 2 * S, 500)
      .map((t) => t.label)
      .filter((l) => l !== "T0");
    expect(labels.length).toBeGreaterThan(1);
    for (const l of labels) expect(l).toMatch(/^T[+-]\d+(\.\d+)? s$/);
  });

  it("should label a span of a minute or more as a clock", () => {
    const labels = relative(0, 90 * S, 500).map((t) => t.label);
    expect(labels).toContain("T+1:00");
  });

  it("should space the ticks evenly", () => {
    const ticks = relative(-3 * S, 7 * S, 800);
    const gaps = ticks.slice(1).map((t, i) => t.position - ticks[i].position);
    for (const g of gaps) expect(g).toBeCloseTo(gaps[0]);
  });

  it("should return no ticks for an empty span", () => {
    expect(relative(S, S, 500)).toEqual([]);
  });
});

describe("TimeTickFactory segments", () => {
  const ns = (d: Date): number => Number(new TimeStamp(d).valueOf());
  const A = [new Date(2026, 10, 17, 10, 0), new Date(2026, 10, 17, 10, 6)];
  const B = [new Date(2026, 10, 18, 14, 30), new Date(2026, 10, 18, 14, 36)];
  // Two segments of six minutes that each fill half of the axis.
  const segments = (seam = 0.5) => [
    {
      position: { lower: -Infinity, upper: seam },
      decimalToDataScale: scale.Scale.scale<number>(0, seam).scale(ns(A[0]), ns(A[1])),
    },
    {
      position: { lower: seam, upper: Infinity },
      decimalToDataScale: scale.Scale.scale<number>(seam, 1).scale(ns(B[0]), ns(B[1])),
    },
  ];
  const create = (seam = 0.5, size = 800): Tick[] =>
    newTickFactory({ type: "time" }).create({
      decimalToDataScale: segments(seam)[0].decimalToDataScale,
      size,
      segments: segments(seam),
    });

  it("should place the ticks of every segment at one interval", () => {
    const ticks = create();
    const first = ticks.filter((t) => t.position < 400);
    const second = ticks.filter((t) => t.position >= 400);
    expect(first.length).toBeGreaterThan(1);
    expect(second.length).toBeGreaterThan(1);
    const step = (t: Tick[]) => t[1].position - t[0].position;
    expect(step(first)).toBeCloseTo(step(second));
  });

  it("should give each tick its date and time as an anchor label", () => {
    const firsts = [0, 1].map((i) => create().find((t) => t.segment === i));
    expect(firsts.map((t) => t?.anchorLabel)).toEqual(["Nov 17 10:00", "Nov 18 14:30"]);
  });

  it("should label the ticks in the short form", () => {
    create().forEach((t) => expect(t.label).toMatch(/^\d+(:\d\d|\.\d+)?s?$/));
  });

  it("should return the ticks it made last for the same segments and size", () => {
    const factory = newTickFactory({ type: "time" });
    const params = { decimalToDataScale: segments()[0].decimalToDataScale, size: 800 };
    const first = factory.create({ ...params, segments: segments() });
    expect(factory.create({ ...params, segments: segments() })).toBe(first);
    expect(factory.create({ ...params, segments: segments(0.4) })).not.toBe(first);
  });

  it("should tick as one axis when one segment is in view", () => {
    const factory = newTickFactory({ type: "time" });
    const [a] = segments(1.5);
    expect(create(1.5)).toEqual(
      factory.create({ decimalToDataScale: a.decimalToDataScale, size: 800 }),
    );
  });
});
