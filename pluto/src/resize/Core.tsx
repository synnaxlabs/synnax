// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/resize/Core.css";

import { direction, location } from "@synnaxlabs/x";
import {
  type DetailedHTMLProps,
  forwardRef,
  type HTMLAttributes,
  type ReactElement,
} from "react";

import { CSS } from "@/css";
import { preventDefault } from "@/util/event";

export interface CoreProps
  extends DetailedHTMLProps<HTMLAttributes<HTMLDivElement>, HTMLDivElement> {
  location: location.Crude;
  size: number;
  onDragStart: (e: React.DragEvent<HTMLDivElement>) => void;
  sizeUnits?: "px" | "%";
  showHandle?: boolean;
}

export const Core = forwardRef<HTMLDivElement, CoreProps>(
  (
    {
      location: cloc,
      style,
      size,
      className,
      children,
      onDragStart,
      sizeUnits = "px",
      showHandle = true,
      ...props
    }: CoreProps,
    ref,
  ): ReactElement => {
    const loc_ = location.construct(cloc);
    const dir = location.direction(loc_);
    const dim = direction.dimension(dir);
    return (
      <div
        className={CSS(CSS.B("resize"), CSS.loc(loc_), CSS.dir(dir), className)}
        style={{ [dim]: `${size}${sizeUnits}`, ...style }}
        ref={ref}
        {...props}
      >
        {children}
        {showHandle && (
          <div
            draggable
            className={CSS(
              CSS.BE("resize", "handle"),
              CSS.bordered(location.swap(loc_)),
            )}
            onDragStart={onDragStart}
            onDrag={preventDefault}
            onDragEnd={preventDefault}
          />
        )}
      </div>
    );
  },
);

Core.displayName = "Resize.Core";
