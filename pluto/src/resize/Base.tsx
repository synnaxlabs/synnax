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
import { type CSSProperties, type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Cursor } from "@/cursor";
import { Flex } from "@/flex";

export interface BaseProps extends Omit<
  Flex.BoxProps,
  "gap" | "size" | "direction" | "x" | "y"
> {
  location: location.Crude;
  size: number;
  decimal?: boolean;
  hideHandle?: boolean;
}

export const Base = ({
  ref,
  location: cloc,
  style: propsStyle,
  size,
  className,
  children,
  onPointerDown,
  decimal = false,
  hideHandle = false,
  ...rest
}: BaseProps): ReactElement => {
  const parsedLocation = location.construct(cloc);
  const dir = location.direction(parsedLocation);
  const style = useMemo<CSSProperties>(() => {
    const dim = direction.dimension(dir);
    const cssSize = decimal ? `${size * 100}%` : `${size}px`;
    return { [dim]: cssSize, ...propsStyle };
  }, [decimal, size, dir, propsStyle]);
  return (
    <Flex.Box
      className={CSS(CSS.B("resize"), CSS.loc(parsedLocation), CSS.dir(dir), className)}
      full={direction.swap(dir)}
      style={style}
      ref={ref}
      direction={dir}
      empty
      {...rest}
    >
      {children}
      {!hideHandle && (
        <div
          className={CSS(
            CSS.BE("resize", "handle"),
            CSS.bordered(location.swap(parsedLocation)),
            Cursor.DRAG_CLASS,
          )}
          onPointerDown={onPointerDown}
        />
      )}
    </Flex.Box>
  );
};
