// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Jumps } from "@/schematic/edge/common/jumps";

const horizontal = (
  key: string,
  order: number,
  y = 50,
  strokeWidth = 2,
  width = strokeWidth,
): Jumps.Polyline => ({
  key,
  order,
  strokeWidth,
  width,
  points: [
    { x: 0, y },
    { x: 100, y },
  ],
});

const vertical = (
  key: string,
  order: number,
  x = 50,
  fromY = 0,
  toY = 100,
  strokeWidth = 2,
  width = strokeWidth,
): Jumps.Polyline => ({
  key,
  order,
  strokeWidth,
  width,
  points: [
    { x, y: fromY },
    { x, y: toY },
  ],
});

describe("findCrossings", () => {
  it.each([
    { name: "vertical edge on top", h: 0, v: 1, winner: "v", loser: "h" },
    { name: "horizontal edge on top", h: 5, v: 1, winner: "h", loser: "v" },
  ])("assigns a crossing to the edge on top ($name)", ({ h, v, winner, loser }) => {
    const result = Jumps.findCrossings([horizontal("h", h), vertical("v", v)]);
    expect(result.get(winner)).toEqual([{ x: 50, y: 50, radius: 5 }]);
    expect(result.has(loser)).toBe(false);
  });

  it.each<{ name: string; polylines: Jumps.Polyline[] }>([
    {
      name: "T-junction where one segment ends on the other",
      polylines: [horizontal("a", 0), vertical("b", 1, 50, 50, 100)],
    },
    {
      name: "crossing within the hop segment's end buffer",
      polylines: [horizontal("a", 0), vertical("b", 1, 50, 45, 100)],
    },
    {
      name: "parallel segments that never form an x/y pair",
      polylines: [horizontal("a", 0, 50), horizontal("b", 1, 50)],
    },
    {
      name: "a single edge crossing itself",
      polylines: [
        {
          key: "a",
          order: 0,
          strokeWidth: 2,
          width: 2,
          points: [
            { x: 0, y: 50 },
            { x: 50, y: 50 },
            { x: 50, y: 100 },
          ],
        },
      ],
    },
  ])("produces no hop for $name", ({ polylines }) => {
    expect(Jumps.findCrossings(polylines).size).toBe(0);
  });

  it("keeps a hop when the crossing nears the crossed segment's end", () => {
    // The horizontal edge is on top and draws the hop; the crossed vertical edge ends
    // only 5px from the crossing, but only the small cross-buffer applies there.
    const result = Jumps.findCrossings([
      horizontal("h", 5),
      vertical("v", 1, 50, 45, 100),
    ]);
    expect(result.get("h")).toEqual([{ x: 50, y: 50, radius: 5 }]);
  });

  it("records a separate crossing per vertical edge over a shared horizontal", () => {
    const result = Jumps.findCrossings([
      horizontal("a", 0),
      vertical("b", 1, 30),
      vertical("c", 2, 70),
    ]);
    expect(result.get("b")).toEqual([{ x: 30, y: 50, radius: 5 }]);
    expect(result.get("c")).toEqual([{ x: 70, y: 50, radius: 5 }]);
    expect(result.has("a")).toBe(false);
  });

  it("widens a hop to clear both strokes", () => {
    const result = Jumps.findCrossings([
      horizontal("h", 0, 50, 2),
      vertical("v", 1, 50, 0, 100, 8),
    ]);
    expect(result.get("v")).toEqual([{ x: 50, y: 50, radius: 8 }]);
  });

  it("clears the full width of the crossed edge", () => {
    const result = Jumps.findCrossings([
      horizontal("h", 0, 50, 2, 14),
      vertical("v", 1, 50, 0, 100, 2),
    ]);
    expect(result.get("v")).toEqual([{ x: 50, y: 50, radius: 11 }]);
  });

  it.each([
    { name: "keeps", fromY: 29, hops: true },
    { name: "drops", fromY: 31, hops: false },
  ])("$name a wide hop by its room to the segment's end", ({ fromY, hops }) => {
    const result = Jumps.findCrossings([
      horizontal("h", 0, 50, 2, 14),
      vertical("v", 1, 50, fromY, 100, 2),
    ]);
    expect(result.has("v")).toBe(hops);
  });

  it.each([
    { name: "keeps", fromY: 29, hops: true },
    { name: "drops", fromY: 31, hops: false },
  ])("$name a jacketed hop by the room of its inner line", ({ fromY, hops }) => {
    const result = Jumps.findCrossings([
      horizontal("h", 0, 50, 2),
      vertical("v", 1, 50, fromY, 100, 2, 14),
    ]);
    expect(result.has("v")).toBe(hops);
  });

  it("adds only the stroke of the edge that hops", () => {
    const result = Jumps.findCrossings([
      horizontal("h", 0, 50, 2),
      vertical("v", 1, 50, 0, 100, 2, 14),
    ]);
    expect(result.get("v")).toEqual([{ x: 50, y: 50, radius: 5 }]);
  });
});
