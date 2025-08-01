// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/Text.css";

import { color } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { type text } from "@/text/core";

export type TextProps<E extends Align.ElementType = Align.ElementType> = Omit<
  Align.SpaceProps<E>,
  "color" | "children" | "direction" | "x" | "y"
> & {
  /* The level of text to display i.e. p, h1, h2 */
  level: text.Level;
  /* The text to display */
  children?: ReactNode;
  /* The color of the text */
  color?: color.Crude | false;
  /* NoWrap prevents the text from wrapping */
  noWrap?: boolean;
  /* Shade sets the shade of the text */
  shade?: text.Shade;
  /* Weight sets the weight of the text */
  weight?: text.Weight;
  code?: boolean;
  /* Ellipsis sets whether to truncate the text */
  ellipsis?: boolean;
};

export const Text = <E extends Align.ElementType = Align.ElementType>({
  ref,
  level = "p",
  el,
  color,
  className,
  style,
  noWrap = false,
  code = false,
  weight,
  ellipsis = false,
  shade,
  ...rest
}: TextProps<E>): ReactElement => (
  // @ts-expect-error - TODO: Generic Elements are weird
  <Align.Space<E>
    el={el ?? level}
    ref={ref}
    x
    style={{ color: parseColor(color), fontWeight: weight, ...style }}
    className={CSS(
      CSS.B("text"),
      code && CSS.M("code"),
      CSS.noWrap(noWrap),
      ellipsis && CSS.M("ellipsis"),
      shade != null && CSS.BM("text", `shade-${shade}`),
      className,
    )}
    gap="small"
    align="center"
    {...rest}
  />
);

export const parseColor = (
  crudeColor?: color.Crude | false,
  shade?: number,
): string | undefined => {
  if (crudeColor != null) {
    if (typeof crudeColor === "boolean") return undefined;
    return color.cssString(crudeColor);
  }
  if (shade != null) return `var(--pluto-gray-l${shade})`;
  return undefined;
};
