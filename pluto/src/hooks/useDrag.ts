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
