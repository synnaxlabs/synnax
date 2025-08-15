// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, DataType, MultiSeries, Series } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type SeriesSource } from "@/telem/aether/telem";
import { SeriesDownsampler } from "@/telem/aether/transformers";

describe("SeriesDownsampler", () => {
  describe("decimation mode", () => {
    it("should take every nth sample", () => {
      const downsampler = new SeriesDownsampler({
        mode: "none",
        windowSize: 3,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4, 5, 6, 7, 8, 9]),
          dataType: DataType.FLOAT32,
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 9), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      const [, resultSeries] = downsampler.value();
      expect(resultSeries.series).toHaveLength(1);
      expect(Array.from(resultSeries.series[0])).toEqual([1, 4, 7]);
    });

    it("should handle incremental updates", () => {
      const downsampler = new SeriesDownsampler({
        mode: "none",
        windowSize: 2,
      });

      const sourceSingle = Series.alloc({
        dataType: DataType.FLOAT32,
        key: "test-series",
        capacity: 8,
      });
      sourceSingle.write(
        new Series({
          data: new Float32Array([1, 2, 3, 4]),
          dataType: DataType.FLOAT32,
        }),
      );

      let source = new MultiSeries([sourceSingle]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 4), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      let [, result] = downsampler.value();
      expect(Array.from(result.series[0])).toEqual([1, 3]);

      // Add more data to the same series
      sourceSingle.write(
        new Series({
          data: new Float32Array([5, 6, 7, 8]),
          dataType: DataType.FLOAT32,
        }),
      );
      source = new MultiSeries([sourceSingle]);
      mockSource.value = () => [bounds.construct(1, 8), source];

      [, result] = downsampler.value();
      expect(Array.from(result.series[0])).toEqual([1, 3, 5, 7]);
    });

    it("should handle partial final window", () => {
      const downsampler = new SeriesDownsampler({
        mode: "none",
        windowSize: 3,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4, 5]),
          dataType: DataType.FLOAT32,
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 5), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      const [, result] = downsampler.value();
      expect(Array.from(result.series[0])).toEqual([1, 4]);
    });
  });

  describe("average mode", () => {
    it("should average samples in each window", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 3,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4, 5, 6, 7, 8, 9]),
          dataType: DataType.FLOAT32,
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 9), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      const [, result] = downsampler.value();
      expect(Array.from(result.series[0])).toEqual([2, 5, 8]); // (1+2+3)/3, (4+5+6)/3, (7+8+9)/3
    });

    it("should handle partial final window", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 3,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4, 5]),
          dataType: DataType.FLOAT32,
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 5), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      const [, result] = downsampler.value();
      expect(Array.from(result.series[0])).toEqual([2, 4.5]); // (1+2+3)/3, (4+5)/2
    });

    it("should handle incremental updates correctly", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 2,
      });

      const sourceSingle = Series.alloc({
        dataType: DataType.FLOAT32,
        key: "avg-test",
        capacity: 10,
      });
      sourceSingle.write(
        new Series({
          data: new Float32Array([2, 4, 6, 8]),
          dataType: DataType.FLOAT32,
        }),
      );

      let source = new MultiSeries([sourceSingle]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(2, 8), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      let [, result] = downsampler.value();
      expect(Array.from(result.series[0])).toEqual([3, 7]); // (2+4)/2, (6+8)/2

      // Add more data to the same series
      sourceSingle.write(
        new Series({
          data: new Float32Array([10, 12]),
          dataType: DataType.FLOAT32,
        }),
      );
      source = new MultiSeries([sourceSingle]);
      mockSource.value = () => [bounds.construct(2, 12), source];

      [, result] = downsampler.value();
      expect(Array.from(result.series[0])).toEqual([3, 7, 11]); // (2+4)/2, (6+8)/2, (10+12)/2
    });
  });

  describe("minmax mode", () => {
    it("should preserve min and max in each window", () => {
      const downsampler = new SeriesDownsampler({
        mode: "minmax",
        windowSize: 3,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 5, 2, 8, 3, 6, 9, 4, 7]),
          dataType: DataType.FLOAT32,
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 9), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      const [, result] = downsampler.value();
      expect(Array.from(result.series[0])).toEqual([
        1,
        5, // min=1, max=5 for [1,5,2]
        3,
        8, // min=3, max=8 for [8,3,6]
        4,
        9, // min=4, max=9 for [9,4,7]
      ]);
    });

    it("should handle partial final window", () => {
      const downsampler = new SeriesDownsampler({
        mode: "minmax",
        windowSize: 3,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 5, 2, 8, 3]),
          dataType: DataType.FLOAT32,
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 8), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      const [, result] = downsampler.value();
      expect(Array.from(result.series[0])).toEqual([
        1,
        5, // min=1, max=5 for [1,5,2]
        3,
        8, // min=3, max=8 for [8,3]
      ]);
    });

    it("should handle incremental updates", () => {
      const downsampler = new SeriesDownsampler({
        mode: "minmax",
        windowSize: 2,
      });

      const sourceSingle = Series.alloc({
        dataType: DataType.FLOAT32,
        key: "minmax-test",
        capacity: 10,
      });
      sourceSingle.write(
        new Series({
          data: new Float32Array([5, 3, 8, 2]),
          dataType: DataType.FLOAT32,
        }),
      );

      let source = new MultiSeries([sourceSingle]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(2, 8), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      let [, result] = downsampler.value();
      expect(Array.from(result.series[0])).toEqual([
        3,
        5, // min=3, max=5 for [5,3]
        2,
        8, // min=2, max=8 for [8,2]
      ]);

      // Add more data to the same series
      sourceSingle.write(
        new Series({
          data: new Float32Array([6, 9]),
          dataType: DataType.FLOAT32,
        }),
      );
      source = new MultiSeries([sourceSingle]);
      mockSource.value = () => [bounds.construct(2, 9), source];

      [, result] = downsampler.value();
      expect(Array.from(result.series[0])).toEqual([
        3,
        5, // min=3, max=5 for [5,3]
        2,
        8, // min=2, max=8 for [8,2]
        6,
        9, // min=6, max=9 for [6,9]
      ]);
    });
  });

  describe("cache management", () => {
    it("should evict series that are removed from the source", () => {
      const downsampler = new SeriesDownsampler({
        mode: "none",
        windowSize: 2,
      });

      // Start with two series
      let source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4]),
          dataType: DataType.FLOAT32,
          key: "series-1",
        }),
        new Series({
          data: new Float32Array([5, 6, 7, 8]),
          dataType: DataType.FLOAT32,
          key: "series-2",
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 8), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      let [, result] = downsampler.value();
      expect(result.series).toHaveLength(2);
      expect(Array.from(result.series[0])).toEqual([1, 3]);
      expect(Array.from(result.series[1])).toEqual([5, 7]);

      // Remove first series
      source = new MultiSeries([
        new Series({
          data: new Float32Array([5, 6, 7, 8]),
          dataType: DataType.FLOAT32,
          key: "series-2",
        }),
      ]);
      mockSource.value = () => [bounds.construct(5, 8), source];

      [, result] = downsampler.value();
      expect(result.series).toHaveLength(1);
      expect(Array.from(result.series[0])).toEqual([5, 7]);
    });

    it("should handle series being added", () => {
      const downsampler = new SeriesDownsampler({
        mode: "none",
        windowSize: 2,
      });

      // Start with one series
      let source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4]),
          dataType: DataType.FLOAT32,
          key: "series-1",
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 4), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      let [, result] = downsampler.value();
      expect(result.series).toHaveLength(1);

      // Add another series
      source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4]),
          dataType: DataType.FLOAT32,
          key: "series-1",
        }),
        new Series({
          data: new Float32Array([5, 6, 7, 8]),
          dataType: DataType.FLOAT32,
          key: "series-2",
        }),
      ]);
      mockSource.value = () => [bounds.construct(1, 8), source];

      [, result] = downsampler.value();
      expect(result.series).toHaveLength(2);
      expect(Array.from(result.series[0])).toEqual([1, 3]);
      expect(Array.from(result.series[1])).toEqual([5, 7]);
    });

    it("should reuse cached downsampled series across multiple calls", () => {
      const downsampler = new SeriesDownsampler({
        mode: "none",
        windowSize: 2,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4]),
          dataType: DataType.FLOAT32,
          key: "cached-series",
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 4), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      const [, result1] = downsampler.value();
      const [, result2] = downsampler.value();

      // Should be the exact same downsampled series object (cached)
      expect(result1.series[0]).toBe(result2.series[0]);
    });
  });

  describe("bounds calculation", () => {
    it("should correctly calculate bounds after downsampling", () => {
      const downsampler = new SeriesDownsampler({
        mode: "average",
        windowSize: 2,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 3, 2, 8, 5, 7]),
          dataType: DataType.FLOAT32,
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 8), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      const [resultBounds, _] = downsampler.value();
      // Averaged values: [2, 5, 6]
      expect(resultBounds).toEqual(bounds.construct(2, 6));
    });

    it("should handle minmax mode bounds correctly", () => {
      const downsampler = new SeriesDownsampler({
        mode: "minmax",
        windowSize: 3,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([5, 1, 3, 9, 2, 4]),
          dataType: DataType.FLOAT32,
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 9), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      const [, result] = downsampler.value();
      // Min/max pairs: [1,5, 2,9]
      expect(result.bounds).toEqual(bounds.construct(1, 9));
    });
  });

  describe("edge cases", () => {
    it("should handle empty source", () => {
      const downsampler = new SeriesDownsampler({
        mode: "none",
        windowSize: 2,
      });

      const source = new MultiSeries([]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(0, 0), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      const [resultBounds, result] = downsampler.value();
      expect(result.series).toHaveLength(0);
      expect(resultBounds).toEqual(bounds.construct(0, 0));
    });

    it("should handle window size of 1", () => {
      const downsampler = new SeriesDownsampler({
        mode: "none",
        windowSize: 1,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 3), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      const [, result] = downsampler.value();
      expect(Array.from(result.series[0])).toEqual([1, 2, 3]);
    });

    it("should handle series with alignment", () => {
      const downsampler = new SeriesDownsampler({
        mode: "none",
        windowSize: 2,
      });

      const source = new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3, 4]),
          dataType: DataType.FLOAT32,
          alignment: 100n,
          key: "aligned-series",
        }),
      ]);

      const mockSource: SeriesSource = {
        value: () => [bounds.construct(1, 4), source],
        onChange: () => () => {},
      };
      downsampler.setSources({ source: mockSource });

      const [, result] = downsampler.value();
      expect(result.series[0].alignment).toBe(100n);
      expect(Array.from(result.series[0])).toEqual([1, 3]);
    });
  });
});
