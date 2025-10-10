// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/flex/Box.css";

import { type color, direction } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";
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

/**
 * Props for the Box component. Extends generic element props with flex layout
 * capabilities.
 *
 * @template E - The HTML element type to render as
 *
 * @example
 * ```tsx
 * // Basic vertical layout with spacing
 * <Box gap="medium">
 *   <div>Item 1</div>
 *   <div>Item 2</div>
 * </Box>
 *
 * // Horizontal centered layout with border
 * <Box x center bordered rounded>
 *   <Button>Action</Button>
 *   <Button>Cancel</Button>
 * </Box>
 *
 * // Full-width container with custom styling
 * <Box full background={11} gap={1.5}>
 *   <Text.H3>Title</Text.H3>
 *   <Text.P>Content goes here</Text.P>
 * </Box>
 * ```
 */
export type BoxProps<E extends Generic.ElementType = "div"> = Omit<
  Generic.OptionalElementProps<E>,
  "color"
> &
  BoxExtensionProps;

/**
 * Extension props that provide flex layout capabilities to the Box component.
 * These props control layout direction, alignment, spacing, styling, and sizing.
 */
export interface BoxExtensionProps {
  /** Whether to show a border around the container */
  bordered?: boolean;
  /** Border color using Theming.Shade, color.Crude, or false to hide */
  borderColor?: Theming.Shade | color.Crude | false;
  /** Border width in pixels */
  borderWidth?: number;
  /** Border radius. true for default rounding, number for specific rem value */
  rounded?: boolean | number;
  /** Whether to remove border radius (sharp corners) */
  sharp?: boolean;

  /** Background color using Theming.Shade values */
  background?: Theming.Shade | color.Crude | false;

  /** Whether the container has no visible spacing/padding */
  empty?: boolean;
  /** Spacing between children. Can be a Component.Size ('small', 'medium', 'large') or
   * a number (in rem units) */
  gap?: Component.Size | number;

  /** The flex direction of the container. Defaults to 'y' (column). Can be 'x', 'y',
   * 'left', 'right', 'top', 'bottom' */
  direction?: direction.Crude;
  /** Shorthand for setting direction to 'x' (row). Overrides direction */
  x?: boolean;
  /** Shorthand for setting direction to 'y' (column). Overrides direction if x is
   * not set */
  y?: boolean;
  /** Whether to reverse the direction of children */
  reverse?: boolean;

  /** Main-axis justification of children. See {@link Justification} for options:
   * 'start', 'center', 'end', 'between', 'around', 'evenly' */
  justify?: Justification;
  /** Cross-axis alignment of children. See {@link Alignment} for options: 'start',
   * 'center', 'end', 'stretch' */
  align?: Alignment;

  /** Flex grow behavior. true sets flex-grow: 1, false leaves unset, number sets
   * specific flex-grow value */
  grow?: boolean | number;
  /** Flex shrink behavior. true sets flex-shrink: 1, false leaves unset, number
   * sets specific flex-shrink value */
  shrink?: boolean | number;
  /** Shorthand for centering both axes (align: 'center', justify: 'center') */
  center?: boolean;
  /** Whether the container should take full width/height. true for both, or specify
   * a direction ('x', 'y') for single axis */
  full?: boolean | direction.Direction;

  /** Whether children should wrap to new lines when they overflow */
  wrap?: boolean;

  /** Height of the container using Component.Size values */
  size?: Component.Size;
  /**
   * Whether to use horizontal layout optimized for packing items into
   * visually consistent groups.
   */
  pack?: boolean;
  /** Text color using Theming.Shade, color.Crude, or false to inherit */
  color?: Theming.Shade | color.Crude | false;
  /** Whether the container should maintain square aspect ratio */
  square?: boolean;
}

export const shouldReverse = (
  direction?: direction.Crude,
  reverse?: boolean,
): boolean => {
  if (reverse != null) return reverse;
  return direction === "right" || direction === "bottom";
};

export const parseDirection = (
  dir?: direction.Crude,
  x?: boolean,
  y?: boolean,
  pack?: boolean,
): direction.Direction | undefined => {
  if (x === true) return "x";
  if (y === true) return "y";
  if (dir != null) return direction.construct(dir);
  if (pack == true) return "x";
  return undefined;
};

const parseFull = (full?: boolean | direction.Direction): string | false => {
  if (full == null || full === false) return false;
  if (full === true) return CSS.M("full");
  return CSS.M("full", full);
};

/**
 * A flexible container component that arranges its children using CSS flexbox.
 * See {@link BoxProps} for all available props and examples.
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
  empty,
  pack,
  wrap,
  center,
  direction: crudeDirection,
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
  square = false,
  ...rest
}: BoxProps<E>): ReactElement => {
  const parsedDirection = parseDirection(crudeDirection, x, y, pack);
  const classNames = [
    className,
    CSS.B("flex"),
    parsedDirection != null && CSS.M("direction", parsedDirection),
    shouldReverse(crudeDirection, reverse) && CSS.M("reverse"),
    parseFull(full),
    pack && CSS.M("pack"),
    justify != null && CSS.M("justify", justify),
    align != null && CSS.M("align", align),
    wrap === true && CSS.M("wrap"),
    empty === true && CSS.M("empty"),
    center === true && CSS.M("center"),
    bordered === true && CSS.M("bordered"),
    sharp === true && CSS.M("sharp"),
    size != null && CSS.M("height", size),
    square === true && CSS.M("square"),
  ];
  style = { borderWidth, ...style } as CSSProperties;

  if (typeof color === "number") classNames.push(CSS.M("color", color.toString()));
  else if (color != null && color !== false) style.color = CSS.colorVar(color);

  if (typeof background === "number")
    classNames.push(CSS.M("bg", background.toString()));
  else if (background != null && background !== false)
    style.backgroundColor = CSS.colorVar(background);

  if (typeof borderColor === "number")
    classNames.push(CSS.M("border-color", borderColor.toString()));
  else if (borderColor != null) style.borderColor = CSS.colorVar(borderColor);

  if (rounded === true) classNames.push(CSS.M("rounded"));
  else if (typeof rounded === "number") style.borderRadius = `${rounded}rem`;

  if (typeof gap === "number") style.gap = `${gap}rem`;
  else if (gap != null) classNames.push(CSS.M("gap", gap));

  if (typeof grow === "number") style.flexGrow = grow;
  else if (grow === true) classNames.push(CSS.M("grow"));

  if (typeof shrink === "number") style.flexShrink = shrink;
  else if (shrink === true) classNames.push(CSS.M("shrink"));

  return (
    <Generic.Element<E>
      className={CSS(...classNames)}
      style={style}
      {...(rest as Generic.ElementProps<E>)}
      el={rest.el ?? "div"}
    />
  );
};
