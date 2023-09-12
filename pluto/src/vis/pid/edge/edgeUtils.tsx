// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, direction, xy } from "@synnaxlabs/x";


export const calculateLineDirection = (source: xy.XY, target: xy.XY): direction.Direction => {
  const xDist = xy.xDistance(source, target);
  const yDist = xy.yDistance(source, target);
  return xDist > yDist ? "x" : "y";
};

export const handleDrag = (
  prevPoints: xy.XY[],
  b: box.Box,
  root: xy.XY,
  index: number,
  zoom: number,
): [number, xy.XY[]] => {
  // The first point in the line.
  const prevOne = prevPoints[index];
  // The second point in the line
  const prevTwo = prevPoints[index + 1];
  // The direction of the line
  const dir = calculateLineDirection(prevOne, prevTwo);

  const translateDir = direction.swap(dir);
  const translateBy = root[translateDir] + box.signedDims(b)[direction.signedDimension(translateDir)] / zoom;
  const nextOne = xy.set(prevOne, translateDir, translateBy);
  const nextTwo = xy.set(prevTwo, translateDir, translateBy);

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

export const formsRoughlyStraightLine = (points: xy.XY[]): boolean => {
  // The objective is to determine if the line is roughly straight.
  const peakXDelta = Math.max(
    ...points.slice(1).map((p, i) => xy.xDistance(p, points[i])),
  );
  const peakYDelta = Math.max(
    ...points.slice(1).map((p, i) => xy.yDistance(p, points[i])),
  );
  return peakXDelta < 10 || peakYDelta < 10;
};

const newConnectorPoints = (source: xy.XY, target: xy.XY): xy.XY[] => [
  source,
  {x: (source.x + target.x) / 2, y: source.y},
  {x: (source.x + target.x) / 2, y: target.y},
  target,
];

export const adjustToSourceOrTarget = (
  sourceX: number,
  sourceY: number,
  targetX: number,
  targetY: number,
  points: xy.XY[],
): xy.XY[] | null => {
  const source = xy.construct(sourceX, sourceY);
  const target = xy.construct(targetX, targetY);
  const prevSource = points[0];
  const prevTarget = points[points.length - 1];

  const linear = formsRoughlyStraightLine([source, target]);
  if (points.length === 0) return newConnectorPoints(source, target);

  const sourceChanged = xy.distance(source, prevSource) > 1;
  const targetChanged = xy.distance(target, prevTarget) > 1;

  if (!sourceChanged && !targetChanged) return null;
  if (linear && points.length === 2) return [source, target];
  if (!linear && points.length === 2) return newConnectorPoints(source, target);

  const next: xy.XY[] = [...points];

  if (sourceChanged) {
    const adjustDir = direction.swap(calculateLineDirection(prevSource, points[1]));
    next[0] = source;
    next[1] = xy.set(next[1], adjustDir, source[adjustDir]);

    if (points.length >= 3 && formsRoughlyStraightLine(next.slice(0, 3)))
      next.splice(1, 1);
  } else if (targetChanged) {
    const adjustDir = direction.swap(calculateLineDirection(
      prevTarget,
      points[points.length - 2],
    ))
    next[points.length - 2] = xy.set(
      next[points.length - 2],
      adjustDir,
      target[adjustDir],
    );
    next[points.length - 1] = target;

    if (points.length >= 3 && formsRoughlyStraightLine(next.slice(-3)))
      next.splice(-2, 1);
  }

  return next;
};
