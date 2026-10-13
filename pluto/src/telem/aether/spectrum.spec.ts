// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, DataType, MultiSeries, Series } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Spectrum, windowSamples } from "@/telem/aether/spectrum";
import { type SeriesValue } from "@/telem/aether/telem";
import { TestSource } from "@/telem/aether/test/source";

const series = (data: number[], dataType: DataType, alignment: bigint): Series =>
  new Series({ data: new Float64Array(data), dataType, alignment });

describe("windowSamples", () => {
  it("should take each sample once when series overlap", () => {
    const stamps = (from: number, to: number) =>
      Array.from({ length: to - from }, (_, i) => (from + i) * 10);
    const values = (from: number, to: number) =>
      Array.from({ length: to - from }, (_, i) => from + i);
    const index = new MultiSeries([
      series(stamps(0, 10), DataType.FLOAT64, 0n),
      series(stamps(5, 15), DataType.FLOAT64, 5n),
    ]);
    const data = new MultiSeries([
      series(values(0, 10), DataType.FLOAT64, 0n),
      series(values(5, 15), DataType.FLOAT64, 5n),
    ]);
    const { y, t } = windowSamples(index, data, { lower: 0, upper: 140 });
    expect(Array.from(y)).toEqual(values(0, 15));
    expect(Array.from(t)).toEqual(stamps(0, 15));
  });

  it("should pair data split into different series than its index", () => {
    const index = new MultiSeries([
      series([0, 10, 20, 30, 40, 50], DataType.FLOAT64, 0n),
    ]);
    const data = new MultiSeries([
      series([0, 1], DataType.FLOAT64, 0n),
      series([2, 3, 4, 5], DataType.FLOAT64, 2n),
    ]);
    const { y, t } = windowSamples(index, data, { lower: 10, upper: 40 });
    expect(Array.from(y)).toEqual([1, 2, 3, 4]);
    expect(Array.from(t)).toEqual([10, 20, 30, 40]);
  });
});

describe("Spectrum", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  const empty: SeriesValue = [bounds.INVALID, new MultiSeries([])];

  it("should pass source changes on at most once a second", () => {
    const data = new TestSource<SeriesValue>(empty);
    const index = new TestSource<SeriesValue>(empty);
    const spectrum = new Spectrum({ output: "magnitude" });
    spectrum.setSources({ data, index });
    const handler = vi.fn();
    spectrum.onChange(handler);
    spectrum.value();
    data.setValue(empty);
    data.setValue(empty);
    index.setValue(empty);
    expect(handler).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1000);
    expect(handler).toHaveBeenCalledTimes(1);
  });
});
