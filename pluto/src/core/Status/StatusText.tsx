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

import { StatusVariant } from "./types";

import { Text, TextProps, TypographyLevel } from "@/core/Typography";

export interface StatusBadgeProps extends Omit<TextProps, "level" | "wrap"> {
  level?: TypographyLevel;
  variant: StatusVariant;
}

const statusVariantIcons: Record<StatusVariant, ReactElement> = {
  info: <Icon.Info />,
  warning: <Icon.Warning />,
  error: <Icon.Close />,
  success: <Icon.Check />,
  loading: <Icon.Warning />,
};

const statusVariantColors: Record<StatusVariant, string> = {
  info: "var(--pluto-text-color)",
  error: "var(--pluto-error-z)",
  warning: "var(--pluto-text-color)",
  success: "var(--pluto-primary-z)",
  loading: "var(--pluto-text-color)",
};

export const StatusText = ({
  variant = "error",
  level = "p",
  ...props
}: StatusBadgeProps): JSX.Element => (
  <Text.WithIcon
    color={statusVariantColors[variant]}
    level={level}
    startIcon={statusVariantIcons[variant]}
    {...props}
  />
);
