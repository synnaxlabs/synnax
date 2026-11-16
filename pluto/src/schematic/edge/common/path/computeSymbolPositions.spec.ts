// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Path } from "@/schematic/edge/common/path";

const positions = (points: xy.XY[], interval: number, clearance?: number): xy.XY[] =>
  Path.computeSymbolPositions(points, interval, clearance).map((p) => p.position);

describe("Path.computeSymbolPositions", () => {
  it("places a symbol every interval", () => {
    expect(
      Path.computeSymbolPositions(
        [
          { x: 0, y: 0 },
          { x: 100, y: 0 },
        ],
        40,
      ),
    ).toEqual([
      { position: { x: 40, y: 0 }, direction: "x" },
      { position: { x: 80, y: 0 }, direction: "x" },
    ]);
  });

  it("drops a symbol within the default clearance of a corner", () => {
    expect(
      positions(
        [
          { x: 0, y: 0 },
          { x: 42, y: 0 },
          { x: 42, y: 100 },
        ],
        40,
      ),
    ).toEqual([
      { x: 42, y: 38 },
      { x: 42, y: 78 },
    ]);
  });

  it("moves a symbol near a corner out to the clearance", () => {
    expect(
      positions(
        [
          { x: 0, y: 0 },
          { x: 200, y: 0 },
          { x: 200, y: 200 },
        ],
        120,
        50,
      ),
    ).toEqual([
      { x: 120, y: 0 },
      { x: 200, y: 50 },
      { x: 200, y: 150 },
    ]);
  });

  it("drops a symbol on a segment shorter than twice the clearance", () => {
    expect(
      positions(
        [
          { x: 0, y: 0 },
          { x: 80, y: 0 },
        ],
        40,
        50,
      ),
    ).toEqual([]);
  });

  it("drops a symbol that would move within half an interval of the last", () => {
    expect(
      positions(
        [
          { x: 0, y: 0 },
          { x: 75, y: 0 },
          { x: 75, y: 200 },
        ],
        40,
        30,
      ),
    ).toEqual([
      { x: 40, y: 0 },
      { x: 75, y: 30 },
      { x: 75, y: 85 },
      { x: 75, y: 125 },
      { x: 75, y: 165 },
    ]);
  });

  const ROUTE: xy.XY[] = [
    { x: 0, y: 0 },
    { x: 130, y: 0 },
    { x: 130, y: 90 },
    { x: 410, y: 90 },
    { x: 410, y: 260 },
    { x: 600, y: 260 },
  ];

  it.each([
    { interval: 40, clearance: 5 },
    { interval: 80, clearance: 29 },
    { interval: 126, clearance: 77 },
  ])(
    "keeps every symbol $clearance px from each corner and end",
    ({ interval, clearance }) => {
      const placed = positions(ROUTE, interval, clearance);
      expect(placed.length).toBeGreaterThan(0);
      placed.forEach((p) =>
        ROUTE.forEach((corner) =>
          expect(xy.distance(p, corner)).toBeGreaterThanOrEqual(clearance),
        ),
      );
    },
  );
});
