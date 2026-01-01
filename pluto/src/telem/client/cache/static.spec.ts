// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, MultiSeries, Series, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { Static } from "@/telem/client/cache/static";

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
});
