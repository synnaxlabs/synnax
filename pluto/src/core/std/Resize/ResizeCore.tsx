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

import { CSS } from "@/core/css";
import { preventDefault } from "@/util/event";

export interface ResizeCoreProps
  extends DetailedHTMLProps<HTMLAttributes<HTMLDivElement>, HTMLDivElement> {
  location: Location;
  size: number;
  onDragStart: (e: React.DragEvent<HTMLDivElement>) => void;
  sizeUnits?: "px" | "%";
  showHandle?: boolean;
}

export const ResizeCore = ({
  location,
  style = {},
  size,
  className,
  children,
  onDragStart,
  sizeUnits = "px",
  showHandle = true,
  ...props
}: ResizeCoreProps): ReactElement => {
  const dir = locToDir(location);
  return (
    <div
      className={CSS(CSS.B("resize"), CSS.loc(location), CSS.dir(dir), className)}
      style={{ [dirToDim(dir)]: `${size}${sizeUnits}`, ...style }}
      {...props}
    >
      {children}
      {showHandle && (
        <div
          draggable
          className={CSS(CSS.BE("resize", "handle"), CSS.bordered(swapLoc(location)))}
          onDragStart={onDragStart}
          onDrag={preventDefault}
          onDragEnd={preventDefault}
        />
      )}
    </div>
  );
};
