// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { HTMLAttributes, PropsWithChildren } from "react";

import clsx from "clsx";

import { Direction } from "@/util/spatial";

import "./Divider.css";

/** The props for the {@link Divider} component. */
export interface DividerProps
  extends PropsWithChildren<HTMLAttributes<HTMLDivElement>> {
  direction?: Direction;
}

/**
 * Divider renders a vertical or horizontal divided to separate content.
 *
 * @param props - The props for the component.
 * @param props.direction - The directio to render the divider in. Can be "horiztonal"
 * or "vertical".
 */
export const Divider = ({
  direction = "vertical",
  className,
  ...props
}: DividerProps): JSX.Element => {
  return (
    <div
      className={clsx("pluto-divider", `pluto-divider--${direction}`, className)}
      {...props}
    />
  );
};
