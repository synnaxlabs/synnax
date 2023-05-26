// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { Button } from "@/core/std/Button";
import "@/core/Tag/Tag.css";
import { Typography, Text, TextProps } from "@/core/std/Typography";
import { CSS } from "@/core/css";
import { ComponentSize } from "@/util/component";

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
}: TagProps): ReactElement => {
  const closeIcon =
    onClose == null ? undefined : (
      <Button.Icon
        aria-label="close"
        className={CSS.BE("tag", "close")}
        onClick={(e) => {
          e.stopPropagation();
          onClose();
        }}
      >
        <Icon.Close />
      </Button.Icon>
    );
  return (
    <Text.WithIcon
      endIcon={closeIcon}
      startIcon={icon}
      className={CSS.B("tag")}
      level={Typography.ComponentSizeLevels[size]}
      noWrap
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
