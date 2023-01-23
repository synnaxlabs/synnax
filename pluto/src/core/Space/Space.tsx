// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSSProperties, ForwardedRef, forwardRef } from "react";

import clsx from "clsx";

import { Generic, GenericProps } from "../Generic";

import { Direction } from "@/spatial";
import { ComponentSize } from "@/util/component";

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

export type SpaceElementType =
  | "div"
  | "header"
  | "nav"
  | "section"
  | "article"
  | "aside"
  | "footer"
  | "button"
  | "dialog";

export interface SpaceExtensionProps {
  empty?: boolean;
  size?: ComponentSize | number | null;
  direction?: Direction;
  reverse?: boolean;
  justify?: SpaceJustification;
  align?: SpaceAlignment;
  grow?: boolean | number;
  wrap?: boolean;
  el?: SpaceElementType;
}

export type SpaceProps<E extends SpaceElementType = "div"> = Omit<
  GenericProps<E>,
  "el"
> &
  SpaceExtensionProps;

const CoreSpace = <E extends SpaceElementType = "div">(
  {
    style,
    align,
    className,
    grow,
    empty = false,
    size = "medium",
    justify = "start",
    reverse = false,
    direction = "y",
    wrap = false,
    el = "div" as E,
    ...props
  }: SpaceProps<E>,
  ref: ForwardedRef<JSX.IntrinsicElements[E]>
): JSX.Element => {
  let gap: number | string | undefined;
  if (empty) [size, gap] = [0, 0];
  else if (typeof size === "number") gap = `${size}rem`;

  style = {
    gap,
    flexDirection: flexDirection(direction, reverse),
    justifyContent: justifications[justify],
    alignItems: align,
    flexWrap: wrap ? "wrap" : "nowrap",
    ...style,
  };

  if (grow != null) style.flexGrow = Number(grow);

  return (
    // @ts-expect-error
    <Generic<E>
      el={el}
      ref={ref}
      className={clsx(
        "pluto-space",
        typeof size === "string" && `pluto-space--${size}`,
        `pluto-space--${direction}`,
        className
      )}
      style={style}
      {...props}
    />
  );
};
CoreSpace.displayName = "Space";

export const Space = forwardRef(CoreSpace) as <E extends SpaceElementType = "div">(
  props: SpaceProps<E>
) => JSX.Element;

type FlexDirection = CSSProperties["flexDirection"];

const flexDirection = (direction: Direction, reverse: boolean): FlexDirection => {
  const base = direction === "x" ? "row" : "column";
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
