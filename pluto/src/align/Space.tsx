// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/align/Space.css";

import { direction } from "@synnaxlabs/x/spatial";
import { type CSSProperties, type ReactElement } from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { Generic } from "@/generic";
import { type Theming } from "@/theming";

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

const CSS_JUSTIFICATIONS: Record<Justification, CSSProperties["justifyContent"]> = {
  start: "flex-start",
  center: "center",
  end: "flex-end",
  spaceBetween: "space-between",
  spaceAround: "space-around",
  spaceEvenly: "space-evenly",
};

/** The justification for the main axis of a space */
export type Justification = (typeof JUSTIFICATIONS)[number];

export type ElementType =
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
  | "form"
  | "main";

export interface CoreExtensionProps {
  el?: ElementType;
  bordered?: boolean;
  borderShade?: Theming.Shade;
  borderWidth?: number;
  rounded?: boolean | number;
  background?: Theming.Shade;
}

export interface SpaceExtensionProps extends CoreExtensionProps {
  empty?: boolean;
  gap?: Component.Size | number;
  direction?: direction.Crude;
  x?: boolean;
  y?: boolean;
  reverse?: boolean;
  justify?: Justification;
  align?: Alignment;
  grow?: boolean | number;
  shrink?: boolean | number;
  wrap?: boolean;
}

export type SpaceProps<E extends ElementType = "div"> = Omit<
  Generic.ElementProps<E>,
  "el"
> &
  SpaceExtensionProps;

export const shouldReverse = (direction: direction.Crude, reverse?: boolean): boolean =>
  reverse ?? (direction === "right" || direction === "bottom");

type FlexDirection = CSSProperties["flexDirection"];

const flexDirection = (
  direction: direction.Direction,
  reverse: boolean,
): FlexDirection => {
  const base = direction === "x" ? "row" : "column";
  return reverse ? (`${base}-reverse` as FlexDirection) : base;
};

export const parseDirection = (
  dir?: direction.Crude,
  x?: boolean,
  y?: boolean,
  def: direction.Direction = "y",
): direction.Direction => {
  if (x) return "x";
  if (y) return "y";
  if (dir != null) return direction.construct(dir);
  return def;
};

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
 * @param props.direction - The direction of the space. Defaults to 'y'. If props.x or
 * props.y are true, this prop is ignored.
 * @param props.x - A boolean value that determines if the space should be oriented
 * horizontally. If true, props.y and props.direction are ignored.
 * @param props.y - A boolean value that determines if the space should be oriented
 * vertically. If true, props.direction is ignored. props.x takes precedence over props.y.
 * @param props.el - The element type to render as. Defaults to 'div'.
 */
export const Space = <E extends ElementType = "div">({
  style,
  align,
  className,
  grow,
  shrink,
  empty = false,
  gap = "medium",
  justify = "start",
  reverse,
  wrap = false,
  direction: propsDir,
  x,
  y,
  ...rest
}: SpaceProps<E>): ReactElement => {
  const dir = parseDirection(propsDir, x, y, "y");
  reverse = shouldReverse(dir, reverse);

  let parsedGap: number | string | undefined;
  if (empty) [parsedGap, parsedGap] = [0, 0];
  else if (typeof gap === "number") parsedGap = `${gap}rem`;

  style = {
    gap: parsedGap,
    flexDirection: flexDirection(dir, reverse),
    justifyContent: CSS_JUSTIFICATIONS[justify],
    alignItems: align,
    flexWrap: wrap ? "wrap" : "nowrap",
    ...style,
  };

  if (grow != null) style.flexGrow = Number(grow);
  if (shrink != null) style.flexShrink = Number(shrink);

  return (
    // @ts-expect-error - TODO: fix generic element props
    <Core<E>
      className={CSS(
        CSS.B("space"),
        CSS.dir(dir),
        typeof gap === "string" && CSS.BM("space", gap),
        className,
      )}
      style={style}
      {...rest}
    />
  );
};

export type CoreProps<E extends ElementType = "div"> = Omit<
  Generic.ElementProps<E>,
  "el"
> &
  CoreExtensionProps & { el?: E };

export const Core = <E extends ElementType = "div">({
  style,
  el = "div" as E,
  bordered = false,
  borderShade,
  borderWidth,
  rounded = false,
  background,
  className,
  ...rest
}: CoreProps<E>): ReactElement => {
  let borderRadius: string | undefined;
  if (rounded != null && typeof rounded === "number") borderRadius = `${rounded}rem`;
  style = {
    borderWidth,
    borderColor: CSS.shadeVar(borderShade),
    borderRadius,
    ...style,
  };
  if (background != null) style.backgroundColor = CSS.shadeVar(background);

  return (
    // @ts-expect-error - TODO: fix generic element props
    <Generic.Element<E>
      el={el}
      className={CSS(
        CSS.bordered(bordered),
        typeof rounded === "boolean" && CSS.rounded(rounded),
        className,
      )}
      style={style}
      {...rest}
    />
  );
};
