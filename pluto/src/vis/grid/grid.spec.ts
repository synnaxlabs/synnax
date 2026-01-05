// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { grid } from "@/vis/grid";

describe("grid", () => {
  describe("position", () => {
    interface Spec {
      name: string;
      grid: grid.Grid;
      key: string;
      expected: xy.XY;
      container: box.Crude;
    }
    const SPECS: Spec[] = [
      {
        name: "Single X-Axis",
        container: box.construct({ x: 0, y: 0 }, { x: 100, y: 100 }),
        key: "x-axis",
        grid: { "x-axis": { key: "x-axis", loc: "bottom", order: 0, size: 50 } },
        expected: { x: 0, y: 50 },
      },
      {
        name: "X-Axis with Y-Axis",
        container: box.construct({ x: 0, y: 0 }, { x: 100, y: 100 }),
        key: "x-axis",
        grid: {
          "x-axis": { key: "x-axis", loc: "bottom", order: 0, size: 50 },
          "y-axis": { key: "y-axis", loc: "left", order: 0, size: 50 },
        },
        expected: { x: 50, y: 50 },
      },
      {
        name: "Y-Axis with X-Axis",
        container: box.construct({ x: 0, y: 0 }, { x: 100, y: 100 }),
        key: "y-axis",
        grid: {
          "x-axis": { key: "x-axis", loc: "bottom", order: 0, size: 50 },
          "y-axis": { key: "y-axis", loc: "left", order: 0, size: 50 },
        },
        expected: { x: 0, y: 0 },
      },
      {
        name: "Y-Axis with X-Axis on Top",
        container: box.construct({ x: 0, y: 0 }, { x: 100, y: 100 }),
        key: "y-axis",
        grid: {
          "x-axis": { key: "x-axis", loc: "top", order: 0, size: 50 },
          "y-axis": { key: "y-axis", loc: "left", order: 0, size: 50 },
        },
        expected: { x: 0, y: 50 },
      },
      {
        name: "Two X-Axes on Bottom - Closest to Visualization",
        container: box.construct({ x: 0, y: 0 }, { x: 100, y: 100 }),
        key: "x-axis",
        grid: {
          "x-axis": { key: "x-axis", loc: "bottom", order: 0, size: 10 },
          "x-axis-2": { key: "x-axis-2", loc: "bottom", order: 1, size: 10 },
        },
        expected: { x: 0, y: 80 },
      },
      {
        name: "Two X-Axes on Bottom - Farthest from Visualization",
        container: box.construct({ x: 0, y: 0 }, { x: 100, y: 100 }),
        key: "x-axis",
        grid: {
          "x-axis": { key: "x-axis", loc: "bottom", order: 1, size: 10 },
          "x-axis-2": { key: "x-axis-2", loc: "bottom", order: 0, size: 10 },
        },
        expected: { x: 0, y: 90 },
      },
      {
        name: "Two X-Axes on Top - Closest to Visualization",
        container: box.construct({ x: 0, y: 0 }, { x: 100, y: 100 }),
        key: "x-axis",
        grid: {
          "x-axis": { key: "x-axis", loc: "top", order: 0, size: 10 },
          "x-axis-2": { key: "x-axis-2", loc: "top", order: 1, size: 10 },
        },
        expected: { x: 0, y: 10 },
      },
      {
        name: "Two X-Axes on Bottom, one Left, one Right",
        container: box.construct({ x: 0, y: 0 }, { x: 100, y: 100 }),
        key: "x-axis",
        grid: {
          "x-axis": { key: "x-axis", loc: "bottom", order: 0, size: 10 },
          "x-axis-2": { key: "x-axis-2", loc: "bottom", order: 1, size: 10 },
          "y-axis": { key: "y-axis", loc: "left", order: 0, size: 10 },
          "y-axis-2": { key: "y-axis-2", loc: "right", order: 0, size: 10 },
        },
        expected: { x: 10, y: 80 },
      },
    ];
    SPECS.forEach(({ key, name, grid: grid_, container, expected }) => {
      it(name, () => {
        const actual = grid.position(key, grid_, box.construct(container));
        expect(actual).toEqual(expected);
      });
    });
  });
});
