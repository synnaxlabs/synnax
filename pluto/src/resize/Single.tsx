// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, location } from "@synnaxlabs/x";
import { clsx } from "clsx";
import {
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { Base, type BaseProps } from "@/resize/Base";

/** Props for the {@link Single} component. */
export interface SingleProps extends Omit<
  BaseProps,
  "showHandle" | "size" | "onResize" | "onDragStart" | "ref"
> {
  initialSize?: number;
  sizeBounds?: Partial<bounds.Bounds>;
  onResize?: (size: number, box: box.Box) => void;
  collapseThreshold?: number;
  onCollapse?: () => void;
}

const COLLAPSED_SIZE = 2;
const DEFAULT_SIZE_BOUNDS = { lower: 100 };

export const Single = ({
  onCollapse,
  onResize,
  location: location_ = "left",
  sizeBounds = DEFAULT_SIZE_BOUNDS,
  initialSize = 200,
  collapseThreshold = Infinity,
  className,
  ...rest
}: SingleProps): ReactElement => {
  const fullSizeBounds = useMemo(
    () =>
      bounds.construct({
        lower: sizeBounds.lower ?? DEFAULT_SIZE_BOUNDS.lower,
        upper: sizeBounds.upper,
      }),
    [sizeBounds.lower, sizeBounds.upper],
  );
  const [size, setSize] = useState(bounds.clamp(fullSizeBounds, initialSize));
  const marker = useRef<number | null>(null);
  const loc = location.construct(location_);

  const calcNextSize = useCallback(
    (b: box.Box) => {
      if (marker.current === null) return 0;
      const dim =
        box.dim(b, location.direction(loc), true) *
        (1 - 2 * Number(["bottom", "right"].includes(loc)));
      const rawNextSize = marker.current + dim;
      const nextSize = bounds.clamp(fullSizeBounds, rawNextSize);
      if ((nextSize - rawNextSize) / fullSizeBounds.lower > collapseThreshold)
        return COLLAPSED_SIZE;
      return nextSize;
    },
    [loc, fullSizeBounds, collapseThreshold],
  );

  const ref = useRef<HTMLDivElement>(null);

  const handleMove = useCallback(
    (dragRegion: box.Box) => {
      const nextSize = calcNextSize(dragRegion);
      setSize(nextSize);
      if (ref.current != null) onResize?.(nextSize, box.construct(ref.current));
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
    setSize((prev) => {
      const nextSize = bounds.clamp(fullSizeBounds, prev);
      marker.current = nextSize;
      return nextSize;
    });
  }, [fullSizeBounds]);

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
