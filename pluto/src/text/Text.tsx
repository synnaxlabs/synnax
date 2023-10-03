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
import { type Level } from "@/text/types";

import "@/text/Text.css";

export interface CoreProps<L extends Level = "h1"> {
  /* The level of text to display i.e. p, h1, h2 */
  level: L;
  /* The text to display */
  children?: (string | number) | Array<string | number>;
  /* The color of the text */
  color?: Color.Crude;
  /* NoWrap prevents the text from wrapping */
  noWrap?: boolean;
}

export type TextProps<L extends Level = "h1"> = Omit<
  Generic.ElementProps<L>,
  "el" | "color" | "children"
> &
  CoreProps<L>;

const CoreText = <L extends Level = "h1">(
  {
    level = "h1" as L,
    color,
    className,
    style,
    children,
    noWrap = false,
    ...props
  }: TextProps<L>,
  ref: ForwardedRef<JSX.IntrinsicElements[L]>,
): ReactElement => (
  // @ts-expect-error
  <Generic.Element<L>
    el={level}
    ref={ref}
    style={{ color: Color.cssString(color), ...style }}
    className={CSS(CSS.B("text"), CSS.BM("text", level), CSS.noWrap(noWrap), className)}
    {...props}
  >
    {children}
  </Generic.Element>
);

export const Text = forwardRef(CoreText) as <L extends Level = "h1">(
  props: TextProps<L>,
) => ReactElement;
