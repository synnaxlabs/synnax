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
import { glTransform } from "@/testutil/telem";

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

        it("should fall back to the wall clock at a buffer split", () => {
          const cache = new Dynamic({ dynamicBufferSize: 2, now: () => WALL });
          const { flushed, allocated } = cache.write(
            new MultiSeries([
              stamped([1, 2, 3], TimeStamp.seconds(10).range(TimeStamp.seconds(13))),
            ]),
          );
          // The split point's timestamp is unknowable from the data series, so the
          // flushed buffer ends at the wall clock and the continuation starts there.
          expect(flushed.series[0].timeRange).toEqual(
            TimeStamp.seconds(10).range(WALL),
          );
          expect(allocated.series[1].timeRange.start).toEqual(WALL);
        });
      });

      it("should correctly allocate a single new buffer when the current one is full", async () => {
        const cache = new Dynamic({ dynamicBufferSize: 2 });
        const ser = f32([1, 2, 3]);
        const { flushed, allocated } = cache.write(new MultiSeries([ser]));
        expect(flushed).toHaveLength(2);
        expect(allocated).toHaveLength(3);
        expect(flushed.series[0]).toBe(allocated.series[0]);
        expect(cache.length).toEqual(1);
      });
      it("should correctly allocate multiple new buffers when the current one is full", () => {
        const cache = new Dynamic({ dynamicBufferSize: 1 });
        const ser = f32([1, 2, 3]);
        const { flushed, allocated } = cache.write(new MultiSeries([ser]));
        expect(flushed).toHaveLength(2);
        expect(allocated).toHaveLength(3);
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
        expect(allocated).toHaveLength(2);
        expect(allocated.timeRange.start.sub(TimeStamp.now()).valueOf()).toBeLessThan(
          TimeSpan.milliseconds(3).valueOf(),
        );
        expect(allocated.timeRange.end.valueOf()).toEqual(TimeStamp.MAX.valueOf());
        expect(flushed).toHaveLength(10);
        expect(flushed.timeRange.span.sub(waitSpan).valueOf()).toBeLessThanOrEqual(
          TimeSpan.milliseconds(20).valueOf(),
        );
        expect(flushed.series[0].data.slice(0, 3)).toEqual(new Float32Array([1, 2, 3]));
        expect(flushed.series[0].data.slice(3, 6)).toEqual(new Float32Array([1, 2, 3]));
        expect(flushed.series[0].data.slice(6, 9)).toEqual(new Float32Array([1, 2, 3]));
        expect(flushed.series[0].data.slice(9)).toEqual(new Float32Array([1]));
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
      it("should keep bigint storage when no transform is provided", () => {
        const cache = new Dynamic({ dynamicBufferSize: 100 });
        const ser = new Series({ data: [42n, 43n], dataType: DataType.INT64 });
        const { allocated } = cache.write(new MultiSeries([ser]));
        expect(allocated.series[0].dataType.equals(DataType.INT64)).toBe(true);
        expect(allocated.series[0].sampleOffset).toBe(0);
        expect(allocated.series[0].at(0)).toBe(42n);
      });
      it("should allocate the buffer with a bigint sampleOffset for bigint data types", () => {
        const nowTs = TimeStamp.seconds(1);
        const cache = new Dynamic({
          dynamicBufferSize: 100,
          transform: glTransform,
          now: () => nowTs,
        });
        const ser = new Series({
          data: [nowTs.valueOf(), nowTs.valueOf() + 1n, nowTs.valueOf() + 2n],
          dataType: DataType.INT64,
        });
        const { allocated } = cache.write(new MultiSeries([ser]));
        expect(allocated.series).toHaveLength(1);
        expect(allocated.series[0].sampleOffset).toBe(ser.data[0]);
        expect(allocated.series[0].dataType.equals(DataType.FLOAT32)).toBe(true);
      });
      it("should allocate the buffer with a numeric sampleOffset for non-bigint data types", () => {
        const cache = new Dynamic({ dynamicBufferSize: 100, transform: glTransform });
        const ser = f32([1, 2, 3]);
        const { allocated } = cache.write(new MultiSeries([ser]));
        expect(allocated.series).toHaveLength(1);
        expect(allocated.series[0].sampleOffset).toBe(0);
      });
      // With a narrowing transform the cache renders into FLOAT32 buffers. FLOAT32
      // only has a 24-bit mantissa, so any integer above ~2^24 quantizes to its
      // local ULP. At wall-clock nanosecond magnitudes (~1.778e18) a single ULP is
      // roughly 10^11 nanoseconds, which is a precision loss of about a hundred
      // seconds per sample. To keep precision usable the cache anchors a per-buffer
      // `sampleOffset` and stores `(value - sampleOffset)` as a small float32.
      // `Series.at(i)` reverses the offset on read.
      //
      // The choice of *which* anchor to use depends on the source type:
      //   - TIMESTAMP: now() is a tight proxy for the values being written, so the
      //     deltas stay small without needing to peek at the data.
      //   - INT64 / UINT64: the value distribution is unconstrained, so we anchor
      //     on the first observed sample. A counter starting at 0 and a counter
      //     near 2^60 both round-trip cleanly.
      //   - Non-bigint: no precision concern, anchor stays at 0.
      describe("bigint channel precision", () => {
        const WALL_CLOCK = TimeStamp.seconds(1778020940);
        const bigCache = (): Dynamic =>
          new Dynamic({
            dynamicBufferSize: 100,
            transform: glTransform,
            now: () => WALL_CLOCK,
          });
        describe("offset anchor selection", () => {
          it("anchors the offset on now() for TIMESTAMP channels", () => {
            const cache = bigCache();
            const firstSample = WALL_CLOCK.valueOf() + 5_000_000_000n;
            const ser = new Series({
              data: [firstSample, firstSample + 1n],
              dataType: DataType.TIMESTAMP,
            });
            const { allocated } = cache.write(new MultiSeries([ser]));
            expect(allocated.series[0].sampleOffset).toBe(WALL_CLOCK.valueOf());
          });
          it("anchors the offset on the first observed sample for INT64 channels", () => {
            const cache = bigCache();
            const ser = new Series({ data: [42n, 43n, 44n], dataType: DataType.INT64 });
            const { allocated } = cache.write(new MultiSeries([ser]));
            expect(allocated.series[0].sampleOffset).toBe(42n);
          });
          it("anchors the offset on the first observed sample for UINT64 channels", () => {
            const cache = bigCache();
            const ser = new Series({
              data: [42n, 43n, 44n],
              dataType: DataType.UINT64,
            });
            const { allocated } = cache.write(new MultiSeries([ser]));
            expect(allocated.series[0].sampleOffset).toBe(42n);
          });
        });
        describe("round-trip correctness", () => {
          it("round-trips small INT64 counter values under a wall-clock now()", () => {
            // Regression for SY-4149: forcing the offset to now() for every bigint
            // channel loses ~10^11 ns of precision per sample for counters near 0.
            const cache = bigCache();
            const ser = new Series({ data: [0n, 1n, 2n], dataType: DataType.INT64 });
            const { allocated } = cache.write(new MultiSeries([ser]));
            const buf = allocated.series[0];
            expect(buf.at(0)).toBe(0n);
            expect(buf.at(1)).toBe(1n);
            expect(buf.at(2)).toBe(2n);
          });
          it("round-trips INT64 counter values above 2^53", () => {
            const first = 1778020940471336960n;
            const cache = bigCache();
            const ser = new Series({
              data: [first, first + 1n, first + 2n],
              dataType: DataType.INT64,
            });
            const { allocated } = cache.write(new MultiSeries([ser]));
            const buf = allocated.series[0];
            expect(buf.at(0)).toBe(first);
            expect(buf.at(1)).toBe(first + 1n);
            expect(buf.at(2)).toBe(first + 2n);
          });
          it("round-trips negative INT64 values", () => {
            const cache = bigCache();
            const ser = new Series({
              data: [-100n, -99n, -98n],
              dataType: DataType.INT64,
            });
            const { allocated } = cache.write(new MultiSeries([ser]));
            const buf = allocated.series[0];
            expect(buf.at(0)).toBe(-100n);
            expect(buf.at(1)).toBe(-99n);
            expect(buf.at(2)).toBe(-98n);
          });
        });
        describe("offset behavior across writes and rotations", () => {
          it("preserves the buffer's offset across multiple writes", () => {
            const cache = bigCache();
            const first = cache.write(
              new MultiSeries([
                new Series({ data: [1000n, 1001n], dataType: DataType.INT64 }),
              ]),
            );
            const offset = first.allocated.series[0].sampleOffset;
            expect(offset).toBe(1000n);
            cache.write(
              new MultiSeries([
                new Series({
                  data: [1002n, 1003n],
                  dataType: DataType.INT64,
                }).reAlign(2n),
              ]),
            );
            const buf = cache.leadingBuffer!;
            expect(buf.sampleOffset).toBe(1000n);
            expect(buf.at(0)).toBe(1000n);
            expect(buf.at(2)).toBe(1002n);
            expect(buf.at(3)).toBe(1003n);
          });
          it("anchors a fresh offset when the buffer rotates due to capacity", () => {
            const cache = new Dynamic({
              dynamicBufferSize: 2,
              transform: glTransform,
              now: () => WALL_CLOCK,
            });
            const ser = new Series({
              data: [500n, 501n, 600n, 601n, 700n],
              dataType: DataType.INT64,
            });
            const { flushed, allocated } = cache.write(new MultiSeries([ser]));
            // Buffer A fills with [500, 501], anchored at 500.
            expect(flushed.series[0].sampleOffset).toBe(500n);
            expect(flushed.series[0].at(0)).toBe(500n);
            expect(flushed.series[0].at(1)).toBe(501n);
            // Buffer B fills with [600, 601], anchored at the first sample that
            // landed in it, not at A's offset.
            expect(flushed.series[1].sampleOffset).toBe(600n);
            expect(flushed.series[1].at(0)).toBe(600n);
            expect(flushed.series[1].at(1)).toBe(601n);
            // Buffer C is the trailing buffer holding [700], anchored at 700.
            expect(allocated.series).toHaveLength(3);
            expect(allocated.series[2].sampleOffset).toBe(700n);
            expect(allocated.series[2].at(0)).toBe(700n);
          });
          it("anchors a fresh offset when alignment mismatch forces rotation", () => {
            const cache = bigCache();
            cache.write(
              new MultiSeries([
                new Series({ data: [10n, 11n], dataType: DataType.INT64 }),
              ]),
            );
            const second = new Series({
              data: [9999n, 10000n],
              dataType: DataType.INT64,
            }).reAlign(50n);
            const { flushed, allocated } = cache.write(new MultiSeries([second]));
            expect(flushed.series[0].sampleOffset).toBe(10n);
            expect(allocated.series[0].sampleOffset).toBe(9999n);
            expect(allocated.series[0].at(0)).toBe(9999n);
            expect(allocated.series[0].at(1)).toBe(10000n);
          });
        });
        describe("known limitations", () => {
          // These tests pin down a constraint of the anchor strategy described
          // above. The cache only feeds the visualization layer, so this affects
          // what is rendered on screen, not the underlying telemetry on disk. Real
          // workloads (roughly monotonic counters, bounded sensors, timestamps
          // seconds apart) do not span enough value inside one rolling buffer to
          // hit it, so the loss is documented rather than fixed.
          it("loses precision on samples far from the buffer's anchor", () => {
            const cache = bigCache();
            const farAway = 1_000_000_000_000_000_000n;
            const ser = new Series({
              data: [0n, 1n, 2n, farAway],
              dataType: DataType.INT64,
            });
            const { allocated } = cache.write(new MultiSeries([ser]));
            const buf = allocated.series[0];
            // Samples near the anchor round-trip exactly.
            expect(buf.at(0)).toBe(0n);
            expect(buf.at(1)).toBe(1n);
            expect(buf.at(2)).toBe(2n);
            // The far sample reads back at the float32 ULP for its magnitude. We
            // intentionally do not assert an exact value (the rounding is platform
            // dependent), only that precision is meaningfully lost.
            const readBack = buf.at(3) as bigint;
            expect(readBack).not.toBe(farAway);
            const drift = readBack > farAway ? readBack - farAway : farAway - readBack;
            expect(drift).toBeGreaterThan(1000n);
          });
        });
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
});
