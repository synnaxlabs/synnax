// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError } from "@/index";
import { SyntheticCore } from "@/testutil";

const DOMAIN = 1n << 32n;
const MS = TimeSpan.MILLISECOND.valueOf();

// One sample per millisecond from the epoch, so sample i is stamped i milliseconds.
const createCore = (now = TimeSpan.seconds(1)): SyntheticCore =>
  new SyntheticCore({
    rate: 1000,
    start: new TimeStamp(0),
    now: new TimeStamp(now.valueOf()),
  });

const ms = (n: number): TimeStamp => TimeStamp.milliseconds(n);

describe("SyntheticCore", () => {
  describe("channels", () => {
    it("should retrieve the index and data channels by key or name", async () => {
      const core = createCore();
      expect(await core.channels.retrieve(core.index.key)).toBe(core.index);
      expect(await core.channels.retrieve("data_0")).toBe(core.data[0]);
      expect(core.data[0].index).toBe(core.index.key);
      expect(core.index.dataType).toEqual(DataType.TIMESTAMP);
    });

    it("should throw a not found error for an unknown channel", async () => {
      await expect(createCore().channels.retrieve(99)).rejects.toThrow(
        new NotFoundError("channel 99 not found"),
      );
    });
  });

  describe("read", () => {
    it("should read raw samples in iterator-sized series", () => {
      const core = createCore(TimeSpan.seconds(250));
      const series = core
        .read(new TimeRange(ms(0), TimeStamp.seconds(250)), [core.index.key])
        .get(core.index.key).series;
      expect(series.map((s) => s.length)).toEqual([100_000, 100_000, 50_000]);
      expect(series.map((s) => s.alignment)).toEqual([
        DOMAIN,
        DOMAIN + 100_000n,
        DOMAIN + 200_000n,
      ]);
      expect(series[1].at(0)).toEqual(100_000n * MS);
    });

    it("should read only the samples written before the clock", () => {
      const core = createCore();
      const series = core
        .read(new TimeRange(ms(990), ms(2000)), [core.data[0].key])
        .get(core.data[0].key);
      expect(series.length).toBe(10);
    });

    it("should stamp each series with its domain inside the read", () => {
      const core = createCore(TimeSpan.seconds(250));
      const read = (tr: TimeRange, limited = false): TimeRange[] =>
        core
          .read(
            tr,
            [core.index.key],
            limited
              ? { variant: "limit", aggregation: "min_max", pointLimit: 10 }
              : undefined,
          )
          .get(core.index.key)
          .series.map((s) => s.timeRange);
      const end = ms(249_999).add(TimeSpan.NANOSECOND);
      expect(read(new TimeRange(ms(-5), TimeStamp.seconds(300)))).toEqual([
        new TimeRange(ms(0), ms(100_000)),
        new TimeRange(ms(100_000), ms(200_000)),
        new TimeRange(ms(200_000), end),
      ]);
      const inside = new TimeRange(ms(10).add(TimeSpan.NANOSECOND), ms(20));
      expect(read(inside)).toEqual([inside]);
      expect(read(inside, true)).toEqual([inside]);
      expect(read(new TimeRange(ms(-5), TimeStamp.seconds(300)), true)).toEqual([
        new TimeRange(ms(0), end),
      ]);
    });

    it("should write only the samples that the lag leaves behind the clock", () => {
      const core = new SyntheticCore({
        rate: 1000,
        start: new TimeStamp(0),
        now: ms(1000),
        lag: TimeSpan.milliseconds(400),
      });
      const series = core
        .read(new TimeRange(ms(0), ms(1000)), [core.index.key])
        .get(core.index.key);
      expect(series.length).toBe(600);
    });

    it("should read samples raw when they fit under the point limit", () => {
      const core = createCore();
      const [s] = core.read(new TimeRange(ms(0), ms(100)), [core.index.key], {
        variant: "limit",
        aggregation: "min_max",
        pointLimit: 100,
      }).series;
      expect(s.length).toBe(100);
      expect(s.alignmentMultiple).toBe(1n);
    });

    it("should reduce min_max groups of twice the decimated size", () => {
      const core = createCore();
      const frame = core.read(
        new TimeRange(ms(0), ms(1000)),
        [core.index.key, core.data[0].key],
        { variant: "limit", aggregation: "min_max", pointLimit: 100 },
      );
      const [time] = frame.get(core.index.key).series;
      const [data] = frame.get(core.data[0].key).series;
      expect(time.length).toBe(100);
      expect(data.length).toBe(100);
      expect(time.alignment).toBe(DOMAIN);
      expect(time.alignmentMultiple).toBe(10n);
      expect([time.at(0), time.at(1), time.at(2)]).toEqual([0n, 19n * MS, 20n * MS]);
    });

    it("should anchor groups at sample zero when a read starts inside one", () => {
      const core = createCore();
      const [time] = core.read(new TimeRange(ms(5), ms(1000)), [core.index.key], {
        variant: "limit",
        aggregation: "min_max",
        pointLimit: 100,
      }).series;
      expect(time.alignment).toBe(DOMAIN);
      expect([time.at(0), time.at(1)]).toEqual([5n * MS, 19n * MS]);
      expect(time.length).toBe(100);
    });

    it("should reduce decimate and average groups of the decimated size", () => {
      const core = createCore();
      const read = (aggregation: "decimate" | "average") =>
        core.read(new TimeRange(ms(0), ms(1000)), [core.index.key], {
          variant: "limit",
          aggregation,
          pointLimit: 100,
        }).series[0];
      const decimated = read("decimate");
      expect(decimated.alignmentMultiple).toBe(10n);
      expect([decimated.at(0), decimated.at(1)]).toEqual([0n, 10n * MS]);
      expect(read("average").at(0)).toBe((9n * MS) / 2n);
    });

    it("should throw on a stride reduction", () => {
      const core = createCore();
      expect(() =>
        core.read(new TimeRange(ms(0), ms(10)), [core.index.key], {
          variant: "stride",
          factor: 2,
        }),
      ).toThrow(new Error("synthetic core does not serve stride reductions"));
    });

    it("should count every read", () => {
      const core = createCore();
      core.read(new TimeRange(ms(0), ms(10)), [core.index.key]);
      core.read(new TimeRange(ms(2000), ms(3000)), [core.index.key]);
      expect(core.reads).toBe(2);
    });
  });

  describe("retype", () => {
    it("should read and stream the text of a channel retyped to string", async () => {
      const core = new SyntheticCore({
        rate: 1000,
        start: new TimeStamp(0),
        now: TimeStamp.seconds(1),
        text: (i) => `line ${i}`,
      });
      const { key } = core.data[0];
      core.retype(key, DataType.STRING);
      const [latest] = core.readLatest([key], 2).get(key).series;
      expect(latest.dataType).toEqual(DataType.STRING);
      expect(latest.toStrings()).toEqual(["line 998", "line 999"]);
      const streamer = core.openStreamer({ channels: [key] });
      core.advance(TimeSpan.milliseconds(1));
      const frame = await streamer.read();
      expect(frame.get(key).series[0].toStrings()).toEqual(["line 1000"]);
      streamer.close();
    });
  });

  describe("advance", () => {
    it("should advance the clock", () => {
      const core = createCore();
      core.advance(TimeSpan.seconds(1));
      expect(core.now()).toEqual(TimeStamp.seconds(2));
    });

    it("should throw when the samples outgrow one alignment domain", () => {
      const core = createCore();
      expect(() => core.advance(TimeSpan.days(50))).toThrow(
        new Error("synthetic core holds at most 2^32 samples"),
      );
    });

    it("should stream the samples written by an advance to a feed", async () => {
      const core = createCore();
      const feed = core.openFeed({ batchDebounce: TimeSpan.ZERO });
      const sub = feed.subscribe({
        keys: [core.index.key],
        limit: { variant: "latest", count: 20 },
      });
      // The read holds the samples written before the stream opened.
      await expect.poll(() => sub.value(core.index.key).status.variant).toBe("success");
      core.advance(TimeSpan.milliseconds(10));
      await expect
        .poll(() => sub.value(core.index.key).series.at(-1))
        .toEqual(1009n * MS);
      sub.close();
      await feed.close();
    });

    it("should serve the latest sample to a feed", async () => {
      const core = createCore();
      const feed = core.openFeed({ batchDebounce: TimeSpan.ZERO });
      const sub = feed.subscribe({
        keys: [core.index.key],
        limit: { variant: "latest", count: 1 },
      });
      await expect
        .poll(() => sub.value(core.index.key).series.at(-1))
        .toEqual(999n * MS);
      sub.close();
      await feed.close();
    });
  });
});
