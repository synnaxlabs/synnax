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
  background?: Theming.Shade;

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
  square = false,
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
    wrap === true && CSS.M("wrap"),
    empty === true && CSS.M("empty"),
    center === true && CSS.BM("flex", "center"),
    parseFull(full),
    bordered === true && CSS.bordered(bordered),
    rounded === true && CSS.rounded(true),
    sharp === true && CSS.M("sharp"),
    size != null && CSS.height(size),
    square === true && CSS.BM("flex", "square"),
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
