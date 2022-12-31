// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  CSSProperties,
  ForwardedRef,
  forwardRef,
  HTMLAttributes,
  PropsWithChildren,
} from "react";

import clsx from "clsx";

import { ComponentSize } from "@/util/component";
import { Direction } from "@/util/spatial";
import "./Space.css";

/** The alignments for the cross axis of a space */
export type SpaceAlignment = "start" | "center" | "end" | "stretch";

/** All possible alignments for the cross axis of a space */
export const SpaceAlignments: readonly SpaceAlignment[] = [
  "start",
  "center",
  "end",
  "stretch",
];

/** The justification for the main axis of a space */
export type SpaceJustification =
  | "start"
  | "center"
  | "end"
  | "spaceBetween"
  | "spaceAround"
  | "spaceEvenly";

/** All possible justifications for the main axis of a space */
export const SpaceJustifications: readonly SpaceJustification[] = [
  "start",
  "center",
  "end",
  "spaceBetween",
  "spaceAround",
  "spaceEvenly",
];

export interface SpaceExtensionProps {
  empty?: boolean;
  size?: ComponentSize | number | null;
  direction?: Direction;
  reverse?: boolean;
  justify?: SpaceJustification;
  align?: SpaceAlignment;
  grow?: boolean | number;
}

export interface SpaceProps
  extends PropsWithChildren<HTMLAttributes<HTMLDivElement>>,
    SpaceExtensionProps {}

export const Space = forwardRef(
  (
    {
      empty = false,
      size = "medium",
      justify = "start",
      reverse = false,
      direction = "vertical",
      grow,
      align,
      className,
      style,
      ...props
    }: SpaceProps,
    ref: ForwardedRef<HTMLDivElement>
  ) => {
    let gap: number | string | undefined;
    if (empty) [size, gap] = [0, 0];
    else if (typeof size === "number") gap = `${size ?? 0}rem`;

    style = {
      gap,
      flexDirection: flexDirection(direction, reverse),
      justifyContent: justifications[justify],
      alignItems: align,
      ...style,
    };

    if (grow != null) style.flexGrow = Number(grow);

    return (
      <div
        className={clsx(
          "pluto-space",
          typeof size === "string" && `pluto-space--${size}`,
          `pluto-space--${direction}`,
          className
        )}
        ref={ref}
        style={style}
        {...props}
      />
    );
  }
);
Space.displayName = "Space";

type FlexDirection = CSSProperties["flexDirection"];

const flexDirection = (direction: Direction, reverse: boolean): FlexDirection => {
  const base = direction === "horizontal" ? "row" : "column";
  return reverse ? ((base + "-reverse") as FlexDirection) : base;
};

const justifications = {
  start: "flex-sart",
  center: "center",
  end: "flex-end",
  spaceBetween: "space-between",
  spaceAround: "space-around",
  spaceEvenly: "space-evenly",
};
