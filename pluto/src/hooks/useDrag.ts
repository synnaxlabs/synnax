// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DragEvent,
  MouseEvent as RMouseEvent,
  useCallback,
  useLayoutEffect as useEffect,
  useState,
} from "react";

export interface UseDragProps {
  onMove?: (e: MouseEvent) => void;
  onStart?: (e: DragEvent | MouseEvent) => void;
  onEnd?: (e: DragEvent | MouseEvent) => void;
}

export interface UseDragReturn {
  onDragStart: (e: DragEvent | RMouseEvent) => void;
  onDrag: (e: DragEvent | RMouseEvent) => void;
  onDragEnd: (e: DragEvent | RMouseEvent) => void;
}

export const useDrag = ({
  onMove: propsOnMouseMove,
  onStart,
  onEnd,
}: UseDragProps): UseDragReturn => {
  const [dragging, setDragging] = useState<boolean>(false);

  const onDragStart = useCallback((e: DragEvent | RMouseEvent): void => {
    e.preventDefault();
    setDragging(true);
    onStart?.(e as DragEvent);
  }, []);

  const onDrag = useCallback(
    (e: DragEvent | RMouseEvent): void => e.preventDefault(),
    []
  );

  const onDragEnd = useCallback(
    (e: DragEvent | RMouseEvent): void => e.preventDefault(),
    []
  );

  useEffect(() => {
    if (!dragging) return;
    const onMouseUp = (e: MouseEvent): void => {
      onEnd?.(e);
      setDragging(false);
    };
    const onMouseMove = (e: MouseEvent): void => {
      e.preventDefault();
      propsOnMouseMove?.(e);
    };
    window.addEventListener("mouseup", onMouseUp);
    window.addEventListener("mousemove", onMouseMove);
    return () => {
      window.removeEventListener("mouseup", onMouseUp);
      window.removeEventListener("mousemove", onMouseMove);
    };
  }, [dragging, propsOnMouseMove]);

  return { onDragStart, onDrag, onDragEnd };
};
