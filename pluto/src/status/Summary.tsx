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
import { Text as BaseText } from "@/text";

export interface SummaryProps
  extends Omit<BaseText.TextProps, "wrap" | "variant">,
    Partial<Omit<status.Status, "key">> {
  hideIcon?: boolean;
}

export const Summary = ({
  level = "p",
  variant,
  status: textStatusVariant,
  description,
  hideIcon = false,
  className,
  children,
  message,
  color,
  ...rest
}: SummaryProps): ReactElement => {
  let icon: Icon.ReactElement | undefined;
  variant ??= textStatusVariant;
  if (!hideIcon) icon = variant === "loading" ? <Icon.Loading /> : <Icon.Circle />;
  const hasDescription = description != null;
  children ??= message;
  const baseText = (
    <BaseText.Text
      className={CSS(className, !hasDescription && CSS.BE("status", "text"))}
      level={level}
      status={variant}
      {...(description == null ? rest : {})}
    >
      {icon}
      {children}
    </BaseText.Text>
  );
  if (!hasDescription) return baseText;
  const descriptionText = (
    <BaseText.Text level="small" color={8}>
      {description}
    </BaseText.Text>
  );
  return (
    <Flex.Box
      y
      className={CSS.BE("status", "text")}
      align="start"
      gap="small"
      {...rest}
    >
      {baseText}
      {descriptionText}
    </Flex.Box>
  );
};
