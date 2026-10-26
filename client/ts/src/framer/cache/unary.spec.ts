// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DataType,
  MultiSeries,
  Series,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { UnexpectedError } from "@/errors";
import { Stamps } from "@/framer/cache/stamps";
import { Unary } from "@/framer/cache/unary";
import { glTransform } from "@/testutil/telem";

const LEADING_ALIGNMENT = (BigInt(0xffffffff) - 1_000_000n) << 32n;

const stamped = (
  startSec: number,
  endSec: number,
  data: number[],
  alignment: bigint,
): MultiSeries =>
  new MultiSeries([
    new Series({
      data: new Float32Array(data),
      dataType: DataType.FLOAT32,
      timeRange: TimeStamp.seconds(startSec).range(TimeStamp.seconds(endSec)),
      alignment,
    }),
  ]);

const newUnary = (): Unary => new Unary({ dynamicBufferSize: 100 });

describe("Unary", () => {
  describe("read with a live leading buffer", () => {
    // The buffer's samples carry provisional leading alignments that cannot pair
    // with fetched data on another channel, so it never claims coverage.
    it("should include the buffer and still report the full gap", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      const { series, gaps } = u.read(
        TimeStamp.seconds(5).range(TimeStamp.seconds(15)),
      );
      expect(series.series).toHaveLength(1);
      expect(series.series[0]).toBe(u.leadingBuffer);
      expect(gaps).toHaveLength(1);
      expect(gaps[0].equals(TimeStamp.seconds(5).range(TimeStamp.seconds(15)))).toBe(
        true,
      );
    });

    it("should report a gap over the buffer's own span", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      const { series, gaps } = u.read(
        TimeStamp.seconds(12).range(TimeStamp.seconds(20)),
      );
      expect(series.series).toHaveLength(1);
      expect(gaps).toHaveLength(1);
      expect(gaps[0].equals(TimeStamp.seconds(12).range(TimeStamp.seconds(20)))).toBe(
        true,
      );
    });

    it("should exclude the buffer when the read ends before it starts", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      const { series, gaps } = u.read(TimeStamp.seconds(2).range(TimeStamp.seconds(8)));
      expect(series.series).toHaveLength(0);
      expect(gaps).toHaveLength(1);
      expect(gaps[0].equals(TimeStamp.seconds(2).range(TimeStamp.seconds(8)))).toBe(
        true,
      );
    });

    // The buffer's own time range ends at the provisional TimeStamp.MAX, which must not
    // let it leak into reads of spans it holds no samples for.
    it("should exclude the buffer when the read starts after its data ends", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      const { series, gaps } = u.read(
        TimeStamp.seconds(14).range(TimeStamp.seconds(20)),
      );
      expect(series.series).toHaveLength(0);
      expect(gaps).toHaveLength(1);
      expect(gaps[0].equals(TimeStamp.seconds(14).range(TimeStamp.seconds(20)))).toBe(
        true,
      );
    });

    it("should include an unstamped buffer for a read spanning the present", () => {
      const u = newUnary();
      u.writeDynamic(
        new MultiSeries([
          new Series({
            data: new Float32Array([1, 2, 3]),
            dataType: DataType.FLOAT32,
            alignment: LEADING_ALIGNMENT,
          }),
        ]),
      );
      const now = TimeStamp.now();
      const { series } = u.read(
        now.sub(TimeSpan.seconds(5)).range(now.add(TimeSpan.seconds(5))),
      );
      expect(series.series).toHaveLength(1);
      expect(series.series[0]).toBe(u.leadingBuffer);
    });

    it("should compute gaps from fetched entries alone", () => {
      const u = newUnary();
      u.writeStatic(stamped(0, 4, [1, 2, 3, 4], 0n));
      u.writeDynamic(stamped(10, 13, [5, 6, 7], LEADING_ALIGNMENT));
      const { series, gaps } = u.read(
        new TimeRange(TimeStamp.seconds(0), TimeStamp.seconds(20)),
      );
      expect(series.series).toHaveLength(2);
      expect(series.series[1]).toBe(u.leadingBuffer);
      expect(gaps).toHaveLength(1);
      expect(gaps[0].start).toEqual(TimeStamp.seconds(4));
      expect(gaps[0].end).toEqual(TimeStamp.seconds(20));
    });
  });

  describe("streamed flushes", () => {
    it("should read the fetched form of a flushed buffer a fetch covers", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      // An alignment gap flushes the first buffer into the static cache.
      u.writeDynamic(stamped(20, 23, [4, 5, 6], LEADING_ALIGNMENT + 100n));
      u.writeStatic(stamped(10, 13, [1, 2, 3], 0n));
      const { series, gaps } = u.read(
        TimeStamp.seconds(10).range(TimeStamp.seconds(13)),
      );
      expect(series.series).toHaveLength(1);
      expect(series.series[0].alignment).toEqual(0n);
      expect(gaps).toHaveLength(0);
    });

    it("should keep a flushed buffer that no fetched write covers", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      u.writeDynamic(stamped(20, 23, [4, 5, 6], LEADING_ALIGNMENT + 100n));
      const { series } = u.read(TimeStamp.seconds(10).range(TimeStamp.seconds(13)));
      expect(series.series).toHaveLength(1);
      expect(series.series[0].alignment).toEqual(LEADING_ALIGNMENT);
    });
  });

  describe("data type changes", () => {
    const float64 = (startSec: number, endSec: number, alignment: bigint) =>
      new MultiSeries([
        new Series({
          data: new Float64Array([7, 8]),
          dataType: DataType.FLOAT64,
          timeRange: TimeStamp.seconds(startSec).range(TimeStamp.seconds(endSec)),
          alignment,
        }),
      ]);

    it("should read only the new type after a streamed type change", () => {
      const u = newUnary();
      u.writeStatic(stamped(0, 10, [1, 2], 0n));
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      u.writeDynamic(float64(13, 15, LEADING_ALIGNMENT + 3n));
      const { series, gaps } = u.read(
        TimeStamp.seconds(0).range(TimeStamp.seconds(15)),
      );
      expect(series.series).toEqual([u.leadingBuffer]);
      expect(series.dataType).toEqual(DataType.FLOAT64);
      expect(gaps).toEqual([TimeStamp.seconds(0).range(TimeStamp.seconds(15))]);
    });

    it("should drop a live buffer of the old type when a fetch changes the type", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      u.writeStatic(float64(0, 20, 0n));
      expect(u.leadingBuffer).toBeNull();
      const { series } = u.read(TimeStamp.seconds(0).range(TimeStamp.seconds(20)));
      expect(series.dataType).toEqual(DataType.FLOAT64);
    });

    it("should read only the new type after a fetched type change", () => {
      const u = newUnary();
      u.writeStatic(stamped(0, 10, [1, 2], 0n));
      u.writeStatic(float64(10, 20, 2n));
      const { series } = u.read(TimeStamp.seconds(0).range(TimeStamp.seconds(20)));
      expect(series.series).toHaveLength(1);
      expect(series.dataType).toEqual(DataType.FLOAT64);
    });
  });

  describe("flushDynamic", () => {
    it("should move the leading buffer to static without claiming coverage", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      u.flushDynamic();
      expect(u.leadingBuffer).toBeNull();
      const { series, gaps } = u.read(
        TimeStamp.seconds(5).range(TimeStamp.seconds(20)),
      );
      expect(series.series).toHaveLength(1);
      expect(Array.from(series.series[0])).toEqual([1, 2, 3]);
      expect(gaps).toHaveLength(1);
      expect(gaps[0].equals(TimeStamp.seconds(5).range(TimeStamp.seconds(20)))).toBe(
        true,
      );
    });

    it("should be a no-op when there is no leading buffer", () => {
      const u = newUnary();
      u.flushDynamic();
      expect(u.leadingBuffer).toBeNull();
      const { gaps } = u.read(TimeStamp.seconds(5).range(TimeStamp.seconds(20)));
      expect(gaps).toHaveLength(1);
    });
  });

  describe("lastWrite", () => {
    it("should be the stamp of the leading buffer's last sample", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      expect(u.lastWrite).toEqual(TimeStamp.seconds(13).sub(TimeSpan.NANOSECOND));
    });

    it("should be null before any write", () => {
      const u = newUnary();
      expect(u.lastWrite).toBeNull();
    });

    it("should be null once the leading buffer is flushed", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      u.flushDynamic();
      expect(u.lastWrite).toBeNull();
    });

    it("should throw after close", () => {
      const u = newUnary();
      u.close();
      expect(() => u.lastWrite).toThrow(UnexpectedError);
    });
  });

  describe("timeAt", () => {
    const T0 = TimeStamp.seconds(1_700_000_000).valueOf();
    // Deltas past 2^24 ns, which a float32 buffer cannot hold exactly.
    const NS = [T0, T0 + 20_000_001n, T0 + 40_000_003n];
    const stamps = (ns: bigint[], alignment: bigint): MultiSeries =>
      new MultiSeries([
        new Series({
          data: new BigInt64Array(ns),
          dataType: DataType.TIMESTAMP,
          timeRange: new TimeStamp(ns[0]).range(new TimeStamp(ns[ns.length - 1] + 1n)),
          alignment,
        }),
      ]);
    const newNarrowed = (): Unary =>
      new Unary({ dynamicBufferSize: 100, transform: glTransform });

    it("should return the exact stamp of a streamed sample through the index", () => {
      const index = newNarrowed();
      index.writeDynamic(stamps(NS, LEADING_ALIGNMENT));
      expect(index.leadingBuffer?.dataType.equals(DataType.FLOAT32)).toBe(true);
      const data = newNarrowed();
      data.index = index;
      expect(data.timeAt(LEADING_ALIGNMENT + 1n)).toEqual(new TimeStamp(NS[1]));
    });

    it("should find a sample after the leading buffer is flushed", () => {
      const index = newNarrowed();
      index.writeDynamic(stamps(NS, LEADING_ALIGNMENT));
      index.flushDynamic();
      index.index = index;
      expect(index.timeAt(LEADING_ALIGNMENT + 2n)).toEqual(new TimeStamp(NS[2]));
    });

    it("should find a fetched sample", () => {
      const index = newNarrowed();
      index.writeStatic(stamps(NS, 100n));
      index.index = index;
      expect(index.timeAt(101n)).toEqual(new TimeStamp(NS[1]));
    });

    it("should read stamps from the main buffer when nothing narrows them", () => {
      const index = newUnary();
      index.writeDynamic(stamps(NS, LEADING_ALIGNMENT));
      index.index = index;
      expect(index.timeAt(LEADING_ALIGNMENT + 1n)).toEqual(new TimeStamp(NS[1]));
    });

    it("should be null without an index", () => {
      const data = newNarrowed();
      expect(data.timeAt(LEADING_ALIGNMENT)).toBeNull();
    });

    it("should stamp a sample from the write that carried it without an index", () => {
      const data = newNarrowed();
      data.stamps = new Stamps(4);
      data.writeDynamic(stamped(10, 11, [1, 2, 3], LEADING_ALIGNMENT));
      data.writeDynamic(stamped(12, 13, [4], LEADING_ALIGNMENT + 3n));
      expect(data.timeAt(LEADING_ALIGNMENT + 2n)).toEqual(TimeStamp.seconds(10));
      expect(data.timeAt(LEADING_ALIGNMENT + 3n)).toEqual(TimeStamp.seconds(12));
      expect(data.timeAt(LEADING_ALIGNMENT + 4n)).toBeNull();
    });

    it("should stamp a fetched sample from the write that carried it without an index", () => {
      const data = newNarrowed();
      data.stamps = new Stamps(4);
      data.writeStatic(stamped(10, 11, [1], 100n));
      expect(data.timeAt(100n)).toEqual(TimeStamp.seconds(10));
    });

    it("should answer from the index alone when one is set", () => {
      const index = newNarrowed();
      index.writeDynamic(stamps(NS, LEADING_ALIGNMENT));
      const data = newNarrowed();
      data.index = index;
      data.stamps = new Stamps(4);
      data.writeDynamic(stamped(10, 11, [1, 2, 3, 4], LEADING_ALIGNMENT));
      expect(data.timeAt(LEADING_ALIGNMENT + 1n)).toEqual(new TimeStamp(NS[1]));
      expect(data.timeAt(LEADING_ALIGNMENT + 3n)).toBeNull();
    });

    it("should be null for an alignment the index does not hold", () => {
      const index = newNarrowed();
      index.writeDynamic(stamps(NS, LEADING_ALIGNMENT));
      index.index = index;
      expect(index.timeAt(LEADING_ALIGNMENT + 3n)).toBeNull();
    });

    it("should be null when the index holds no timestamps", () => {
      const index = newUnary();
      index.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      index.index = index;
      expect(index.timeAt(LEADING_ALIGNMENT)).toBeNull();
    });

    it("should keep a flushed stamp a fetch covers", () => {
      const index = newNarrowed();
      index.writeDynamic(stamps(NS, LEADING_ALIGNMENT));
      index.flushDynamic();
      index.writeStatic(stamps(NS, 100n));
      index.index = index;
      expect(index.timeAt(LEADING_ALIGNMENT + 1n)).toEqual(new TimeStamp(NS[1]));
      expect(index.timeAt(101n)).toEqual(new TimeStamp(NS[1]));
    });

    it("should keep a flushed stamp a fetch covers when nothing narrows it", () => {
      const index = newUnary();
      index.writeDynamic(stamps(NS, LEADING_ALIGNMENT));
      index.flushDynamic();
      index.writeStatic(stamps(NS, 100n));
      index.index = index;
      expect(index.timeAt(LEADING_ALIGNMENT + 1n)).toEqual(new TimeStamp(NS[1]));
    });

    it("should forget a sample the cache collected", () => {
      const index = new Unary({
        dynamicBufferSize: 100,
        transform: glTransform,
        staleEntryThreshold: TimeSpan.ZERO,
      });
      index.writeDynamic(stamps(NS, LEADING_ALIGNMENT));
      index.flushDynamic();
      index.index = index;
      expect(index.timeAt(LEADING_ALIGNMENT)).not.toBeNull();
      index.gc();
      expect(index.timeAt(LEADING_ALIGNMENT)).toBeNull();
    });

    it("should keep a retained stamp past the stale threshold", () => {
      const index = new Unary({
        dynamicBufferSize: 100,
        transform: glTransform,
        staleEntryThreshold: TimeSpan.ZERO,
      });
      index.writeDynamic(stamps(NS, LEADING_ALIGNMENT));
      index.flushDynamic();
      index.index = index;
      index.gc([{ lower: LEADING_ALIGNMENT + 2n, upper: LEADING_ALIGNMENT + 3n }]);
      expect(index.timeAt(LEADING_ALIGNMENT)).toEqual(new TimeStamp(NS[0]));
      index.gc();
      expect(index.timeAt(LEADING_ALIGNMENT)).toBeNull();
    });

    it("should keep a retained stamp when nothing narrows it", () => {
      const index = new Unary({
        dynamicBufferSize: 100,
        staleEntryThreshold: TimeSpan.ZERO,
      });
      index.writeDynamic(stamps(NS, LEADING_ALIGNMENT));
      index.flushDynamic();
      index.index = index;
      index.gc([{ lower: LEADING_ALIGNMENT, upper: LEADING_ALIGNMENT + 1n }]);
      expect(index.timeAt(LEADING_ALIGNMENT + 2n)).toEqual(new TimeStamp(NS[2]));
      index.gc();
      expect(index.timeAt(LEADING_ALIGNMENT + 2n)).toBeNull();
    });

    it("should throw after close", () => {
      const u = newUnary();
      u.close();
      expect(() => u.timeAt(0n)).toThrow(UnexpectedError);
    });
  });

  describe("markFetched", () => {
    const TR = TimeStamp.seconds(10).range(TimeStamp.seconds(13));

    it("should stop reporting a fetched span as a gap", () => {
      const u = newUnary();
      expect(u.read(TR).gaps).toHaveLength(1);
      u.markFetched(TR);
      expect(u.read(TR).gaps).toHaveLength(0);
    });

    it("should leave the spans around a fetched one as gaps", () => {
      const u = newUnary();
      u.markFetched(TR);
      const { gaps } = u.read(TimeStamp.seconds(5).range(TimeStamp.seconds(20)));
      expect(gaps.map((g) => g.toString())).toEqual([
        TimeStamp.seconds(5).range(TimeStamp.seconds(10)).toString(),
        TimeStamp.seconds(13).range(TimeStamp.seconds(20)).toString(),
      ]);
    });

    it("should ignore a zero-length span", () => {
      const u = newUnary();
      u.markFetched(TimeStamp.seconds(10).range(TimeStamp.seconds(10)));
      expect(u.read(TR).gaps).toHaveLength(1);
    });

    it("should cover the exact copy alongside the narrowed one", () => {
      const T0 = TimeStamp.seconds(10).valueOf();
      const index = new Unary({ dynamicBufferSize: 100, transform: glTransform });
      index.writeStatic(
        new MultiSeries([
          new Series({
            data: new BigInt64Array([T0, T0 + 1n, T0 + 2n]),
            dataType: DataType.TIMESTAMP,
            timeRange: TR,
            alignment: 100n,
          }),
        ]),
      );
      index.index = index;
      index.markFetched(TR);
      expect(index.read(TR).gaps).toHaveLength(0);
      expect(index.timeAt(101n)).toEqual(new TimeStamp(T0 + 1n));
    });

    it("should throw after close", () => {
      const u = newUnary();
      u.close();
      expect(() => u.markFetched(TR)).toThrow(UnexpectedError);
    });
  });

  describe("heldAlignments", () => {
    it("should report the spans of series a consumer holds", () => {
      const u = newUnary();
      u.writeStatic(stamped(10, 13, [1, 2, 3], 100n));
      expect(u.heldAlignments()).toEqual([]);
      const tr = TimeStamp.seconds(10).range(TimeStamp.seconds(13));
      u.read(tr).series.series.forEach((s) => s.acquire());
      expect(u.heldAlignments()).toEqual([{ lower: 100n, upper: 103n }]);
    });

    it("should report held static and leading spans together", () => {
      const u = newUnary();
      u.writeStatic(stamped(10, 13, [1, 2, 3], 100n));
      u.writeDynamic(stamped(20, 23, [4, 5, 6], LEADING_ALIGNMENT));
      const tr = TimeStamp.seconds(10).range(TimeStamp.seconds(13));
      u.read(tr).series.series.forEach((s) => s.acquire());
      u.leadingBuffer?.acquire();
      expect(u.heldAlignments()).toEqual([
        { lower: 100n, upper: 103n },
        { lower: LEADING_ALIGNMENT, upper: LEADING_ALIGNMENT + 3n },
      ]);
    });

    it("should report only the held series among several", () => {
      const u = newUnary();
      u.writeStatic(stamped(10, 13, [1, 2, 3], 100n));
      u.writeStatic(stamped(20, 23, [4, 5, 6], 200n));
      const tr = TimeStamp.seconds(20).range(TimeStamp.seconds(23));
      u.read(tr).series.series.forEach((s) => s.acquire());
      expect(u.heldAlignments()).toEqual([{ lower: 200n, upper: 203n }]);
    });

    it("should report a held leading buffer", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      expect(u.heldAlignments()).toEqual([]);
      u.leadingBuffer?.acquire();
      expect(u.heldAlignments()).toEqual([
        { lower: LEADING_ALIGNMENT, upper: LEADING_ALIGNMENT + 3n },
      ]);
    });

    it("should throw after close", () => {
      const u = newUnary();
      u.close();
      expect(() => u.heldAlignments()).toThrow(UnexpectedError);
    });
  });
});
