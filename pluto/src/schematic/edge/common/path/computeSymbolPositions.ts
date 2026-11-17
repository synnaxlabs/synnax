// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type direction, xy } from "@synnaxlabs/x";

export interface SymbolPosition {
  position: xy.XY;
  direction: direction.Direction;
}

const JOINT_REMOVE_THRESHOLD = 5;
/// @brief share of the interval within which a symbol near a corner is dropped instead
/// of moved out to the clearance.
const DROP_RATIO = 1 / 8;

/**
 * @returns a symbol position every interval along the points. A symbol closer than
 * clearance to a corner or an end moves out to the clearance, unless it is very near
 * the corner, its segment is too short, or the move puts it too near the last symbol.
 */
export const computeSymbolPositions = (
  points: xy.XY[],
  interval: number,
  clearance: number = JOINT_REMOVE_THRESHOLD,
): SymbolPosition[] => {
  const positions: SymbolPosition[] = [];
  const drop = Math.min(clearance, interval * DROP_RATIO);
  let lastPlaced = -Infinity;
  let currentLength = 0;
  let nextPosition = interval;
  for (let i = 0; i < points.length - 1; i++) {
    const start = points[i];
    const end = points[i + 1];
    const length = xy.distance(start, end);
    const direction: direction.Direction = Math.abs(end.x - start.x) < 1 ? "y" : "x";
    while (currentLength + length >= nextPosition) {
      const wanted = nextPosition - currentLength;
      nextPosition += interval;
      if (wanted < drop || length - wanted < drop) continue;
      const along = Math.min(Math.max(wanted, clearance), length - clearance);
      if (along < clearance || currentLength + along - lastPlaced < interval / 2)
        continue;
      lastPlaced = currentLength + along;
      const t = along / length;
      const position = {
        x: start.x + t * (end.x - start.x),
        y: start.y + t * (end.y - start.y),
      };
      positions.push({ position, direction });
    }
    currentLength += length;
  }
  return positions;
};
