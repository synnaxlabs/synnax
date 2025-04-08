// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/WithIcon.css";

import { toArray } from "@synnaxlabs/x";
import { Children, type Key, type ReactElement, type ReactNode } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { type text } from "@/text/core";
import { type CoreProps, evalColor, Text } from "@/text/Text";
import { isValidElement } from "@/util/children";

interface IconProps {
  key?: Key | null;
  color?: string;
  style?: React.CSSProperties;
}

type IconElement = ReactElement<IconProps>;

export type WithIconProps<
  E extends Align.ElementType = "div",
  L extends text.Level = "h1",
> = Omit<Align.SpaceProps<E>, "children" | "color"> &
  CoreProps<L> & {
    startIcon?: false | IconElement | IconElement[];
    endIcon?: false | IconElement | IconElement[];
    divided?: boolean;
    noWrap?: boolean;
  };

export const WithIcon = <
  E extends Align.ElementType = "div",
  L extends text.Level = text.Level,
>({
  level = "h1" as L,
  divided = false,
  startIcon,
  endIcon,
  children,
  color: crudeColor,
  className,
  noWrap = false,
  shade,
  weight,
  ...rest
}: WithIconProps<E, L>): ReactElement => {
  const color = evalColor(crudeColor, shade);
  const startIcons = Children.toArray(startIcon);
  const endIcons = Children.toArray(endIcon);
  const formatted = formatChildren(level, children, color, shade, weight);
  return (
    // @ts-expect-error - level type errors
    <Align.Space<E>
      className={CSS(
        CSS.B("text-icon"),
        CSS.BM("text-icon", level),
        CSS.noWrap(noWrap),
        className,
      )}
      x
      size="small"
      align="center"
      {...rest}
      style={{ ...rest.style, color }}
    >
      {startIcons}
      {divided && startIcon != null && <Divider.Divider y />}
      {formatted}
      {divided && endIcon != null && <Divider.Divider y />}
      {endIcons}
    </Align.Space>
  );
};

export const formatChildren = <L extends text.Level>(
  level: L,
  children: ReactNode = [],
  color?: string,
  shade?: number,
  weight?: text.Weight,
): ReactElement | ReactElement[] => {
  const arr = toArray(children);
  const o: ReactElement[] = [];
  let buff: Array<ReactNode> = [];
  const props = { color, level, shade, weight };
  arr.forEach((child) => {
    if (child == null) return;
    if (
      typeof child === "string" ||
      typeof child === "number" ||
      !isValidElement(child)
    )
      buff.push(child);
    else {
      if (buff.length > 0) {
        o.push(
          // @ts-expect-error - level type errors
          <Text<L> key={buff[0]} {...props}>
            {buff}
          </Text>,
        );
        buff = [];
      }
      o.push(child);
    }
  });
  if (buff.length > 0)
    o.push(
      // @ts-expect-error- level type errors
      <Text<L> key={buff[0]} {...props}>
        {buff}
      </Text>,
    );
  if (o.length === 1) return o[0];
  return o;
};
