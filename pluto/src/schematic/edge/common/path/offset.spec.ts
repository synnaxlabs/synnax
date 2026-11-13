// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Path } from "@/schematic/edge/common/path";

const expectPoints = (actual: xy.XY[], expected: xy.XY[]): void => {
  expect(actual).toHaveLength(expected.length);
  actual.forEach((p, i) => {
    expect(p.x).toBeCloseTo(expected[i].x);
    expect(p.y).toBeCloseTo(expected[i].y);
  });
};

describe("Path.offset", () => {
  it("draws a parallel run at the distance", () => {
    expectPoints(
      Path.offset(
        [
          { x: 0, y: 0 },
          { x: 100, y: 0 },
        ],
        6,
      ),
      [
        { x: 0, y: 6 },
        { x: 100, y: 6 },
      ],
    );
  });

  it("draws on the other side at a negative distance", () => {
    expectPoints(
      Path.offset(
        [
          { x: 0, y: 0 },
          { x: 100, y: 0 },
        ],
        -6,
      ),
      [
        { x: 0, y: -6 },
        { x: 100, y: -6 },
      ],
    );
  });

  it("keeps the distance from both runs at a corner", () => {
    expectPoints(
      Path.offset(
        [
          { x: 0, y: 0 },
          { x: 100, y: 0 },
          { x: 100, y: 100 },
        ],
        6,
      ),
      [
        { x: 0, y: 6 },
        { x: 94, y: 6 },
        { x: 94, y: 100 },
      ],
    );
  });

  it("drops a point that does not turn the line", () => {
    expectPoints(
      Path.offset(
        [
          { x: 0, y: 0 },
          { x: 50, y: 0 },
          { x: 100, y: 0 },
        ],
        6,
      ),
      [
        { x: 0, y: 6 },
        { x: 100, y: 6 },
      ],
    );
  });

  it("draws no spike where the line doubles back", () => {
    const points = Path.offset(
      [
        { x: 0, y: 4 },
        { x: 0, y: -10 },
        { x: 0, y: 100 },
        { x: 100, y: 100 },
      ],
      6,
    );
    expectPoints(points, [
      { x: -6, y: 4 },
      { x: -6, y: 106 },
      { x: 100, y: 106 },
    ]);
  });

  it("keeps an end run longer than the distance", () => {
    expectPoints(
      Path.offset(
        [
          { x: 0, y: 0 },
          { x: 0, y: -14 },
          { x: 100, y: -14 },
        ],
        6,
      ),
      [
        { x: 6, y: 0 },
        { x: 6, y: -8 },
        { x: 100, y: -8 },
      ],
    );
  });

  it.each([
    {
      name: "a start",
      distance: 24,
      points: [
        { x: 0, y: 0 },
        { x: 0, y: -14 },
        { x: 100, y: -14 },
      ],
      expected: [
        { x: 24, y: 10 },
        { x: 100, y: 10 },
      ],
    },
    {
      name: "an end",
      distance: -24,
      points: [
        { x: 100, y: -14 },
        { x: 0, y: -14 },
        { x: 0, y: 0 },
      ],
      expected: [
        { x: 100, y: 10 },
        { x: 24, y: 10 },
      ],
    },
  ])(
    "cuts off $name run shorter than the distance",
    ({ points, distance, expected }) => {
      expectPoints(Path.offset(points, distance), expected);
    },
  );

  it("returns no points for a line with no length", () => {
    expect(
      Path.offset(
        [
          { x: 5, y: 5 },
          { x: 5, y: 5 },
        ],
        6,
      ),
    ).toEqual([]);
  });
});
