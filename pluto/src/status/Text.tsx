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

export interface TextProps extends Omit<BaseText.WithIconProps, "level" | "wrap"> {
  level?: BaseText.Level;
  hideIcon?: boolean;
  noColor?: boolean;
  variant: Variant;
}

const Core = ({
  variant = "info",
  level = "p",
  hideIcon = false,
  className,
  ...props
}: TextProps): ReactElement => (
  <BaseText.WithIcon
    color={variantColors[variant]}
    className={CSS(className, CSS.B("status-text"))}
    level={level}
    startIcon={!hideIcon && variant === "loading" ? <Icon.Loading /> : <Icon.Circle />}
    {...props}
  />
);

export interface TextCenteredProps extends TextProps {}

const Centered = ({ style, ...props }: TextCenteredProps): ReactElement => (
  <Align.Center style={style} grow>
    <Core {...props} />
  </Align.Center>
);

type CoreTextType = typeof Core;

export interface TextType extends CoreTextType {
  Centered: typeof Centered;
}

export const Text: TextType = Object.assign(Core, { Centered });
