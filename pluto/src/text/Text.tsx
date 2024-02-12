// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ForwardedRef, forwardRef, type ReactElement } from "react";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Generic } from "@/generic";
import { type text } from "@/text/core";

import "@/text/Text.css";

/* Shade sets the shade color of the text */
export type Shade = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10;

export interface CoreProps<L extends text.Level = text.Level> {
  /* The level of text to display i.e. p, h1, h2 */
  level: L;
  /* The text to display */
  children?: (string | number | boolean) | Array<string | number | boolean>;
  /* The color of the text */
  color?: Color.Crude;
  /* NoWrap prevents the text from wrapping */
  noWrap?: boolean;
  shade?: Shade;
  /* Weight sets the weight of the text */
  weight?: number;
}

export type TextProps<L extends text.Level = "h1"> = Omit<
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
  // @ts-expect-error
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

export const Text = forwardRef(CoreText) as <L extends text.Level = "h1">(
  props: TextProps<L>,
) => ReactElement;

const evalColor = (color?: Color.Crude, shade?: number): string | undefined => {
  if (color != null) return Color.cssString(color);
  if (shade != null) return Color.cssString(`var(--pluto-gray-l${shade})`);
  return undefined;
};
