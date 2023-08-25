// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Children, cloneElement, isValidElement, ReactElement } from "react";

import { Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { Text } from "@/text";
import { Level } from "@/text/types";

import "@/text/WithIcon.css";

export type WithIconProps<
  E extends Align.SpaceElementType = "div",
  L extends Level = "h1"
> = Omit<Align.SpaceProps<E>, "children" | "color"> &
  Omit<Text.CoreProps<L>, "children"> & {
    startIcon?: false | ReactElement | ReactElement[];
    endIcon?: false | ReactElement | ReactElement[];
    children?: string | number | ReactElement;
    divided?: boolean;
    noWrap?: boolean;
  };

export const WithIcon = <
  E extends Align.SpaceElementType = "div",
  L extends Level = "h1"
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

  let children_ = null;
  if (children != null) {
    if (isValidElement(children)) {
      children_ = children;
    } else {
      children_ = (
        // @ts-expect-error
        <Text.Text<L> color={color} level={level}>
          {children}
        </Text.Text>
      );
    }
  }

  return (
    // @ts-expect-error
    <Align.Space<E>
      className={CSS(
        CSS.B("text-icon"),
        CSS.BM("text-icon", level),
        CSS.noWrap(noWrap),
        className
      )}
      direction="x"
      size="small"
      align="center"
      {...props}
    >
      {startIcons}
      {divided && startIcon != null && <Divider.Divider direction="y" />}
      {children_}
      {divided && endIcon != null && <Divider.Divider direction="y" />}
      {endIcons}
    </Align.Space>
  );
};

const formatIcons = (
  icon: false | ReactElement | ReactElement[],
  color?: string
): ReactElement[] => {
  if (icon === false) return [];
  return (Children.toArray(icon) as ReactElement[]).map((icon) =>
    cloneElement(icon, {
      ...icon.props,
      color,
      style: { ...icon.props.style },
    })
  );
};
