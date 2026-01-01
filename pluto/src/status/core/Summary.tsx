// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { primitive, type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text as BaseText } from "@/text";

export interface SummaryProps
  extends
    Omit<BaseText.TextProps, "wrap" | "variant" | "status">,
    Partial<Omit<status.Status, "key">> {
  hideIcon?: boolean;
  status?: status.Status;
}

export const Summary = ({
  level = "p",
  variant,
  description,
  hideIcon = false,
  status,
  className,
  children,
  message,
  color,
  ...rest
}: SummaryProps): ReactElement => {
  let icon: Icon.ReactElement | undefined;
  if (status != null) {
    const { key: _, ...restStatus } = status;
    return <Summary {...rest} {...restStatus} />;
  }
  if (!hideIcon) icon = variant === "loading" ? <Icon.Loading /> : <Icon.Circle />;
  const hasDescription = primitive.isNonZero(description);
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
    <Flex.Box y align="start" gap="small" center {...rest}>
      {baseText}
      {descriptionText}
    </Flex.Box>
  );
};

export interface RemoteSummaryProps {
  statusKey: string;
}

// export const RemoteSummary = ({ statusKey }: RemoteSummaryProps): ReactElement => {
//   const res = useRetrieve({ key: statusKey });
//   const { key, ...rest } = res.data ?? res.status;
//   return <Summary key={key} {...rest} />;
// };
