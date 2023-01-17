// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DetailedHTMLProps, HTMLAttributes } from "react";

import clsx from "clsx";

import { Location, dirFromLoc, swapLoc } from "@/spatial";

export interface ResizeCoreProps
  extends DetailedHTMLProps<HTMLAttributes<HTMLDivElement>, HTMLDivElement> {
  location: Location;
  size: number;
  showHandle?: boolean;
}

export const ResizeCore = ({
  location,
  style = {},
  size,
  className,
  children,
  onDragStart,
  showHandle = true,
  ...props
}: ResizeCoreProps): JSX.Element => {
  const direction = dirFromLoc(location);
  const parsedStyle: React.CSSProperties = { ...style };
  if (direction === "x") parsedStyle.height = size;
  else parsedStyle.width = size;
  return (
    <div
      className={clsx(
        "pluto-resize-panel",
        `pluto-resize-panel--${location}`,
        `pluto-resize-panel--${direction}`,
        showHandle && `pluto-bordered--${swapLoc(location)}`,
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
