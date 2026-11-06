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
    { name: "on the line", jump: { x: 50, y: 50, radius: 5 } },
    { name: "a few pixels off the line", jump: { x: 50, y: 54, radius: 5 } },
  ])("hops up over a jump $name on a horizontal run", ({ jump }) => {
    expect(Path.rounded(HORIZONTAL, [jump])).toContain(
      "L43,50Q45,50 45,48A5,5 0 0 1 55,48Q55,50 57,50L100,50",
    );
  });

  it("draws a hop at the crossing's radius", () => {
    expect(Path.rounded(HORIZONTAL, [{ x: 50, y: 50, radius: 8 }])).toContain(
      "L40,50Q42,50 42,48A8,8 0 0 1 58,48Q58,50 60,50L100,50",
    );
  });

  it("keeps wide hops from overlapping on a run", () => {
    const path = Path.rounded(HORIZONTAL, [
      { x: 40, y: 50, radius: 8 },
      { x: 55, y: 50, radius: 8 },
    ]);
    expect(path.match(/A/g)).toHaveLength(1);
  });

  it("hops over a crossing past the snap distance with a snap margin", () => {
    const crossing = { x: 50, y: 74, radius: 5 };
    expect(Path.rounded(HORIZONTAL, [crossing])).toEqual("M0,50L100,50");
    expect(Path.rounded(HORIZONTAL, [crossing], 10)).toContain("A5,5");
  });

  it.each([
    { name: "start", x: 12 },
    { name: "end", x: 88 },
  ])("shrinks a wide hop to fit near the run's $name", ({ x }) => {
    expect(Path.rounded(HORIZONTAL, [{ x, y: 50, radius: 12 }])).toContain("A10,10");
  });

  it("shrinks a wide hop to clear the hop before it", () => {
    const path = Path.rounded(HORIZONTAL, [
      { x: 16, y: 50, radius: 23 },
      { x: 50, y: 50, radius: 23 },
    ]);
    expect(path).toContain("A14,14");
    expect(path).toContain("A16,16");
  });

  it("ignores a wide hop with less room than a default hop", () => {
    expect(Path.rounded(HORIZONTAL, [{ x: 6, y: 50, radius: 12 }])).toEqual(
      "M0,50L100,50",
    );
  });

  it("hops left over a jump on a vertical run", () => {
    expect(
      Path.rounded(
        [
          { x: 0, y: 0 },
          { x: 0, y: 100 },
        ],
        [{ x: 0, y: 50, radius: 5 }],
      ),
    ).toContain("L0,43Q0,45 -2,45A5,5 0 0 0 -2,55Q0,55 0,57L0,100");
  });

  it.each([
    { name: "is not on the path", jump: { x: 50, y: 50, radius: 5 } },
    { name: "is too close to a run's end", jump: { x: 2, y: 0, radius: 5 } },
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
