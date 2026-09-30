// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, MultiSeries, Series, TimeStamp } from "@synnaxlabs/x";
import { allocSuite } from "@synnaxlabs/x/bench";
import { test } from "vitest";

import { Unary } from "@/framer/cache/unary";

const SAMPLES = 10;
const WRITES_PER_BUFFER = 10_000;

const data = new Float32Array(SAMPLES);

// One stamped frame per write, with the monotonic alignments and time ranges the
// streamer feeds the cache in steady state. Built up front so the loop measures the
// cache alone.
const frames: MultiSeries[] = Array.from(
  { length: WRITES_PER_BUFFER },
  (_, i) =>
    new MultiSeries([
      new Series({
        data,
        dataType: DataType.FLOAT32,
        timeRange: TimeStamp.milliseconds(i).range(TimeStamp.milliseconds(i + 1)),
        alignment: BigInt(i * SAMPLES),
      }),
    ]),
);

const newCache = (): Unary =>
  new Unary({ dynamicBufferSize: SAMPLES * WRITES_PER_BUFFER });

// Rotates the cache before its buffer flushes into the static cache, so every
// iteration measures a write into the leading buffer alone.
const newWriter = (readLastWrite: boolean): (() => TimeStamp | null) => {
  let cache = newCache();
  let i = 0;
  return () => {
    if (i === WRITES_PER_BUFFER) {
      cache.close();
      cache = newCache();
      i = 0;
    }
    cache.writeDynamic(frames[i++]);
    return readLastWrite ? cache.lastWrite : null;
  };
};

test("Unary.writeDynamic", async ({ bench }) => {
  const write = newWriter(false);
  const writeAndRead = newWriter(true);
  await bench.compare(
    bench("write", () => {
      write();
    }),
    bench("write + lastWrite", () => {
      writeAndRead();
    }),
  );
});

allocSuite("Unary.writeDynamic 10 samples", [["write", newWriter(false)]]);
