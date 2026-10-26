// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, location } from "@synnaxlabs/x";
import { type ReactElement, useRef, useState } from "react";

import { Cursor } from "@/cursor";
import { Base, type BaseProps } from "@/resize/Base";

export interface HandlerExtra {
  box: box.Box;
  dragSize: number;
}

/** Props for the {@link Single} component. */
export interface SingleProps extends Omit<
  BaseProps,
  "hideHandle" | "size" | "onResize" | "onPointerDown" | "ref"
> {
  size?: number;
  sizeBounds?: Partial<bounds.Bounds>;
  onResize?: (size: number, extra: HandlerExtra) => void;
  onResizeEnd?: (size: number, extra: HandlerExtra) => void;
}

const DEFAULT_SIZE = 200;
const DEFAULT_SIZE_BOUNDS = { lower: 100 };

export const Single = ({
  onResize,
  onResizeEnd,
  location: propsLoc = "left",
  sizeBounds,
  size = DEFAULT_SIZE,
  className,
  ...rest
}: SingleProps): ReactElement => {
  const fullSizeBounds = bounds.construct({
    ...sizeBounds,
    lower: sizeBounds?.lower ?? DEFAULT_SIZE_BOUNDS.lower,
  });
  const clamped = bounds.clamp(fullSizeBounds, size);
  const [dragSize, setDragSize] = useState<number | null>(null);
  const rendered = dragSize ?? clamped;
  const marker = useRef<number>(clamped);
  const loc = location.construct(propsLoc);
  const ref = useRef<HTMLDivElement>(null);

  const calcNextSize = (dragRegion: box.Box): [clamped: number, raw: number] => {
    const signedDim = box.dim(dragRegion, location.direction(loc), true);
    const isInverted = loc === "bottom" || loc === "right";
    const dim = isInverted ? -signedDim : signedDim;
    const rawNextSize = marker.current + dim;
    return [bounds.clamp(fullSizeBounds, rawNextSize), rawNextSize];
  };

  const handleStart = () => {
    marker.current = clamped;
  };

  const handleMove = (dragRegion: box.Box) => {
    const [nextSize, rawNextSize] = calcNextSize(dragRegion);
    setDragSize(nextSize);
    if (ref.current != null)
      onResize?.(nextSize, {
        box: box.construct(ref.current),
        dragSize: rawNextSize,
      });
  };

  const handleEnd = (dragRegion: box.Box) => {
    const [nextSize, dragSize] = calcNextSize(dragRegion);
    setDragSize(null);
    if (ref.current != null)
      onResizeEnd?.(nextSize, { box: box.construct(ref.current), dragSize });
  };

  const handleDragStart = Cursor.useDrag({
    onMove: handleMove,
    onStart: handleStart,
    onEnd: handleEnd,
    preventDefault: true,
  });

  return (
    <Base
      ref={ref}
      location={loc}
      size={rendered}
      onPointerDown={handleDragStart}
      className={className}
      {...rest}
    />
  );
};
