// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/align/Space.css";

import { direction } from "@synnaxlabs/x/spatial";
import {
  type CSSProperties,
  type ForwardedRef,
  forwardRef,
  type ReactElement,
} from "react";

import { CSS } from "@/css";
import { Generic } from "@/generic";
import { type Theming } from "@/theming";
import { type ComponentSize } from "@/util/component";

/** All possible alignments for the cross axis of a space */
export const ALIGNMENTS = ["start", "center", "end", "stretch"] as const;

/** The alignments for the cross axis of a space */
export type Alignment = (typeof ALIGNMENTS)[number];

/** All possible justifications for the main axis of a space */
export const JUSTIFICATIONS = [
  "start",
  "center",
  "end",
  "spaceBetween",
  "spaceAround",
  "spaceEvenly",
] as const;

/** The justification for the main axis of a space */
export type Justification = (typeof JUSTIFICATIONS)[number];

export type SpaceElementType =
  | "div"
  | "header"
  | "nav"
  | "section"
  | "article"
  | "aside"
  | "footer"
  | "button"
  | "dialog"
  | "a"
  | "form";

export interface SpaceExtensionProps {
  empty?: boolean;
  size?: ComponentSize | number;
  direction?: direction.Crude;
  reverse?: boolean;
  justify?: Justification;
  align?: Alignment;
  grow?: boolean | number;
  shrink?: boolean | number;
  wrap?: boolean;
  el?: SpaceElementType;
  bordered?: boolean;
  borderShade?: Theming.Shade;
  rounded?: boolean;
  background?: Theming.Shade;
}

export type SpaceProps<E extends SpaceElementType = "div"> = Omit<
  Generic.ElementProps<E>,
  "el"
> &
  SpaceExtensionProps;

export const shouldReverse = (direction: direction.Crude, reverse?: boolean): boolean =>
  reverse ?? (direction === "right" || direction === "bottom");

const CoreSpace = <E extends SpaceElementType>(
  {
    style,
    align,
    className,
    grow,
    shrink,
    empty = false,
    size = "medium",
    justify = "start",
    reverse,
    direction: direction_ = "y",
    wrap = false,
    bordered = false,
    borderShade,
    rounded = false,
    el = "div",
    background,
    ...props
  }: SpaceProps<E>,
  ref: ForwardedRef<JSX.IntrinsicElements[E]>,
): ReactElement => {
  const dir = direction.construct(direction_);
  reverse = shouldReverse(direction_, reverse);

  let gap: number | string | undefined;
  if (empty) [size, gap] = [0, 0];
  else if (typeof size === "number") gap = `${size}rem`;

  style = {
    gap: `${gap} ${gap}`,
    flexDirection: flexDirection(dir, reverse),
    justifyContent: justifications[justify],
    alignItems: align,
    flexWrap: wrap ? "wrap" : "nowrap",
    ...style,
  };

  if (grow != null) style.flexGrow = Number(grow);
  if (shrink != null) style.flexShrink = Number(shrink);
  if (background != null) style.backgroundColor = CSS.shadeVar(background);

  return (
    // @ts-expect-error - TODO: fix generic element props
    <Generic.Element<E>
      el={el}
      ref={ref}
      className={CSS(
        CSS.B("space"),
        CSS.dir(dir),
        CSS.bordered(bordered),
        CSS.rounded(rounded),
        typeof size === "string" && CSS.BM("space", size),
        className,
      )}
      style={style}
      {...props}
    />
  );
};
CoreSpace.displayName = "Space";

/**
 * A component that orients its children in a row or column and adds
 * space between them. This is essentially a thin wrapped around a
 * flex component that makes it more 'reacty' to use.
 *
 * @param props - The props for the component. All unlisted props will be passed
 * to the underlying root element.
 * @param props.align - The off axis alignment of the children. The 'off' axis is the
 * opposite direction of props.direction. For example, if direction is 'x', then the
 * off axis is 'y'. See the {@link Alignment} for available options.
 * @param props.justify - The main axis justification of the children. The 'main' axis
 * is the same direction as props.direction. For example, if direction is 'x', then the
 * main axis is 'x'. See the {@link Justification} for available options.
 * @param props.grow - A boolean or number value that determines if the space should
 * grow in the flex-box sense. A value of true will set css flex-grow to 1. A value of
 * false will leave the css flex-grow unset. A number value will set the css flex-grow
 * to that number.
 * @param props.size - A string or number value that determines the amount of spacing
 * between items. If set to "small", "medium", or "large", the spacing will be determined
 * by the theme. If set to a number, the spacing will be that number of rem.
 * @param props.wrap - A boolean value that determines if the space should wrap its
 * children.
 * @param props.el - The element type to render as. Defaults to 'div'.
 */
export const Space = forwardRef(CoreSpace) as <E extends SpaceElementType = "div">(
  props: SpaceProps<E>,
) => ReactElement;

type FlexDirection = CSSProperties["flexDirection"];

const flexDirection = (
  direction: direction.Direction,
  reverse: boolean,
): FlexDirection => {
  const base = direction === "x" ? "row" : "column";
  return reverse ? (`${base}-reverse` as FlexDirection) : base;
};

const justifications: Record<Justification, CSSProperties["justifyContent"]> = {
  start: "flex-start",
  center: "center",
  end: "flex-end",
  spaceBetween: "space-between",
  spaceAround: "space-around",
  spaceEvenly: "space-evenly",
};
