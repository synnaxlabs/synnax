import clsx from "clsx";
import { HTMLAttributes, useEffect, useState } from "react";
import { getDirection, Location } from "../../util/spatial";
import "./ResizePanel.css";

export interface ResizePanelProps extends HTMLAttributes<HTMLDivElement> {
  location: Location;
  initialSize?: number;
  minSize?: number;
  maxSize?: number;
}

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
  }
};

export default function ResizePanel({
  children,
  location,
  minSize = 100,
  maxSize = Infinity,
  initialSize = 200,
  className,
  style,
  ...props
}: ResizePanelProps) {
  const [size, prevSize] = useState(initialSize);
  const [dragging, setDragging] = useState(false);
  const direction = getDirection(location);

  useEffect(() => {
    if (dragging) {
      const onMouseMove = (e: MouseEvent) => {
        prevSize((prevSize: number) => {
          const movement = parseMovement(location, e);
          if (prevSize + movement < minSize) return minSize;
          if (prevSize + movement > maxSize) return maxSize;
          return prevSize + movement;
        });
      };
      const onMouseUp = () => setDragging(false);
      document.addEventListener("mousemove", onMouseMove);
      document.addEventListener("mouseup", onMouseUp);
      return () => {
        document.removeEventListener("mousemove", onMouseMove);
        document.removeEventListener("mouseup", onMouseUp);
      };
    }
  }, [dragging]);

  const parsedStyle: React.CSSProperties = { ...style };
  if (direction === "horizontal") {
    parsedStyle.height = size;
  } else {
    parsedStyle.width = size;
  }

  return (
    <div
      className={clsx(
        "pluto-resize-panel",
        `pluto-resize-panel--${location}`,
        `pluto-resize-panel--${direction}`,
        className
      )}
      style={parsedStyle}
      {...props}
    >
      {children}
      <div
        draggable
        className="pluto-resize-panel__handle"
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
