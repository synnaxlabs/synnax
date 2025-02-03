// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds } from "@synnaxlabs/x/spatial";
import {
  type CrudeTimeRange,
  DataType,
  Series,
  TimeRange,
  TimeSpan,
} from "@synnaxlabs/x/telem";
import { describe, expect, it } from "vitest";

import { buildDrawOperations, type DrawOperation } from "@/vis/line/aether/line";

describe("line", () => {
  describe("buildDrawOperations", () => {
    interface SpecEntry {
      timeRange: CrudeTimeRange;
      alignmentBounds: bounds.Bounds<bigint>;
    }

    interface SpecExpected {
      xSeries: number;
      ySeries: number;
      xOffset: number;
      yOffset: number;
      count: number;
    }

    interface Spec {
      name: string;
      x: SpecEntry[];
      y: SpecEntry[];
      expected: SpecExpected[];
    }

    const buildSeriesFromEntries = (entries: SpecEntry[]): Series[] =>
      entries.map(
        ({ alignmentBounds, timeRange }) =>
          new Series({
            data: new Float32Array(
              Number(alignmentBounds.upper - alignmentBounds.lower),
            ),
            dataType: DataType.FLOAT32,
            timeRange: new TimeRange(timeRange.start, timeRange.end),
            alignment: alignmentBounds.lower,
          }),
      );

    const CLEARLY_DISTINCT: Spec = {
      name: "clearly distinct",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 100n },
        },
        {
          timeRange: { start: 100, end: 200 },
          alignmentBounds: { lower: 100n, upper: 200n },
        },
      ],
      y: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 100n },
        },
        {
          timeRange: { start: 100, end: 200 },
          alignmentBounds: { lower: 100n, upper: 200n },
        },
      ],
      expected: [
        { xSeries: 0, ySeries: 0, xOffset: 0, yOffset: 0, count: 100 },
        { xSeries: 1, ySeries: 1, xOffset: 0, yOffset: 0, count: 100 },
      ],
    };

    const COMPLETE_OVERLAP_ON_X: Spec = {
      name: "complete overlap on x",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 100n },
        },
      ],
      y: [
        {
          timeRange: { start: 0, end: 50 },
          alignmentBounds: { lower: 0n, upper: 50n },
        },
        {
          timeRange: { start: 50, end: 100 },
          alignmentBounds: { lower: 50n, upper: 100n },
        },
      ],
      expected: [
        { ySeries: 0, xSeries: 0, xOffset: 0, yOffset: 0, count: 50 },
        { ySeries: 1, xSeries: 0, xOffset: 50, yOffset: 0, count: 50 },
      ],
    };

    const PARTIAL_OVERLAP_ON_Y_AFTER: Spec = {
      name: "partial overlap on x",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 100n },
        },
      ],
      y: [
        {
          timeRange: { start: 50, end: 150 },
          alignmentBounds: { lower: 50n, upper: 150n },
        },
      ],
      expected: [{ xSeries: 0, ySeries: 0, xOffset: 50, yOffset: 0, count: 50 }],
    };

    const Y_COMPLETELY_CONTAINS_X: Spec = {
      name: "y completely contains x",
      x: [
        {
          timeRange: { start: 50, end: 100 },
          alignmentBounds: { lower: 50n, upper: 100n },
        },
      ],
      y: [
        {
          timeRange: { start: 0, end: 200 },
          alignmentBounds: { lower: 0n, upper: 200n },
        },
      ],
      expected: [{ xSeries: 0, ySeries: 0, xOffset: 0, yOffset: 50, count: 50 }],
    };

    const X_COMPLETELY_CONTAINS_Y: Spec = {
      name: "x completely contains y",
      x: [
        {
          timeRange: { start: 0, end: 200 },
          alignmentBounds: { lower: 0n, upper: 200n },
        },
      ],
      y: [
        {
          timeRange: { start: 50, end: 100 },
          alignmentBounds: { lower: 50n, upper: 100n },
        },
      ],
      expected: [{ xSeries: 0, ySeries: 0, xOffset: 50, yOffset: 0, count: 50 }],
    };

    const MULTIPLE_PARTIAL_OVERLAPS: Spec = {
      name: "multiple partial overlaps",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 100n },
        },
        {
          timeRange: { start: 100, end: 150 },
          alignmentBounds: { lower: 100n, upper: 150n },
        },
      ],
      y: [
        {
          timeRange: { start: 25, end: 75 },
          alignmentBounds: { lower: 25n, upper: 75n },
        },
        {
          timeRange: { start: 75, end: 125 },
          alignmentBounds: { lower: 75n, upper: 125n },
        },
        {
          timeRange: { start: 125, end: 175 },
          alignmentBounds: { lower: 125n, upper: 175n },
        },
      ],
      expected: [
        { xSeries: 0, ySeries: 0, xOffset: 25, yOffset: 0, count: 50 },
        { xSeries: 0, ySeries: 1, xOffset: 75, yOffset: 0, count: 25 },
        { xSeries: 1, ySeries: 1, xOffset: 0, yOffset: 25, count: 25 },
        { xSeries: 1, ySeries: 2, xOffset: 25, yOffset: 0, count: 25 },
      ],
    };

    const ALIGN_OVERLAP_TIME_RANGE_NO_OVERLAP: Spec = {
      name: "align overlap time range no overlap",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 100n },
        },
      ],
      y: [
        {
          timeRange: { start: 100, end: 150 },
          alignmentBounds: { lower: 50n, upper: 150n },
        },
      ],
      expected: [],
    };

    const SPECS: Spec[] = [
      CLEARLY_DISTINCT,
      COMPLETE_OVERLAP_ON_X,
      PARTIAL_OVERLAP_ON_Y_AFTER,
      Y_COMPLETELY_CONTAINS_X,
      X_COMPLETELY_CONTAINS_Y,
      MULTIPLE_PARTIAL_OVERLAPS,
      ALIGN_OVERLAP_TIME_RANGE_NO_OVERLAP,
    ];

    SPECS.forEach(({ name, x, y, expected }) => {
      it(`spec ${name}`, () => {
        const xSeries = buildSeriesFromEntries(x);
        const ySeries = buildSeriesFromEntries(y);
        const downSample = 1;
        const drawOperations = buildDrawOperations(
          xSeries,
          ySeries,
          downSample,
          0,
          TimeSpan.ZERO,
        );
        expect(drawOperations.length).toBe(expected.length);
        drawOperations.forEach((drawOperation: DrawOperation, i: number) => {
          expect(drawOperation.x).toBe(xSeries[expected[i].xSeries]);
          expect(drawOperation.y).toBe(ySeries[expected[i].ySeries]);
          expect(drawOperation.xOffset).toBe(expected[i].xOffset);
          expect(drawOperation.yOffset).toBe(expected[i].yOffset);
        });
      });
    });
  });
});
