// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSSProperties, ForwardedRef, ReactElement, forwardRef } from "react";

import { Direction, LooseDirectionT } from "@synnaxlabs/x";

import { CSS } from "@/core/css";
import { Generic, GenericProps } from "@/core/std/Generic";
import { ComponentSize } from "@/util/component";

import "@/core/std/Space/Space.css";

/** All possible alignments for the cross axis of a space */
export const SpaceAlignments = ["start", "center", "end", "stretch"] as const;

/** The alignments for the cross axis of a space */
export type SpaceAlignment = typeof SpaceAlignments[number];

/** All possible justifications for the main axis of a space */
export const SpaceJustifications = [
  "start",
  "center",
  "end",
  "spaceBetween",
  "spaceAround",
  "spaceEvenly",
] as const;

/** The justification for the main axis of a space */
export type SpaceJustification = typeof SpaceJustifications[number];

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
  size?: ComponentSize | number;
  direction?: LooseDirectionT;
  reverse?: boolean;
  justify?: SpaceJustification;
  align?: SpaceAlignment;
  grow?: boolean | number;
  shrink?: boolean | number;
  wrap?: boolean;
  el?: SpaceElementType;
  bordered?: boolean;
  rounded?: boolean;
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
    shrink,
    empty = false,
    size = "medium",
    justify = "start",
    reverse = false,
    direction: direction_ = "y",
    wrap = false,
    bordered = false,
    rounded = false,
    el = "div" as E,
    ...props
  }: SpaceProps<E>,
  ref: ForwardedRef<JSX.IntrinsicElements[E]>
): ReactElement => {
  const direction = new Direction(direction_);

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
  if (shrink != null) style.flexShrink = Number(shrink);

  return (
    // @ts-expect-error
    <Generic<E>
      el={el}
      ref={ref}
      className={CSS(
        CSS.B("space"),
        CSS.dir(direction),
        CSS.bordered(bordered),
        CSS.rounded(rounded),
        typeof size === "string" && CSS.BM("space", size),
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
) => ReactElement;

type FlexDirection = CSSProperties["flexDirection"];

const flexDirection = (direction: Direction, reverse: boolean): FlexDirection => {
  const base = direction.isX ? "row" : "column";
  return reverse ? ((base + "-reverse") as FlexDirection) : base;
};

const justifications: Record<SpaceJustification, CSSProperties["justifyContent"]> = {
  start: "flex-start",
  center: "center",
  end: "flex-end",
  spaceBetween: "space-between",
  spaceAround: "space-around",
  spaceEvenly: "space-evenly",
};
