// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, Direction, XY } from "@synnaxlabs/x";

export const calculateLineDirection = (source: XY, target: XY): Direction => {
  const xDist = source.xDistanceTo(target);
  const yDist = source.yDistanceTo(target);
  return xDist > yDist ? Direction.X : Direction.Y;
};

export const handleDrag = (
  prevPoints: XY[],
  b: Box,
  root: XY,
  index: number,
  zoom: number
): [number, XY[]] => {
  // The first point in the line.
  const prevOne = prevPoints[index];
  // The second point in the line
  const prevTwo = prevPoints[index + 1];
  // The direction of the line
  const dir = calculateLineDirection(prevOne, prevTwo);

  const translateDir = dir.inverse;
  const translateBy = root[translateDir.crude] + b[translateDir.signedDimension] / zoom;
  const nextOne = prevOne.set(translateDir, translateBy);
  const nextTwo = prevTwo.set(translateDir, translateBy);

  const next = [...prevPoints];
  const isFirst = index === 0;
  const isLast = index === prevPoints.length - 2;
  if (isFirst) {
    next.unshift(prevPoints[index]);
    index++;
  }
  if (isLast) next.push(next[index + 1]);

  next[index] = nextOne;
  next[index + 1] = nextTwo;

  const maybeCompressForward = next.slice(index, index + 3).filter(Boolean);
  if (
    maybeCompressForward.length === 3 &&
    formsRoughlyStraightLine(maybeCompressForward)
  ) {
    next.splice(index + 1, 1);
  }
  const maybeCompressBackward = next.slice(index - 1, index + 2).filter(Boolean);
  if (
    maybeCompressBackward.length === 3 &&
    formsRoughlyStraightLine(maybeCompressBackward)
  ) {
    next.splice(index, 1);
    return [index - 1, next];
  }

  return [index, next];
};

export const formsRoughlyStraightLine = (points: XY[]): boolean => {
  // The objective is to determine if the line is roughly straight.
  const peakXDelta = Math.max(
    ...points.slice(1).map((p, i) => p.xDistanceTo(points[i]))
  );
  const peakYDelta = Math.max(
    ...points.slice(1).map((p, i) => p.yDistanceTo(points[i]))
  );
  return peakXDelta < 10 || peakYDelta < 10;
};

const newConnectorPoints = (source: XY, target: XY): XY[] => [
  source,
  new XY((source.x + target.x) / 2, source.y),
  new XY((source.x + target.x) / 2, target.y),
  target,
];

export const adjustToSourceOrTarget = (
  sourceX: number,
  sourceY: number,
  targetX: number,
  targetY: number,
  points: XY[]
): XY[] | null => {
  const source = new XY(sourceX, sourceY);
  const target = new XY(targetX, targetY);
  const prevSource = points[0];
  const prevTarget = points[points.length - 1];

  const linear = formsRoughlyStraightLine([source, target]);
  if (points.length === 0) return newConnectorPoints(source, target);

  const sourceChanged = source.distanceTo(prevSource) > 1;
  const targetChanged = target.distanceTo(prevTarget) > 1;

  if (!sourceChanged && !targetChanged) return null;
  if (linear && points.length === 2) return [source, target];
  if (!linear && points.length === 2) return newConnectorPoints(source, target);

  const next: XY[] = [...points];

  if (sourceChanged) {
    const adjustDir = calculateLineDirection(prevSource, points[1]).inverse;
    next[0] = source;
    next[1] = next[1].set(adjustDir, source[adjustDir.crude]);

    if (points.length >= 3 && formsRoughlyStraightLine(next.slice(0, 3)))
      next.splice(1, 1);
  } else if (targetChanged) {
    const adjustDir = calculateLineDirection(
      prevTarget,
      points[points.length - 2]
    ).inverse;
    next[points.length - 2] = next[points.length - 2].set(
      adjustDir,
      target[adjustDir.crude]
    );
    next[points.length - 1] = target;

    if (points.length >= 3 && formsRoughlyStraightLine(next.slice(-3)))
      next.splice(-2, 1);
  }

  return next;
};
