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

import { CSS } from "@/css";
import { Generic } from "@/generic";
import { type text } from "@/text/core";

export interface CoreProps<L extends text.Level = text.Level> {
  /* The level of text to display i.e. p, h1, h2 */
  level: L;
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
}

export type TextProps<L extends text.Level = text.Level> = Omit<
  Generic.ElementProps<L>,
  "el" | "color" | "children"
> &
  CoreProps<L>;

export const Text = <L extends text.Level = text.Level>({
  ref,
  level = "p" as L,
  color,
  className,
  style,
  noWrap = false,
  code = false,
  shade,
  weight,
  ...rest
}: TextProps<L>): ReactElement => (
  // @ts-expect-error - TODO: Generic Elements are weird
  <Generic.Element<L>
    el={level}
    ref={ref}
    style={{ color: parseColor(color, shade), fontWeight: weight, ...style }}
    className={CSS(
      CSS.B("text"),
      code && CSS.M("code"),
      CSS.BM("text", level),
      CSS.noWrap(noWrap),
      className,
    )}
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
