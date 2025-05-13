// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tag/Tag.css";

import { Icon } from "@synnaxlabs/media";
import { color, type Optional } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { type Icon as PIcon } from "@/icon";
import { Text } from "@/text";
import { type ComponentSize } from "@/util/component";

export interface TagProps
  extends Optional<Omit<Text.TextProps, "size" | "wrap">, "level"> {
  icon?: PIcon.Element;
  onClose?: () => void;
  color?: color.Crude;
  size?: ComponentSize;
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
  if (icon == null && pColor != null) icon = <Icon.Circle fill={cssColor} />;
  const closeIcon =
    onClose == null ? undefined : (
      <Button.Icon
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
      </Button.Icon>
    );
  return (
    // @ts-expect-error - TODO: Generic Elements are weird
    <Text.WithIcon
      startIcon={icon}
      endIcon={closeIcon}
      className={CSS(
        className,
        CSS.B("tag"),
        CSS.size(size),
        onClose != null && CSS.BM("tag", "closeable"),
      )}
      level={Text.ComponentSizeLevels[size]}
      noWrap
      align="center"
      size="small"
      onDragStart={onDragStart}
      {...rest}
    >
      {children}
    </Text.WithIcon>
  );
};
