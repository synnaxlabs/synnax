// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/Text.css";

import {
  type ForwardedRef,
  forwardRef,
  type JSX,
  type ReactElement,
  type ReactNode,
} from "react";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Generic } from "@/generic";
import { type text } from "@/text/core";

export interface CoreProps<L extends text.Level = text.Level> {
  /* The level of text to display i.e. p, h1, h2 */
  level: L;
  /* The text to display */
  children?: ReactNode;
  /* The color of the text */
  color?: Color.Crude | boolean;
  /* NoWrap prevents the text from wrapping */
  noWrap?: boolean;
  shade?: text.Shade;
  /* Weight sets the weight of the text */
  weight?: text.Weight;
}

export type TextProps<L extends text.Level = text.Level> = Omit<
  Generic.ElementProps<L>,
  "el" | "color" | "children"
> &
  CoreProps<L>;

const CoreText = <L extends text.Level = text.Level>(
  {
    level = "p" as L,
    color,
    className,
    style,
    children,
    noWrap = false,
    shade,
    weight,
    ...props
  }: TextProps<L>,
  ref: ForwardedRef<JSX.IntrinsicElements[L]>,
): ReactElement => (
  // @ts-expect-error - TODO: Generic Elements are weird
  <Generic.Element<L>
    el={level}
    ref={ref}
    style={{ color: evalColor(color, shade), fontWeight: weight, ...style }}
    className={CSS(CSS.B("text"), CSS.BM("text", level), CSS.noWrap(noWrap), className)}
    {...props}
  >
    {children}
  </Generic.Element>
);

export const Text = forwardRef(CoreText) as <L extends text.Level = text.Level>(
  props: TextProps<L>,
) => ReactElement;

export const evalColor = (
  color?: Color.Crude | boolean,
  shade?: number,
): string | undefined => {
  if (color != null) {
    if (typeof color === "boolean") return undefined;
    return Color.cssString(color) as string;
  }
  if (shade != null) return `var(--pluto-gray-l${shade})`;
  return undefined;
};
