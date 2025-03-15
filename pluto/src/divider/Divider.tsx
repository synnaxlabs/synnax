// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/divider/Divider.css";

import { type direction, type location } from "@synnaxlabs/x/spatial";
import { type HTMLAttributes, type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/css";

/** The props for the {@link Divider} component. */
export interface DividerProps
  extends PropsWithChildren<HTMLAttributes<HTMLDivElement>> {
  direction?: direction.Direction;
  padded?: boolean | location.Location;
}

/**
 * Divider renders a vertical or horizontal divided to separate content.
 *
 * @param props - The props for the component.
 * @param props.direction - The direction to render the divider in.
 */
export const Divider = ({
  direction = "y",
  className,
  padded = false,
  ...rest
}: DividerProps): ReactElement => {
  if (padded === true) padded = "center";
  return (
    <div
      className={CSS(
        CSS.B("divider"),
        CSS.dir(direction),
        padded !== false && CSS.BM("divider", "padded"),
        typeof padded === "string" && CSS.loc(padded),
        className,
      )}
      {...rest}
    />
  );
};
