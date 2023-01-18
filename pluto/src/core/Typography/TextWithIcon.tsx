// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Children, cloneElement, ReactElement } from "react";

import clsx from "clsx";

import { Divider } from "@/core/Divider";

import { CoreTextProps, Text } from "./Text";

import { Space, SpaceProps } from "@/core/Space";

import "./TextWithIcon.css";

export interface TextWithIconProps<E extends HTMLElement = HTMLDivElement>
  extends Omit<SpaceProps<E>, "children">,
    CoreTextProps {
  startIcon?: ReactElement | ReactElement[];
  endIcon?: ReactElement | ReactElement[];
  children?: string | number;
  divided?: boolean;
}

export const TextWithIcon = <E extends HTMLElement = HTMLDivElement>({
  level = "h1",
  divided = false,
  startIcon,
  endIcon,
  children,
  color,
  className,
  ...props
}: TextWithIconProps<E>): JSX.Element => {
  const startIcons = startIcon != null && formatIcons(startIcon, color);
  const endIcons = endIcon != null && formatIcons(endIcon, color);
  return (
    <Space<E>
      className={clsx("pluto-text-icon", `pluto-text-icon--${level}`, className)}
      direction="x"
      size="small"
      align="center"
      {...props}
    >
      {startIcons}
      {divided && startIcon != null && <Divider direction="y" />}
      {children != null && (
        <Text color={color} level={level}>
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
