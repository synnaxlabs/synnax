// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { AiOutlineClose } from "react-icons/ai";

import { Typography, Text, TextProps } from "@/core/Typography";
import { ComponentSize } from "@/util/component";

import "./Tag.css";

export interface TagProps extends Omit<TextProps, "level" | "size" | "wrap"> {
  icon?: React.ReactElement;
  onClose?: () => void;
  color?: string;
  size?: ComponentSize;
  variant?: "filled" | "outlined";
}

export const Tag = ({
  children = "",
  size = "medium",
  variant = "filled",
  color = "var(--pluto-primary-z)",
  icon,
  onClose,
  style,
  ...props
}: TagProps): JSX.Element => {
  const closeIcon =
    onClose == null ? undefined : (
      <AiOutlineClose
        aria-label="close"
        className="pluto-tag__close"
        onClick={(e) => {
          e.stopPropagation();
          onClose();
        }}
      />
    );
  return (
    <Text.WithIcon
      endIcon={closeIcon}
      startIcon={icon}
      className="pluto-tag"
      level={Typography.ComponentSizeLevels[size]}
      style={{
        border: `var(--pluto-border-width) solid ${color}`,
        backgroundColor: variant === "filled" ? color : "transparent",
        ...style,
      }}
      {...props}
    >
      {children}
    </Text.WithIcon>
  );
};
