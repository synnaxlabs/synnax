import { DetailedHTMLProps, HTMLAttributes } from "react";

import clsx from "clsx";

import { Location, getDirection, swapLocation } from "@/util";

export interface ResizeCoreProps
  extends DetailedHTMLProps<HTMLAttributes<HTMLDivElement>, HTMLDivElement> {
  location: Location;
  size: number;
  showHandle?: boolean;
}

export const ResizeCore = ({
  location,
  style,
  size,
  className,
  children,
  onDragStart,
  showHandle = true,
  ...props
}: ResizeCoreProps): JSX.Element => {
  const direction = getDirection(location);
  const parsedStyle: React.CSSProperties = { ...style, overflow: "hidden" };
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
        showHandle && `pluto-bordered--${swapLocation(location)}`,
        className
      )}
      style={parsedStyle}
      {...props}
    >
      {children}
      {showHandle && (
        <div
          draggable
          className="pluto-resize-panel__handle"
          data-testid="resize-handle"
          onDragStart={(e) => {
            e.preventDefault();
            onDragStart?.(e);
          }}
          onDrag={(e) => e.preventDefault()}
          onDragEnd={(e) => e.preventDefault()}
        ></div>
      )}
    </div>
  );
};
