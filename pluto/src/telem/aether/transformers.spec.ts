// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, MultiSeries, Series, TimeRange } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { TestSource } from "@/telem/aether/test/source";
import {
  RollingAverage,
  ScaleNumber,
  SeriesDownsampler,
  StringifyNumber,
  WithinBounds,
} from "@/telem/aether/transformers";

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

describe("StringifyNumber", () => {
  it("formats a number value with prefix and suffix", () => {
    const t = new StringifyNumber({ precision: 2, prefix: "$", suffix: " USD" });
    t.setSources({ in: new TestSource(42) });
    expect(t.value()).toBe("$42.00 USD");
  });

  it("preserves full precision for a bigint value above 2^53 in standard notation", () => {
    const t = new StringifyNumber({ precision: 0, notation: "standard" });
    t.setSources({ in: new TestSource(1778020940471336960n) });
    expect(t.value()).toBe("1778020940471336960");
  });

  it("does not crash and returns a finite string for a bigint value", () => {
    const t = new StringifyNumber({ precision: 2, notation: "scientific" });
    t.setSources({ in: new TestSource(1778020940471336960n) });
    expect(t.value()).toBe("1.78ᴇ18");
  });

  it("returns an empty string for NaN", () => {
    const t = new StringifyNumber({ precision: 2 });
    t.setSources({ in: new TestSource(NaN) });
    expect(t.value()).toBe("");
  });
});

describe("RollingAverage", () => {
  it("returns the value unchanged when windowSize is less than 2", () => {
    const t = new RollingAverage({ windowSize: 1 });
    t.setSources({ in: new TestSource(42) });
    expect(t.value()).toBe(42);
  });

  it("coerces a bigint value to a number", () => {
    const t = new RollingAverage({ windowSize: 1 });
    t.setSources({ in: new TestSource(123n) });
    expect(t.value()).toBe(123);
  });
});

describe("ScaleNumber", () => {
  it("applies the scale and offset to a number value", () => {
    const t = new ScaleNumber({ scale: { scale: 2, offset: 3 } });
    t.setSources({ in: new TestSource(10) });
    expect(t.value()).toBe(23);
  });

  it("coerces a bigint value to a number before scaling", () => {
    const t = new ScaleNumber({ scale: { scale: 2, offset: 0 } });
    t.setSources({ in: new TestSource(50n) });
    expect(t.value()).toBe(100);
  });

  it("returns NaN when given NaN", () => {
    const t = new ScaleNumber({ scale: { scale: 2, offset: 3 } });
    t.setSources({ in: new TestSource(NaN) });
    expect(Number.isNaN(t.value())).toBe(true);
  });
});

describe("WithinBounds", () => {
  it("returns true for a value inside the bounds", () => {
    const t = new WithinBounds({ trueBound: { lower: 5, upper: 15 } });
    t.setSources({ in: new TestSource(10) });
    expect(t.value()).toBe(true);
  });

  it("returns false for a value outside the bounds", () => {
    const t = new WithinBounds({ trueBound: { lower: 5, upper: 15 } });
    t.setSources({ in: new TestSource(20) });
    expect(t.value()).toBe(false);
  });

  // Staleness counts arrivals, so a sample that leaves the boolean unchanged must still
  // reach the listener.
  it("notifies on every sample, even when the boolean stays the same", () => {
    const t = new WithinBounds({ trueBound: { lower: 5, upper: 15 } });
    const source = new TestSource(10);
    t.setSources({ in: source });
    const handler = vi.fn();
    t.onChange(handler);
    source.setValue(11);
    source.setValue(12);
    expect(handler).toHaveBeenCalledTimes(2);
  });
});
