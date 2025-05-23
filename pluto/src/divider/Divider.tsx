// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/divider/Divider.css";

import { type location } from "@synnaxlabs/x";
import { type HTMLAttributes, type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";

/** The props for the {@link Divider} component. */
export interface DividerProps
  extends Omit<HTMLAttributes<HTMLDivElement>, "children">,
    Pick<Align.SpaceProps, "x" | "y" | "direction"> {
  padded?: boolean | location.Location;
  shade?: number;
}

/**
 * Divider renders a vertical or horizontal divided to separate content.
 *
 * @param props - The props for the component.
 * @param props.direction - The direction to render the divider in.
 */
export const Divider = ({
  direction,
  className,
  padded = false,
  shade = 3,
  style,
  ...rest
}: DividerProps): ReactElement => {
  if (padded === true) padded = "center";
  return (
    <Align.Space
      className={CSS(
        CSS.B("divider"),
        padded !== false && CSS.BM("divider", "padded"),
        typeof padded === "string" && CSS.loc(padded),
        className,
      )}
      style={{
        ...style,
        borderColor: CSS.shadeVar(shade),
      }}
      {...rest}
    />
  );
};
