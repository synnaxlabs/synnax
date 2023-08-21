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

import { Align } from "@/align";
import { Color } from "@/color";
import { Variant } from "@/status/types";
import { Text as CoreText } from "@/text";

export interface StatusTextDigest {
  variant: Variant;
  children?: string | number;
}

export interface StatusTextProps
  extends Omit<CoreText.TextProps, "level" | "wrap">,
    StatusTextDigest {
  level?: CoreText.Level;
  hideIcon?: boolean;
}

const statusVariantColors: Record<Variant, Color.Crude> = {
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
  <CoreText.WithIcon
    color={statusVariantColors[variant]}
    level={level}
    startIcon={!hideIcon && <Icon.Circle />}
    {...props}
  />
);

export interface StatusTextCenteredProps extends StatusTextProps {}

const StatusTextCentered = (props: StatusTextCenteredProps): ReactElement => (
  <Align.Center>
    <CoreStatusText {...props} />
  </Align.Center>
);

type CoreStatusTextType = typeof CoreStatusText;

export interface StatusTextType extends CoreStatusTextType {
  Centered: typeof StatusTextCentered;
}

export const Text = CoreStatusText as StatusTextType;

Text.Centered = StatusTextCentered;
