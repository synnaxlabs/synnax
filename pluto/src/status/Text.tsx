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
import { Text as BaseText } from "@/text";

export interface TextDigest {
  variant: Variant;
  children?: string | number;
}

export interface TextProps
  extends Omit<BaseText.TextProps, "level" | "wrap">,
    TextDigest {
  level?: BaseText.Level;
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

const CoreText = ({
  variant = "info",
  level = "p",
  hideIcon = false,
  ...props
}: TextProps): ReactElement => (
  <BaseText.WithIcon
    color={statusVariantColors[variant]}
    level={level}
    startIcon={!hideIcon && <Icon.Circle />}
    {...props}
  />
);

export interface TextCenteredProps extends TextProps {}

const TextCentered = (props: TextCenteredProps): ReactElement => (
  <Align.Center>
    <CoreText {...props} />
  </Align.Center>
);

type CoreTextType = typeof CoreText;

export interface TextType extends CoreTextType {
  Centered: typeof TextCentered;
}

export const Text = CoreText as TextType;

Text.Centered = TextCentered;
