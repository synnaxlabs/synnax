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
import { Unary } from "@/framer/cache/unary";

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
    it("should evict a flushed buffer when a fetched write covers it", () => {
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
    it("should be the end of the leading buffer's last stamped write", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      expect(u.lastWrite).toEqual(TimeStamp.seconds(13));
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

  describe("latest", () => {
    const sample = (): MultiSeries => stamped(10, 11, [1], 0n);

    it("should be null before a read is stored", () => {
      const u = newUnary();
      expect(u.latest).toBeNull();
    });

    it("should serve a read stored under the current version", () => {
      const u = newUnary();
      const series = sample();
      u.storeLatest(series, u.version);
      expect(u.latest).toBe(series);
    });

    it("should refuse a read whose version a live write moved", () => {
      const u = newUnary();
      const version = u.version;
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      u.storeLatest(sample(), version);
      expect(u.latest).toBeNull();
    });

    it("should refuse a read whose version a flush moved", () => {
      const u = newUnary();
      const version = u.version;
      u.flushDynamic();
      u.storeLatest(sample(), version);
      expect(u.latest).toBeNull();
    });

    it("should stop serving once a live write lands", () => {
      const u = newUnary();
      u.storeLatest(sample(), u.version);
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      expect(u.latest).toBeNull();
    });

    it("should stop serving once the leading buffer is flushed", () => {
      const u = newUnary();
      u.storeLatest(sample(), u.version);
      u.flushDynamic();
      expect(u.latest).toBeNull();
    });

    it("should serve a read stored after the last change", () => {
      const u = newUnary();
      u.writeDynamic(stamped(10, 13, [1, 2, 3], LEADING_ALIGNMENT));
      u.flushDynamic();
      const series = sample();
      u.storeLatest(series, u.version);
      expect(u.latest).toBe(series);
    });

    it("should throw after close", () => {
      const u = newUnary();
      u.close();
      expect(() => u.latest).toThrow(UnexpectedError);
      expect(() => u.version).toThrow(UnexpectedError);
      expect(() => u.storeLatest(sample(), 0)).toThrow(UnexpectedError);
    });
  });
});
