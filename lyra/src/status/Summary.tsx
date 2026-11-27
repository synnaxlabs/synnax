// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/status/Summary.css";

import { primitive } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import type { Icon } from "@/icon";
import type { Status } from "@/status/core";
import { Indicator } from "@/status/Indicator";
import { Text as BaseText } from "@/text";

/** Props for {@link Summary}. Pass a whole `status`, or its parts one by one. */
export interface SummaryProps
  extends
    Omit<BaseText.TextProps, "wrap" | "variant" | "status" | "color">,
    Partial<Pick<Status, "variant" | "message" | "description">> {
  hideIcon?: boolean;
  status?: Status;
}

/**
 * Renders a status as an {@link Indicator}, its message, and its description. Use it
 * wherever an error or a result has to read inline.
 */
export const Summary = ({
  level = "p",
  variant,
  description,
  hideIcon = false,
  status,
  className,
  children,
  message,
  center,
  ...rest
}: SummaryProps): ReactElement => {
  let icon: Icon.ReactElement | undefined;
  if (status != null) ({ variant, message, description } = status);
  if (!hideIcon) icon = <Indicator variant={variant} />;
  const hasDescription = primitive.isNonZero(description);
  if (children == null) children = message;
  const baseText = (
    <BaseText.Text
      className={CSS.cls(className, !hasDescription && CSS.BE("status", "text"))}
      level={level}
      status={variant}
      {...(hasDescription ? {} : { center, ...rest })}
    >
      {icon}
      {children}
    </BaseText.Text>
  );
  if (!hasDescription) return baseText;
  const descriptionText = (
    <BaseText.Text level="small" color={9} className={CSS.BE("status", "description")}>
      {description}
    </BaseText.Text>
  );
  const centered = center ?? true;
  return (
    <Flex.Box
      y
      align={centered ? "center" : "start"}
      gap="small"
      center={centered}
      className={CSS.BE("status", "summary")}
      {...rest}
    >
      {baseText}
      {descriptionText}
    </Flex.Box>
  );
};
