// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Dynamic } from "@/telem/client/cache/dynamic";

describe("DynamicCache", () => {
  describe("write", () => {
    it("Should correctly allocate a buffer", () => {
      const cache = new Dynamic({ dynamicBufferSize: 100, dataType: DataType.FLOAT32 });
      const arr = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const { flushed, allocated } = cache.write([arr]);
      expect(flushed).toHaveLength(0);
      expect(allocated).toHaveLength(1);
      expect(allocated[0].timeRange.start.sub(TimeStamp.now()).valueOf()).toBeLessThan(
        TimeSpan.milliseconds(1).valueOf(),
      );
      expect(allocated[0].timeRange.end.valueOf()).toEqual(TimeStamp.MAX.valueOf());
      expect(cache.length).toEqual(arr.length);
    });
    it("Should not allocate a new buffer when the current buffer has sufficient space", () => {
      const cache = new Dynamic({ dynamicBufferSize: 100, dataType: DataType.FLOAT32 });
      const arr = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      cache.write([arr]);
      const { flushed, allocated } = cache.write([arr.reAlign(3n)]);
      expect(flushed).toHaveLength(0);
      expect(allocated).toHaveLength(0);
      expect(cache.length).toEqual(arr.length * 2);
    });
    it("should correctly allocate a single new buffer when the current one is full", async () => {
      const cache = new Dynamic({ dynamicBufferSize: 2, dataType: DataType.FLOAT32 });
      const arr = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const { flushed, allocated } = cache.write([arr]);
      expect(flushed).toHaveLength(1);
      expect(allocated).toHaveLength(2);
      expect(flushed[0]).toBe(allocated[0]);
      expect(cache.length).toEqual(1);
    });
    it("should correctly allocate multiple new buffers when the current one is full", () => {
      const cache = new Dynamic({ dynamicBufferSize: 1, dataType: DataType.FLOAT32 });
      const arr = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const { flushed, allocated } = cache.write([arr]);
      expect(flushed).toHaveLength(2);
      expect(allocated).toHaveLength(3);
      expect(cache.length).toEqual(1);
    });
    it("it should correctly set multiple writes", async () => {
      const cache = new Dynamic({ dynamicBufferSize: 10, dataType: DataType.FLOAT32 });
      const arr = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const res1 = cache.write([arr]);
      expect(res1.allocated).toHaveLength(1);
      expect(res1.flushed).toHaveLength(0);
      expect(
        res1.allocated[0].timeRange.start.sub(TimeStamp.now()).valueOf(),
      ).toBeLessThan(TimeSpan.milliseconds(1).valueOf());
      expect(res1.allocated[0].timeRange.end.valueOf()).toEqual(
        TimeStamp.MAX.valueOf(),
      );
      const res2 = cache.write([arr.reAlign(3n)]);
      expect(res2.allocated).toHaveLength(0);
      expect(res2.flushed).toHaveLength(0);
      const res3 = cache.write([arr.reAlign(6n)]);
      expect(res3.allocated).toHaveLength(0);
      expect(res3.flushed).toHaveLength(0);
      const waitSpan = TimeSpan.milliseconds(10);
      await new Promise((resolve) => setTimeout(resolve, waitSpan.milliseconds));
      const { flushed, allocated } = cache.write([arr.reAlign(9n)]);
      expect(allocated).toHaveLength(1);
      expect(allocated[0].timeRange.start.sub(TimeStamp.now()).valueOf()).toBeLessThan(
        TimeSpan.milliseconds(3).valueOf(),
      );
      expect(allocated[0].timeRange.end.valueOf()).toEqual(TimeStamp.MAX.valueOf());
      expect(flushed).toHaveLength(1);
      expect(flushed[0].timeRange.span.sub(waitSpan).valueOf()).toBeLessThanOrEqual(
        TimeSpan.milliseconds(5).valueOf(),
      );
      expect(flushed[0].data.slice(0, 3)).toEqual(new Float32Array([1, 2, 3]));
      expect(flushed[0].data.slice(3, 6)).toEqual(new Float32Array([1, 2, 3]));
      expect(flushed[0].data.slice(6, 9)).toEqual(new Float32Array([1, 2, 3]));
      expect(flushed[0].data.slice(9)).toEqual(new Float32Array([1]));
    });
    it("should allocate a new buffer if the two series are out of alignment", () => {
      const cache = new Dynamic({ dynamicBufferSize: 10, dataType: DataType.FLOAT32 });
      const s1 = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const { flushed, allocated } = cache.write([s1]);
      expect(flushed).toHaveLength(0);
      expect(allocated).toHaveLength(1);
      const s2 = s1.reAlign(5n);
      const { flushed: f2, allocated: a2 } = cache.write([s2]);
      expect(f2).toHaveLength(1);
      expect(a2).toHaveLength(1);
    });
    it("in the same write, it should allocate a new buffer if the two series are out of alignment", () => {
      const cache = new Dynamic({ dynamicBufferSize: 10, dataType: DataType.FLOAT32 });
      const s1 = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const s2 = s1.reAlign(5n);
      const { flushed, allocated } = cache.write([s1, s2]);
      expect(flushed).toHaveLength(1);
      expect(allocated[1].timeRange.start.sub(TimeStamp.now()).valueOf()).toBeLessThan(
        TimeSpan.milliseconds(10).valueOf(),
      );
      expect(allocated[1].timeRange.end.valueOf()).toEqual(TimeStamp.MAX.valueOf());
      expect(flushed[0]).toBe(allocated[0]);
      expect(allocated).toHaveLength(2);
    });
    it("should allocate a buffer properly using a TimeSpan", () => {
      let nowF = () => TimeStamp.seconds(1);
      const now = () => {
        return nowF();
      };
      const cache = new Dynamic({
        dynamicBufferSize: TimeSpan.minutes(5),
        dataType: DataType.FLOAT32,
        testingNow: now,
      });
      const arr = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const res1 = cache.write([arr]);
      expect(res1.allocated).toHaveLength(1);
      expect(res1.flushed).toHaveLength(0);
      nowF = () => TimeStamp.seconds(2);
      const res2 = cache.write([arr.reAlign(3n)]);
      expect(res2.allocated).toHaveLength(0);
      expect(res2.flushed).toHaveLength(0);

      nowF = () => TimeStamp.seconds(3);
      const res3 = cache.write([arr.reAlign(6n)]);
      expect(res3.allocated).toHaveLength(0);
      expect(res3.flushed).toHaveLength(0);
      expect(cache.length).toBe(9);
    });
  });
});
