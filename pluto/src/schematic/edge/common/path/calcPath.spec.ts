// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { calcPath } from "@/schematic/edge/common/path/calcPath";

describe("calcPath", () => {
  it("draws a straight segment", () => {
    expect(
      calcPath([
        { x: 0, y: 0 },
        { x: 100, y: 0 },
      ]),
    ).toEqual("M0,0L100,0");
  });

  it("rounds corners with a quadratic curve", () => {
    expect(
      calcPath([
        { x: 0, y: 0 },
        { x: 100, y: 0 },
        { x: 100, y: 100 },
      ]),
    ).toEqual("M0,0L94,0Q100,0 100,6L100,100");
  });

  it("draws a hop arc over a jump point on a straight run", () => {
    const path = calcPath(
      [
        { x: 0, y: 50 },
        { x: 100, y: 50 },
      ],
      [{ x: 50, y: 50 }],
    );
    // approach, foot fillet, lifted semicircle, foot fillet, then the rest of the run.
    expect(path).toContain("L43,50Q45,50 45,48A5,5 0 0 1 55,48Q55,50 57,50L100,50");
  });

  it("snaps a jump that sits a few pixels off the drawn line onto the run", () => {
    const path = calcPath(
      [
        { x: 0, y: 50 },
        { x: 100, y: 50 },
      ],
      [{ x: 50, y: 54 }],
    );
    // the off-line jump still snaps onto the run, hopping at x=50.
    expect(path).toContain("L43,50Q45,50 45,48A5,5 0 0 1 55,48Q55,50 57,50L100,50");
  });

  it("ignores a jump point that is not on the path", () => {
    expect(
      calcPath(
        [
          { x: 0, y: 50 },
          { x: 100, y: 50 },
        ],
        [{ x: 50, y: 200 }],
      ),
    ).toEqual("M0,50L100,50");
  });

  it("ignores jumps too close to a run's end", () => {
    expect(
      calcPath(
        [
          { x: 0, y: 0 },
          { x: 100, y: 0 },
        ],
        [{ x: 2, y: 0 }],
      ),
    ).toEqual("M0,0L100,0");
  });
});
