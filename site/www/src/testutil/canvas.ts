// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { vi } from "vitest";

export type Point = [x: number, y: number];

export interface Stroke {
  style: string;
  points: Point[];
}

export interface Fill {
  style: string;
  /** The center and radius of each arc in the path. */
  arcs: Array<[x: number, y: number, r: number]>;
}

export interface CanvasRecord {
  strokes: Stroke[];
  fills: Fill[];
  scales: Point[];
}

/**
 * Gives every canvas a 2D context that records the paths drawn on it, since jsdom has
 * no canvas.
 */
export const recordCanvas = (): CanvasRecord => {
  const record: CanvasRecord = { strokes: [], fills: [], scales: [] };
  let points: Point[] = [];
  let arcs: Fill["arcs"] = [];
  const context = {
    strokeStyle: "",
    fillStyle: "",
    lineWidth: 1,
    lineJoin: "miter",
    lineCap: "butt",
    scale: (x: number, y: number) => record.scales.push([x, y]),
    setTransform: () => {},
    clearRect: () => {},
    beginPath: () => {
      points = [];
      arcs = [];
    },
    moveTo: (x: number, y: number) => points.push([x, y]),
    lineTo: (x: number, y: number) => points.push([x, y]),
    arc: (x: number, y: number, r: number) => arcs.push([x, y, r]),
    stroke: () => record.strokes.push({ style: context.strokeStyle, points }),
    fill: () => record.fills.push({ style: context.fillStyle, arcs }),
  };
  vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(
    context as unknown as CanvasRenderingContext2D,
  );
  return record;
};
