// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PerformanceObserver } from "node:perf_hooks";
import v8 from "node:v8";
import vm from "node:vm";

import { DataType, TimeRange } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { Codec } from "@/framer/codec";
import { Frame, type Payload } from "@/framer/frame";

// Exposes V8's gc() to this process even when the runner did not pass --expose-gc.
const exposeGC = (): (() => void) | null => {
  try {
    v8.setFlagsFromString("--expose-gc");
    const gc = vm.runInNewContext("gc") as () => void;
    v8.setFlagsFromString("--no-expose-gc");
    return gc;
  } catch {
    return null;
  }
};

const gc = exposeGC();

export const makeKeys = (n: number): channel.Key[] =>
  Array.from({ length: n }, (_, i) => i + 1);

export interface PayloadOptions {
  /** Base alignment for every series in the frame. */
  alignment?: bigint;
  /** Give every series a distinct time range and alignment, defeating the codec's
   * equal-props fast paths. */
  varied?: boolean;
}

/** Builds a frame payload with one float32 series per key. */
export const makeFramePayload = (
  keys: channel.Key[],
  samplesPerChannel: number,
  { alignment = 0n, varied = false }: PayloadOptions = {},
): Payload => {
  const data = new Float32Array(samplesPerChannel);
  for (let i = 0; i < data.length; i++) data[i] = i;
  return {
    keys: [...keys],
    series: keys.map((_, i) => ({
      dataType: DataType.FLOAT32,
      data: data.buffer.slice(0),
      alignment: varied ? alignment + BigInt(i) : alignment,
      timeRange: varied ? new TimeRange(i * 1000, (i + 1) * 1000) : TimeRange.ZERO,
    })),
  };
};

/** Builds a sequence of frames with monotonically increasing alignments, the shape
 * the streaming run loop ingests in steady state. */
export const makeFrameSequence = (
  keys: channel.Key[],
  samplesPerChannel: number,
  count: number,
): Frame[] =>
  Array.from(
    { length: count },
    (_, i) =>
      new Frame(
        makeFramePayload(keys, samplesPerChannel, {
          alignment: BigInt(i * samplesPerChannel),
        }),
      ),
  );

export const makeCodec = (keys: channel.Key[]): Codec =>
  new Codec(
    keys,
    keys.map(() => DataType.FLOAT32),
  );

export interface AllocMeasurement {
  /** Average heap bytes allocated per operation. NaN when GC is not exposed. */
  bytesPerOp: number;
  /** Iterations the measurement settled at. */
  iterations: number;
}

const countGC = (fn: () => void): number => {
  let runs = 0;
  const obs = new PerformanceObserver((list) => {
    runs += list.getEntries().length;
  });
  obs.observe({ entryTypes: ["gc"] });
  fn();
  obs.disconnect();
  return runs;
};

/**
 * Measures heap bytes allocated per call of fn. Requires --expose-gc; returns NaN
 * bytes otherwise. Halves the iteration count until a run completes without a GC
 * cycle, so the heapUsed delta counts every allocation instead of survivors only.
 */
export const measureAllocs = (fn: () => void, iters = 512): AllocMeasurement => {
  if (gc == null) return { bytesPerOp: NaN, iterations: 0 };
  for (let i = 0; i < 32; i++) fn();
  let n = iters;
  while (n >= 4) {
    gc();
    gc();
    const before = process.memoryUsage().heapUsed;
    let after = 0;
    const count = n;
    const gcRuns = countGC(() => {
      for (let i = 0; i < count; i++) fn();
      after = process.memoryUsage().heapUsed;
    });
    if (gcRuns === 0) return { bytesPerOp: (after - before) / n, iterations: n };
    n = Math.floor(n / 2);
  }
  return { bytesPerOp: NaN, iterations: 0 };
};

/** Async variant of {@link measureAllocs}. Timer and promise machinery pollute the
 * numbers slightly; use for coarse comparisons only. */
export const measureAllocsAsync = async (
  fn: () => Promise<void>,
  iters = 64,
): Promise<AllocMeasurement> => {
  if (gc == null) return { bytesPerOp: NaN, iterations: 0 };
  for (let i = 0; i < 8; i++) await fn();
  gc();
  gc();
  const before = process.memoryUsage().heapUsed;
  for (let i = 0; i < iters; i++) await fn();
  const after = process.memoryUsage().heapUsed;
  return { bytesPerOp: (after - before) / iters, iterations: iters };
};

interface AllocRow {
  case: string;
  "bytes/op": number | string;
  iters: number;
}

const toRow = (name: string, m: AllocMeasurement): AllocRow => ({
  case: name,
  "bytes/op": Number.isNaN(m.bytesPerOp)
    ? "n/a (gc unavailable)"
    : Math.round(m.bytesPerOp),
  iters: m.iterations,
});

/** Measures and prints an allocation table for the given cases. */
export const allocSuite = (title: string, cases: [string, () => void][]): void => {
  const rows = cases.map(([name, fn]) => toRow(name, measureAllocs(fn)));
  console.log(`\nallocations: ${title}`);
  console.table(rows);
};

/** Async variant of {@link allocSuite}. */
export const allocSuiteAsync = async (
  title: string,
  cases: [string, () => Promise<void>][],
  iters?: number,
): Promise<void> => {
  const rows: AllocRow[] = [];
  for (const [name, fn] of cases)
    rows.push(toRow(name, await measureAllocsAsync(fn, iters)));
  console.log(`\nallocations: ${title}`);
  console.table(rows);
};
