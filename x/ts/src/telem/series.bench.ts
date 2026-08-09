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

describe("data access", () => {
  const partial = Series.alloc({ capacity: 1000, dataType: DataType.FLOAT32 });
  partial.write(new Series({ data: new Float32Array(500) }));
  const full = new Series({
    data: new Float32Array(1000).buffer,
    dataType: DataType.FLOAT32,
  });
  bench("partial buffer", () => partial.data.length);
  bench("full buffer", () => full.data.length);
});
