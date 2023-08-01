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

import { CrudeColor } from "@/core/color";
import { Space } from "@/core/std/Space";
import { StatusVariant } from "@/core/std/Status/types";
import { Text, TextProps, TypographyLevel } from "@/core/std/Typography";

export interface StatusTextDigest {
  variant: StatusVariant;
  children?: string | number;
}

export interface StatusTextProps
  extends Omit<TextProps, "level" | "wrap">,
    StatusTextDigest {
  level?: TypographyLevel;
  hideIcon?: boolean;
}

const statusVariantColors: Record<StatusVariant, CrudeColor> = {
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
    startIcon={!hideIcon && <Icon.Circle />}
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
