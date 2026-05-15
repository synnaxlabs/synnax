// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { direction } from "@synnaxlabs/x/direction";
import { xy } from "@synnaxlabs/x/xy";
export interface SymbolPosition {
  position: xy.XY;
  direction: direction.Direction;
}

const JOINT_REMOVE_THRESHOLD = 5;

export const computeSymbolPositions = (
  points: xy.XY[],
  interval: number,
): SymbolPosition[] => {
  const positions: SymbolPosition[] = [];
  const segmentLengths: number[] = [];
  for (let i = 0; i < points.length - 1; i++)
    segmentLengths.push(xy.distance(points[i], points[i + 1]));
  let currentLength = 0;
  let nextPosition = interval;
  for (let i = 0; i < segmentLengths.length; i++) {
    const length = segmentLengths[i];
    const start = points[i];
    const end = points[i + 1];
    while (currentLength + length >= nextPosition) {
      const t = (nextPosition - currentLength) / length;
      const position = {
        x: start.x + t * (end.x - start.x),
        y: start.y + t * (end.y - start.y),
      };
      let direction: direction.Direction = "x";
      if (Math.abs(end.x - start.x) < 1) direction = "y";
      nextPosition += interval;
      if (
        xy.distance(position, end) < JOINT_REMOVE_THRESHOLD ||
        xy.distance(position, start) < JOINT_REMOVE_THRESHOLD
      )
        continue;
      positions.push({ position, direction });
    }
    currentLength += length;
  }
  return positions;
};
