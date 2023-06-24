// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";

import { ColorT } from "@/core/color";
import { Space } from "@/core/std/Space";
import { StatusVariant } from "@/core/std/Status/types";
import { Text, TextProps, TypographyLevel } from "@/core/std/Typography";

export interface StatusTextProps extends Omit<TextProps, "level" | "wrap"> {
  level?: TypographyLevel;
  hideIcon?: boolean;
  variant?: StatusVariant;
}

const statusVariantIcons: Record<StatusVariant, ReactElement> = {
  info: <Icon.Info />,
  warning: <Icon.Warning />,
  error: <Icon.Close />,
  success: <Icon.Check />,
  loading: <Icon.Warning />,
  disabled: <Icon.Warning />,
};

const statusVariantColors: Record<StatusVariant, ColorT> = {
  info: "var(--pluto-text-color)",
  error: "var(--pluto-error-z)",
  warning: "var(--pluto-text-color)",
  success: "var(--pluto-primary-z)",
  loading: "var(--pluto-text-color)",
  disabled: "var(--pluto-gray-p0)",
};

const CoreStatusText = ({
  variant = "info",
  level = "p",
  hideIcon = false,
  ...props
}: StatusTextProps): ReactElement => (
  <Text.WithIcon
    color={statusVariantColors[variant]}
    level={level}
    startIcon={!hideIcon && statusVariantIcons[variant]}
    {...props}
  />
);

export interface StatusTextCenteredProps extends StatusTextProps {}

const StatusTextCentered = (props: StatusTextCenteredProps): ReactElement => (
  <Space.Centered>
    <CoreStatusText {...props} />
  </Space.Centered>
);

type CoreStatusTextType = typeof CoreStatusText;

export interface StatusTextType extends CoreStatusTextType {
  Centered: typeof StatusTextCentered;
}

export const StatusText = CoreStatusText as StatusTextType;

StatusText.Centered = StatusTextCentered;
