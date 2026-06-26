// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, location } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef, useState } from "react";

import { CSS } from "@/css";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { Base, type BaseProps } from "@/resize/Base";

/** Props for the {@link Single} component. */
export interface SingleProps extends Omit<
  BaseProps,
  "hideHandle" | "size" | "onResize" | "onDragStart" | "ref"
> {
  /**
   * The size of the pane, in pixels. Acts as the source of truth whenever the handle is
   * not being actively dragged; while a drag is in progress the pane renders its
   * transient drag size instead. Defaults to 200.
   */
  size?: number;
  sizeBounds?: Partial<bounds.Bounds>;
  /**
   * Called continuously while the handle is being dragged with the live size and the
   * pane's box. Use for transient, per-frame side effects (e.g. repainting an overlay);
   * do not persist from here, as it fires once per pointer move.
   */
  onResize?: (size: number, box: box.Box) => void;
  /**
   * Called once when a drag gesture ends with the committed size and the pane's box.
   * This is the callback to persist the new size to the source of truth.
   */
  onResizeEnd?: (size: number, box: box.Box) => void;
  collapseThreshold?: number;
  onCollapse?: () => void;
}

const COLLAPSED_SIZE = 2;
const DEFAULT_SIZE = 200;
const DEFAULT_SIZE_BOUNDS = { lower: 100 };

export const Single = ({
  onCollapse,
  onResize,
  onResizeEnd,
  location: propsLoc = "left",
  sizeBounds,
  size = DEFAULT_SIZE,
  collapseThreshold = Infinity,
  className,
  ...rest
}: SingleProps): ReactElement => {
  const fullSizeBounds = useMemo(
    () =>
      bounds.construct({
        ...sizeBounds,
        lower: sizeBounds?.lower ?? DEFAULT_SIZE_BOUNDS.lower,
      }),
    [sizeBounds?.lower, sizeBounds?.upper],
  );
  const clamped = bounds.clamp(fullSizeBounds, size);
  const [dragSize, setDragSize] = useState<number | null>(null);
  const rendered = dragSize ?? clamped;
  const marker = useRef<number>(clamped);
  const loc = location.construct(propsLoc);
  const ref = useRef<HTMLDivElement>(null);

  const calcNextSize = useCallback(
    (b: box.Box) => {
      const signedDim = box.dim(b, location.direction(loc), true);
      const isInverted = loc === "bottom" || loc === "right";
      const dim = isInverted ? -signedDim : signedDim;
      const rawNextSize = marker.current + dim;
      const nextSize = bounds.clamp(fullSizeBounds, rawNextSize);
      if ((nextSize - rawNextSize) / fullSizeBounds.lower > collapseThreshold)
        return COLLAPSED_SIZE;
      return nextSize;
    },
    [loc, fullSizeBounds, collapseThreshold],
  );

  const handleStart = useCallback(() => {
    marker.current = clamped;
  }, [clamped]);

  const handleMove = useCallback(
    (dragRegion: box.Box) => {
      const nextSize = calcNextSize(dragRegion);
      setDragSize(nextSize);
      if (ref.current != null) onResize?.(nextSize, box.construct(ref.current));
    },
    [onResize, calcNextSize],
  );

  const handleEnd = useCallback(
    (dragRegion: box.Box) => {
      const nextSize = calcNextSize(dragRegion);
      setDragSize(null);
      if (nextSize === COLLAPSED_SIZE) return onCollapse?.();
      if (ref.current != null) onResizeEnd?.(nextSize, box.construct(ref.current));
    },
    [onResizeEnd, onCollapse, calcNextSize],
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
      size={rendered}
      onDragStart={handleDragStart}
      className={CSS(className, CSS.expanded(rendered !== COLLAPSED_SIZE))}
      {...rest}
    />
  );
};
