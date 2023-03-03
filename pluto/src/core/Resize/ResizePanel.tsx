// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DetailedHTMLProps, HTMLAttributes } from "react";

import { Location, locToDir, swapLoc, dirToDim } from "@synnaxlabs/x";
import clsx from "clsx";

import { preventDefault } from "@/util/event";

export interface ResizePanelProps
  extends DetailedHTMLProps<HTMLAttributes<HTMLDivElement>, HTMLDivElement> {
  location: Location;
  size: number;
  onDragStart: (e: React.DragEvent<HTMLDivElement>) => void;
  sizeUnits?: "px" | "%";
  showHandle?: boolean;
}

export const ResizePanel = ({
  location,
  style = {},
  size,
  className,
  children,
  onDragStart,
  sizeUnits = "px",
  showHandle = true,
  ...props
}: ResizePanelProps): JSX.Element => {
  const dir = locToDir(location);
  return (
    <div
      className={clsx(
        "pluto-resize",
        `pluto-resize--${location}`,
        `pluto-resize--${dir}`,
        className
      )}
      style={{
        [dirToDim(dir)]: `${size}${sizeUnits}`,
        ...style,
      }}
      {...props}
    >
      {children}
      {showHandle && (
        <div
          draggable
          className={clsx(
            "pluto-resize__handle",
            showHandle && `pluto-bordered--${swapLoc(location)}`
          )}
          onDragStart={onDragStart}
          onDrag={preventDefault}
          onDragEnd={preventDefault}
        />
      )}
    </div>
  );
};
