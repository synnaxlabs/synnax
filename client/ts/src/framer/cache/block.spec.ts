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

import {
  align,
  countBefore,
  RAW_LEVEL,
  RawBlock,
  rawIndex,
  rawKey,
  split,
} from "@/framer/cache/block";
import { Dynamic } from "@/framer/cache/dynamic";
import { tileSpan } from "@/framer/cache/grid";
import { UnexpectedError } from "@/index";

const seconds = (s: number): TimeStamp => TimeStamp.seconds(s);

// Returns index samples at the given seconds, spanning the first to just past the last.
const createTimestamps = (alignment: bigint, ...at: number[]): Series =>
  new Series({
    data: new BigInt64Array(at.map((s) => seconds(s).valueOf())),
    dataType: DataType.TIMESTAMP,
    alignment,
    timeRange: new TimeRange(seconds(at[0]), seconds(at[at.length - 1]).add(1n)),
  });

const createValues = (
  alignment: bigint,
  values: number[],
  timeRange: TimeRange = seconds(1).range(seconds(2)),
): Series => new Series({ data: new Float32Array(values), alignment, timeRange });

const blockAlignment = (index: number, offset: number): bigint =>
  (BigInt(index) << 32n) | BigInt(offset);

describe("block", () => {
  describe("rawIndex", () => {
    it("should return the index of the raw block that holds a timestamp", () => {
      const span = tileSpan(RAW_LEVEL);
      expect(rawIndex(TimeStamp.ZERO)).toBe(0);
      expect(rawIndex(new TimeStamp(span.valueOf() - 1n))).toBe(0);
      expect(rawIndex(new TimeStamp(span.valueOf()))).toBe(1);
      expect(rawIndex(new TimeStamp(span.valueOf() * 7n + 5n))).toBe(7);
    });

    it("should round the index of a timestamp before the epoch down", () => {
      expect(rawIndex(new TimeStamp(-1n))).toBe(-1);
    });
  });

  describe("rawKey", () => {
    it("should key a raw block by channel and index", () => {
      expect(rawKey(12, 3)).toBe("12/raw/3");
    });
  });

  describe("align", () => {
    it("should count offsets of a series from the base", () => {
      const ts = createTimestamps(500n, 1, 2, 3, 4);
      const [aligned] = align(2, 10, [ts], [createValues(501n, [1, 2])]);
      expect(aligned.alignment).toBe(blockAlignment(2, 11));
      expect(Array.from(aligned)).toEqual([1, 2]);
    });

    it("should join index series with consecutive Core alignments", () => {
      const ts = [createTimestamps(500n, 1, 2), createTimestamps(502n, 3, 4)];
      const [aligned] = align(0, 0, ts, [createValues(501n, [1, 2, 3])]);
      expect(aligned.alignment).toBe(blockAlignment(0, 1));
    });

    it("should continue offsets across a jump in Core alignment", () => {
      const ts = [createTimestamps(500n, 1, 2), createTimestamps(9000n, 3, 4, 5)];
      const aligned = align(0, 0, ts, [
        createValues(500n, [1, 2]),
        createValues(9001n, [4, 5]),
      ]);
      expect(aligned.map((s) => s.alignment)).toEqual([
        blockAlignment(0, 0),
        blockAlignment(0, 3),
      ]);
    });

    it("should align index series to themselves", () => {
      const ts = [createTimestamps(500n, 1, 2), createTimestamps(9000n, 3)];
      const aligned = align(4, 7, ts, ts);
      expect(aligned.map((s) => s.alignment)).toEqual([
        blockAlignment(4, 7),
        blockAlignment(4, 9),
      ]);
      expect(Array.from(aligned[1])).toEqual([seconds(3).valueOf()]);
    });

    it("should keep the data and time range of each series", () => {
      const tr = seconds(5).range(seconds(9));
      const values = createValues(500n, [7, 8], tr);
      const [aligned] = align(0, 0, [createTimestamps(500n, 5, 6)], [values]);
      expect(aligned.data).toBe(values.data);
      expect(aligned.timeRange).toEqual(tr);
    });

    it("should skip empty series", () => {
      const empty = new Series({ data: new Float32Array([]), alignment: 7n });
      expect(align(0, 0, [], [empty])).toEqual([]);
    });

    it("should throw when no index samples share a series' Core alignment", () => {
      const ts = createTimestamps(500n, 1, 2);
      const aligned = () => align(3, 0, [ts], [createValues(501n, [1, 2])]);
      expect(aligned).toThrow(UnexpectedError);
      expect(aligned).toThrow("no index samples share alignment 501 in raw block 3");
    });
  });

  describe("split", () => {
    const span = tileSpan(RAW_LEVEL);
    // Returns index samples at the given offsets from the start of raw block 0.
    const createIndex = (alignment: bigint, ...ms: number[]): Series =>
      new Series({
        data: new BigInt64Array(ms.map((m) => TimeSpan.milliseconds(m).valueOf())),
        dataType: DataType.TIMESTAMP,
        alignment,
        timeRange: new TimeRange(0n, span.mult(3).valueOf()),
      });

    it("should split series at block edges and cut their time ranges", () => {
      const edge = Number(span.milliseconds);
      const ts = createIndex(500n, 1, edge - 1, edge, 2 * edge + 1);
      const values = createValues(500n, [1, 2, 3, 4], ts.timeRange);
      const parts = [...split([ts], [values]).values()];
      expect(parts.map((p) => p.series.map((s) => Array.from(s)))).toEqual([
        [[1, 2]],
        [[3]],
        [[4]],
      ]);
      expect(parts.map((p) => p.series[0].alignment)).toEqual([500n, 502n, 503n]);
      expect(parts.map((p) => p.timestamps[0].timeRange)).toEqual([
        new TimeRange(0n, span.valueOf()),
        new TimeRange(span.valueOf(), span.mult(2).valueOf()),
        new TimeRange(span.mult(2).valueOf(), span.mult(3).valueOf()),
      ]);
    });

    it("should split a series that spans index series of two blocks", () => {
      const edge = Number(span.milliseconds);
      const ts = [createIndex(500n, 1, 2), createIndex(502n, edge + 1)];
      const parts = [
        ...split(ts, [createValues(501n, [2, 3], ts[0].timeRange)]).values(),
      ];
      expect(parts.map((p) => p.series.map((s) => s.alignmentBounds))).toEqual([
        [{ lower: 501n, upper: 502n }],
        [{ lower: 502n, upper: 503n }],
      ]);
    });

    it("should give each block of a cut series its own buffer", () => {
      const edge = Number(span.milliseconds);
      const ts = createIndex(500n, 1, edge + 1);
      const parts = [
        ...split([ts], [createValues(500n, [1, 2], ts.timeRange)]).values(),
      ];
      const buffers = parts.flatMap((p) => [p.timestamps[0], p.series[0]]);
      buffers.forEach((s) =>
        expect(s.buffer.byteLength).toBe(s.byteCapacity.valueOf()),
      );
    });

    it("should make parts only for the blocks that hold index samples", () => {
      const edge = Number(span.milliseconds);
      const ts = createIndex(500n, 1, 2 * edge + 1);
      const parts = split([ts], [createValues(500n, [1, 2], ts.timeRange)]);
      expect([...parts.keys()]).toEqual([0, 2]);
    });

    it("should keep a series in one block as it is", () => {
      const ts = createIndex(500n, 1, 2);
      const values = createValues(500n, [1, 2], ts.timeRange);
      const [part] = split([ts], [values]).values();
      expect(part.series[0].buffer).toBe(values.buffer);
    });

    it("should leave out samples with no index sample of their Core alignment", () => {
      const ts = createIndex(500n, 1, 2);
      const [part] = split([ts], [createValues(501n, [1, 2])]).values();
      expect(part.series.map((s) => Array.from(s))).toEqual([[1]]);
      expect(part.series[0].alignment).toBe(501n);
    });
  });

  describe("countBefore", () => {
    it("should count the samples stamped before a timestamp across series", () => {
      const ts = [createTimestamps(0n, 1, 2, 3), createTimestamps(3n, 5, 8)];
      expect(countBefore(ts, seconds(0))).toBe(0);
      expect(countBefore(ts, seconds(3))).toBe(2);
      expect(countBefore(ts, seconds(6))).toBe(4);
      expect(countBefore(ts, seconds(9))).toBe(5);
      expect(countBefore([], seconds(9))).toBe(0);
    });
  });

  describe("RawBlock", () => {
    const fetched = (offset: number, values: number[], tr: TimeRange): Series =>
      createValues(blockAlignment(0, offset), values, tr);

    it("should span one raw tile of the grid", () => {
      const block = new RawBlock(1, 3);
      expect(block.range).toEqual(
        new TimeRange(tileSpan(RAW_LEVEL).mult(3), tileSpan(RAW_LEVEL).mult(4)),
      );
      expect(block.coveredEnd).toEqual(block.range.start);
    });

    describe("covers", () => {
      it("should cover nothing before a fill", () => {
        const block = new RawBlock(1, 0);
        expect(block.covers(seconds(1).range(seconds(2)))).toBe(false);
      });

      it("should cover a read that ends at or before the covered end", () => {
        const block = new RawBlock(1, 0);
        block.fill([], seconds(10), 0);
        expect(block.covers(seconds(1).range(seconds(10)))).toBe(true);
        expect(block.covers(seconds(9).range(seconds(11)))).toBe(false);
      });

      it("should cover a read past the block once the whole block is covered", () => {
        const block = new RawBlock(1, 0);
        block.fill([], block.range.end, 0);
        expect(block.covers(seconds(1).range(seconds(200)))).toBe(true);
      });
    });

    describe("fill", () => {
      it("should record the covered end and the index samples before it", () => {
        const block = new RawBlock(1, 0);
        block.fill([fetched(0, [1, 2], seconds(1).range(seconds(3)))], seconds(5), 2);
        expect(block.coveredEnd).toEqual(seconds(5));
        expect(block.coveredCount).toBe(2);
        expect(block.fetched.flatMap((s) => Array.from(s))).toEqual([1, 2]);
      });

      it("should stamp the first fill only", () => {
        const block = new RawBlock(1, 0);
        expect(block.fetchedAt).toEqual(TimeStamp.ZERO);
        block.fill([], seconds(5), 0);
        const first = block.fetchedAt;
        expect(first.after(TimeStamp.ZERO)).toBe(true);
        block.fill([], seconds(9), 0);
        expect(block.fetchedAt).toBe(first);
      });

      it("should keep the series it already holds when a tail overlaps them", () => {
        const block = new RawBlock(1, 0);
        const head = fetched(0, [1, 2, 3], seconds(1).range(seconds(4)));
        block.fill([head], seconds(3), 2);
        block.fill([fetched(2, [3, 4], seconds(3).range(seconds(5)))], seconds(6), 4);
        expect(block.fetched[0]).toBe(head);
        expect(block.fetched.flatMap((s) => Array.from(s))).toEqual([1, 2, 3, 4]);
      });

      it("should start a trimmed series where the series before it ends", () => {
        const block = new RawBlock(1, 0);
        block.fill(
          [fetched(0, [1, 2, 3], seconds(1).range(seconds(4)))],
          seconds(3),
          2,
        );
        block.fill([fetched(2, [3, 4], seconds(3).range(seconds(5)))], seconds(6), 4);
        expect(block.fetched[1].timeRange).toEqual(seconds(4).range(seconds(5)));
        expect(block.fetched[1].alignment).toBe(blockAlignment(0, 3));
      });

      it("should end a trimmed series where the series after it starts", () => {
        const block = new RawBlock(1, 0);
        const next = fetched(4, [5, 6], seconds(5).range(seconds(7)));
        block.fill(
          [
            fetched(0, [1, 2], seconds(1).range(seconds(3))),
            fetched(2, [3, 4], seconds(3).range(seconds(5))),
            next,
          ],
          seconds(3),
          2,
        );
        // The Core's series edges moved, so the tail's first series runs into the next.
        block.fill(
          [fetched(2, [3, 4, 5], seconds(3).range(seconds(6)))],
          seconds(6),
          5,
        );
        expect(block.fetched[1].timeRange).toEqual(seconds(3).range(seconds(5)));
        expect(block.fetched[2]).toBe(next);
        expect(block.fetched.flatMap((s) => Array.from(s))).toEqual([1, 2, 3, 4, 5, 6]);
      });
    });

    describe("expire", () => {
      it("should drop the fetched samples, their coverage, and the seal", () => {
        const block = new RawBlock(1, 0);
        block.fill([fetched(0, [1], seconds(1).range(seconds(2)))], seconds(5), 1);
        block.sealed = true;
        expect(block.expire()).toBe(true);
        expect(block.fetched).toEqual([]);
        expect(block.coveredEnd).toEqual(block.range.start);
        expect(block.coveredCount).toBe(0);
        expect(block.sealed).toBe(false);
      });

      it("should report a block with nothing to drop", () => {
        expect(new RawBlock(1, 0).expire()).toBe(false);
      });
    });

    describe("seriesIn", () => {
      it("should return the fetched series that overlap a range", () => {
        const block = new RawBlock(1, 0);
        const early = fetched(0, [1], seconds(1).range(seconds(2)));
        const late = fetched(1, [2], seconds(8).range(seconds(9)));
        block.fill([early, late], seconds(9), 2);
        expect(block.seriesIn(seconds(7).range(seconds(20)))).toEqual([late]);
      });

      it("should return the open buffer after the fetched series", () => {
        const block = new RawBlock(1, 0);
        const early = fetched(0, [1], seconds(1).range(seconds(2)));
        block.fill([early], seconds(4), 1);
        const buffer = new Dynamic({ dynamicBufferSize: 10 });
        buffer.write(
          new MultiSeries([createValues(9n, [1, 2], seconds(4).range(seconds(6)))]),
        );
        block.buffer = buffer;
        expect(block.seriesIn(seconds(0).range(seconds(20)))).toEqual([
          early,
          buffer.leadingBuffer,
        ]);
      });

      it("should judge the open buffer by the span of its samples", () => {
        const block = new RawBlock(1, 0);
        const buffer = new Dynamic({ dynamicBufferSize: 10 });
        buffer.write(
          new MultiSeries([createValues(9n, [1, 2], seconds(4).range(seconds(6)))]),
        );
        block.buffer = buffer;
        expect(block.seriesIn(seconds(7).range(seconds(20)))).toEqual([]);
        expect(block.seriesIn(seconds(5).range(seconds(20)))).toEqual([
          buffer.leadingBuffer,
        ]);
      });
    });

    describe("held", () => {
      it("should not hold an empty block", () => {
        expect(new RawBlock(1, 0).held).toBe(false);
      });

      it("should hold a block while the stream writes into it", () => {
        const block = new RawBlock(1, 0);
        block.buffer = new Dynamic({ dynamicBufferSize: 10 });
        expect(block.held).toBe(true);
      });

      it("should hold a block while a fetch fills it", () => {
        const block = new RawBlock(1, 0);
        block.pending = Promise.resolve();
        expect(block.held).toBe(true);
      });

      it("should hold a block while a subscription pins it", () => {
        const block = new RawBlock(1, 0);
        block.fill([fetched(0, [1], seconds(1).range(seconds(2)))], seconds(5), 1);
        block.pins++;
        expect(block.held).toBe(true);
        block.pins--;
        expect(block.held).toBe(false);
      });

      it("should not hold a block for a series a caller acquires", () => {
        const block = new RawBlock(1, 0);
        const series = fetched(0, [1], seconds(1).range(seconds(2)));
        block.fill([series], seconds(5), 1);
        series.acquire();
        expect(block.held).toBe(false);
        series.release();
      });
    });

    it("should stop counting the bytes of the series it drops", () => {
      const block = new RawBlock(1, 0);
      block.fill([fetched(0, [1, 2], seconds(1).range(seconds(5)))], seconds(5), 2);
      expect(block.bytes).toBe(2 * 4);
      block.expire();
      expect(block.bytes).toBe(0);
    });

    it("should count the bytes of every series it holds", () => {
      const block = new RawBlock(1, 0);
      block.fill([fetched(0, [1, 2], seconds(1).range(seconds(3)))], seconds(5), 2);
      const buffer = new Dynamic({ dynamicBufferSize: 10 });
      buffer.write(
        new MultiSeries([createValues(9n, [1], seconds(11).range(seconds(12)))]),
      );
      block.buffer = buffer;
      expect(block.bytes).toBe(2 * 4 + 10 * 4);
    });
  });
});
