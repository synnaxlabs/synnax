// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tag/Tag.css";

import { color, type Optional } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button } from "@/button";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Text } from "@/text";

export interface TagProps
  extends Optional<Omit<Text.TextProps, "size" | "wrap">, "level"> {
  icon?: Icon.ReactElement;
  onClose?: () => void;
  color?: color.Crude;
  size?: Component.Size;
  variant?: "filled" | "outlined";
}

export const Tag = ({
  children = "",
  size = "medium",
  color: pColor,
  icon,
  onClose,
  className,
  onDragStart,
  ...rest
}: TagProps): ReactElement => {
  const cssColor = color.cssString(pColor);
  if (icon == null && pColor != null) icon = <Icon.Circle color={cssColor} />;
  const closeIcon =
    onClose == null ? undefined : (
      <Button.Button
        aria-label="close"
        size="small"
        className={CSS.BE("tag", "close")}
        shade={1}
        sharp
        onClick={(e) => {
          e.stopPropagation();
          onClose();
        }}
      >
        <Icon.Close />
      </Button.Button>
    );
  return (
    <Text.Text
      className={CSS(
        className,
        CSS.B("tag"),
        CSS.height(size),
        onClose != null && CSS.BM("tag", "closeable"),
      )}
      level={Text.COMPONENT_SIZE_LEVELS[size]}
      noWrap
      align="center"
      gap="small"
      onDragStart={onDragStart}
      {...rest}
    >
      {children}
      {closeIcon}
    </Text.Text>
  );
};
