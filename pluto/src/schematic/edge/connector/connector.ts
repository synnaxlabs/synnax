// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, direction, location, xy } from "@synnaxlabs/x";
import { z } from "zod";

export interface CheckIntegrityProps {
  sourcePos: xy.XY;
  targetPos: xy.XY;
  next: Segment[];
  prev: Segment[];
}

export const checkIntegrity = ({
  sourcePos,
  targetPos,
  next,
  prev,
}: CheckIntegrityProps): boolean => {
  const theoTarget = travelSegments(sourcePos, ...next);
  if (xy.distance(theoTarget, targetPos) < 1) return true;
  console.warn("Connector is out of sync with target");
  console.log("Theoretical target", theoTarget);
  console.log("Actual target", targetPos);
  console.log("Segments", next);
  console.log("Prev segments", prev);
  return false;
};

export interface ChangeOrientationProps {
  orientation: location.Outer;
  segments: Segment[];
}

export const changeSourceOrientation = (props: ChangeOrientationProps): Segment[] =>
  compressSegments(internalChangeOrientation(props, false));

export const changeTargetOrientation = (props: ChangeOrientationProps): Segment[] =>
  compressSegments(internalChangeOrientation(props, true));

const internalChangeOrientation = (
  { orientation, segments }: ChangeOrientationProps,
  reverse: boolean,
): Segment[] => {
  const stumpsPidX = reverse ? segments.length - 1 : 0;
  let next = [...segments];
  const stump = next[stumpsPidX];
  const currOrientation = segmentOrientation(stump);
  if (currOrientation === orientation) return next;
  const dir = direction.construct(orientation);
  const newStumpLength = setOrientationOnLength(orientation, STUMP_LENGTH);
  next = moveNodeInDirection(dir, xy.set(xy.ZERO, dir, -newStumpLength), next, reverse);
  const newStump = { direction: dir, length: newStumpLength };
  if (reverse) next.push(newStump);
  else next.unshift(newStump);
  return next;
};

export interface PrepareNodeProps {
  sourceStumpTip: xy.XY;
  sourceOrientation: location.Outer;
  sourceBox: box.Box;
  targetStumpTip: xy.XY;
  targetOrientation: location.Outer;
  targetBox: box.Box;
}

export const prepareNode = ({
  sourceStumpTip: sourcePos,
  sourceOrientation,
  sourceBox,
  targetStumpTip: targetPos,
  targetOrientation,
  targetBox,
}: PrepareNodeProps): Segment | undefined => {
  // This is the case where the final connection line will not overlap with the node,
  // so we don't need to create an extra segment.
  if (!needToGoAround({ sourcePos, sourceOrientation, targetPos })) return;

  const sourceDirection = direction.construct(sourceOrientation);
  const swappedSourceDirection = direction.swap(sourceDirection);

  // This is the direction we need to travel in
  let orientationToTravelIn = orientationFromLength(
    swappedSourceDirection,
    targetPos[swappedSourceDirection] - sourcePos[swappedSourceDirection],
  );

  // We need to grab the edge of the node in this direction
  const nodeEdge = box.loc(sourceBox, orientationToTravelIn);

  // If they are pointing in opposite directions, we need to check if we need to
  // go completely around the node if there isn't enough space for a connector
  // between the two nodes.
  if (location.swap(sourceOrientation) === targetOrientation) {
    // In this case we do need to go around the node.
    const targetNodeEdge = box.loc(targetBox, location.swap(orientationToTravelIn));
    if (Math.abs(nodeEdge - targetNodeEdge) < STUMP_LENGTH)
      orientationToTravelIn = location.swap(orientationToTravelIn) as location.Outer;
  }
  // We need to travel from the source to the edge of the node plus MIN_LENGTH
  return {
    direction: swappedSourceDirection,
    length:
      nodeEdge -
      sourcePos[swappedSourceDirection] +
      setOrientationOnLength(orientationToTravelIn, STUMP_LENGTH),
  };
};

export const segmentZ = z.object({
  direction: direction.direction,
  length: z.number(),
});

export type Segment = z.infer<typeof segmentZ>;

export const travelSegments = (source: xy.XY, ...segments: Segment[]): xy.XY => {
  let current = source;
  for (const segment of segments)
    current = xy.translate(current, segment.direction, segment.length);
  return current;
};

export const segmentsToPoints = (
  source: xy.XY,
  segments: Segment[],
  zoom: number,
  applyTransform: boolean,
): xy.XY[] => {
  try {
    let current = source;
    const points: xy.XY[] = [
      source,
      ...segments.map((s) => {
        current = xy.translate(current, s.direction, s.length);
        return current;
      }),
    ];
    if (!applyTransform) return points;

    const firstSeg = segments[0];
    const firstSegOrientation = orientationFromLength(
      firstSeg.direction,
      firstSeg.length,
    );
    const firstMag = orientationMagnitude(firstSegOrientation);
    const zoomMultiplier = 4 * (2 - zoom ** 0.5);
    points[0] = xy.translate(points[0], {
      [firstSeg.direction]: -1 * firstMag * zoomMultiplier,
      [direction.swap(firstSeg.direction)]: 0,
    } as const as xy.XY);
    const lastSeg = segments[segments.length - 1];
    const lastSegOrientation = orientationFromLength(lastSeg.direction, lastSeg.length);
    const lastMag = orientationMagnitude(lastSegOrientation);
    points[points.length - 1] = xy.translate(points[points.length - 1], {
      [lastSeg.direction]: lastMag * zoomMultiplier,
      [direction.swap(lastSeg.direction)]: 0,
    } as const as xy.XY);

    return points;
  } catch {
    return [];
  }
};

export interface BuildNew {
  sourceBox: box.Box;
  targetBox: box.Box;
  sourcePos: xy.XY;
  targetPos: xy.XY;
  sourceOrientation: location.Outer;
  targetOrientation: location.Outer;
}

export const STUMP_LENGTH = 10;

const orientationMagnitude = (or: location.Outer): number =>
  or === "top" || or === "left" ? -1 : 1;

const setOrientationOnLength = (or: location.Outer, length: number): number =>
  or === "top" || or === "left" ? -length : length;

const orientationFromLength = (
  direction: direction.Direction,
  length: number,
): location.Outer => {
  if (direction === "x") return length > 0 ? "right" : "left";
  return length > 0 ? "bottom" : "top";
};

const segmentOrientation = (segment: Segment): location.Outer =>
  orientationFromLength(segment.direction, segment.length);

export interface NeedToGoAroundSourceProps {
  sourcePos: xy.XY;
  targetPos: xy.XY;
  sourceOrientation: location.Outer;
}

export const needToGoAround = ({
  sourcePos,
  targetPos,
  sourceOrientation,
}: NeedToGoAroundSourceProps): boolean => {
  const sourceDirection = direction.construct(sourceOrientation);
  const delta = targetPos[sourceDirection] - sourcePos[sourceDirection];
  return setOrientationOnLength(sourceOrientation, delta) < 0;
};

export const stump = (orientation: location.Outer): Segment => ({
  direction: direction.construct(orientation),
  length: setOrientationOnLength(orientation, STUMP_LENGTH),
});

const STUMPS = {
  top: stump("top"),
  bottom: stump("bottom"),
  left: stump("left"),
  right: stump("right"),
};

const COMPRESSION_THRESHOLD = 4;
const DIRECT_REMOVAL_THRESHOLD = 0.25;

export const compressSegments = (segments: Segment[]): Segment[] =>
  removeSameOrientationSegments(
    removeShortSegments(removeSameOrientationSegments(segments)),
  );

const removeShortSegments = (segments: Segment[]): Segment[] => {
  const next: Segment[] = [...segments];
  const ok = segments.findIndex((seg, i) => {
    // If it's below the compression threshold and the user is making it smaller,
    // then we compress.
    const mag = Math.abs(seg.length);
    if (mag < COMPRESSION_THRESHOLD) {
      if (i === 0 || i === segments.length - 1) return false;
      if (mag < DIRECT_REMOVAL_THRESHOLD) return true;
      if (segments.length <= 3) return false;
      if (i + 2 < segments.length) {
        const toAdjust = next[i + 2];
        if (toAdjust.direction !== seg.direction) return false;
        next[i + 2] = { ...toAdjust, length: toAdjust.length + seg.length };
      } else {
        const toAdjust = next[i - 2];
        if (toAdjust.direction !== seg.direction) return false;
        next[i - 2] = { ...toAdjust, length: toAdjust.length + seg.length };
      }
      return true;
    }
    return false;
  });
  if (ok !== -1) {
    // splice out the short segment
    next.splice(ok, 1);
    return next;
  }
  return next;
};

const removeSameOrientationSegments = (segments: Segment[]): Segment[] => {
  const next: Segment[] = [...segments];
  const idx = segments.findIndex((seg, i) => {
    const prevSeg = next[i - 1];
    if (i === 0 || seg.direction !== prevSeg.direction) return false;

    if (i === 1) {
      const stumpIdx = i - 1;
      const segIdx = i;
      const stumpOrientation = segmentOrientation(prevSeg);
      const segOrientation = segmentOrientation(seg);
      if (stumpOrientation === segOrientation) {
        next[i - 1] = {
          direction: next[i - 1].direction,
          length: next[i - 1].length + next[i].length,
        };
        next.splice(i, 1);
        return true;
      }
      if (Math.abs(prevSeg.length) === STUMP_LENGTH) return false;

      const stumpLength = setOrientationOnLength(stumpOrientation, STUMP_LENGTH);
      const delta = next[stumpIdx].length - stumpLength;
      const nextLength = next[segIdx].length + delta;
      if (Math.abs(nextLength) < STUMP_LENGTH) return false;
      next[stumpIdx] = { ...next[stumpIdx], length: stumpLength };
      next[segIdx] = { ...next[segIdx], length: nextLength };
      return true;
    }

    if (i === segments.length - 1) {
      const stumpIdx = i;
      const segIdx = i - 1;
      const stumpOrientation = segmentOrientation(seg);
      const segOrientation = segmentOrientation(prevSeg);
      if (stumpOrientation === segOrientation) {
        next[i - 1] = {
          direction: next[i - 1].direction,
          length: next[i - 1].length + next[i].length,
        };
        next.splice(i, 1);
        return true;
      }

      if (Math.abs(seg.length) === STUMP_LENGTH) return false;

      const stumpLength = setOrientationOnLength(stumpOrientation, STUMP_LENGTH);
      const delta = next[stumpIdx].length - stumpLength;
      const nextLength = next[segIdx].length + delta;
      if (Math.abs(nextLength) < STUMP_LENGTH) return false;
      next[stumpIdx] = { ...next[stumpIdx], length: stumpLength };
      next[segIdx] = { ...next[segIdx], length: nextLength };
      return true;
    }

    // splice out the short segment
    next[i - 1] = {
      direction: next[i - 1].direction,
      length: next[i - 1].length + next[i].length,
    };
    next.splice(i, 1);
    return true;
  });
  if (idx !== -1) return removeSameOrientationSegments(next);
  return next;
};

export const buildNew = (props: BuildNew): Segment[] =>
  compressSegments(internalNewConnector(props));

const internalNewConnector = ({
  sourceBox,
  targetBox,
  sourcePos,
  targetPos,
  targetOrientation,
  sourceOrientation,
}: BuildNew): Segment[] => {
  let sourceStumpOrientation = sourceOrientation;
  let targetStumpOrientation = targetOrientation;

  let sourceStump = { ...STUMPS[sourceOrientation] };
  let sourceStumpTip = travelSegments(sourcePos, sourceStump);

  const targetStump = { ...STUMPS[targetOrientation] };
  let targetStumpTip = travelSegments(targetPos, targetStump);

  const xDist = Math.abs(sourceStumpTip.x - targetStumpTip.x);
  const yDist = Math.abs(sourceStumpTip.y - targetStumpTip.y);
  if (xDist < 2 * STUMP_LENGTH && yDist < 10) {
    sourceStump.length -= xDist / 2;
    targetStump.length += xDist / 2;
    sourceStumpTip = travelSegments(sourcePos, sourceStump);
    targetStumpTip = travelSegments(targetPos, targetStump);
  }

  const segments = [sourceStump];
  const extraSourceSeg = prepareNode({
    sourceStumpTip,
    sourceOrientation,
    sourceBox,
    targetStumpTip,
    targetOrientation,
    targetBox,
  });
  if (extraSourceSeg != null) {
    segments.push(extraSourceSeg);
    sourceStumpTip = travelSegments(sourceStumpTip, extraSourceSeg);
    sourceStumpOrientation = segmentOrientation(extraSourceSeg);
    sourceStump = extraSourceSeg;
  }

  const extraTargetSeg = prepareNode({
    sourceStumpTip: targetStumpTip,
    sourceOrientation: targetStumpOrientation,
    sourceBox: targetBox,
    targetStumpTip: sourceStumpTip,
    targetOrientation: sourceStumpOrientation,
    targetBox: sourceBox,
  });

  if (extraTargetSeg != null) {
    targetStumpTip = travelSegments(targetStumpTip, extraTargetSeg);
    targetStumpOrientation = orientationFromLength(
      extraTargetSeg.direction,
      extraTargetSeg.length,
    );
  }

  const addTargetSegments = (): void => {
    // We need to swap the orientations of the target stumps so that
    // they create the correct path when traversing the segments.
    if (extraTargetSeg != null) {
      extraTargetSeg.length *= -1;
      segments.push(extraTargetSeg);
    }
    targetStump.length *= -1;
    segments.push(targetStump);
  };

  // Here is where we draw the final connection line.
  // In this case we split the delta in half and draw three lines.
  if (location.swap(sourceStumpOrientation) === targetOrientation) {
    const dir = direction.construct(sourceStumpOrientation);
    // push three segments on
    // first segment is in same direction as source stump orientation and half way to the
    // target stump tip
    const dist = (targetStumpTip[dir] - sourceStumpTip[dir]) / 2;
    segments.push({ direction: dir, length: dist });
    const swappedDir = direction.swap(dir);
    // second segment is in the swapped direction of the source stump orientation and
    // the distance between the source stump tip and the target stump tip
    segments.push({
      direction: swappedDir,
      length: targetStumpTip[swappedDir] - sourceStumpTip[swappedDir],
    });
    // third segment is in the same direction as the source stump orientation and the
    // remaining distance to the target stump tip
    segments.push({ direction: dir, length: dist });
    addTargetSegments();
    return segments;
  }

  // In this case we draw two lines.
  // Check the delta in the direction of the source stump
  const delta =
    targetStumpTip[sourceStump.direction] - sourceStumpTip[sourceStump.direction];
  let swapped = direction.swap(sourceStump.direction);
  const swappedDelta = sourceStumpTip[swapped] - targetStumpTip[swapped];
  // Check if the delta is in the same direction as the source stump
  let firstSeg: Segment;
  if (
    orientationFromLength(sourceStump.direction, delta) === sourceStumpOrientation &&
    orientationFromLength(swapped, swappedDelta) === targetStumpOrientation
  )
    // This means we're good to go in this direction
    firstSeg = {
      direction: sourceStump.direction,
      length: delta,
    };
  else {
    // This means we need to go orthogonally
    firstSeg = {
      direction: swapped,
      length: targetStumpTip[swapped] - sourceStumpTip[swapped],
    };
    swapped = direction.swap(swapped);
  }

  segments.push(firstSeg);
  // All we need to do next is draw a line
  const secondSeg = {
    direction: swapped,
    length: targetStumpTip[swapped] - sourceStumpTip[swapped],
  };
  segments.push(secondSeg);
  addTargetSegments();
  return segments;
};

export interface MoveConnectorProps {
  segments: Segment[];
  index: number;
  magnitude: number;
}

export const dragSegment = (props: MoveConnectorProps): Segment[] =>
  compressSegments(internalDragSegment(props));

const internalDragSegment = ({
  segments,
  index,
  magnitude,
}: MoveConnectorProps): Segment[] => {
  const next = [...segments];
  const seg = next[index];
  const dir = direction.swap(seg.direction);
  const orientation = segmentOrientation(seg);

  // If we're dragging on a stump, we need to add two new segments.
  if (index === 0) {
    next.unshift({ direction: dir, length: magnitude });
    const stumpLength = setOrientationOnLength(orientation, STUMP_LENGTH);
    next.unshift({ direction: seg.direction, length: stumpLength });
    // Move the index up by two since we added two segments
    index += 2;
    // Since we added a new stump in the same direction as the old one, we need to
    // subtract the stump length from the segment.
    next[index] = { ...next[index], length: next[index].length - stumpLength };
  }
  // If it's not the stump just move it directly.
  else
    next[index - 1] = {
      direction: next[index - 1].direction,
      length: next[index - 1].length + magnitude,
    };

  // If we're dragging on the target stump, we need to add two new segments.
  if (index === next.length - 1) {
    next.push({ direction: dir, length: -magnitude });
    const stumpLength = setOrientationOnLength(orientation, STUMP_LENGTH);
    next.push({ direction: seg.direction, length: stumpLength });
    next[index] = { ...next[index], length: next[index].length - stumpLength };
  } else
    next[index + 1] = {
      direction: next[index + 1].direction,
      length: next[index + 1].length - magnitude,
    };

  return next;
};

const findIndexBackwards = (
  segments: Segment[],
  cb: (seg: Segment, i: number) => boolean,
): number => {
  for (let i = segments.length - 1; i >= 0; i--) if (cb(segments[i], i)) return i;
  return -1;
};

const findNonTargetIdx = (
  segments: Segment[],
  cb: (seg: Segment, i: number) => boolean,
  reverse = false,
  ...exclude: number[]
): number => {
  const internalCB = (seg: Segment, i: number): boolean =>
    !exclude.includes(i) && cb(seg, i);
  return reverse
    ? findIndexBackwards(segments, internalCB)
    : segments.findIndex(internalCB);
};

export interface MoveNodeProps {
  delta: xy.XY;
  segments: Segment[];
}

export const moveSourceNode = ({ delta, segments }: MoveNodeProps): Segment[] =>
  compressSegments(moveNodeInternal(delta, segments, false));

export const moveTargetNode = ({ delta, segments }: MoveNodeProps): Segment[] =>
  compressSegments(moveNodeInternal(delta, segments, true));

const moveNodeInternal = (
  delta: xy.XY,
  segments: Segment[],
  reverse: boolean,
): Segment[] => {
  let next = [...segments];
  if (delta.x !== 0) next = moveNodeInDirection("x", delta, next, reverse);
  if (delta.y !== 0) next = moveNodeInDirection("y", delta, next, reverse);
  return next;
};

const canAdjustStump = (
  dir: direction.Direction,
  seg: Segment,
  delta: xy.XY,
): boolean => {
  if (seg.direction !== dir) return false;
  const next = { ...seg, length: seg.length - delta[dir] };
  const firstSegOrientation = segmentOrientation(seg);
  const nextFirstSegOrientation = segmentOrientation(next);
  const nextLengthMag = Math.abs(next.length);
  const prevLengthMag = Math.abs(seg.length);
  const isAboveMinLength =
    nextLengthMag > prevLengthMag || nextLengthMag > STUMP_LENGTH;
  const isSameOrientation = firstSegOrientation === nextFirstSegOrientation;
  return isAboveMinLength && isSameOrientation;
};

const moveNodeInDirection = (
  dir: direction.Direction,
  delta: xy.XY,
  segments: Segment[],
  reverse: boolean,
): Segment[] => {
  const swappedDirection = direction.swap(dir);
  // We'd always like to adjust the stump closest to the node if possible, but only
  // if compressing it won't make it too small OR cause it to reverse its orientation.
  const stumpIdx = reverse ? segments.length - 1 : 0;
  const stump = segments[stumpIdx];
  if (canAdjustStump(dir, stump, delta)) {
    const oppositeStump = segments[segments.length - 1];
    if (oppositeStump.direction === dir && Math.abs(stump.length) < STUMP_LENGTH) {
      segments[segments.length - 1] = {
        ...oppositeStump,
        length: oppositeStump.length - delta[dir],
      };
      return segments;
    }
    segments[stumpIdx] = { ...stump, length: stump.length - delta[dir] };
    return segments;
  }

  const oppositeStumpIdx = reverse ? 0 : segments.length - 1;
  const isX = (seg: Segment): boolean => seg.direction === dir;
  const idxToAdjust = findNonTargetIdx(segments, isX, reverse, stumpIdx);
  if (idxToAdjust === oppositeStumpIdx) {
    const oppositeStump = segments[oppositeStumpIdx];
    if (canAdjustStump(dir, oppositeStump, delta)) {
      segments[oppositeStumpIdx] = {
        ...oppositeStump,
        length: oppositeStump.length - delta[dir],
      };
      return segments;
    }
    // If the stump is in the right direction and its larger than the opposite stump
    if (stump.direction === dir && Math.abs(stump.length) > oppositeStump.length)
      segments[stumpIdx] = { ...stump, length: stump.length - delta[dir] };
    else
      segments[oppositeStumpIdx] = {
        ...oppositeStump,
        length: oppositeStump.length - delta[dir],
      };
    return segments;
  }
  // This means that there is only one segment in the 'swappedDirection' direction in the whole
  // connector, so we split it in half and add a new segment.
  if (idxToAdjust === -1)
    if (segments.length === 1) {
      if (segments[0].direction === dir)
        return [{ direction: dir, length: segments[0].length - delta[dir] }];
      return [
        { direction: swappedDirection, length: segments[0].length / 2 },
        { direction: dir, length: -delta[dir] },
        { direction: swappedDirection, length: segments[0].length / 2 },
      ];
    } else {
      if (stump.direction === dir) {
        // just adjust the stump
        segments[stumpIdx] = { ...stump, length: stump.length - delta[dir] };
        return segments;
      }
      return [
        { direction: swappedDirection, length: segments[0].length },
        { direction: dir, length: -delta[dir] },
        ...segments.slice(1),
      ];
    }

  const prev = segments[idxToAdjust];
  segments[idxToAdjust] = { ...prev, length: prev.length - delta[dir] };
  return segments;
};
