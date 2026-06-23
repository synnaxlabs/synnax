// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds, box, clamp, location } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef } from "react";

import { CSS } from "@/css";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { Base, type BaseProps } from "@/resize/Base";

export interface SingleProps extends Omit<
  BaseProps,
  "showHandle" | "onResize" | "onDragStart" | "ref"
> {
  onResize: (size: number, box: box.Box) => void;
  sizeBounds?: Partial<bounds.Bounds>;
  collapseThreshold?: number;
  onCollapse?: () => void;
}

const COLLAPSED_SIZE = 2;
const DEFAULT_MIN_SIZE = 100;

export const Single = ({
  onCollapse,
  onResize,
  location: propsLocation = "left",
  size,
  sizeBounds,
  collapseThreshold = Infinity,
  className,
  ...rest
}: SingleProps): ReactElement => {
  const marker = useRef<number | null>(null);
  const loc = location.construct(propsLocation);
  const clampSize = useCallback(
    (value: number) => clamp(value, sizeBounds?.lower, sizeBounds?.upper),
    [sizeBounds],
  );
  const clampedSize = clampSize(size);

  const calcNextSize = useCallback(
    (b: box.Box) => {
      if (marker.current === null) return 0;
      const dim =
        box.dim(b, location.direction(loc), true) *
        (1 - 2 * Number(["bottom", "right"].includes(loc)));
      const rawNextSize = marker.current + dim;
      const nextSize = clampSize(rawNextSize);
      if (
        (nextSize - rawNextSize) / (sizeBounds?.lower ?? DEFAULT_MIN_SIZE) >
        collapseThreshold
      )
        return COLLAPSED_SIZE;
      return nextSize;
    },
    [loc, sizeBounds, collapseThreshold],
  );

  const ref = useRef<HTMLDivElement>(null);

  const handleMove = useCallback(
    (dragRegion: box.Box) => {
      const nextSize = calcNextSize(dragRegion);
      if (nextSize === COLLAPSED_SIZE || ref.current == null) return;
      onResize?.(nextSize, box.construct(ref.current));
    },
    [onResize, calcNextSize],
  );

  const handleStart = useCallback(() => {
    marker.current = clampedSize;
  }, [clampedSize]);

  const handleEnd = useCallback(
    (box: box.Box) => calcNextSize(box) === COLLAPSED_SIZE && onCollapse?.(),
    [onCollapse, calcNextSize],
  );

  const handleDragStart = useCursorDrag({
    onMove: handleMove,
    onStart: handleStart,
    onEnd: handleEnd,
  });

  return (
    <Base
      ref={ref}
      location={loc}
      size={clampedSize}
      onDragStart={handleDragStart}
      className={CSS(className, CSS.expanded(clampedSize !== COLLAPSED_SIZE))}
      {...rest}
    />
  );
};
