// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type bounds,
  type MultiSeries,
  type scale,
  type Series,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { forEachPair } from "@/vis/line/aether/bounds";

export const DEFAULT_OVERLAP_THRESHOLD = TimeSpan.milliseconds(2);

export interface DrawOperation {
  x: Series;
  y: Series;
  xOffset: number;
  yOffset: number;
  count: number;
  /** The time span the op draws: where its x and y series overlap. */
  span: TimeRange;
}

export const buildDrawOperations = (
  xSeries: MultiSeries,
  ySeries: MultiSeries,
  overlapThreshold: TimeSpan,
): DrawOperation[] => {
  const ops: DrawOperation[] = [];
  forEachPair(xSeries.series, ySeries.series, overlapThreshold, (x, y, start, end) => {
    const mult = x.alignmentMultiple;
    ops.push({
      x,
      y,
      xOffset: Number((start - x.alignment) / mult),
      yOffset: Number((start - y.alignment) / mult),
      count: Number((end - start) / mult),
      span: new TimeRange(
        TimeStamp.max(x.timeRange.start, y.timeRange.start),
        TimeStamp.min(x.timeRange.end, y.timeRange.end),
      ),
    });
  });
  return ops;
};

/**
 * @returns the [first, count] vertex ranges of op that hold only finite x and y
 * samples. A range with one vertex draws no segment, so it is left out.
 */
export const finiteRuns = (op: DrawOperation): Array<[number, number]> => {
  const total = op.count;
  const gaps = [
    ...nonFiniteVertices(op.x, op.xOffset, total),
    ...nonFiniteVertices(op.y, op.yOffset, total),
  ].sort((a, b) => a.lower - b.lower);
  const runs: Array<[number, number]> = [];
  let first = 0;
  for (const { lower, upper } of gaps) {
    if (lower - first > 1) runs.push([first, lower - first]);
    first = Math.max(first, upper);
  }
  if (total - first > 1) runs.push([first, total - first]);
  return runs;
};

const nonFiniteVertices = (
  series: Series,
  offset: number,
  total: number,
): bounds.Bounds[] => {
  const vertices: bounds.Bounds[] = [];
  for (const { lower, upper } of series.nonFiniteRanges) {
    const v = {
      lower: Math.max(0, lower - offset),
      upper: Math.min(total, upper - offset),
    };
    if (v.lower >= total) break;
    if (v.lower < v.upper) vertices.push(v);
  }
  return vertices;
};

/** @returns true when b starts in time where a ends, so a line may join them. */
export const touching = (a: DrawOperation, b: DrawOperation): boolean =>
  a.span.end.equals(b.span.start);

/**
 * @returns interleaved decimal x and y of the segments joining each op to the next in
 * time, leaving out any segment with a non-finite end. Only ops that touch in time
 * join. Time between two ops holds no samples, and ops that overlap hold the same span
 * twice, so a joining segment would run back in time.
 */
export const bridgeVertices = (ops: DrawOperation[], s: scale.XY): Float32Array => {
  const vertices: number[] = [];
  const sorted = ops.toSorted((a, b) => TimeStamp.sort(a.span.start, b.span.start));
  for (let i = 1; i < sorted.length; i++) {
    const a = sorted[i - 1];
    const b = sorted[i];
    if (!touching(a, b)) continue;
    const last = Math.max(a.count - 1, 0);
    const segment = [
      s.x.pos(Number(a.x.at(a.xOffset + last, true))),
      s.y.pos(Number(a.y.at(a.yOffset + last, true))),
      s.x.pos(Number(b.x.at(b.xOffset, true))),
      s.y.pos(Number(b.y.at(b.yOffset, true))),
    ];
    if (segment.every(Number.isFinite)) vertices.push(...segment);
  }
  return new Float32Array(vertices);
};
