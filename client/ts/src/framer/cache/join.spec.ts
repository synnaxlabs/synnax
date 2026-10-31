// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeRange, TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { alignmentOf, RAW_LEVEL, rawIndex } from "@/framer/cache/block";
import { tileRange } from "@/framer/cache/grid";
import { Joins, type Wait } from "@/framer/cache/join";
import { Store } from "@/framer/cache/store";

const INDEX = 1;
const DATA = 2;
const BLOCK = 100;
const RANGE = tileRange({ level: RAW_LEVEL, index: BLOCK });

const at = (seconds: number): TimeStamp => RANGE.start.add(TimeSpan.seconds(seconds));

const spanOf = (seconds: number[]): TimeRange =>
  new TimeRange(at(seconds[0]), at(seconds[seconds.length - 1]).add(1));

const createIndex = (seconds: number[], alignment: bigint): Series =>
  new Series({
    data: new BigInt64Array(seconds.map((s) => at(s).valueOf())),
    dataType: DataType.TIMESTAMP,
    alignment,
    timeRange: spanOf(seconds),
  });

const createData = (
  values: number[],
  alignment: bigint,
  timeRange: TimeRange = TimeRange.ZERO,
): Series => new Series({ data: new Float32Array(values), alignment, timeRange });

const createFrame = (...entries: Array<[channel.Key, Series]>) =>
  new Map(entries.map(([k, s]) => [k, [s]]));

const createJoins = () => {
  const store = new Store({ dynamicBufferSize: 100 });
  const joins = new Joins(store);
  joins.join(DATA, INDEX);
  return { store, joins };
};

// Lands an empty catch-up read for both members and writes one placed frame.
const createJoined = () => {
  const { store, joins } = createJoins();
  joins.landed(INDEX, waitsOf(joins, INDEX, DATA));
  joins.write(
    createFrame([INDEX, createIndex([1, 2], 0n)], [DATA, createData([10, 20], 0n)]),
  );
  return { store, joins };
};

// Returns the waits of the given members, as a catch-up read of them begins.
const waitsOf = (joins: Joins, ...keys: channel.Key[]): Wait[] =>
  joins.waiting(INDEX).filter((w) => keys.includes(w.key));

const keysOf = (joins: Joins): channel.Key[] => joins.waiting(INDEX).map((w) => w.key);

const leadingOf = (store: Store, ch: channel.Key): Series | null =>
  store.open(ch)?.buffer?.leadingBuffer ?? null;

const streamed = (store: Store, ch: channel.Key): number[] =>
  Array.from(leadingOf(store, ch) ?? []).map(Number);

// Returns the series of the channel's raw blocks in time order, the open one last.
const seriesOf = (store: Store, ch: channel.Key, ...blocks: number[]): Series[] => {
  const fetched = blocks.flatMap((b) => [...store.raw(ch, b).fetched]);
  const leading = leadingOf(store, ch);
  return leading == null ? fetched : [...fetched, leading];
};

// Returns whether each series after the first starts where the one before it ends.
const touches = (series: Series[]): boolean[] =>
  series.slice(1).map((s, i) => series[i].timeRange.end.equals(s.timeRange.start));

// Fills the block with a read of both members' samples at the seconds.
const fillRead = (store: Store, seconds: number[]): void => {
  const span = spanOf(seconds);
  const alignment = alignmentOf(BLOCK, 0);
  store
    .raw(INDEX, BLOCK)
    .fill([createIndex(seconds, alignment)], span.end, seconds.length);
  store
    .raw(DATA, BLOCK)
    .fill([createData(seconds, alignment, span)], span.end, seconds.length);
};

const NEXT_DOMAIN = alignmentOf(1, 0);

describe("Joins", () => {
  describe("landed", () => {
    it("should hold the series of waiting members until the join lands", () => {
      const { store, joins } = createJoins();
      const waiting = joins.write(
        createFrame(
          [INDEX, createIndex([1, 2, 3], 0n)],
          [DATA, createData([10, 20, 30], 0n)],
        ),
      );
      expect(waiting.size).toBe(0);
      expect(leadingOf(store, DATA)).toBeNull();
      expect(keysOf(joins)).toEqual([INDEX, DATA]);
      expect(joins.landed(INDEX, waitsOf(joins, INDEX, DATA)).size).toBe(0);
      expect(keysOf(joins)).toEqual([]);
      expect(joins.joined(DATA)).toBe(true);
      expect(streamed(store, DATA)).toEqual([10, 20, 30]);
      expect(leadingOf(store, DATA)?.alignment).toBe(alignmentOf(BLOCK, 0));
      expect(store.raw(DATA, BLOCK).coveredEnd).toEqual(at(3).add(1));
    });

    it("should keep a member waiting that waited again since its read began", () => {
      const { joins } = createJoins();
      const waits = joins.waiting(INDEX);
      joins.flush(DATA);
      joins.landed(INDEX, waits);
      expect(joins.joined(DATA)).toBe(false);
      expect(keysOf(joins)).toEqual([DATA]);
    });

    it("should keep a member waiting that joined again since its read began", () => {
      const { joins } = createJoins();
      joins.join(INDEX, INDEX);
      const waits = joins.waiting(INDEX);
      joins.leave(DATA, INDEX);
      joins.join(DATA, INDEX);
      joins.landed(INDEX, waits);
      expect(joins.joined(DATA)).toBe(false);
      expect(keysOf(joins)).toEqual([DATA]);
    });

    it("should keep a data member waiting until its index lands", () => {
      const { joins } = createJoins();
      joins.landed(INDEX, waitsOf(joins, DATA));
      expect(joins.joined(DATA)).toBe(false);
      expect(keysOf(joins)).toEqual([INDEX]);
    });

    it("should give streamed samples the block offsets after the read's samples", () => {
      const { store, joins } = createJoins();
      const fetched = spanOf([1, 2, 3, 4]);
      store
        .raw(INDEX, BLOCK)
        .fill([createIndex([1, 2, 3, 4], alignmentOf(BLOCK, 0))], fetched.end, 4);
      store
        .raw(DATA, BLOCK)
        .fill(
          [createData([10, 20, 30, 40], alignmentOf(BLOCK, 0), fetched)],
          fetched.end,
          4,
        );
      joins.write(
        createFrame(
          [INDEX, createIndex([3, 4, 5, 6], 10n)],
          [DATA, createData([30, 40, 50, 60], 10n)],
        ),
      );
      expect(joins.landed(INDEX, waitsOf(joins, INDEX, DATA)).size).toBe(0);
      expect(streamed(store, DATA)).toEqual([50, 60]);
      expect(leadingOf(store, DATA)?.alignment).toBe(alignmentOf(BLOCK, 4));
      expect(store.raw(INDEX, BLOCK).coveredCount).toBe(6);
      expect(store.raw(DATA, BLOCK).coveredCount).toBe(6);
    });

    it("should place streamed samples that a read's coverage claimed", () => {
      const { store, joins } = createJoins();
      // A read claims coverage past its newest sample when a writer lags.
      const claimed = at(10);
      const fetched = spanOf([1, 2, 3, 4]);
      store
        .raw(INDEX, BLOCK)
        .fill([createIndex([1, 2, 3, 4], alignmentOf(BLOCK, 0))], claimed, 4);
      store
        .raw(DATA, BLOCK)
        .fill(
          [createData([10, 20, 30, 40], alignmentOf(BLOCK, 0), fetched)],
          claimed,
          4,
        );
      joins.write(
        createFrame(
          [INDEX, createIndex([4, 5, 6], 10n)],
          [DATA, createData([40, 50, 60], 10n)],
        ),
      );
      expect(joins.landed(INDEX, waitsOf(joins, INDEX, DATA)).size).toBe(0);
      expect(streamed(store, DATA)).toEqual([50, 60]);
      expect(leadingOf(store, DATA)?.alignment).toBe(alignmentOf(BLOCK, 4));
      expect(store.raw(INDEX, BLOCK).coveredEnd).toEqual(claimed);
    });

    it("should place samples the data read missed and the index read held", () => {
      const { store, joins } = createJoins();
      store
        .raw(INDEX, BLOCK)
        .fill(
          [createIndex([1, 2, 3, 4, 5, 6], alignmentOf(BLOCK, 0))],
          at(6).add(1),
          6,
        );
      store
        .raw(DATA, BLOCK)
        .fill(
          [createData([10, 20, 30, 40], alignmentOf(BLOCK, 0), spanOf([1, 2, 3, 4]))],
          at(4).add(1),
          4,
        );
      joins.write(
        createFrame(
          [INDEX, createIndex([5, 6, 7, 8], 10n)],
          [DATA, createData([50, 60, 70, 80], 10n)],
        ),
      );
      expect(joins.landed(INDEX, waitsOf(joins, INDEX, DATA)).size).toBe(0);
      expect(joins.bounced(DATA)).toBe(false);
      expect(streamed(store, DATA)).toEqual([50, 60, 70, 80]);
      expect(leadingOf(store, DATA)?.alignment).toBe(alignmentOf(BLOCK, 4));
    });

    it("should give fresh samples the offsets after a block's newest sample", () => {
      const { store, joins } = createJoins();
      // The read holds samples 5 and 6 past its coverage, which ends at sample 4.
      store
        .raw(INDEX, BLOCK)
        .fill(
          [createIndex([1, 2, 3, 4, 5, 6], alignmentOf(BLOCK, 0))],
          at(4).add(1),
          4,
        );
      joins.write(createFrame([INDEX, createIndex([5, 6, 7, 8], 10n)]));
      joins.landed(INDEX, waitsOf(joins, INDEX));
      expect(leadingOf(store, INDEX)?.length).toBe(2);
      expect(leadingOf(store, INDEX)?.alignment).toBe(alignmentOf(BLOCK, 6));
    });

    it("should send back a member whose read missed samples before the stream", () => {
      const { store, joins } = createJoins();
      const fetched = spanOf([1, 2, 3, 4]);
      store
        .raw(INDEX, BLOCK)
        .fill([createIndex([1, 2, 3, 4], alignmentOf(BLOCK, 0))], fetched.end, 4);
      joins.write(
        createFrame(
          [INDEX, createIndex([5, 6], 10n)],
          [DATA, createData([50, 60], 10n)],
        ),
      );
      expect(joins.landed(INDEX, waitsOf(joins, INDEX, DATA))).toEqual(
        new Set([INDEX]),
      );
      expect(keysOf(joins)).toEqual([DATA]);
      expect(joins.bounced(DATA)).toBe(true);
      expect(joins.catchUpFrom(DATA, at(1000))).toBe(BLOCK);
      expect(leadingOf(store, DATA)).toBeNull();
    });

    it("should place a historyless member's first samples at their index offsets", () => {
      const { store, joins } = createJoins();
      const fetched = spanOf([1, 2, 3, 4]);
      store
        .raw(INDEX, BLOCK)
        .fill([createIndex([1, 2, 3, 4], alignmentOf(BLOCK, 0))], fetched.end, 4);
      joins.landed(INDEX, waitsOf(joins, INDEX));
      joins.write(
        createFrame(
          [INDEX, createIndex([5, 6], 10n)],
          [DATA, createData([50, 60], 10n)],
        ),
      );
      expect(joins.landed(INDEX, waitsOf(joins, DATA), true).size).toBe(0);
      expect(streamed(store, DATA)).toEqual([50, 60]);
      expect(leadingOf(store, DATA)?.alignment).toBe(alignmentOf(BLOCK, 4));
    });
  });

  describe("write", () => {
    it("should leave the series of a channel outside every join out of the store", () => {
      const { store, joins } = createJoined();
      const other = 3;
      const series = createData([1, 2], 0n, spanOf([1, 2]));
      expect(joins.write(createFrame([other, series]))).toEqual(new Set());
      expect(store.open(other)).toBeNull();
      expect(store.findRaw(other, BLOCK)).toBeUndefined();
    });

    it("should place untimed data samples by their Core alignment", () => {
      const { store } = createJoined();
      expect(streamed(store, DATA)).toEqual([10, 20]);
      expect(store.raw(DATA, BLOCK).coveredEnd).toEqual(at(2).add(1));
    });

    it("should place data written in the same frame ahead of its index", () => {
      const { store, joins } = createJoined();
      joins.write(
        createFrame([DATA, createData([30], 2n)], [INDEX, createIndex([3], 2n)]),
      );
      expect(streamed(store, DATA)).toEqual([10, 20, 30]);
    });

    it("should place data framed apart from its index after later index frames", () => {
      const { store, joins } = createJoined();
      joins.write(createFrame([INDEX, createIndex([3, 4], 2n)]));
      joins.write(createFrame([INDEX, createIndex([5, 6], 4n)]));
      expect(joins.write(createFrame([DATA, createData([30, 40], 2n)]))).toEqual(
        new Set(),
      );
      joins.write(createFrame([DATA, createData([50, 60], 4n)]));
      expect(streamed(store, DATA)).toEqual([10, 20, 30, 40, 50, 60]);
      expect(store.raw(DATA, BLOCK).coveredEnd).toEqual(at(6).add(1));
    });

    it("should take the time range of data samples from their index samples", () => {
      const { store, joins } = createJoined();
      joins.write(createFrame([INDEX, createIndex([3, 4], 2n)]));
      joins.write(createFrame([DATA, createData([30, 40], 2n, spanOf([50, 60]))]));
      expect(store.raw(DATA, BLOCK).coveredEnd).toEqual(at(4).add(1));
    });

    it("should drop data series the stream carried before their index", () => {
      const { store, joins } = createJoins();
      joins.write(createFrame([DATA, createData([10, 20], 0n)]));
      joins.write(
        createFrame([INDEX, createIndex([3, 4], 2n)], [DATA, createData([30, 40], 2n)]),
      );
      expect(joins.landed(INDEX, waitsOf(joins, INDEX, DATA)).size).toBe(0);
      expect(joins.joined(DATA)).toBe(true);
      expect(streamed(store, DATA)).toEqual([30, 40]);
    });

    it("should send the join back to wait on a gap in its index stream", () => {
      const { store, joins } = createJoined();
      const waiting = joins.write(
        createFrame([INDEX, createIndex([4, 5], 5n)], [DATA, createData([40, 50], 5n)]),
      );
      expect(waiting).toEqual(new Set([INDEX]));
      expect(keysOf(joins)).toEqual([INDEX, DATA]);
      expect(store.raw(INDEX, BLOCK).coveredCount).toBe(2);
      expect(joins.catchUpFrom(INDEX, at(1000))).toBe(BLOCK);
    });

    it("should continue the index stream into a new Core domain", () => {
      const { joins } = createJoined();
      const waiting = joins.write(
        createFrame([INDEX, createIndex([3], alignmentOf(1, 0))]),
      );
      expect(waiting.size).toBe(0);
      expect(
        joins.write(createFrame([INDEX, createIndex([4], alignmentOf(2, 5))])),
      ).toEqual(new Set([INDEX]));
    });

    describe("runs", () => {
      it("should keep a new run in one block apart from the run before it", () => {
        const { store, joins } = createJoined();
        joins.write(
          createFrame(
            [INDEX, createIndex([5, 6], NEXT_DOMAIN)],
            [DATA, createData([30, 40], NEXT_DOMAIN)],
          ),
        );
        [INDEX, DATA].forEach((ch) => {
          const series = seriesOf(store, ch, BLOCK);
          expect(series.map((s) => s.length)).toEqual([2, 2]);
          expect(touches(series)).toEqual([false]);
          expect(series[0].timeRange).toEqual(spanOf([1, 2]));
          expect(series[1].timeRange.start).toEqual(at(5));
          expect(series[1].alignment).toBe(alignmentOf(BLOCK, 2));
        });
      });

      it.each([8, 1000])(
        "should keep the series of one run touching across blocks with %d sample buffers",
        (dynamicBufferSize) => {
          const store = new Store({ dynamicBufferSize });
          const joins = new Joins(store);
          joins.join(DATA, INDEX);
          joins.landed(INDEX, waitsOf(joins, INDEX, DATA));
          const seconds = Array.from({ length: 400 }, (_, i) => i / 2);
          for (let i = 0; i < seconds.length; i += 3) {
            const chunk = seconds.slice(i, i + 3);
            const alignment = BigInt(i);
            joins.write(
              createFrame(
                [INDEX, createIndex(chunk, alignment)],
                [DATA, createData(chunk, alignment)],
              ),
            );
          }
          const blocks = [BLOCK, BLOCK + 1, BLOCK + 2, BLOCK + 3];
          [INDEX, DATA].forEach((ch) => {
            const series = seriesOf(store, ch, ...blocks);
            expect(series.length).toBeGreaterThan(blocks.length);
            expect(touches(series)).not.toContain(false);
          });
        },
      );

      it("should continue the samples of a read when a stream joins mid-run", () => {
        const { store, joins } = createJoins();
        fillRead(store, [1, 2]);
        joins.landed(INDEX, waitsOf(joins, INDEX, DATA));
        joins.write(
          createFrame(
            [INDEX, createIndex([3, 4], 2n)],
            [DATA, createData([30, 40], 2n)],
          ),
        );
        [INDEX, DATA].forEach((ch) =>
          expect(touches(seriesOf(store, ch, BLOCK))).toEqual([true]),
        );
      });

      it("should keep a run apart from a read's samples of the run before it", () => {
        const { store, joins } = createJoins();
        fillRead(store, [1, 2]);
        joins.landed(INDEX, waitsOf(joins, INDEX, DATA));
        joins.write(
          createFrame(
            [INDEX, createIndex([3, 4], NEXT_DOMAIN)],
            [DATA, createData([30, 40], NEXT_DOMAIN)],
          ),
        );
        [INDEX, DATA].forEach((ch) =>
          expect(touches(seriesOf(store, ch, BLOCK))).toEqual([false]),
        );
      });
    });

    it("should not wait on a gap in a join with a calculated member", () => {
      const { store, joins } = createJoins();
      store.setChannel(
        new channel.Channel({
          key: DATA,
          name: "calculated",
          dataType: DataType.FLOAT32,
          index: INDEX,
          virtual: true,
          expression: "return a + b",
        }).payload,
      );
      joins.landed(INDEX, waitsOf(joins, INDEX, DATA));
      joins.write(
        createFrame([INDEX, createIndex([1, 2], 0n)], [DATA, createData([10, 20], 0n)]),
      );
      const waiting = joins.write(
        createFrame([INDEX, createIndex([3], 6n)], [DATA, createData([30], 6n)]),
      );
      expect(waiting.size).toBe(0);
      expect(streamed(store, DATA)).toEqual([10, 20, 30]);
    });

    it("should send back only a data member whose alignment has no index samples", () => {
      const { store, joins } = createJoined();
      const waiting = joins.write(createFrame([DATA, createData([90], 50n)]));
      expect(waiting).toEqual(new Set([INDEX]));
      expect(keysOf(joins)).toEqual([DATA]);
      expect(joins.joined(INDEX)).toBe(true);
      expect(store.open(DATA)).toBeNull();
    });

    it("should send back a member when the stream changes its data type", () => {
      const { store, joins } = createJoined();
      const waiting = joins.write(
        createFrame(
          [INDEX, createIndex([3], 2n)],
          [DATA, new Series({ data: new Float64Array([30]), alignment: 2n })],
        ),
      );
      expect(waiting).toEqual(new Set([INDEX]));
      expect(keysOf(joins)).toEqual([DATA]);
      expect(store.dataType(DATA)).toEqual(DataType.FLOAT64);
    });

    it("should send back a member when a read changed its data type", () => {
      const { store, joins } = createJoined();
      store.retainDataType(DATA, DataType.FLOAT64);
      const waiting = joins.write(
        createFrame(
          [INDEX, createIndex([3], 2n)],
          [DATA, new Series({ data: new Float64Array([30]), alignment: 2n })],
        ),
      );
      expect(waiting).toEqual(new Set([INDEX]));
      expect(keysOf(joins)).toEqual([DATA]);
    });
  });

  describe("invalidate", () => {
    it("should send back a member whose open block the change touches", () => {
      const { joins } = createJoined();
      expect(joins.invalidate(DATA, new TimeRange(at(1), at(2)))).toEqual(
        new Set([INDEX]),
      );
      expect(keysOf(joins)).toEqual([DATA]);
    });

    it("should keep a member joined when the change misses its open block", () => {
      const { joins } = createJoined();
      const earlier = tileRange({ level: RAW_LEVEL, index: BLOCK - 1 });
      expect(joins.invalidate(DATA, earlier).size).toBe(0);
      expect(joins.joined(DATA)).toBe(true);
    });

    it("should send back every join on a change to every channel", () => {
      const { joins } = createJoined();
      expect(joins.invalidate()).toEqual(new Set([INDEX]));
      expect(keysOf(joins)).toEqual([INDEX, DATA]);
    });
  });

  describe("flush", () => {
    it("should send back the member and fetch from the block of now", () => {
      const { store, joins } = createJoined();
      expect(joins.flush(DATA)).toEqual(new Set([INDEX]));
      expect(keysOf(joins)).toEqual([DATA]);
      expect(store.open(DATA)).toBeNull();
      expect(joins.catchUpFrom(DATA, at(1000))).toBe(rawIndex(at(1000)));
    });
  });

  describe("bounced", () => {
    it("should be true for a member that waits again before it placed a sample", () => {
      const { joins } = createJoins();
      joins.landed(INDEX, waitsOf(joins, INDEX, DATA));
      joins.flush(DATA);
      expect(joins.bounced(DATA)).toBe(true);
    });

    it("should be false for a member that placed a sample before it waited", () => {
      const { joins } = createJoined();
      joins.flush(DATA);
      expect(joins.bounced(DATA)).toBe(false);
    });
  });

  describe("catchUpFrom", () => {
    it("should start an index member's read at its first held sample", () => {
      const { joins } = createJoins();
      joins.write(createFrame([INDEX, createIndex([1], 0n)]));
      expect(joins.catchUpFrom(INDEX, at(1000))).toBe(BLOCK);
    });

    it("should start a data member's read at its first held time range", () => {
      const { joins } = createJoins();
      joins.write(createFrame([DATA, createData([10], 0n, spanOf([1]))]));
      expect(joins.catchUpFrom(DATA, at(1000))).toBe(BLOCK);
    });

    it("should start an untimed data member's read at the given time", () => {
      const { joins } = createJoins();
      joins.write(createFrame([DATA, createData([10], 0n)]));
      const later = at(1000);
      expect(joins.catchUpFrom(DATA, later)).toBe(joins.catchUpFrom(INDEX, later));
      expect(joins.catchUpFrom(DATA, later)).toBeGreaterThan(BLOCK);
    });
  });

  describe("leave", () => {
    it("should keep a member until each of its joins leaves", () => {
      const { store, joins } = createJoined();
      joins.join(DATA, INDEX);
      joins.leave(DATA, INDEX);
      expect(joins.has(DATA)).toBe(true);
      expect(store.open(DATA)).not.toBeNull();
      joins.leave(DATA, INDEX);
      expect(joins.has(DATA)).toBe(false);
      expect(joins.has(INDEX)).toBe(false);
      expect(store.open(DATA)).toBeNull();
      expect(keysOf(joins)).toEqual([]);
    });
  });
});
