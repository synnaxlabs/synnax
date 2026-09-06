// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DataType,
  MultiSeries,
  Series,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { afterAll, bench, type BenchOptions, describe } from "vitest";

import { GL_TRANSFORM } from "@/telem/aether/convertSeries";
import { windowBounds } from "@/vis/line/aether/bounds";
import { buildDrawOperations, DEFAULT_OVERLAP_THRESHOLD } from "@/vis/line/aether/line";

// Measures back-fill fetch speed under render-loop contention and the per-frame cost
// the plot pays once the back-fill lands.

// Samples per iterator response (client framer/iterator.ts).
const CHUNK = 1e5;
const FRAME_INTERVAL_MS = 1000 / 60;
// Cells above MAX_CHUNKS are dropped since run cost grows quadratically with chunks.
const MAX_CHUNKS = 2600;

interface Spec {
  name: string;
  rate: number;
  span: TimeSpan;
  channels: number;
  sharedIndex: boolean;
}

const RATES: [string, number][] = [
  ["1Hz", 1],
  ["100Hz", 100],
  ["1kHz", 1e3],
];

const SPANS: [string, TimeSpan][] = [
  ["1d", TimeSpan.days(1)],
  ["7d", TimeSpan.days(7)],
  ["30d", TimeSpan.days(30)],
  ["1y", TimeSpan.days(365)],
];

const chunkCount = ({ rate, span }: Pick<Spec, "rate" | "span">): number =>
  Math.ceil((rate * span.seconds) / CHUNK);

const MATRIX: Spec[] = RATES.flatMap(([rateName, rate]) =>
  SPANS.map(([spanName, span]) => ({
    name: `${rateName} ${spanName}`,
    rate,
    span,
    channels: 1,
    sharedIndex: true,
  })),
).filter((s) => chunkCount(s) <= MAX_CHUNKS);

// Channel scaling is measured on one small cell to keep the suite bounded.
const CHANNEL_SPECS: Spec[] = [2, 5, 10].flatMap((channels) =>
  [true, false].map((sharedIndex) => ({
    name: `100Hz 1d ${channels}ch ${sharedIndex ? "shared index" : "separate indexes"}`,
    rate: 100,
    span: TimeSpan.days(1),
    channels,
    sharedIndex,
  })),
);

const fillRamp = (x: BigInt64Array, y: Float32Array, period: bigint): void => {
  for (let i = 0; i < x.length; i++) {
    x[i] = BigInt(i) * period;
    y[i] = i % 1000;
  }
};

interface Chunks {
  x: Series[];
  ys: Series[][];
}

const buildChunks = ({ rate, span, channels }: Spec): Chunks => {
  const total = Math.round(rate * span.seconds);
  const period = BigInt(Math.round(1e9 / rate));
  interface Template {
    x: BigInt64Array<ArrayBuffer>;
    y: Float32Array<ArrayBuffer>;
  }
  const templates = new Map<number, Template>();
  const template = (len: number): Template => {
    let t = templates.get(len);
    if (t == null) {
      const x = new BigInt64Array(new ArrayBuffer(len * 8));
      const y = new Float32Array(new ArrayBuffer(len * 4));
      fillRamp(x, y, period);
      t = { x, y };
      templates.set(len, t);
    }
    return t;
  };
  const x: Series[] = [];
  const ys: Series[][] = Array.from({ length: channels }, () => []);
  for (let i = 0; i < total; i += CHUNK) {
    const len = Math.min(CHUNK, total - i);
    const t = template(len);
    const start = period * BigInt(i);
    const timeRange = new TimeRange(
      new TimeStamp(start),
      new TimeStamp(start + period * BigInt(len)),
    );
    const alignment = BigInt(i);
    x.push(
      new Series({ data: t.x, dataType: DataType.TIMESTAMP, timeRange, alignment }),
    );
    for (const y of ys)
      y.push(
        new Series({ data: t.y, dataType: DataType.FLOAT32, timeRange, alignment }),
      );
  }
  return { x, ys };
};

// One data-path frame per line, omitting GL upload, so slowdowns are a lower bound.
const frame = (x: MultiSeries, y: MultiSeries): void => {
  windowBounds(x, y, x.bounds, DEFAULT_OVERLAP_THRESHOLD, y.bounds);
  buildDrawOperations(x, y, 1, 0, "decimate", DEFAULT_OVERLAP_THRESHOLD);
};

// Drain frames walk only the short live window that is on screen during back-fill.
const buildLive = (): { x: MultiSeries; y: MultiSeries } => {
  const len = 2000;
  const period = BigInt(1e7);
  const xData = new BigInt64Array(new ArrayBuffer(len * 8));
  const yData = new Float32Array(new ArrayBuffer(len * 4));
  fillRamp(xData, yData, period);
  const x: Series[] = [];
  const y: Series[] = [];
  for (let i = 0; i < 3; i++) {
    const start = period * BigInt(i * len);
    const timeRange = new TimeRange(
      new TimeStamp(start),
      new TimeStamp(start + period * BigInt(len)),
    );
    const alignment = BigInt(i * len);
    x.push(
      GL_TRANSFORM.convert(
        new Series({ data: xData, dataType: DataType.TIMESTAMP, timeRange, alignment }),
      ),
    );
    y.push(
      new Series({ data: yData, dataType: DataType.FLOAT32, timeRange, alignment }),
    );
  }
  return { x: new MultiSeries(x), y: new MultiSeries(y) };
};

const LIVE = buildLive();

interface Landed {
  xs: MultiSeries[];
  ys: MultiSeries[];
}

// The real reader stages the whole drain and lands it in the cache as one batch.
const drain = (chunks: Chunks, spec: Spec, rendering: boolean): Landed => {
  const { channels, sharedIndex } = spec;
  const stagedX: Series[][] = Array.from({ length: channels }, () => []);
  const stagedY: Series[][] = Array.from({ length: channels }, () => []);
  let last = performance.now();
  for (let i = 0; i < chunks.x.length; i++) {
    const shared = sharedIndex ? GL_TRANSFORM.convert(chunks.x[i]) : null;
    for (let ch = 0; ch < channels; ch++) {
      stagedX[ch].push(shared ?? GL_TRANSFORM.convert(chunks.x[i]));
      stagedY[ch].push(GL_TRANSFORM.convert(chunks.ys[ch][i]));
    }
    if (rendering && performance.now() - last >= FRAME_INTERVAL_MS) {
      frame(LIVE.x, LIVE.y);
      last = performance.now();
    }
  }
  return {
    xs: stagedX.map((s) => new MultiSeries(s)),
    ys: stagedY.map((s) => new MultiSeries(s)),
  };
};

// Weight approximates timestamp conversions, the dominant per-chunk fetch cost.
const opts = (spec: Spec): BenchOptions | undefined => {
  const weight = chunkCount(spec) * (spec.sharedIndex ? 1 : spec.channels);
  if (weight > 300)
    return { time: 0, iterations: 1, warmupTime: 0, warmupIterations: 0 };
  if (weight > 20)
    return { time: 0, iterations: 3, warmupTime: 0, warmupIterations: 1 };
  return undefined;
};

for (const spec of [...MATRIX, ...CHANNEL_SPECS]) {
  const chunks = buildChunks(spec);
  const o = opts(spec);
  describe(`${spec.name} (${chunkCount(spec)} chunks)`, () => {
    // The drain benches keep their landed result so frame after load reuses it.
    let landed: Landed | null = null;
    afterAll(() => {
      landed = null;
    });
    const exec = (rendering: boolean): void => {
      landed = drain(chunks, spec, rendering);
    };
    bench("fetch only", () => exec(false), o);
    bench("fetch during render loop", () => exec(true), o);
    // One frame over the landed back-fill, the recurring cost the plot pays after.
    bench(
      "frame after load",
      () => {
        if (landed == null) throw new Error("no landed back-fill");
        for (let ch = 0; ch < spec.channels; ch++) frame(landed.xs[ch], landed.ys[ch]);
      },
      {
        ...o,
        setup: () => {
          landed ??= drain(chunks, spec, false);
        },
      },
    );
  });
}
