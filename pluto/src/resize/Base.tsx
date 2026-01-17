// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/resize/Base.css";

import { direction, location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { preventDefault } from "@/util/event";

export type BaseProps = Omit<
  Flex.BoxProps<"div">,
  "gap" | "size" | "direction" | "x" | "y"
> & {
  location: location.Crude;
  size: number;
  onDragStart: (e: React.DragEvent<HTMLDivElement>) => void;
  sizeUnits?: "px" | "%";
  showHandle?: boolean;
};

export const Base = ({
  ref,
  location: cloc,
  style,
  size,
  className,
  children,
  onDragStart,
  sizeUnits = "px",
  showHandle = true,
  ...rest
}: BaseProps): ReactElement => {
  const parsedLocation = location.construct(cloc);
  const dir = location.direction(parsedLocation);
  const dim = direction.dimension(dir);
  return (
    <Flex.Box
      className={CSS(CSS.B("resize"), CSS.loc(parsedLocation), className)}
      style={{ [dim]: `${size}${sizeUnits}`, ...style }}
      ref={ref}
      direction={dir}
      empty
      {...rest}
    >
      {children}
      {showHandle && (
        <div
          draggable
          className={CSS(
            CSS.BE("resize", "handle"),
            CSS.bordered(location.swap(parsedLocation)),
          )}
          onDragStart={onDragStart}
          onDrag={preventDefault}
          onDragEnd={preventDefault}
        />
      )}
    </Flex.Box>
  );
};
