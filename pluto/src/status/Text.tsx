// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { VARIANT_COLORS } from "@/status/colors";
import { Text as BaseText } from "@/text";

export interface TextProps
  extends Omit<BaseText.TextProps, "level" | "wrap" | "variant">,
    Partial<Omit<status.Status, "key">> {
  level?: BaseText.Level;
  hideIcon?: boolean;
}

export const Text = ({
  variant = "info",
  level = "p",
  description,
  hideIcon = false,
  className,
  children,
  color,
  ...rest
}: TextProps): ReactElement => {
  let icon: Icon.ReactElement | undefined;
  if (!hideIcon) icon = variant === "loading" ? <Icon.Loading /> : <Icon.Circle />;
  const baseText = (
    <BaseText.Text
      color={color ?? VARIANT_COLORS[variant]}
      className={CSS(className, CSS.BE("status", "text"))}
      level={level}
      {...rest}
    >
      {icon}
      {children}
    </BaseText.Text>
  );
  if (description == null) return baseText;
  const descriptionText = (
    <BaseText.Text level="small" {...rest} color={8} style={{ maxWidth: 150 }}>
      {description}
    </BaseText.Text>
  );
  return (
    <Flex.Box y align="start" gap="small">
      {baseText}
      {descriptionText}
    </Flex.Box>
  );
};
