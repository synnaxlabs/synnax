// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bench, describe } from "vitest";

import { DataType, Series } from "@/telem";

// Streaming caches append small series into a large allocated buffer at high
// rates, then read the partial view back on every render.
describe("write", () => {
  // Wire-decoded series are ArrayBuffer-backed, matching the streaming path.
  const source = new Series({
    data: new Float32Array(10).buffer,
    dataType: DataType.FLOAT32,
  });
  let target = Series.alloc({ capacity: 1_000_000, dataType: DataType.FLOAT32 });
  bench("10smp into allocated buffer", () => {
    if (target.write(source) === 0)
      target = Series.alloc({ capacity: 1_000_000, dataType: DataType.FLOAT32 });
  });
});

// Rolling plots query y bounds over a moving sub-range every frame. The block
// summaries should keep the warm query cost flat as the series grows, while a
// naive scan grows linearly with the window.
describe("boundsFor", () => {
  const makeSeries = (samples: number): Series => {
    const data = new Float32Array(samples);
    for (let i = 0; i < data.length; i++) data[i] = Math.sin(i);
    return new Series({ data, dataType: DataType.FLOAT32 });
  };
  const sink = { bounds: 0 };
  const small = makeSeries(10_000);
  const large = makeSeries(1_500_000);
  // Warm the block summaries so the benchmarks measure steady-state queries.
  small.boundsFor(1, 10_000);
  large.boundsFor(1, 1_500_000);
  bench("warm sub-range query 10k", () => {
    sink.bounds += small.boundsFor(1, 9_999).lower;
  });
  bench("warm sub-range query 1.5m", () => {
    sink.bounds += large.boundsFor(1, 1_499_999).lower;
  });
  const raw = large.data as Float32Array;
  bench("naive full scan 1.5m", () => {
    let min = Infinity;
    let max = -Infinity;
    for (let i = 1; i < 1_499_999; i++) {
      const v = raw[i];
      if (v < min) min = v;
      if (v > max) max = v;
    }
    sink.bounds += min + max;
  });
});

describe("data access", () => {
  const partial = Series.alloc({ capacity: 1000, dataType: DataType.FLOAT32 });
  partial.write(new Series({ data: new Float32Array(500) }));
  const full = new Series({
    data: new Float32Array(1000).buffer,
    dataType: DataType.FLOAT32,
  });
  // Accumulating keeps the read observable, so the optimizer cannot drop the access
  // the benchmark is timing.
  const sink = { full: 0, partial: 0 };
  bench("partial buffer", () => {
    sink.partial += partial.data.length;
  });
  bench("full buffer", () => {
    sink.full += full.data.length;
  });
});
