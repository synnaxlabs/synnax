// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/icon/Icon.css";

import { color } from "@synnaxlabs/x";
import type { FC as ReactFC, ReactElement as BaseReactElement, Ref } from "react";
import type { IconBaseProps } from "react-icons";

import { CSS } from "@/css";
import type { Theming } from "@/theming";

/** Props for every icon in the set. */
export interface IconProps extends Omit<IconBaseProps, "color" | "children"> {
  ref?: Ref<SVGSVGElement>;
  /** Fill color. A number selects that step on the theme gray scale. */
  color?: color.Crude | Theming.Shade;
}

/**
 * A type representing a React element that renders an icon. This element encapsulates
 * the visual representation of an icon component, typically rendered as an SVG with the
 * specified base properties.
 */
export interface ReactElement extends BaseReactElement<IconProps> {}

/** An icon component. Every member of the icon set has this type. */
export interface FC extends ReactFC<IconProps> {}

interface WrapIconOpts {
  className?: string;
}

/**
 * A raw SVG component, as generated from an icon file. Wrap it with
 * {@link wrapSVGIcon}.
 */
export interface SVGFC extends ReactFC<IconBaseProps> {}

const parseColor = (c?: color.Crude | Theming.Shade): string | undefined => {
  if (typeof c === "number") return `var(--pluto-gray-l${c})`;
  return color.cssString(c);
};

/**
 * Turns a raw SVG component into an {@link FC}, applying the icon class names and
 * theme-aware color parsing. Icons are decorative by default: without an explicit
 * `aria-label` they are hidden from the accessibility tree, so they never leak into
 * an ancestor's accessible name.
 */
// This and the factories below are marked pure, so a bundler drops unused icons.
/* #__NO_SIDE_EFFECTS__ */
export const wrapSVGIcon = (
  Base: SVGFC,
  name: string,
  { className }: WrapIconOpts = {},
): FC => {
  const typeClass = CSS.BM("icon", name);
  const O: FC = ({ className: pClassName, color: c, ...rest }) => {
    c = parseColor(c);
    return (
      <Base
        className={CSS.cls(CSS.B("icon"), pClassName, className, typeClass)}
        aria-hidden={rest["aria-label"] == null ? true : undefined}
        color={c}
        {...rest}
      />
    );
  };
  O.displayName = Base.displayName || Base.name;
  return O;
};

/**
 * @returns props that draw an icon in a user-picked color, with the lightness moved
 * into a band that keeps 3:1 contrast on the theme's surfaces. The color's alpha is
 * ignored. Without a color, the icon keeps the color it inherits.
 *
 * @example <Glyph {...Icon.tint(panel.color)} />
 */
export const tint = (c?: color.Crude): Pick<IconProps, "className" | "style"> => {
  if (c == null) return {};
  return {
    className: CSS.cls(CSS.symbolColored, CSS.BM("symbol-colored", "icon")),
    // Inline, so the pick outranks the default icon colors that tabs and trees set.
    style: {
      [CSS.variable("symbol-color")]: color.rgbString(c),
      color: "var(--pluto-symbol-display)",
    },
  };
};

/**
 * Returns an icon that fills one SVG path drawn on a 24 by 24 grid.
 *
 * @param name - The icon's name, added to its class names.
 * @param d - The path data.
 */
/* #__NO_SIDE_EFFECTS__ */
export const createFromPath = (name: string, d: string): FC =>
  wrapSVGIcon(
    (props) => (
      <svg
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg"
        stroke="currentColor"
        fill="currentColor"
        strokeWidth={0}
        {...props}
      >
        <path d={d} />
      </svg>
    ),
    name,
  );
