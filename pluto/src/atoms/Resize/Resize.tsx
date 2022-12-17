import { useEffect, useState } from "react";

import { Location } from "../../util/spatial";

import "./Resize.css";
import { ResizeCore, ResizeCoreProps } from "./ResizeCore";

export interface ResizePanelProps extends Omit<ResizeCoreProps, "showHandle" | "size"> {
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
}: ResizePanelProps): JSX.Element => {
  const [size, setSize] = useState<number>(initialSize);
  const [dragging, setDragging] = useState(false);

  useEffect(() => {
    if (!dragging) return;
    const onMouseMove = (e: MouseEvent): void => {
      setSize((prevSize: number) => {
        return calcNextSize(e, location, prevSize, minSize, maxSize);
      });
      onResize?.(size);
    };
    const onMouseUp = (): void => setDragging(false);
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
    return () => {
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };
  }, [dragging, onResize]);

  return (
    <ResizeCore
      draggable
      location={location}
      size={size}
      onDragStart={(e) => {
        setDragging(true);
        e.preventDefault();
      }}
      onDrag={(e) => e.preventDefault()}
      onDragEnd={(e) => e.preventDefault()}
      {...props}
    />
  );
};

export const parseMovement = (location: Location, e: MouseEvent): number => {
  switch (location) {
    case "top":
      return e.movementY;
    case "bottom":
      return -e.movementY;
    case "left":
      return e.movementX;
    case "right":
      return -e.movementX;
    case "center":
      return 0;
  }
};

export const calcNextSize = (
  e: MouseEvent,
  location: Location,
  prevSize: number,
  minSize: number,
  maxSize: number
): number => {
  const movement = parseMovement(location, e);
  if (prevSize + movement < minSize) return minSize;
  if (prevSize + movement > maxSize) return maxSize;
  return prevSize + movement;
};

export const anyExceedsBounds = (nums: number[], min: number, max: number): boolean => {
  return nums.some((num) => num < min || num > max);
};
