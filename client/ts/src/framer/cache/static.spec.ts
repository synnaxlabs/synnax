// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  bounds,
  DataType,
  MultiSeries,
  Series,
  sleep,
  type TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { Static } from "@/framer/cache/static";

// NOTE: Most of the insertion algorithm logic is not implemented in the static cache,
// but inside the x/ts/src/spatial/bounds module, where there are comprehensive tests.
// These tests are more focused on reading than writing.
describe("StaticReadCache", () => {
  describe("read and write", () => {
    test("simple write", () => {
      const c = new Static({});
      const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([1]),
            dataType: DataType.FLOAT32,
            timeRange: tr,
            alignment: 0n,
          }),
        ]),
      );
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3)),
      );
      expect(series).toHaveLength(1);
      expect(gaps).toHaveLength(0);
    });
    test("should correctly return leading and trailing gaps", () => {
      const c = new Static({});
      const tr = TimeStamp.seconds(2).spanRange(TimeSpan.seconds(3));
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([1]),
            dataType: DataType.FLOAT32,
            timeRange: tr,
            alignment: 0n,
          }),
        ]),
      );
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(1).spanRange(TimeSpan.seconds(6)),
      );
      expect(series).toHaveLength(1);
      expect(gaps).toHaveLength(2);
      expect(gaps[0].start).toEqual(TimeStamp.seconds(1));
      expect(gaps[0].end).toEqual(TimeStamp.seconds(2));
      expect(gaps[1].start).toEqual(TimeStamp.seconds(5));
      expect(gaps[1].end).toEqual(TimeStamp.seconds(7));
    });
    // Input:
    // [1,2]
    //      [,,][4,5]
    test("internal gaps", () => {
      const c = new Static({});
      const tr1 = TimeStamp.seconds(1).range(TimeStamp.seconds(3));
      const tr2 = TimeStamp.seconds(4).range(TimeStamp.seconds(6));
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([1, 2]),
            dataType: DataType.FLOAT32,
            timeRange: tr1,
            alignment: 1n,
          }),
        ]),
      );
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([4, 5]),
            dataType: DataType.FLOAT32,
            timeRange: tr2,
            alignment: 4n,
          }),
        ]),
      );
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(1).range(TimeStamp.seconds(7)),
      );
      expect(series).toHaveLength(4);
      expect(gaps).toHaveLength(2);
      expect(gaps[0].start).toEqual(TimeStamp.seconds(3));
      expect(gaps[0].end).toEqual(TimeStamp.seconds(4));
      expect(gaps[1].start).toEqual(TimeStamp.seconds(6));
      expect(gaps[1].end).toEqual(TimeStamp.seconds(7));
    });
    test("overlapping insert on a string series trims by sample", () => {
      const c = new Static({});
      c.write(
        new MultiSeries([
          new Series({
            data: ["a", "bb", "ccc", "dddd"],
            timeRange: TimeStamp.seconds(1).range(TimeStamp.seconds(3)),
            alignment: 0n,
          }),
        ]),
      );
      c.write(
        new MultiSeries([
          new Series({
            data: ["ccc", "dddd", "ee", "f"],
            timeRange: TimeStamp.seconds(2).range(TimeStamp.seconds(4)),
            alignment: 2n,
          }),
        ]),
      );
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(1).range(TimeStamp.seconds(4)),
      );
      expect(gaps).toHaveLength(0);
      const strings = series.series.flatMap((s) => s.toStrings());
      expect(strings).toEqual(["a", "bb", "ccc", "dddd", "ee", "f"]);
    });

    // Input:
    // [2,3,4,5]
    //     [4,5,6]
    // Expected
    //
    // [2,3,4,5][6]
    test("insert after overlap last", () => {
      const c = new Static({});
      const tr1 = TimeStamp.seconds(2).range(TimeStamp.seconds(6));
      const tr2 = TimeStamp.seconds(4).range(TimeSpan.seconds(7));
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([2, 3, 4, 5]),
            dataType: DataType.FLOAT32,
            timeRange: tr1,
            alignment: 2n,
          }),
        ]),
      );
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([4, 5, 6]),
            dataType: DataType.FLOAT32,
            timeRange: tr2,
            alignment: 4n,
          }),
        ]),
      );
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(2).range(TimeSpan.seconds(7)),
      );
      expect(series).toHaveLength(5);
      expect(gaps).toHaveLength(0);
    });
    // Input
    //     [3,4,5,6]
    // [1,2,3]
    //
    // Expected
    //
    // [1,2][3,4,5,6]
    test("insert before overlap first", () => {
      const c = new Static({});
      const tr1 = TimeStamp.seconds(3).range(TimeSpan.seconds(7));
      const tr2 = TimeStamp.seconds(1).range(TimeSpan.seconds(4));
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([3, 4, 5, 6]),
            dataType: DataType.FLOAT32,
            timeRange: tr1,
            alignment: 3n,
          }),
        ]),
      );
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([1, 2, 3]),
            dataType: DataType.FLOAT32,
            timeRange: tr2,
            alignment: 1n,
          }),
        ]),
      );
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(1).range(TimeSpan.seconds(7)),
      );
      expect(series).toHaveLength(6);
      expect(gaps).toHaveLength(0);
    });

    // Input
    // [1,2,3,4]
    // [5,6,7,8]
    //
    // Expected
    // [5,6,7,8]
    test("completely overlapping series", () => {
      const c = new Static({});
      const tr1 = TimeStamp.seconds(1).range(TimeSpan.seconds(4));
      const tr2 = TimeStamp.seconds(1).range(TimeSpan.seconds(4));
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([1, 2, 3, 4]),
            dataType: DataType.FLOAT32,
            timeRange: tr1,
            alignment: 0n,
          }),
        ]),
      );
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([5, 6, 7, 8]),
            dataType: DataType.FLOAT32,
            timeRange: tr2,
            alignment: 0n,
          }),
        ]),
      );
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(2).range(TimeStamp.seconds(4)),
      );
      expect(series.series).toHaveLength(1);
      expect(series.series[0].data).toEqual(new Float32Array([5, 6, 7, 8]));
      expect(gaps).toHaveLength(0);
    });
  });
  describe("streamed entries", () => {
    // Streamed data carries leading-region alignments that never match the
    // positional alignments of the same samples read back from disk, so the two
    // forms of one sample can coexist in the cache. Evicting streamed entries is
    // what prevents that.
    const LEADING_ALIGNMENT = (BigInt(0xffffffff) - 1_000_000n) << 32n;
    const streamed = (tr: TimeRange, data: number[], offset = 0n) =>
      new MultiSeries([
        new Series({
          data: new Float32Array(data),
          dataType: DataType.FLOAT32,
          timeRange: tr,
          alignment: LEADING_ALIGNMENT + offset,
        }),
      ]);
    const fetched = (tr: TimeRange, data: number[], alignment = 0n) =>
      new MultiSeries([
        new Series({
          data: new Float32Array(data),
          dataType: DataType.FLOAT32,
          timeRange: tr,
          alignment,
        }),
      ]);

    it("should return streamed entries without letting them claim coverage", () => {
      const c = new Static({});
      const tr = TimeStamp.seconds(10).range(TimeStamp.seconds(20));
      c.write(streamed(tr, [1, 2]), true);
      const { series, gaps } = c.dirtyRead(tr);
      expect(series.series).toHaveLength(1);
      expect(gaps).toHaveLength(1);
      expect(gaps[0].equals(tr)).toBe(true);
    });

    it("should compute gaps from fetched entries around a streamed one", () => {
      const c = new Static({});
      c.write(
        streamed(TimeStamp.seconds(12).range(TimeStamp.seconds(16)), [1, 2]),
        true,
      );
      c.write(fetched(TimeStamp.seconds(10).range(TimeStamp.seconds(12)), [1, 2]));
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(10).range(TimeStamp.seconds(20)),
      );
      expect(series.series).toHaveLength(2);
      expect(gaps).toHaveLength(1);
      expect(gaps[0].equals(TimeStamp.seconds(12).range(TimeStamp.seconds(20)))).toBe(
        true,
      );
    });

    it("should evict a streamed entry when a fetched write overlaps it", () => {
      const c = new Static({});
      const tr = TimeStamp.seconds(10).range(TimeStamp.seconds(20));
      c.write(streamed(tr, [1, 2]), true);
      c.write(fetched(tr, [1, 2]));
      const { series, gaps } = c.dirtyRead(tr);
      expect(series.series).toHaveLength(1);
      expect(series.series[0].alignment).toEqual(0n);
      expect(gaps).toHaveLength(0);
    });

    it("should keep a streamed entry a fetched write only partially covers", () => {
      // The fetch may be stamped wider than the committed data it returned, so a
      // partial cover must not drop the streamed samples it did not replace.
      const c = new Static({});
      c.write(
        streamed(TimeStamp.seconds(12).range(TimeStamp.seconds(20)), [1, 2]),
        true,
      );
      c.write(fetched(TimeStamp.seconds(10).range(TimeStamp.seconds(15)), [1, 2]));
      const { series } = c.dirtyRead(
        TimeStamp.seconds(10).range(TimeStamp.seconds(20)),
      );
      expect(series.series).toHaveLength(2);
    });

    it("should keep streamed entries that do not overlap the write", () => {
      const c = new Static({});
      c.write(
        streamed(TimeStamp.seconds(30).range(TimeStamp.seconds(40)), [3, 4]),
        true,
      );
      c.write(fetched(TimeStamp.seconds(10).range(TimeStamp.seconds(20)), [1, 2]));
      const { series } = c.dirtyRead(
        TimeStamp.seconds(10).range(TimeStamp.seconds(40)),
      );
      expect(series.series).toHaveLength(2);
    });

    it("should not evict streamed entries on a streamed write", () => {
      const c = new Static({});
      c.write(
        streamed(TimeStamp.seconds(10).range(TimeStamp.seconds(20)), [1, 2]),
        true,
      );
      c.write(
        streamed(TimeStamp.seconds(15).range(TimeStamp.seconds(25)), [3, 4], 100n),
        true,
      );
      const { series } = c.dirtyRead(
        TimeStamp.seconds(10).range(TimeStamp.seconds(25)),
      );
      expect(series.series).toHaveLength(2);
    });

    it("should not evict fetched entries", () => {
      const c = new Static({});
      const tr = TimeStamp.seconds(10).range(TimeStamp.seconds(20));
      c.write(fetched(tr, [1, 2]));
      c.write(fetched(TimeStamp.seconds(30).range(TimeStamp.seconds(40)), [3, 4], 20n));
      const { series } = c.dirtyRead(
        TimeStamp.seconds(10).range(TimeStamp.seconds(40)),
      );
      expect(series.series).toHaveLength(2);
    });

    // Streamed entries arrive continuously, so a collector that misses them leaks.
    it("should garbage collect streamed entries alongside fetched ones", async () => {
      const c = new Static({ staleEntryThreshold: TimeSpan.milliseconds(5) });
      const streamedTR = TimeStamp.seconds(30).range(TimeStamp.seconds(40));
      const fetchedTR = TimeStamp.seconds(10).range(TimeStamp.seconds(20));
      c.write(streamed(streamedTR, [3, 4]), true);
      c.write(fetched(fetchedTR, [1, 2]));
      const read = () =>
        c.dirtyRead(TimeStamp.seconds(10).range(TimeStamp.seconds(40))).series.series;
      expect(read()).toHaveLength(2);
      await sleep.sleep(TimeSpan.milliseconds(10));
      expect(c.gc().purgedSeries).toEqual(2);
      expect(read()).toHaveLength(0);
    });
  });

  describe("garbage collection", () => {
    it("should correctly garbage collect series that have a reference count of zero", async () => {
      const c = new Static({ staleEntryThreshold: TimeSpan.milliseconds(5) });
      const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([1]),
            dataType: DataType.FLOAT32,
            timeRange: tr,
            alignment: 0n,
          }),
        ]),
      );
      const read = () =>
        c.dirtyRead(TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3))).series;
      expect(c.gc().purgedSeries).toEqual(0);
      expect(read()).toHaveLength(1);
      await expect.poll(async () => c.gc().purgedSeries === 1).toBe(true);
      expect(read()).toHaveLength(0);
    });
    it("should not garbage collect series that have a reference count greater than zero", async () => {
      const c = new Static({ staleEntryThreshold: TimeSpan.milliseconds(5) });
      const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([1]),
            dataType: DataType.FLOAT32,
            timeRange: tr,
            alignment: 0n,
          }),
        ]),
      );
      const read = () =>
        c.dirtyRead(TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3))).series;
      read().series.forEach((s) => s.acquire());
      expect(c.gc().purgedSeries).toEqual(0);
      expect(read().series.length).toEqual(1);
      c.dirtyRead(TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3)));
      expect(c.gc().purgedSeries).toEqual(0);
      expect(read().series.length).toEqual(1);
    });
  });
  describe("close", () => {
    it("should remove all series from the cache", () => {
      const c = new Static({});
      const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
      c.write(
        new MultiSeries([
          new Series({
            data: new Float32Array([1]),
            dataType: DataType.FLOAT32,
            timeRange: tr,
            alignment: 0n,
          }),
        ]),
      );
      c.close();
      expect(
        c.dirtyRead(TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3))).series,
      ).toHaveLength(0);
    });
  });
  describe("integrity repair", () => {
    // Reports honest bounds until the corrupt flag flips, simulating an insertion
    // bug that leaves stored entries overlapping.
    class CorruptedSeries extends Series {
      corrupt = false;
      override get alignmentBounds(): bounds.Bounds<bigint> {
        if (this.corrupt) return bounds.construct(0n, 3n);
        return super.alignmentBounds;
      }
    }
    const plain = (startSec: number, data: number[], alignment: bigint) =>
      new Series({
        data: new Float32Array(data),
        dataType: DataType.FLOAT32,
        timeRange: TimeStamp.seconds(startSec).range(TimeStamp.seconds(startSec + 3)),
        alignment,
      });

    it("should evict overlapping entries instead of failing every later write", () => {
      const c = new Static({});
      c.write(new MultiSeries([plain(0, [1, 2, 3], 0n)]));
      const corrupted = new CorruptedSeries({
        data: new Float32Array([4, 5, 6]),
        dataType: DataType.FLOAT32,
        timeRange: TimeStamp.seconds(10).range(TimeStamp.seconds(13)),
        alignment: 10n,
      });
      c.write(new MultiSeries([corrupted]));
      corrupted.corrupt = true;
      c.write(new MultiSeries([plain(20, [7, 8, 9], 20n)]));
      const { series } = c.dirtyRead(TimeStamp.seconds(0).range(TimeStamp.seconds(25)));
      expect(series.series).toHaveLength(1);
      expect(series.series[0].alignment).toEqual(20n);
    });

    it("should keep accepting writes after a repair", () => {
      const c = new Static({});
      const corrupted = new CorruptedSeries({
        data: new Float32Array([4, 5, 6]),
        dataType: DataType.FLOAT32,
        timeRange: TimeStamp.seconds(10).range(TimeStamp.seconds(13)),
        alignment: 10n,
      });
      c.write(new MultiSeries([plain(0, [1, 2, 3], 0n)]));
      c.write(new MultiSeries([corrupted]));
      corrupted.corrupt = true;
      c.write(new MultiSeries([plain(20, [7, 8, 9], 20n)]));
      c.write(new MultiSeries([plain(30, [10, 11, 12], 30n)]));
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(20).range(TimeStamp.seconds(33)),
      );
      expect(series.series).toHaveLength(2);
      expect(gaps).toHaveLength(1);
    });
  });

  describe("integrity", () => {
    it("should accept many sequential non-overlapping writes without error", () => {
      const c = new Static({});
      const n = 300;
      for (let i = 0; i < n; i++) {
        const start = TimeStamp.seconds(i * 10);
        c.write(
          new MultiSeries([
            new Series({
              data: new Float32Array([i]),
              dataType: DataType.FLOAT32,
              timeRange: start.range(start.add(TimeSpan.seconds(1))),
              alignment: BigInt(i * 10),
            }),
          ]),
        );
      }
      const { series } = c.dirtyRead(
        TimeStamp.seconds(0).spanRange(TimeSpan.seconds(n * 10)),
      );
      expect(series.length).toEqual(n);
    });
  });
});
