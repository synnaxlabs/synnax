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
  type TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Dynamic } from "@/framer/cache/dynamic";

const f32 = (data: number[]): Series =>
  new Series({ data: new Float32Array(data), dataType: DataType.FLOAT32 });

describe("DynamicCache", () => {
  describe("fixed density channel", () => {
    describe("write", () => {
      it("Should correctly allocate a buffer", () => {
        const cache = new Dynamic({ dynamicBufferSize: 100 });
        const ser = f32([1, 2, 3]);
        const { flushed, allocated } = cache.write(new MultiSeries([ser]));
        expect(flushed).toHaveLength(0);
        expect(allocated).toHaveLength(3);
        expect(allocated.timeRange.start.sub(TimeStamp.now()).valueOf()).toBeLessThan(
          TimeSpan.milliseconds(1).valueOf(),
        );
        expect(allocated.timeRange.end.valueOf()).toEqual(TimeStamp.MAX.valueOf());
        expect(cache.length).toEqual(ser.length);
      });
      it("Should not allocate a new buffer when the current buffer has sufficient space", () => {
        const cache = new Dynamic({ dynamicBufferSize: 100 });
        const ser = f32([1, 2, 3]);
        cache.write(new MultiSeries([ser]));
        const { flushed, allocated } = cache.write(new MultiSeries([ser.reAlign(3n)]));
        expect(flushed).toHaveLength(0);
        expect(allocated).toHaveLength(0);
        expect(cache.length).toEqual(ser.length * 2);
      });
      it("should coalesce an overlapping frame instead of allocating a new fragment", () => {
        const cache = new Dynamic({ dynamicBufferSize: 100 });
        cache.write(new MultiSeries([f32([1, 2, 3])])); // occupies [0, 3)
        // Incoming series steps back by 2 (alignment 1, covers [1, 4)); the first two
        // samples duplicate what the buffer already holds.
        const overlapping = f32([2, 3, 4]).reAlign(1n);
        const { flushed, allocated } = cache.write(new MultiSeries([overlapping]));
        expect(flushed).toHaveLength(0);
        expect(allocated).toHaveLength(0);
        expect(cache.length).toEqual(4); // [0, 4): 3 held + 1 genuinely new sample
      });
      it("should drop a single re-sent sample instead of duplicating it", () => {
        const cache = new Dynamic({ dynamicBufferSize: 100 });
        cache.write(new MultiSeries([f32([1, 2, 3])])); // occupies [0, 3)
        // Steps back by exactly one: the first sample duplicates the buffer's last.
        const overlapping = f32([3, 4]).reAlign(2n);
        const { flushed, allocated } = cache.write(new MultiSeries([overlapping]));
        expect(flushed).toHaveLength(0);
        expect(allocated).toHaveLength(0);
        expect(cache.length).toEqual(4); // [0, 4): the duplicate is trimmed
        expect(cache.leadingBuffer?.data.slice(0, 4)).toEqual(
          new Float32Array([1, 2, 3, 4]),
        );
      });
      it("should reset the buffer across a single-sample gap", () => {
        const cache = new Dynamic({ dynamicBufferSize: 100 });
        cache.write(new MultiSeries([f32([1, 2, 3])])); // occupies [0, 3)
        // Alignment 4 leaves sample 3 missing. Appending would write the series as
        // if contiguous, shifting the buffer's alignment off the Core's counter.
        const gapped = f32([5, 6]).reAlign(4n);
        const { flushed, allocated } = cache.write(new MultiSeries([gapped]));
        expect(flushed).toHaveLength(3);
        expect(allocated).toHaveLength(2);
        expect(cache.leadingBuffer?.alignment).toEqual(4n);
        expect(cache.length).toEqual(2);
      });
      it("should drop a fully-duplicate overlapping frame", () => {
        const cache = new Dynamic({ dynamicBufferSize: 100 });
        cache.write(new MultiSeries([f32([1, 2, 3, 4, 5])])); // occupies [0, 5)
        const dup = f32([2, 3]).reAlign(1n); // [1, 3), entirely within the buffer
        const { flushed, allocated } = cache.write(new MultiSeries([dup]));
        expect(flushed).toHaveLength(0);
        expect(allocated).toHaveLength(0);
        expect(cache.length).toEqual(5); // unchanged
      });
      it("should reset the buffer when alignment regresses behind its start", () => {
        const cache = new Dynamic({ dynamicBufferSize: 100 });
        cache.write(new MultiSeries([f32([1, 2, 3]).reAlign(1000n)]));
        // The alignment counter rewound (e.g. the Core restarted). The write must
        // land in a fresh buffer instead of being dropped as a duplicate.
        const rewound = f32([9]).reAlign(5n);
        const { flushed, allocated } = cache.write(new MultiSeries([rewound]));
        expect(flushed).toHaveLength(3);
        expect(allocated).toHaveLength(1);
        expect(cache.length).toEqual(1);
        expect(cache.leadingBuffer?.alignment).toEqual(5n);
        expect(cache.leadingBuffer?.at(0)).toEqual(9);
        // Subsequent writes continue from the new epoch.
        const next = cache.write(new MultiSeries([f32([10]).reAlign(6n)]));
        expect(next.flushed).toHaveLength(0);
        expect(cache.length).toEqual(2);
      });
      it("should reset the buffer when the incoming data type changes", () => {
        const cache = new Dynamic({ dynamicBufferSize: 100 });
        cache.write(new MultiSeries([f32([1, 2, 3])]));
        const f64 = new Series({
          data: new Float64Array([4, 5]),
          dataType: DataType.FLOAT64,
        }).reAlign(3n);
        const { flushed, allocated } = cache.write(new MultiSeries([f64]));
        expect(flushed).toHaveLength(3);
        expect(allocated).toHaveLength(2);
        expect(cache.leadingBuffer?.dataType.equals(DataType.FLOAT64)).toBe(true);
        expect(cache.leadingBuffer?.at(0)).toEqual(4);
      });
      describe("time range stamping", () => {
        const WALL = TimeStamp.seconds(100);
        const stamped = (data: number[], tr: TimeRange, alignment = 0n): Series =>
          new Series({
            data: new Float32Array(data),
            dataType: DataType.FLOAT32,
            timeRange: tr,
            alignment,
          });

        it("should stamp the buffer with the data's time range", () => {
          const cache = new Dynamic({ dynamicBufferSize: 100, now: () => WALL });
          const { allocated } = cache.write(
            new MultiSeries([
              stamped([1, 2, 3], TimeStamp.seconds(10).range(TimeStamp.seconds(13))),
            ]),
          );
          expect(allocated.series[0].timeRange.start).toEqual(TimeStamp.seconds(10));
          cache.write(
            new MultiSeries([
              stamped(
                [4, 5, 6],
                TimeStamp.seconds(13).range(TimeStamp.seconds(16)),
                3n,
              ),
            ]),
          );
          // A large alignment gap flushes the buffer and allocates a new one.
          const { flushed } = cache.write(
            new MultiSeries([
              stamped([7], TimeStamp.seconds(30).range(TimeStamp.seconds(31)), 100n),
            ]),
          );
          expect(flushed.series[0].timeRange).toEqual(
            TimeStamp.seconds(10).range(TimeStamp.seconds(16)),
          );
          expect(cache.leadingBuffer?.timeRange.start).toEqual(TimeStamp.seconds(30));
        });

        it("should fall back to the wall clock for unstamped series", () => {
          const cache = new Dynamic({ dynamicBufferSize: 100, now: () => WALL });
          const { allocated } = cache.write(new MultiSeries([f32([1, 2, 3])]));
          expect(allocated.series[0].timeRange.start).toEqual(WALL);
          const { flushed } = cache.write(new MultiSeries([f32([7]).reAlign(100n)]));
          expect(flushed.series[0].timeRange.end).toEqual(WALL);
        });

        it("should fall back to the wall clock when an unstamped series follows a stamped one", () => {
          const cache = new Dynamic({ dynamicBufferSize: 100, now: () => WALL });
          cache.write(
            new MultiSeries([
              stamped([1, 2, 3], TimeStamp.seconds(10).range(TimeStamp.seconds(13))),
            ]),
          );
          cache.write(new MultiSeries([f32([4, 5]).reAlign(3n)]));
          const { flushed } = cache.write(new MultiSeries([f32([7]).reAlign(100n)]));
          expect(flushed.series[0].timeRange).toEqual(
            TimeStamp.seconds(10).range(WALL),
          );
        });

        it("should grow a buffer rather than split a series across two", () => {
          const cache = new Dynamic({ dynamicBufferSize: 2, now: () => WALL });
          const { flushed, allocated } = cache.write(
            new MultiSeries([
              stamped([1, 2, 3], TimeStamp.seconds(10).range(TimeStamp.seconds(13))),
            ]),
          );
          // Splitting would put the boundary at a timestamp the data does not carry.
          expect(flushed).toHaveLength(0);
          expect(allocated).toHaveLength(3);
          expect(cache.dataTimeRange).toEqual(
            TimeStamp.seconds(10).range(TimeStamp.seconds(13)),
          );
        });

        it("should stamp a rotation with the last written series' end", () => {
          const cache = new Dynamic({ dynamicBufferSize: 2, now: () => WALL });
          cache.write(
            new MultiSeries([
              stamped([1, 2], TimeStamp.seconds(10).range(TimeStamp.seconds(12))),
            ]),
          );
          // The next series does not fit, so the full buffer is flushed at the end of
          // the series it holds, and the new one starts there.
          const { flushed } = cache.write(
            new MultiSeries([
              stamped([3, 4], TimeStamp.seconds(12).range(TimeStamp.seconds(14)), 2n),
            ]),
          );
          expect(flushed.series[0].timeRange).toEqual(
            TimeStamp.seconds(10).range(TimeStamp.seconds(12)),
          );
          expect(cache.leadingBuffer?.timeRange.start).toEqual(TimeStamp.seconds(12));
        });

        describe("runs", () => {
          // Returns the capacity of each buffer the writes of the lengths allocate.
          const capacitiesOf = (cache: Dynamic, lengths: number[]): number[] => {
            const capacities: number[] = [];
            let alignment = 0n;
            lengths.forEach((length) => {
              const series = f32(new Array<number>(length).fill(1)).reAlign(alignment);
              const { allocated } = cache.write(new MultiSeries([series]));
              capacities.push(...allocated.series.map((s) => s.capacity));
              alignment += BigInt(length);
            });
            return capacities;
          };

          it("should double a run's buffers up to the configured size", () => {
            const cache = new Dynamic({ dynamicBufferSize: 400 });
            cache.startRun();
            expect(capacitiesOf(cache, new Array<number>(8).fill(100))).toEqual([
              100, 200, 400, 400,
            ]);
          });

          it("should hold a first series that outgrows the small buffer whole", () => {
            const cache = new Dynamic({ dynamicBufferSize: 1000 });
            cache.startRun();
            expect(capacitiesOf(cache, [150, 150, 300])).toEqual([150, 300, 600]);
          });

          it("should never exceed a configured size below the small buffer", () => {
            const cache = new Dynamic({ dynamicBufferSize: 8 });
            cache.startRun();
            expect(capacitiesOf(cache, [8])).toEqual([8]);
          });

          it("should keep growing a run's buffers across a flush", () => {
            const cache = new Dynamic({ dynamicBufferSize: 400 });
            cache.startRun();
            capacitiesOf(cache, [100]);
            cache.flush();
            expect(capacitiesOf(cache, [100])).toEqual([200]);
          });

          it("should use the configured size when no run started", () => {
            const cache = new Dynamic({ dynamicBufferSize: 400 });
            expect(capacitiesOf(cache, [100])).toEqual([400]);
          });
        });

        it("should keep a rotation touching the buffer it follows", () => {
          const cache = new Dynamic({ dynamicBufferSize: 2, now: () => WALL });
          cache.write(
            new MultiSeries([
              stamped([1, 2], TimeStamp.seconds(10).range(TimeStamp.seconds(12))),
            ]),
          );
          const { flushed } = cache.write(
            new MultiSeries([
              stamped([3, 4], TimeStamp.seconds(13).range(TimeStamp.seconds(15)), 2n),
            ]),
          );
          expect(cache.leadingBuffer?.timeRange.start).toEqual(
            flushed.series[0].timeRange.end,
          );
        });

        it("should start a rotation after a trim where the flushed buffer ends", () => {
          const cache = new Dynamic({ dynamicBufferSize: 2, now: () => WALL });
          cache.write(
            new MultiSeries([
              stamped([1, 2], TimeStamp.seconds(10).range(TimeStamp.seconds(12))),
            ]),
          );
          // The re-sent frame starts at second 11 but its first new sample is at
          // second 12. Trimming keeps the frame's own start, so the buffer would
          // claim a second of time it holds nothing for.
          const { flushed } = cache.write(
            new MultiSeries([
              stamped(
                [2, 3, 4],
                TimeStamp.seconds(11).range(TimeStamp.seconds(14)),
                1n,
              ),
            ]),
          );
          expect(flushed.series[0].timeRange).toEqual(
            TimeStamp.seconds(10).range(TimeStamp.seconds(12)),
          );
          expect(cache.leadingBuffer?.timeRange.start).toEqual(TimeStamp.seconds(12));
          expect(
            cache.dataTimeRange?.overlapsWith(
              TimeStamp.seconds(11).range(TimeStamp.milliseconds(11500)),
            ),
          ).toEqual(false);
        });
      });

      it("should correctly allocate a single new buffer when the current one is full", async () => {
        const cache = new Dynamic({ dynamicBufferSize: 2 });
        const first = cache.write(new MultiSeries([f32([1, 2])]));
        expect(first.flushed).toHaveLength(0);
        const { flushed, allocated } = cache.write(
          new MultiSeries([f32([3]).reAlign(2n)]),
        );
        expect(flushed).toHaveLength(2);
        expect(allocated).toHaveLength(1);
        expect(flushed.series[0]).toBe(first.allocated.series[0]);
        expect(cache.length).toEqual(1);
      });
      it("should correctly allocate multiple new buffers when the current one is full", () => {
        const cache = new Dynamic({ dynamicBufferSize: 1 });
        cache.write(new MultiSeries([f32([1])]));
        const second = cache.write(new MultiSeries([f32([2]).reAlign(1n)]));
        const third = cache.write(new MultiSeries([f32([3]).reAlign(2n)]));
        expect(second.flushed).toHaveLength(1);
        expect(third.flushed).toHaveLength(1);
        expect(cache.length).toEqual(1);
      });
      it("it should correctly set multiple writes", async () => {
        const cache = new Dynamic({ dynamicBufferSize: 10 });
        const ser = f32([1, 2, 3]);
        const res1 = cache.write(new MultiSeries([ser]));
        expect(res1.allocated).toHaveLength(3);
        expect(res1.flushed).toHaveLength(0);
        expect(
          res1.allocated.timeRange.start.sub(TimeStamp.now()).valueOf(),
        ).toBeLessThan(TimeSpan.milliseconds(1).valueOf());
        expect(res1.allocated.timeRange.end.valueOf()).toEqual(TimeStamp.MAX.valueOf());
        const res2 = cache.write(new MultiSeries([ser.reAlign(3n)]));
        expect(res2.allocated).toHaveLength(0);
        expect(res2.flushed).toHaveLength(0);
        const res3 = cache.write(new MultiSeries([ser.reAlign(6n)]));
        expect(res3.allocated).toHaveLength(0);
        expect(res3.flushed).toHaveLength(0);
        const waitSpan = TimeSpan.milliseconds(10);
        await new Promise((resolve) => setTimeout(resolve, waitSpan.milliseconds));
        const { flushed, allocated } = cache.write(new MultiSeries([ser.reAlign(9n)]));
        expect(allocated).toHaveLength(3);
        expect(allocated.timeRange.start.sub(TimeStamp.now()).valueOf()).toBeLessThan(
          TimeSpan.milliseconds(3).valueOf(),
        );
        expect(allocated.timeRange.end.valueOf()).toEqual(TimeStamp.MAX.valueOf());
        // The fourth series does not fit in the space left, so the buffer flushes with
        // the nine samples it holds instead of taking part of it.
        expect(flushed).toHaveLength(9);
        // Slack for a loaded runner: the buffer's span tracks wall time, so the bound
        // only has to rule out a span untethered from it.
        expect(flushed.timeRange.span.sub(waitSpan).valueOf()).toBeLessThanOrEqual(
          TimeSpan.milliseconds(500).valueOf(),
        );
        expect(flushed.series[0].data.slice(0, 3)).toEqual(new Float32Array([1, 2, 3]));
        expect(flushed.series[0].data.slice(3, 6)).toEqual(new Float32Array([1, 2, 3]));
        expect(flushed.series[0].data.slice(6, 9)).toEqual(new Float32Array([1, 2, 3]));
      });
      it("should allocate a new buffer if the two series are out of alignment", () => {
        const cache = new Dynamic({ dynamicBufferSize: 10 });
        const s1 = f32([1, 2, 3]);
        const { flushed, allocated } = cache.write(new MultiSeries([s1]));
        expect(flushed).toHaveLength(0);
        expect(allocated).toHaveLength(3);
        const s2 = s1.reAlign(5n);
        const { flushed: f2, allocated: a2 } = cache.write(new MultiSeries([s2]));
        expect(f2).toHaveLength(3);
        expect(a2).toHaveLength(3);
      });
      it("in the same write, it should allocate a new buffer if the two series are out of alignment", () => {
        const cache = new Dynamic({ dynamicBufferSize: 10 });
        const s1 = f32([1, 2, 3]);
        const s2 = s1.reAlign(5n);
        const { flushed, allocated } = cache.write(new MultiSeries([s1, s2]));
        expect(flushed).toHaveLength(3);
        expect(allocated.timeRange.start.sub(TimeStamp.now()).valueOf()).toBeLessThan(
          TimeSpan.milliseconds(10).valueOf(),
        );
        expect(allocated.timeRange.end.valueOf()).toEqual(TimeStamp.MAX.valueOf());
        expect(flushed.series[0]).toBe(allocated.series[0]);
        expect(allocated).toHaveLength(6);
      });
      it("should fit a variable-length series that outgrows the byte estimate", () => {
        // Capacity for variable types is estimated in bytes per sample, so a series of
        // long strings needs more room than the sample count suggests.
        const cache = new Dynamic({ dynamicBufferSize: 1 });
        const ser = new Series({
          data: ["a".repeat(200), "b".repeat(200)],
          dataType: DataType.STRING,
        });
        const { flushed, allocated } = cache.write(new MultiSeries([ser]));
        expect(flushed).toHaveLength(0);
        expect(allocated).toHaveLength(2);
        expect(Array.from(cache.leadingBuffer as Series)).toEqual([
          "a".repeat(200),
          "b".repeat(200),
        ]);
      });
      it("should derive the buffer's data type from the first written series", () => {
        const cache = new Dynamic({ dynamicBufferSize: 100 });
        const ser = new Series({
          data: new Float64Array([1, 2, 3]),
          dataType: DataType.FLOAT64,
        });
        const { allocated } = cache.write(new MultiSeries([ser]));
        expect(allocated.series[0].dataType.equals(DataType.FLOAT64)).toBe(true);
        expect(allocated.series[0].sampleOffset).toBe(0);
      });
      it("should keep bigint storage exact", () => {
        const cache = new Dynamic({ dynamicBufferSize: 100 });
        const ser = new Series({ data: [42n, 43n], dataType: DataType.INT64 });
        const { allocated } = cache.write(new MultiSeries([ser]));
        expect(allocated.series[0].dataType.equals(DataType.INT64)).toBe(true);
        expect(allocated.series[0].sampleOffset).toBe(0);
        expect(allocated.series[0].at(0)).toBe(42n);
      });
      it("should allocate a buffer properly using a TimeSpan", () => {
        let nowF = () => TimeStamp.seconds(1);
        const now = () => nowF();
        const cache = new Dynamic({ dynamicBufferSize: TimeSpan.minutes(5), now });
        const ser = f32([1, 2, 3]);
        const res1 = cache.write(new MultiSeries([ser]));
        expect(res1.allocated).toHaveLength(3);
        expect(res1.flushed).toHaveLength(0);
        nowF = () => TimeStamp.seconds(2);
        const res2 = cache.write(new MultiSeries([ser.reAlign(3n)]));
        expect(res2.allocated).toHaveLength(0);
        expect(res2.flushed).toHaveLength(0);

        nowF = () => TimeStamp.seconds(3);
        const res3 = cache.write(new MultiSeries([ser.reAlign(6n)]));
        expect(res3.allocated).toHaveLength(0);
        expect(res3.flushed).toHaveLength(0);
        expect(cache.length).toBe(9);
      });
    });
  });

  describe("dataTimeRange", () => {
    const WALL = TimeStamp.seconds(100);
    const stamped = (data: number[], tr: TimeRange, alignment = 0n): Series =>
      new Series({
        data: new Float32Array(data),
        dataType: DataType.FLOAT32,
        timeRange: tr,
        alignment,
      });

    it("should be null when there is no buffer", () => {
      const cache = new Dynamic({ dynamicBufferSize: 100 });
      expect(cache.dataTimeRange).toBeNull();
    });

    it("should end at the last stamped write, not the buffer's MAX end", () => {
      const cache = new Dynamic({ dynamicBufferSize: 100, now: () => WALL });
      cache.write(
        new MultiSeries([
          stamped([1, 2, 3], TimeStamp.seconds(10).range(TimeStamp.seconds(13))),
        ]),
      );
      cache.write(
        new MultiSeries([
          stamped([4, 5], TimeStamp.seconds(13).range(TimeStamp.seconds(15)), 3n),
        ]),
      );
      expect(cache.leadingBuffer?.timeRange.end).toEqual(TimeStamp.MAX);
      expect(cache.dataTimeRange).toEqual(
        TimeStamp.seconds(10).range(TimeStamp.seconds(15)),
      );
    });

    it("should fall back to the wall clock for unstamped data", () => {
      const cache = new Dynamic({ dynamicBufferSize: 100, now: () => WALL });
      cache.write(new MultiSeries([f32([1, 2, 3])]));
      expect(cache.dataTimeRange).toEqual(WALL.range(WALL));
    });

    // A rotation used to stamp the new buffer with the wall clock while its samples
    // stayed far in the past, which read back as a range spanning both.
    it("should keep a rotated buffer's range inside its own data", () => {
      const cache = new Dynamic({ dynamicBufferSize: 2, now: () => WALL });
      cache.write(
        new MultiSeries([
          stamped([1, 2], TimeStamp.seconds(10).range(TimeStamp.seconds(12))),
        ]),
      );
      cache.write(
        new MultiSeries([
          stamped([3, 4], TimeStamp.seconds(12).range(TimeStamp.seconds(14)), 2n),
        ]),
      );
      const range = cache.dataTimeRange;
      expect(range?.isValid).toBe(true);
      expect(range).toEqual(TimeStamp.seconds(12).range(TimeStamp.seconds(14)));
      // WALL is 100s, so a read of the present must not reach the rotated buffer.
      expect(
        range?.overlapsWith(TimeStamp.seconds(90).range(TimeStamp.seconds(110))),
      ).toBe(false);
    });

    it("should be null again after a flush", () => {
      const cache = new Dynamic({ dynamicBufferSize: 100 });
      cache.write(
        new MultiSeries([
          stamped([1, 2, 3], TimeStamp.seconds(10).range(TimeStamp.seconds(13))),
        ]),
      );
      cache.flush();
      expect(cache.dataTimeRange).toBeNull();
    });
  });

  describe("compaction", () => {
    // 100k samples of float32 is 400KB, so a buffer holding a handful of them wastes
    // far more than the floor and sits well under the fill fraction.
    const ROOMY = 100_000;

    it("should right-size a buffer that rotates nearly empty", () => {
      const cache = new Dynamic({ dynamicBufferSize: ROOMY });
      const first = cache.write(new MultiSeries([f32([1, 2, 3])]));
      const original = first.allocated.series[0];
      // A series of its own size does not fit beside it, forcing a rotation.
      const { flushed } = cache.write(
        new MultiSeries([f32(Array.from({ length: ROOMY }, (_, i) => i)).reAlign(3n)]),
      );
      const compacted = flushed.series[0];
      expect(compacted).not.toBe(original);
      expect(Array.from(compacted)).toEqual([1, 2, 3]);
      expect(compacted.alignment).toEqual(original.alignment);
      expect(compacted.timeRange).toEqual(original.timeRange);
      expect(compacted.key).toEqual(original.key);
      expect(compacted.byteCapacity.valueOf()).toEqual(compacted.byteLength.valueOf());
    });

    it("should leave a buffer a consumer holds alone", () => {
      const cache = new Dynamic({ dynamicBufferSize: ROOMY });
      const first = cache.write(new MultiSeries([f32([1, 2, 3])]));
      const original = first.allocated.series[0];
      // Copying would strand the holder on an object the cache no longer serves.
      original.acquire();
      const { flushed } = cache.write(
        new MultiSeries([f32(Array.from({ length: ROOMY }, (_, i) => i)).reAlign(3n)]),
      );
      expect(flushed.series[0]).toBe(original);
      original.release();
    });

    it("should leave every buffer alone when compaction is disabled", () => {
      const cache = new Dynamic({ dynamicBufferSize: ROOMY, compactionDisabled: true });
      const first = cache.write(new MultiSeries([f32([1, 2, 3])]));
      const original = first.allocated.series[0];
      const { flushed } = cache.write(
        new MultiSeries([f32(Array.from({ length: ROOMY }, (_, i) => i)).reAlign(3n)]),
      );
      expect(flushed.series[0]).toBe(original);
    });

    it("should leave a nearly full buffer alone", () => {
      const cache = new Dynamic({ dynamicBufferSize: ROOMY });
      const fill = Array.from({ length: ROOMY - 1 }, (_, i) => i);
      const first = cache.write(new MultiSeries([f32(fill)]));
      const original = first.allocated.series[0];
      const { flushed } = cache.write(
        new MultiSeries([f32([1, 2]).reAlign(BigInt(fill.length))]),
      );
      expect(flushed.series[0]).toBe(original);
    });

    it("should leave a buffer whose waste is under the floor alone", () => {
      // 100 float32 samples is 400 bytes of capacity, so nothing is worth copying.
      const cache = new Dynamic({ dynamicBufferSize: 100 });
      const first = cache.write(new MultiSeries([f32([1, 2, 3])]));
      const original = first.allocated.series[0];
      const { flushed } = cache.write(
        new MultiSeries([f32(Array.from({ length: 100 }, (_, i) => i)).reAlign(3n)]),
      );
      expect(flushed.series[0]).toBe(original);
    });

    it("should leave a buffer that rotates inside its own write alone", () => {
      const cache = new Dynamic({ dynamicBufferSize: ROOMY });
      // Both series land in one write, so the buffer the first allocates is flushed
      // before any subscriber has seen it. The response has to name one object, or
      // a subscriber reading back what it streamed would see two.
      const { flushed, allocated } = cache.write(
        new MultiSeries([
          f32([1, 2, 3]),
          f32(Array.from({ length: ROOMY }, (_, i) => i)).reAlign(3n),
        ]),
      );
      expect(flushed.series[0]).toBe(allocated.series[0]);
    });

    it("should right-size a variable-length buffer", () => {
      const cache = new Dynamic({ dynamicBufferSize: 10_000 });
      const first = cache.write(
        new MultiSeries([new Series({ data: ["one"], dataType: DataType.STRING })]),
      );
      const original = first.allocated.series[0];
      const { flushed } = cache.write(
        new MultiSeries([
          new Series({
            data: Array.from({ length: 10_000 }, () => "x".repeat(40)),
            dataType: DataType.STRING,
          }).reAlign(1n),
        ]),
      );
      const compacted = flushed.series[0];
      expect(compacted).not.toBe(original);
      expect(Array.from(compacted)).toEqual(["one"]);
      expect(compacted.byteCapacity.valueOf()).toEqual(compacted.byteLength.valueOf());
    });
  });
});
