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

import { useCursorDrag } from "@/hooks/useCursorDrag";

import { ResizeCore, ResizeCoreProps } from "./ResizeCore";

import { Box, dirFromLoc, Location, swapDir } from "@/spatial";

import "./Resize.css";

export interface ResizeProps
  extends Omit<ResizeCoreProps, "showHandle" | "size" | "onResize"> {
  location: Location;
  initialSize?: number;
  minSize?: number;
  maxSize?: number;
  onResize?: (size: number) => void;
}

export const Resize = ({
  location = "left",
  minSize = 100,
  maxSize = Infinity,
  initialSize = 200,
  onResize,
  ...props
}: ResizeProps): JSX.Element => {
  const [size, setSize] = useState(clamp(initialSize, minSize, maxSize));
  const marker = useRef<number | null>(null);
  const swappedDir = swapDir(dirFromLoc(location));

  const onMouseMove = useCallback(
    (box: Box) => {
      if (marker.current === null) marker.current = size;
      const nextSize = clamp(
        marker.current + box.dim(swappedDir, true),
        minSize,
        maxSize
      );
      setSize(nextSize);
      onResize?.(nextSize);
    },
    [onResize, location, minSize, maxSize, size]
  );

  useEffect(() => {
    setSize((prev) => clamp(prev, minSize, maxSize));
  }, [minSize, maxSize]);

  const handleDragStart = useCursorDrag({
    onMove: onMouseMove,
    onEnd: () => {
      marker.current = null;
    },
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
