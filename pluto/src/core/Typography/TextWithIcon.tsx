// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Children, cloneElement, ReactElement } from "react";

import { CoreTextProps, Text } from "./Text";
import { TypographyLevel } from "./types";

import { Divider } from "@/core/Divider";
import { Space, SpaceProps, SpaceElementType } from "@/core/Space";
import { CSS } from "@/css";

import "./TextWithIcon.css";

export type TextWithIconProps<
  E extends SpaceElementType = "div",
  L extends TypographyLevel = "h1"
> = Omit<SpaceProps<E>, "children"> &
  CoreTextProps<L> & {
    startIcon?: ReactElement | ReactElement[];
    endIcon?: ReactElement | ReactElement[];
    children?: string | number;
    divided?: boolean;
    noWrap?: boolean;
  };

export const TextWithIcon = <
  E extends SpaceElementType = "div",
  L extends TypographyLevel = "h1"
>({
  level = "h1" as L,
  divided = false,
  startIcon,
  endIcon,
  children,
  color,
  className,
  noWrap = false,
  ...props
}: TextWithIconProps<E, L>): JSX.Element => {
  const startIcons = startIcon != null && formatIcons(startIcon, color);
  const endIcons = endIcon != null && formatIcons(endIcon, color);
  return (
    // @ts-expect-error
    <Space<E>
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
      {divided && startIcon != null && <Divider direction="y" />}
      {children != null && (
        // @ts-expect-error
        <Text<L> color={color} level={level}>
          {children}
        </Text>
      )}
      {divided && endIcon != null && <Divider direction="y" />}
      {endIcons}
    </Space>
  );
};

const formatIcons = (
  icon: ReactElement | ReactElement[],
  color?: string
): JSX.Element[] =>
  (Children.toArray(icon) as ReactElement[]).map((icon) =>
    cloneElement(icon, {
      ...icon.props,
      style: { fill: color, ...icon.props.style },
    })
  );
