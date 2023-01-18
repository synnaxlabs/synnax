// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useRef, useState } from "react";

import { clamp } from "@synnaxlabs/x";

import { ResizeCore, ResizeCoreProps } from "./ResizeCore";

import { Box, locToDir, Location } from "@/spatial";
import { useCursorDrag } from "@/spatial/useCursorDrag";

import "./Resize.css";

export interface ResizeProps
  extends Omit<ResizeCoreProps, "showHandle" | "size" | "onResize" | "onDragStart"> {
  location: Location;
  initialSize?: number;
  minSize?: number;
  maxSize?: number;
  onResize?: (size: number) => void;
  collapseThreshold?: number;
  onCollapse?: () => void;
}

const COLLAPSED_SIZE = 0;

export const Resize = ({
  onCollapse,
  onResize,
  location = "left",
  minSize = 100,
  maxSize = Infinity,
  initialSize = 200,
  collapseThreshold = Infinity,
  ...props
}: ResizeProps): JSX.Element => {
  const [size, setSize] = useState(clamp(initialSize, minSize, maxSize));
  const marker = useRef<number | null>(null);

  const calcNextSize = useCallback(
    (box: Box) => {
      if (marker.current === null) return 0;
      const dir = locToDir(location);
      const dim =
        box.dim(dir, true) * (1 - 2 * Number(["bottom", "right"].includes(location)));
      const rawNextSize = marker.current + dim;
      const nextSize = clamp(rawNextSize, minSize, maxSize);
      if ((nextSize - rawNextSize) / minSize > collapseThreshold) return COLLAPSED_SIZE;
      return nextSize;
    },
    [location, minSize, maxSize, collapseThreshold]
  );

  const handleMove = useCallback(
    (box: Box) => {
      const nextSize = calcNextSize(box);
      setSize(nextSize);
      onResize?.(nextSize);
    },
    [onResize, calcNextSize]
  );

  const handleStart = useCallback(
    () =>
      setSize((prev) => {
        marker.current = prev;
        return prev;
      }),
    [setSize]
  );

  const handleEnd = useCallback(
    (box: Box) => calcNextSize(box) === COLLAPSED_SIZE && onCollapse?.(),
    [onCollapse, calcNextSize]
  );

  useEffect((): void => {
    const nextSize = clamp(initialSize, minSize, maxSize);
    marker.current = nextSize;
    setSize(nextSize);
  }, [initialSize, minSize, maxSize]);

  const handleDragStart = useCursorDrag({
    onMove: handleMove,
    onStart: handleStart,
    onEnd: handleEnd,
  });

  return (
    <ResizeCore
      draggable
      location={location}
      size={size}
      onDragStart={handleDragStart}
      {...props}
    />
  );
};
