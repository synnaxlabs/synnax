// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, MultiSeries, Series, TimeRange, TimeSpan } from "@synnaxlabs/x";
import { allocSuite } from "@synnaxlabs/x/bench";
import { bench, describe } from "vitest";

import { Dynamic } from "@/framer/cache/dynamic";

const makeSeries = (samples: number, alignment: bigint): Series => {
  const data = new Float32Array(samples);
  for (let i = 0; i < data.length; i++) data[i] = i;
  return new Series({
    data,
    dataType: DataType.FLOAT32,
    alignment,
    timeRange: TimeRange.ZERO,
  });
};

// Steady state: every write lands in the current buffer without flushing. This is
// the per-channel, per-frame cost of the streaming run loop. The time-based size
// variant is the Feed's default configuration.
const makeSteadyCase = (
  samples: number,
  dynamicBufferSize: number | TimeSpan = 100_000,
): (() => void) => {
  const dynamic = new Dynamic({ dynamicBufferSize });
  const inputs = Array.from(
    { length: 512 },
    (_, i) => new MultiSeries([makeSeries(samples, BigInt(i * samples))]),
  );
  let i = 0;
  return () => {
    dynamic.write(inputs[i]);
    i = (i + 1) % inputs.length;
  };
};

// Flush-heavy: every write overflows the buffer, exercising allocation and flush.
const makeFlushCase = (): (() => void) => {
  const dynamic = new Dynamic({ dynamicBufferSize: 64 });
  const inputs = Array.from(
    { length: 512 },
    (_, i) => new MultiSeries([makeSeries(128, BigInt(i * 128))]),
  );
  let i = 0;
  return () => {
    dynamic.write(inputs[i]);
    i = (i + 1) % inputs.length;
  };
};

describe("write", () => {
  bench("steady 10smp", makeSteadyCase(10));
  bench("steady 10smp timespan-sized", makeSteadyCase(10, TimeSpan.seconds(60)));
  bench("steady 100smp", makeSteadyCase(100));
  bench("flush every write", makeFlushCase());
});

allocSuite("dynamic cache write", [
  ["steady 10smp", makeSteadyCase(10)],
  ["steady 10smp timespan-sized", makeSteadyCase(10, TimeSpan.seconds(60))],
  ["flush every write", makeFlushCase()],
]);
