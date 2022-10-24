import clsx from "clsx";
import { HTMLAttributes, useEffect, useRef, useState } from "react";
import { getDirection, Location, swapLocation } from "../../util/spatial";
import "./Resize.css";
import ResizeMultiple from "./ResizeMultiple";

export interface ResizePanelProps extends HTMLAttributes<HTMLDivElement> {
  location: Location;
  initialSize?: number;
  minSize?: number;
  maxSize?: number;
  onResize?: (size: number) => void;
}
function Resize({
  children,
  location = "left",
  minSize = 100,
  maxSize = Infinity,
  initialSize = 200,
  onResize,
  className,
  style,
  ...props
}: ResizePanelProps) {
  const [size, setSize] = useState<number>(initialSize);
  const [dragging, setDragging] = useState(false);
  const direction = getDirection(location);

  useEffect(() => {
    if (!dragging) return;
    const onMouseMove = (e: MouseEvent) => {
      setSize((prevSize: number) => {
        return calcNextSize(e, location, prevSize, minSize, maxSize);
      });
      onResize?.(size);
    };
    const onMouseUp = () => setDragging(false);
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
    return () => {
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };
  }, [dragging, onResize]);

  const parsedStyle: React.CSSProperties = { ...style };
  if (direction === "horizontal") {
    parsedStyle.height = size || initialSize;
  } else {
    parsedStyle.width = size || initialSize;
  }

  return (
    <div
      className={clsx(
        "pluto-resize-panel",
        `pluto-resize-panel--${location}`,
        `pluto-resize-panel--${direction}`,
        `pluto-bordered--${swapLocation(location)}`,
        className
      )}
      style={parsedStyle}
      {...props}
    >
      {children}
      <div
        draggable
        className="pluto-resize-panel__handle"
        data-testid="resize-handle"
        onDragStart={(e) => {
          setDragging(true);
          e.preventDefault();
        }}
        onDrag={(e) => e.preventDefault()}
        onDragEnd={(e) => e.preventDefault()}
      ></div>
    </div>
  );
}

Resize.Multiple = ResizeMultiple;

export default Resize;

const parseMovement = (location: Location, e: MouseEvent) => {
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
) => {
  const movement = parseMovement(location, e);
  if (prevSize + movement < minSize) return minSize;
  if (prevSize + movement > maxSize) return maxSize;
  return prevSize + movement;
};
