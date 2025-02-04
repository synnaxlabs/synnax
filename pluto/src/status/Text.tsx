// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { type Variant } from "@/status/aether/types";
import { variantColors } from "@/status/colors";
import { Text as BaseText } from "@/text";

export interface TextDigest {
  variant: Variant;
}

export interface TextProps
  extends Omit<BaseText.WithIconProps, "level" | "wrap">,
    Omit<TextDigest, "children"> {
  level?: BaseText.Level;
  hideIcon?: boolean;
  noColor?: boolean;
}

const CoreText = ({
  variant = "info",
  level = "p",
  hideIcon = false,
  className,
  ...props
}: TextProps): ReactElement => {
  const icon = variant === "loading" ? <Icon.Loading /> : <Icon.Circle />;
  return (
    <BaseText.WithIcon
      color={variantColors[variant]}
      className={CSS(className, CSS.B("status-text"))}
      level={level}
      startIcon={!hideIcon && icon}
      {...props}
    />
  );
};

export interface TextCenteredProps extends TextProps {}

const TextCentered = ({ style, ...props }: TextCenteredProps): ReactElement => (
  <Align.Center style={style} grow>
    <CoreText {...props} />
  </Align.Center>
);

type CoreTextType = typeof CoreText;

export interface TextType extends CoreTextType {
  Centered: typeof TextCentered;
}

export const Text = CoreText as TextType;

Text.Centered = TextCentered;
