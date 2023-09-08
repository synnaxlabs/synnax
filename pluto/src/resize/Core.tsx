// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type DetailedHTMLProps, type HTMLAttributes, type ReactElement } from "react";

import { Location, type LooseLocationT } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { preventDefault } from "@/util/event";

import "@/resize/Core.css";

export interface CoreProps
  extends DetailedHTMLProps<HTMLAttributes<HTMLDivElement>, HTMLDivElement> {
  location: LooseLocationT;
  size: number;
  onDragStart: (e: React.DragEvent<HTMLDivElement>) => void;
  sizeUnits?: "px" | "%";
  showHandle?: boolean;
}

export const Core = ({
  location: loc_,
  style,
  size,
  className,
  children,
  onDragStart,
  sizeUnits = "px",
  showHandle = true,
  ...props
}: CoreProps): ReactElement => {
  const loc = new Location(loc_);
  return (
    <div
      className={CSS(CSS.B("resize"), CSS.loc(loc), CSS.dir(loc.direction), className)}
      style={{ [loc.direction.dimension]: `${size}${sizeUnits}`, ...style }}
      {...props}
    >
      {children}
      {showHandle && (
        <div
          draggable
          className={CSS(CSS.BE("resize", "handle"), CSS.bordered(loc.inverse.crude))}
          onDragStart={onDragStart}
          onDrag={preventDefault}
          onDragEnd={preventDefault}
        />
      )}
    </div>
  );
};
