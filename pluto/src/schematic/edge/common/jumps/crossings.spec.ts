// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { findCrossings, type Polyline } from "@/schematic/edge/common/jumps/crossings";

const horizontal = (key: string, order: number, y = 50): Polyline => ({
  key,
  order,
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
): Polyline => ({
  key,
  order,
  points: [
    { x, y: fromY },
    { x, y: toY },
  ],
});

describe("findCrossings", () => {
  it("assigns a crossing of a horizontal and vertical edge to the edge on top", () => {
    const result = findCrossings([horizontal("a", 0), vertical("b", 1)]);
    expect(result.get("b")).toEqual([{ x: 50, y: 50 }]);
    expect(result.has("a")).toBe(false);
  });

  it("assigns to the horizontal edge when it renders on top", () => {
    const result = findCrossings([horizontal("a", 5), vertical("b", 1)]);
    expect(result.get("a")).toEqual([{ x: 50, y: 50 }]);
    expect(result.has("b")).toBe(false);
  });

  it("ignores a T-junction where one segment ends on the other", () => {
    const result = findCrossings([horizontal("a", 0), vertical("b", 1, 50, 50, 100)]);
    expect(result.size).toBe(0);
  });

  it("ignores crossings within the end buffer of a segment", () => {
    const result = findCrossings([horizontal("a", 0), vertical("b", 1, 50, 45, 100)]);
    expect(result.size).toBe(0);
  });

  it("ignores parallel (collinear-capable) segments that never form an x/y pair", () => {
    const result = findCrossings([horizontal("a", 0, 50), horizontal("b", 1, 50)]);
    expect(result.size).toBe(0);
  });

  it("does not cross an edge with itself", () => {
    const elbow: Polyline = {
      key: "a",
      order: 0,
      points: [
        { x: 0, y: 50 },
        { x: 50, y: 50 },
        { x: 50, y: 100 },
      ],
    };
    expect(findCrossings([elbow]).size).toBe(0);
  });

  it("records a separate crossing per vertical edge over a shared horizontal", () => {
    const result = findCrossings([
      horizontal("a", 0),
      vertical("b", 1, 30),
      vertical("c", 2, 70),
    ]);
    expect(result.get("b")).toEqual([{ x: 30, y: 50 }]);
    expect(result.get("c")).toEqual([{ x: 70, y: 50 }]);
    expect(result.has("a")).toBe(false);
  });
});
