// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { xy, location, box, direction } from "@synnaxlabs/x";

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
  // so we don't neeed to create an extra segment.
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
    if (Math.abs(nodeEdge - targetNodeEdge) < MIN_LENGTH)
      orientationToTravelIn = location.swap(orientationToTravelIn) as location.Outer;
  }
  // We need to travel from the source to the edge of the node plus MIN_LENGTH
  return {
    direction: swappedSourceDirection,
    length:
      nodeEdge -
      sourcePos[swappedSourceDirection] +
      setOrientationOnLength(orientationToTravelIn, MIN_LENGTH),
  };
};

export interface Segment {
  direction: direction.Direction;
  length: number;
}

export const travelSegments = (source: xy.XY, ...segments: Segment[]): xy.XY => {
  let current = source;
  for (const segment of segments) {
    current = xy.translate(current, segment.direction, segment.length);
  }
  return current;
};

export const segmentsToPoints = (
  source: xy.XY,
  segments: Segment[],
  zoom: number,
  applyTransform: boolean,
): xy.XY[] => {
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
  points[0] = xy.translate(points[0], {
    [firstSeg.direction]: -1 * firstMag * 4 * zoom,
    [direction.swap(firstSeg.direction)]: 0,
  } as const as xy.XY);
  const lastSeg = segments[segments.length - 1];
  const lastSegOrientation = orientationFromLength(lastSeg.direction, lastSeg.length);
  const lastMag = orientationMagnitude(lastSegOrientation);
  points[points.length - 1] = xy.translate(points[points.length - 1], {
    [lastSeg.direction]: lastMag * 4 * zoom,
    [direction.swap(lastSeg.direction)]: 0,
  } as const as xy.XY);

  return points;
};

const handleSourceOrientationChange = (
  segments: Segment[],
  orientation: location.Outer,
): Segment[] => {
  const firstSeg = segments[0];
  const firstSegOrientation = orientationFromLength(
    firstSeg.direction,
    firstSeg.length,
  );
  if (firstSegOrientation === orientation) return segments;
  if (segments.length === 1) return;
};

export interface BuildNewConnectorProps {
  sourceBox: box.Box;
  targetBox: box.Box;
  sourcePos: xy.XY;
  targetPos: xy.XY;
  sourceOrientation: location.Outer;
  targetOrientation: location.Outer;
}

const MIN_LENGTH = 6;

const orientationMagnitude = (or: location.Outer) =>
  or === "top" || or === "left" ? -1 : 1;

const setOrientationOnLength = (or: location.Outer, length: number) =>
  or === "top" || or === "left" ? -length : length;

const orientationFromLength = (
  direction: direction.Direction,
  length: number,
): location.Outer => {
  if (direction === "x") return length > 0 ? "right" : "left";
  return length > 0 ? "bottom" : "top";
};

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
  length: setOrientationOnLength(orientation, MIN_LENGTH),
});

const STUMPS = {
  top: stump("top"),
  bottom: stump("bottom"),
  left: stump("left"),
  right: stump("right"),
};

const COMPRESSION_THRESHOLD = 4;
const DIRECT_REMOVAL_THRESHOLD = 0.25;

export const compressSegments = (segments: Segment[], prev: Segment[]): Segment[] => {
  return removeSameDirectionSegments(
    removeShortSegments(removeSameDirectionSegments(segments), prev),
  );
};

const removeShortSegments = (segments: Segment[], prev: Segment[]): Segment[] => {
  const next: Segment[] = [...segments];
  const ok = segments.findIndex((seg, i) => {
    // If it's below the compression threshold and the user is making it smaller,
    // then we compress.
    const mag = Math.abs(seg.length);
    if (mag < COMPRESSION_THRESHOLD) {
      if (mag < DIRECT_REMOVAL_THRESHOLD) return true;
      if (segments.length <= 3 || i === 0 || i === segments.length - 1) return false;
      if (i + 2 < segments.length) {
        next[i + 2] = {
          direction: next[i + 2].direction,
          length: next[i + 2].length + seg.length,
        };
      } else {
        next[i - 2] = {
          direction: next[i - 2].direction,
          length: next[i - 2].length + seg.length,
        };
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

const removeSameDirectionSegments = (segments: Segment[]): Segment[] => {
  const next: Segment[] = [...segments];
  const idx = segments.findIndex(
    (seg, i) => i !== 0 && seg.direction === segments[i - 1].direction,
  );
  if (idx !== -1) {
    next[idx - 1] = {
      direction: next[idx - 1].direction,
      length: next[idx - 1].length + next[idx].length,
    };
    next.splice(idx, 1);
    return removeSameDirectionSegments(next);
  }
  return next;
};

export const newConnector = (props: BuildNewConnectorProps): Segment[] =>
  removeSameDirectionSegments(internalNewConnector(props));

const internalNewConnector = ({
  sourceBox,
  targetBox,
  sourcePos,
  targetPos,
  targetOrientation,
  sourceOrientation,
}: BuildNewConnectorProps): Segment[] => {
  let sourceStumpOrientation = sourceOrientation;
  let targetStumpOrientation = targetOrientation;

  let sourceStump = { ...STUMPS[sourceOrientation] };
  let sourceStumpTip = travelSegments(sourcePos, sourceStump);

  const targetStump = { ...STUMPS[targetOrientation] };
  let targetStumpTip = travelSegments(targetPos, targetStump);

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
    sourceStumpOrientation = orientationFromLength(
      extraSourceSeg.direction,
      extraSourceSeg.length,
    );
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
  ) {
    // This means we're good to go in this direction
    firstSeg = {
      direction: sourceStump.direction,
      length: delta,
    };
  } else {
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

export const moveConnector = (props: MoveConnectorProps): Segment[] =>
  compressSegments(internalMoveConnector(props), props.segments);

const internalMoveConnector = ({
  segments,
  index,
  magnitude,
}: MoveConnectorProps): Segment[] => {
  const next = [...segments];
  const seg = next[index];
  const dir = direction.swap(seg.direction);
  const orientation = orientationFromLength(seg.direction, seg.length);
  if (index === 0) {
    console.log("here");
    next.unshift({ direction: dir, length: magnitude });
    next.unshift({
      direction: seg.direction,
      length: setOrientationOnLength(orientation, MIN_LENGTH),
    });
    index += 2;
  } else
    next[index - 1] = {
      direction: next[index - 1].direction,
      length: next[index - 1].length + magnitude,
    };
  if (index === next.length - 1) {
    next.push({ direction: dir, length: -magnitude });
    next.push({ direction: seg.direction, length: MIN_LENGTH });
    next[index] = {
      direction: next[index].direction,
      length: next[index].length - MIN_LENGTH,
    };
  } else {
    next[index + 1] = {
      direction: next[index + 1].direction,
      length: next[index + 1].length - magnitude,
    };
  }
  console.log(next);
  return next;
};

const findIndexBackwards = (
  segments: Segment[],
  cb: (seg: Segment, i: number) => boolean,
): number => {
  for (let i = segments.length - 1; i >= 0; i--) {
    if (cb(segments[i], i)) return i;
  }
  return -1;
};

const findIndex = (
  segments: Segment[],
  cb: (seg: Segment, i: number) => boolean,
  reverse = false,
): number => (reverse ? findIndexBackwards(segments, cb) : segments.findIndex(cb));

export const handleMoveSource = (
  prevS: xy.XY,
  nextS: xy.XY,
  segments: Segment[],
): Segment[] =>
  compressSegments(handleMoveSourceInternal(prevS, nextS, segments), segments);

export const handleMoveTarget = (
  prevS: xy.XY,
  nextS: xy.XY,
  segments: Segment[],
): Segment[] =>
  compressSegments(handleMoveSourceInternal(prevS, nextS, segments, true), segments);

const handleMoveSourceInternal = (
  prevS: xy.XY,
  nextS: xy.XY,
  segments: Segment[],
  reverse = false,
): Segment[] => {
  const delta = xy.translation(prevS, nextS);
  const next = [...segments];

  if (delta.x !== 0) {
    // We want to find the first segment in the 'x' direction that still has a magnitude
    // length greater than MIN_LENGTH after we apply the delta.
    let idx = findIndex(
      next,
      (seg) => {
        if (seg.direction !== "x") return false;
        const newLength = seg.length - delta.x;
        return Math.abs(newLength) > MIN_LENGTH * 2;
      },
      reverse,
    );
    if (idx === -1) {
      // just use the first one in the correct direction
      idx = findIndex(next, (seg) => seg.direction === "x", reverse);
      if (idx === -1) {
        // This means that there is only one segment in the 'y' direction in the whole
        // connector, so we split it in half and add a new segment.
        return [
          {
            direction: "y",
            length: segments[0].length / 2,
          },
          {
            direction: "x",
            length: -delta.x,
          },
          {
            direction: "y",
            length: segments[0].length / 2,
          },
        ];
      }
    }
    next[idx] = {
      direction: next[idx].direction,
      length: next[idx].length - delta.x,
    };
  }

  // same theory applies here
  if (delta.y !== 0) {
    let idx = findIndex(
      next,
      (seg) => {
        if (seg.direction !== "y") return false;
        const newLength = seg.length - delta.y;
        return Math.abs(newLength) > MIN_LENGTH;
      },
      reverse,
    );
    if (idx === -1) {
      idx = findIndex(next, (seg) => seg.direction === "y", reverse);
      if (idx === -1) {
        return [
          {
            direction: "x",
            length: segments[0].length / 2,
          },
          {
            direction: "y",
            length: -delta.y,
          },
          {
            direction: "x",
            length: segments[0].length / 2,
          },
        ];
      }
    }
    next[idx] = {
      direction: next[idx].direction,
      length: next[idx].length - delta.y,
    };
  }

  return next;
};
