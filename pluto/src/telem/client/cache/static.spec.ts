// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, test, it } from "vitest";

import { Static } from "@/telem/client/cache/static";

// NOTE: Most of the insertion algorithm logic is not implemented in the static cache,
// but inside the x/ts/src/spatial/bounds module, where there are comprehensive tests.
// These tests are more focused on reading than writing.
describe("StaticReadCache", () => {
  describe("read and write", () => {
    test("simple write", () => {
      const c = new Static({});
      const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
      c.write([
        new Series({
          data: new Float32Array([1]),
          dataType: DataType.FLOAT32,
          timeRange: tr,
          alignment: 0,
        }),
      ]);
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3)),
      );
      expect(series).toHaveLength(1);
      expect(gaps).toHaveLength(0);
    });
    test("should correctly return leading and trailing gaps", () => {
      const c = new Static({});
      const tr = TimeStamp.seconds(2).spanRange(TimeSpan.seconds(3));
      c.write([
        new Series({
          data: new Float32Array([1]),
          dataType: DataType.FLOAT32,
          timeRange: tr,
          alignment: 0,
        }),
      ]);
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
      c.write([
        new Series({
          data: new Float32Array([1, 2]),
          dataType: DataType.FLOAT32,
          timeRange: tr1,
          alignment: 1,
        }),
      ]);
      c.write([
        new Series({
          data: new Float32Array([4, 5]),
          dataType: DataType.FLOAT32,
          timeRange: tr2,
          alignment: 4,
        }),
      ]);
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(1).range(TimeStamp.seconds(7)),
      );
      expect(series).toHaveLength(2);
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
      c.write([
        new Series({
          data: new Float32Array([2, 3, 4, 5]),
          dataType: DataType.FLOAT32,
          timeRange: tr1,
          alignment: 2,
        }),
      ]);
      c.write([
        new Series({
          data: new Float32Array([4, 5, 6]),
          dataType: DataType.FLOAT32,
          timeRange: tr2,
          alignment: 4,
        }),
      ]);
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(2).range(TimeSpan.seconds(7)),
      );
      expect(series).toHaveLength(2);
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
      c.write([
        new Series({
          data: new Float32Array([3, 4, 5, 6]),
          dataType: DataType.FLOAT32,
          timeRange: tr1,
          alignment: 3,
        }),
      ]);
      c.write([
        new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
          timeRange: tr2,
          alignment: 1,
        }),
      ]);
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(1).range(TimeSpan.seconds(7)),
      );
      expect(series).toHaveLength(2);
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
      c.write([
        new Series({
          data: new Float32Array([1, 2, 3, 4]),
          dataType: DataType.FLOAT32,
          timeRange: tr1,
          alignment: 0,
        }),
      ]);
      c.write([
        new Series({
          data: new Float32Array([5, 6, 7, 8]),
          dataType: DataType.FLOAT32,
          timeRange: tr2,
          alignment: 0,
        }),
      ]);
      const { series, gaps } = c.dirtyRead(
        TimeStamp.seconds(2).range(TimeStamp.seconds(4)),
      );
      expect(series).toHaveLength(1);
      expect(series[0].data).toEqual(new Float32Array([5, 6, 7, 8]));
      expect(gaps).toHaveLength(0);
    });
  });
  describe("garbage collection", () => {
    it("should correctly garbage collect series that have a reference count of zero", async () => {
      const c = new Static({ staleEntryThreshold: TimeSpan.milliseconds(5) });
      const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
      c.write([
        new Series({
          data: new Float32Array([1]),
          dataType: DataType.FLOAT32,
          timeRange: tr,
          alignment: 0,
        }),
      ]);
      c.dirtyRead(TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3)));
      expect(c.gc().purgedSeries).toEqual(0);
      expect(
        c.dirtyRead(TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3))).series,
      ).toHaveLength(1);
      await new Promise((resolve) => setTimeout(resolve, 8));
      expect(c.gc().purgedSeries).toEqual(1);
      expect(
        c.dirtyRead(TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3))).series,
      ).toHaveLength(0);
    });
  });
  describe("close", () => {
    it("should remove all series from the cache", () => {
      const c = new Static({});
      const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
      c.write([
        new Series({
          data: new Float32Array([1]),
          dataType: DataType.FLOAT32,
          timeRange: tr,
          alignment: 0,
        }),
      ]);
      c.close();
      expect(
        c.dirtyRead(TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3))).series,
      ).toHaveLength(0);
    });
  });
});
