// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Children, cloneElement, type ReactElement } from "react";

import { toArray } from "@synnaxlabs/x";

import { Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { type CoreProps, Text } from "@/text/Text";
import { type Level } from "@/text/types";
import { isValidElement } from "@/util/children";

import "@/text/WithIcon.css";

type ValidChild = string | number | ReactElement;

export type WithIconProps<
  E extends Align.SpaceElementType = "div",
  L extends Level = "h1",
> = Omit<Align.SpaceProps<E>, "children" | "color"> &
  Omit<CoreProps<L>, "children"> & {
    startIcon?: false | ReactElement | ReactElement[];
    endIcon?: false | ReactElement | ReactElement[];
    children?: ValidChild | ValidChild[];
    divided?: boolean;
    noWrap?: boolean;
  };

export const WithIcon = <
  E extends Align.SpaceElementType = "div",
  L extends Level = "h1",
>({
  level = "h1" as L,
  divided = false,
  startIcon,
  endIcon,
  children,
  color: crudeColor,
  className,
  noWrap = false,
  ...props
}: WithIconProps<E, L>): ReactElement => {
  const color = Color.cssString(crudeColor);
  const startIcons = startIcon != null && formatIcons(startIcon, color);
  const endIcons = endIcon != null && formatIcons(endIcon, color);
  const formatted = formatChildren(level, children, color);
  return (
    // @ts-expect-error - level type errors
    <Align.Space<E>
      className={CSS(
        CSS.B("text-icon"),
        CSS.BM("text-icon", level),
        CSS.noWrap(noWrap),
        className,
      )}
      direction="x"
      size="small"
      align="center"
      {...props}
    >
      {startIcons}
      {divided && startIcon != null && <Divider.Divider direction="y" />}
      {formatted}
      {divided && endIcon != null && <Divider.Divider direction="y" />}
      {endIcons}
    </Align.Space>
  );
};

const formatIcons = (
  icon: false | ReactElement | ReactElement[],
  color?: string,
): ReactElement[] => {
  if (icon === false) return [];
  return (Children.toArray(icon) as ReactElement[]).map((icon) =>
    cloneElement(icon, {
      ...icon.props,
      color,
      style: { ...icon.props.style },
    }),
  );
};

const formatChildren = <L extends Level>(
  level: L,
  children: ValidChild | ValidChild[] = [],
  color?: string,
): ReactElement[] => {
  const arr = toArray(children);
  const o: ReactElement[] = [];
  let buff: Array<string | number> = [];
  arr.forEach((child) => {
    if (
      typeof child === "string" ||
      typeof child === "number" ||
      !isValidElement(child)
    ) {
      buff.push(child);
    } else {
      if (buff.length > 0) {
        o.push(
          // @ts-expect-error - level type errors
          <Text<L> color={color} level={level}>
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
      <Text<L> color={color} level={level}>
        {buff}
      </Text>,
    );
  return o;
};
