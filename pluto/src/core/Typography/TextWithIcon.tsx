// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Children, ReactElement } from "react";

import clsx from "clsx";

import { Divider } from "@/core/Divider";

import { CoreTextProps, Text } from "./Text";

import { Space, SpaceProps } from "@/core/Space";

import "./TextWithIcon.css";

export interface TextWithIconProps extends Omit<SpaceProps, "children">, CoreTextProps {
  startIcon?: ReactElement | ReactElement[];
  endIcon?: ReactElement | ReactElement[];
  children?: string | number;
  divided?: boolean;
}

export const TextWithIcon = ({
  level = "h1",
  divided = false,
  startIcon,
  endIcon,
  children,
  color,
  className,
  ...props
}: TextWithIconProps): JSX.Element => {
  const startIcons = startIcon != null && Children.toArray(startIcon);
  const endIcons = endIcon != null && Children.toArray(endIcon);
  return (
    <Space
      className={clsx("pluto-text-icon", `pluto-text-icon-${level}`, className)}
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
