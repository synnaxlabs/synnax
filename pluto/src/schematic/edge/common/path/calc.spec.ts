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

const HORIZONTAL: xy.XY[] = [
  { x: 0, y: 50 },
  { x: 100, y: 50 },
];

describe("Path.rounded", () => {
  it("draws a straight segment", () => {
    expect(Path.rounded(HORIZONTAL)).toEqual("M0,50L100,50");
  });

  it("rounds corners with a quadratic curve", () => {
    expect(
      Path.rounded([
        { x: 0, y: 0 },
        { x: 100, y: 0 },
        { x: 100, y: 100 },
      ]),
    ).toEqual("M0,0L94,0Q100,0 100,6L100,100");
  });

  // A jump exactly on the line and one a few pixels off both hop at the same place: the
  // off-line jump snaps onto the run (the polyline is reconstructed from the flow store
  // and can sit slightly off the drawn line).
  it.each([
    { name: "on the line", jump: { x: 50, y: 50 } },
    { name: "a few pixels off the line", jump: { x: 50, y: 54 } },
  ])("hops up over a jump $name on a horizontal run", ({ jump }) => {
    expect(Path.rounded(HORIZONTAL, [jump])).toContain(
      "L43,50Q45,50 45,48A5,5 0 0 1 55,48Q55,50 57,50L100,50",
    );
  });

  it("hops left over a jump on a vertical run", () => {
    expect(
      Path.rounded(
        [
          { x: 0, y: 0 },
          { x: 0, y: 100 },
        ],
        [{ x: 0, y: 50 }],
      ),
    ).toContain("L0,43Q0,45 -2,45A5,5 0 0 0 -2,55Q0,55 0,57L0,100");
  });

  it.each([
    { name: "is not on the path", jump: { x: 50, y: 50 } },
    { name: "is too close to a run's end", jump: { x: 2, y: 0 } },
  ])("ignores a jump that $name", ({ jump }) => {
    expect(
      Path.rounded(
        [
          { x: 0, y: 0 },
          { x: 100, y: 0 },
        ],
        [jump],
      ),
    ).toEqual("M0,0L100,0");
  });
});
