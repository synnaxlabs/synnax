// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/Text.css";

import { type color } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { type Generic } from "@/generic";
import { type text } from "@/text/core";

export interface ExtensionProps extends Flex.BoxExtensionProps {
  /* The level of text to display i.e. p, h1, h2 */
  level?: text.Level;
  /* The text to display */
  children?: ReactNode;
  /* The color of the text */
  color?: text.Shade | color.Crude | false;
  /* NoWrap prevents the text from wrapping */
  noWrap?: boolean;
  /* Shade sets the shade of the text */
  /* Weight sets the weight of the text */
  weight?: text.Weight;
  code?: boolean;
  /* Ellipsis sets whether to truncate the text */
  ellipsis?: boolean;
}

export type TextProps<E extends Generic.ElementType = "p"> = Omit<
  Generic.OptionalElementProps<E>,
  "color"
> &
  ExtensionProps;

export const Text = <E extends Generic.ElementType = "p">({
  level = "p",
  color,
  className,
  style,
  noWrap = false,
  code = false,
  weight,
  el,
  ellipsis = false,
  ...rest
}: TextProps<E>): ReactElement => (
  <Flex.Box<E>
    direction="x"
    el={(el ?? level) as E}
    style={{ fontWeight: weight, color: CSS.colorVar(color), ...style }}
    className={CSS(
      CSS.B("text"),
      code && CSS.M("code"),
      CSS.level(level),
      CSS.noWrap(noWrap),
      ellipsis && CSS.M("ellipsis"),
      className,
    )}
    gap="small"
    {...(rest as Flex.BoxProps<E>)}
  />
);
