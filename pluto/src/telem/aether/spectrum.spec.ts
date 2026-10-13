// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, MultiSeries, Series } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { windowSamples } from "@/telem/aether/spectrum";

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
});
