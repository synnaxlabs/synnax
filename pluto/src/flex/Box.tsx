// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/flex/Box.css";

import { type color } from "@synnaxlabs/x";
import { direction } from "@synnaxlabs/x/spatial";
import { type ReactElement } from "react";
import z from "zod";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { Generic } from "@/generic";
import { type Theming } from "@/theming";

/** All possible alignments for the cross axis of a space */
export const ALIGNMENTS = ["start", "center", "end", "stretch"] as const;
export const alignmentZ = z.enum(ALIGNMENTS);

/** The alignments for the cross axis of a space */
export type Alignment = z.infer<typeof alignmentZ>;

/** All possible justifications for the main axis of a space */
export const JUSTIFICATIONS = [
  "start",
  "center",
  "end",
  "between",
  "around",
  "evenly",
] as const;
export const justificationZ = z.enum(JUSTIFICATIONS);

/** The justification for the main axis of a space */
export type Justification = z.infer<typeof justificationZ>;

export type BoxProps<E extends Generic.ElementType = "div"> = Omit<
  Generic.OptionalElementProps<E>,
  "color"
> &
  BoxExtensionProps;

export interface BoxExtensionProps {
  // border
  bordered?: boolean;
  borderColor?: Theming.Shade | color.Crude | false;
  borderWidth?: number;
  rounded?: boolean | number;
  sharp?: boolean;
  // background
  background?: Theming.Shade;
  // gap
  empty?: boolean;
  gap?: Component.Size | number;
  // direction
  direction?: direction.Crude;
  x?: boolean;
  y?: boolean;
  reverse?: boolean;
  // popsitioning
  justify?: Justification;
  align?: Alignment;
  // sizing
  grow?: boolean | number;
  shrink?: boolean | number;
  center?: boolean;
  full?: boolean | direction.Direction;
  // wrapping
  wrap?: boolean;
  // sizing
  size?: Component.Size;
  pack?: boolean;
  color?: Theming.Shade | color.Crude | false;
}

export const shouldReverse = (direction: direction.Crude, reverse?: boolean): boolean =>
  reverse ?? (direction === "right" || direction === "bottom");

export const parseDirection = (
  dir?: direction.Crude,
  x?: boolean,
  y?: boolean,
  pack?: boolean,
): direction.Direction => {
  if (x === true) return "x";
  if (y === true) return "y";
  if (dir != null) return direction.construct(dir);
  return pack ? "x" : "y";
};

const parseFull = (full?: boolean | direction.Direction): string | false => {
  if (full == null || full === false) return false;
  if (full === true) return CSS.BM("flex", "full");
  return CSS.BM("flex", `full-${full}`);
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
export const Box = <E extends Generic.ElementType = "div">({
  style,
  align,
  className,
  grow,
  shrink,
  gap,
  color,
  justify,
  reverse,
  empty = false,
  pack = false,
  wrap = false,
  center = false,
  direction: propsDir,
  rounded,
  sharp,
  borderWidth,
  borderColor,
  full,
  background,
  bordered,
  x,
  y,
  size,
  ...rest
}: BoxProps<E>): ReactElement => {
  const dir = parseDirection(propsDir, x, y, pack);
  const classNames = [
    CSS.BE("flex", "box"),
    CSS.dir(dir),
    pack && CSS.BM("flex", "pack"),
    shouldReverse(dir, reverse) && CSS.M("reverse"),
    justify != null && CSS.BM("justify", justify),
    align != null && CSS.BM("align", align),
    wrap && CSS.M("wrap"),
    empty && CSS.M("empty"),
    center && CSS.BM("flex", "center"),
    parseFull(full),
    bordered != null && CSS.bordered(bordered),
    rounded != null && rounded !== false && CSS.rounded(true),
    sharp != null && sharp !== false && CSS.M("sharp"),
    size != null && CSS.height(size),
    className,
  ];
  style = {
    borderWidth,
    borderColor: CSS.colorVar(borderColor),
    ...style,
  };
  if (rounded != null && typeof rounded === "number")
    // @ts-expect-error - CSS.var returns a string, but we're using it as a CSS property
    style[CSS.var("flex-border-radius")] = `${rounded}rem`;

  if (typeof gap === "number") style.gap = `${gap}rem`;
  else if (gap != null) classNames.push(CSS.BM("gap", gap));
  if (grow != null) style.flexGrow = Number(grow);
  if (shrink != null) style.flexShrink = Number(shrink);
  if (background != null) style.backgroundColor = CSS.colorVar(background);
  if (color != null && color !== false) style.color = CSS.colorVar(color);
  return (
    <Generic.Element<E>
      className={CSS(...classNames)}
      style={style}
      {...(rest as Generic.ElementProps<E>)}
      el={rest.el ?? "div"}
    />
  );
};
