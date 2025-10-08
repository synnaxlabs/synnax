// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, MultiSeries, Series, TimeRange } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { SeriesDownsampler } from "@/telem/aether/transformers";

describe("SeriesDownsampler", () => {
  describe("decimate mode", () => {
    it("should return the original series", () => {
      const downsampler = new SeriesDownsampler({
        mode: "decimate",
        windowSize: 5,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4, 5, 6, 7, 8, 9, 10]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 10n),
          alignment: 0n,
        }),
      ]);

      const result = downsampler.transform(source);
      expect(result).toEqual(source);
    });

    it("should handle series with sample offset", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 3,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4, 5, 6]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 6n),
          alignment: 0n,
          sampleOffset: 1,
        }),
      ]);

      const result = downsampler.transform(source);
      expect(result.series[0].length).toBe(2);
    });
  });

  describe("average mode", () => {
    it("should average data with exact window alignment", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 3,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4, 5, 6, 7, 8, 9]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 9n),
          alignment: 0n,
        }),
      ]);

      const result = downsampler.transform(source);
      expect(result.series[0].length).toBe(3);
      expect(result.series[0].at(0)).toBe(2);
      expect(result.series[0].at(1)).toBe(5);
      expect(result.series[0].at(2)).toBe(8);
    });

    it("should handle partial windows at the end", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 4,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4, 5, 6, 7]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 7n),
          alignment: 0n,
        }),
      ]);

      const result = downsampler.transform(source);
      expect(result.series[0].length).toBe(1);
      expect(result.series[0].at(0)).toBe(2.5);
    });

    it("should handle multiple series", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 2,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 4n),
          alignment: 0n,
        }),
        new Series({
          data: new Float32Array([10, 20, 30, 40]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 4n),
          alignment: 0n,
        }),
      ]);

      const result = downsampler.transform(source);
      expect(result.series.length).toBe(2);

      expect(result.series[0].at(0)).toBe(1.5);
      expect(result.series[0].at(1)).toBe(3.5);

      expect(result.series[1].at(0)).toBe(15);
      expect(result.series[1].at(1)).toBe(35);
    });

    it("should handle NaN values", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 3,
      });

      const data = new Float32Array([1, NaN, 3, 4, 5, 6]);
      const source = new MultiSeries([
        new Series({
          data,
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 6n),
          alignment: 0n,
        }),
      ]);

      const result = downsampler.transform(source);
      expect(result.series[0].length).toBe(2);
      expect(Number.isNaN(result.series[0].at(0))).toBe(true);
      expect(result.series[0].at(1)).toBe(5);
    });

    it("should cache and reuse downsampled series", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 2,
      });

      const source1 = new MultiSeries([
        new Series({
          key: "series1",
          data: new Float32Array([1, 2, 3, 4]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 4n),
          alignment: 0n,
        }),
      ]);

      const result1 = downsampler.transform(source1);
      expect(result1.series[0].length).toBe(2);

      const source2 = new MultiSeries([
        new Series({
          key: "series1",
          data: new Float32Array([1, 2, 3, 4, 5, 6]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 6n),
          alignment: 0n,
        }),
      ]);

      const result2 = downsampler.transform(source2);
      expect(result2.series[0].length).toBe(2);
      expect(result2.series[0].at(1)).toBe(3.5);
    });

    it("should evict old series from cache", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 2,
      });

      const source1 = new MultiSeries([
        new Series({
          key: "series1",
          data: new Float32Array([1, 2, 3, 4]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 4n),
          alignment: 0n,
        }),
      ]);

      downsampler.transform(source1);

      const source2 = new MultiSeries([
        new Series({
          key: "series2",
          data: new Float32Array([10, 20, 30, 40]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 4n),
          alignment: 0n,
        }),
      ]);

      const result2 = downsampler.transform(source2);
      expect(result2.series[0].at(0)).toBe(15);
      expect(result2.series[0].at(1)).toBe(35);
    });
  });

  describe("edge cases", () => {
    it("should return source when windowSize is 1", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 1,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4, 5]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 5n),
          alignment: 0n,
        }),
      ]);

      const result = downsampler.transform(source);
      expect(result).toEqual(source);
    });

    it("should return source when windowSize is 0", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 0,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 3n),
          alignment: 0n,
        }),
      ]);

      const result = downsampler.transform(source);
      expect(result).toEqual(source);
    });

    it("should handle empty series", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 3,
      });

      const source = new MultiSeries([]);
      const result = downsampler.transform(source);
      expect(result.series.length).toBe(0);
    });

    it("should handle series with minimal data", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 3,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 2n),
          alignment: 0n,
        }),
      ]);

      const result = downsampler.transform(source);
      expect(result.series[0].length).toBe(0);
    });

    it("should preserve series metadata", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 2,
      });

      const source = new MultiSeries([
        new Series({
          key: "test-series",
          data: new Float32Array([1, 2, 3, 4]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(100n, 200n),
          alignment: 10n,
        }),
      ]);

      const result = downsampler.transform(source);
      expect(result.series[0].dataType).toEqual(DataType.FLOAT32);
      expect(result.series[0].timeRange.start.valueOf()).toBe(100n);
      expect(result.series[0].timeRange.end.valueOf()).toBe(200n);
      expect(result.series[0].alignment).toBe(10n);
      expect(result.series[0].alignmentMultiple).toBe(2n);
    });

    it("should handle different data types", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 2,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float64Array([1.5, 2.5, 3.5, 4.5]),
          dataType: DataType.FLOAT64,
          timeRange: new TimeRange(0n, 4n),
          alignment: 0n,
        }),
      ]);

      const result = downsampler.transform(source);
      expect(result.series[0].dataType).toEqual(DataType.FLOAT64);
      expect(result.series[0].at(0)).toBe(2);
      expect(result.series[0].at(1)).toBe(4);
    });

    it("should throw error for mismatched cache keys", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 2,
      });

      const source1 = new MultiSeries([
        new Series({
          key: "series1",
          data: new Float32Array([1, 2]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 2n),
          alignment: 0n,
        }),
        new Series({
          key: "series2",
          data: new Float32Array([3, 4]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 2n),
          alignment: 0n,
        }),
      ]);

      downsampler.transform(source1);

      const source2 = new MultiSeries([
        new Series({
          key: "series1",
          data: new Float32Array([1, 2, 3, 4]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 4n),
          alignment: 0n,
        }),
        new Series({
          key: "series3",
          data: new Float32Array([5, 6, 7, 8]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 4n),
          alignment: 0n,
        }),
      ]);

      expect(() => downsampler.transform(source2)).toThrow(
        /expected series with key series3 to be in cache/,
      );
    });
  });

  describe("performance considerations", () => {
    it("should handle large datasets efficiently", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 100,
      });

      const largeData = new Float32Array(10000);
      for (let i = 0; i < largeData.length; i++) largeData[i] = i;

      const source = new MultiSeries([
        new Series({
          data: largeData,
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 10000n),
          alignment: 0n,
        }),
      ]);

      const result = downsampler.transform(source);
      expect(result.series[0].length).toBe(100);
      expect(result.series[0].at(0)).toBe(49.5);
      expect(result.series[0].at(99)).toBe(9949.5);
    });

    it("should efficiently append to cached series", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 2,
      });

      const source1 = new MultiSeries([
        new Series({
          key: "stream",
          data: new Float32Array([0, 1, 2, 3]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 4n),
          alignment: 0n,
        }),
      ]);

      const result1 = downsampler.transform(source1);
      expect(result1.series[0].length).toBe(2);
      expect(result1.series[0].at(0)).toBe(0.5);
      expect(result1.series[0].at(1)).toBe(2.5);

      const source2 = new MultiSeries([
        new Series({
          key: "stream",
          data: new Float32Array([0, 1, 2, 3, 4, 5, 6, 7]),
          dataType: DataType.FLOAT32,
          timeRange: new TimeRange(0n, 8n),
          alignment: 0n,
        }),
      ]);

      const result2 = downsampler.transform(source2);
      expect(result2.series[0].length).toBe(2);
      expect(result2.series[0].at(0)).toBe(0.5);
      expect(result2.series[0].at(1)).toBe(2.5);
    });
  });
});
