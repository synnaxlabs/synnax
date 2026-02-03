// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, clamp, location } from "@synnaxlabs/x";
import { clsx } from "clsx";
import { type ReactElement, useCallback, useEffect, useRef, useState } from "react";

import { CSS } from "@/css";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { Base, type BaseProps } from "@/resize/Base";

/** Props for the {@link Single} component. */
export interface SingleProps extends Omit<
  BaseProps,
  "showHandle" | "size" | "onResize" | "onDragStart" | "ref"
> {
  initialSize?: number;
  minSize?: number;
  maxSize?: number;
  onResize?: (size: number, box: box.Box) => void;
  collapseThreshold?: number;
  onCollapse?: () => void;
}

const COLLAPSED_SIZE = 2;
const DEFAULT_MIN_SIZE = 100;

/**
 * A panel that can be resized in one direction by dragging its handle.
 *
 * @param props - The component props. All unused props will be passed to the div
 * containing the content.
 * @param props.location - The the location of the panel. The drag handle will be on the opposite side.
 * @param props.minSize - The smallest size the panel can be resized to.
 * @param props.maxSize - The largest size the panel can be resized to.
 * @param props.onResize - A callback executed when the panel is resized.
 */
export const Single = ({
  onCollapse,
  onResize,
  location: location_ = "left",
  minSize,
  maxSize,
  initialSize = 200,
  collapseThreshold = Infinity,
  className,
  ...rest
}: SingleProps): ReactElement => {
  const [size, setSize] = useState(clamp(initialSize, minSize, maxSize));
  const marker = useRef<number | null>(null);
  const loc = location.construct(location_);

  const calcNextSize = useCallback(
    (b: box.Box) => {
      if (marker.current === null) return 0;
      const dim =
        box.dim(b, location.direction(loc), true) *
        (1 - 2 * Number(["bottom", "right"].includes(loc)));
      const rawNextSize = marker.current + dim;
      const nextSize = clamp(rawNextSize, minSize, maxSize);
      if ((nextSize - rawNextSize) / (minSize ?? DEFAULT_MIN_SIZE) > collapseThreshold)
        return COLLAPSED_SIZE;
      return nextSize;
    },
    [loc, minSize, maxSize, collapseThreshold],
  );

  const ref = useRef<HTMLDivElement>(null);

  const handleMove = useCallback(
    (dragRegion: box.Box) => {
      const nextSize = calcNextSize(dragRegion);
      setSize(nextSize);
      if (ref.current == null) return;
      onResize?.(nextSize, box.construct(ref.current));
    },
    [onResize, calcNextSize],
  );

  const handleStart = useCallback(
    () =>
      setSize((prev) => {
        marker.current = prev;
        return prev;
      }),
    [setSize],
  );

  const handleEnd = useCallback(
    (box: box.Box) => calcNextSize(box) === COLLAPSED_SIZE && onCollapse?.(),
    [onCollapse, calcNextSize],
  );

  useEffect(() => {
    if (minSize == null || maxSize == null) return;
    setSize((prev) => {
      const nextSize = clamp(prev, minSize, maxSize);
      marker.current = nextSize;
      return nextSize;
    });
  }, [minSize, maxSize]);

  const handleDragStart = useCursorDrag({
    onMove: handleMove,
    onStart: handleStart,
    onEnd: handleEnd,
  });

  return (
    <Base
      ref={ref}
      location={loc}
      size={size}
      onDragStart={handleDragStart}
      className={clsx(className, CSS.expanded(size !== COLLAPSED_SIZE))}
      {...rest}
    />
  );
};
