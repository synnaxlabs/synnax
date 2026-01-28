// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, MultiSeries, Series, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Dynamic } from "@/telem/client/cache/dynamic";

describe("DynamicCache", () => {
  describe("fixed density channel", () => {
    describe("write", () => {
      it("Should correctly allocate a buffer", () => {
        const cache = new Dynamic({
          dynamicBufferSize: 100,
          dataType: DataType.FLOAT32,
        });
        const ser = new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
        });
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
        const cache = new Dynamic({
          dynamicBufferSize: 100,
          dataType: DataType.FLOAT32,
        });
        const ser = new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
        });
        cache.write(new MultiSeries([ser]));
        const { flushed, allocated } = cache.write(new MultiSeries([ser.reAlign(3n)]));
        expect(flushed).toHaveLength(0);
        expect(allocated).toHaveLength(0);
        expect(cache.length).toEqual(ser.length * 2);
      });
      it("should correctly allocate a single new buffer when the current one is full", async () => {
        const cache = new Dynamic({ dynamicBufferSize: 2, dataType: DataType.FLOAT32 });
        const ser = new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
        });
        const { flushed, allocated } = cache.write(new MultiSeries([ser]));
        expect(flushed).toHaveLength(2);
        expect(allocated).toHaveLength(3);
        expect(flushed.series[0]).toBe(allocated.series[0]);
        expect(cache.length).toEqual(1);
      });
      it("should correctly allocate multiple new buffers when the current one is full", () => {
        const cache = new Dynamic({ dynamicBufferSize: 1, dataType: DataType.FLOAT32 });
        const ser = new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
        });
        const { flushed, allocated } = cache.write(new MultiSeries([ser]));
        expect(flushed).toHaveLength(2);
        expect(allocated).toHaveLength(3);
        expect(cache.length).toEqual(1);
      });
      it("it should correctly set multiple writes", async () => {
        const cache = new Dynamic({
          dynamicBufferSize: 10,
          dataType: DataType.FLOAT32,
        });
        const ser = new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
        });
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
        const cache = new Dynamic({
          dynamicBufferSize: 10,
          dataType: DataType.FLOAT32,
        });
        const s1 = new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
        });
        const { flushed, allocated } = cache.write(new MultiSeries([s1]));
        expect(flushed).toHaveLength(0);
        expect(allocated).toHaveLength(3);
        const s2 = s1.reAlign(5n);
        const { flushed: f2, allocated: a2 } = cache.write(new MultiSeries([s2]));
        expect(f2).toHaveLength(3);
        expect(a2).toHaveLength(3);
      });
      it("in the same write, it should allocate a new buffer if the two series are out of alignment", () => {
        const cache = new Dynamic({
          dynamicBufferSize: 10,
          dataType: DataType.FLOAT32,
        });
        const s1 = new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
        });
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
      it("should allocate a buffer properly using a TimeSpan", () => {
        let nowF = () => TimeStamp.seconds(1);
        const now = () => nowF();
        const cache = new Dynamic({
          dynamicBufferSize: TimeSpan.minutes(5),
          dataType: DataType.FLOAT32,
          now,
        });
        const ser = new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
        });
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
