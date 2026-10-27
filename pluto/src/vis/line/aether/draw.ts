// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  bounds,
  math,
  type MultiSeries,
  type scale,
  type Series,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { seriesOverlap } from "@/vis/line/aether/bounds";

export const DEFAULT_OVERLAP_THRESHOLD = TimeSpan.milliseconds(2);

export interface DrawOperation {
  x: Series;
  y: Series;
  xOffset: number;
  yOffset: number;
  count: number;
}

export const buildDrawOperations = (
  xSeries: MultiSeries,
  ySeries: MultiSeries,
  overlapThreshold: TimeSpan,
): DrawOperation[] => {
  if (xSeries.series.length === 0 || ySeries.series.length === 0) return [];
  const ops: DrawOperation[] = [];
  xSeries.series.forEach((x) =>
    ySeries.series.forEach((y) => {
      if (!seriesOverlap(x, y, overlapThreshold)) return;
      let xAlignmentOffset = 0n;
      let yAlignmentOffset = 0n;
      // This means that the x series starts before the y series.
      if (x.alignment < y.alignment) xAlignmentOffset = y.alignment - x.alignment;
      // This means that the y series starts before the x series.
      else if (y.alignment < x.alignment) yAlignmentOffset = x.alignment - y.alignment;
      // The total number of alignment steps that are common to the two series.
      const alignmentCount = math.min(
        bounds.span(x.alignmentBounds) - xAlignmentOffset,
        bounds.span(y.alignmentBounds) - yAlignmentOffset,
      );
      if (alignmentCount === 0n) return;
      const count = Number(alignmentCount / x.alignmentMultiple);
      const xOffset = Number(xAlignmentOffset / x.alignmentMultiple);
      const yOffset = Number(yAlignmentOffset / y.alignmentMultiple);
      ops.push({ x, y, xOffset, yOffset, count });
    }),
  );
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

/** @returns the time span an op draws: where its x and y series overlap. */
const opSpan = ({ x, y }: DrawOperation): TimeRange =>
  new TimeRange(
    TimeStamp.max(x.timeRange.start, y.timeRange.start),
    TimeStamp.min(x.timeRange.end, y.timeRange.end),
  );

/**
 * @returns interleaved decimal x and y of the segments joining each op to the next in
 * time, leaving out any segment with a non-finite end. Ops whose time spans overlap
 * are not joined: they hold the same span twice, so a joining segment would run back in
 * time.
 */
export const bridgeVertices = (ops: DrawOperation[], s: scale.XY): Float32Array => {
  const vertices: number[] = [];
  const spans = ops
    .map((op) => ({ op, span: opSpan(op) }))
    .sort((a, b) => TimeStamp.sort(a.span.start, b.span.start));
  for (let i = 1; i < spans.length; i++) {
    const { op: a, span: aSpan } = spans[i - 1];
    const { op: b, span: bSpan } = spans[i];
    if (bSpan.start.before(aSpan.end)) continue;
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
