// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { type Icon as PIcon } from "@/icon";
import { VARIANT_COLORS } from "@/status/colors";
import { Text as BaseText } from "@/text";

export interface TextProps extends Omit<BaseText.WithIconProps, "level" | "wrap"> {
  level?: BaseText.Level;
  hideIcon?: boolean;
  noColor?: boolean;
  variant?: status.Variant;
}

const Core = ({
  variant = "info",
  level = "p",
  hideIcon = false,
  className,
  ...rest
}: TextProps): ReactElement => {
  let icon: PIcon.Element | undefined;
  if (!hideIcon) icon = variant === "loading" ? <Icon.Loading /> : <Icon.Circle />;
  return (
    <BaseText.WithIcon
      color={VARIANT_COLORS[variant]}
      className={CSS(className, CSS.B("status-text"))}
      level={level}
      startIcon={icon}
      {...rest}
    />
  );
};

export interface TextCenteredProps extends TextProps {}

const Centered = ({ style, ...rest }: TextCenteredProps): ReactElement => (
  <Align.Center style={style} grow>
    <Core {...rest} />
  </Align.Center>
);

type CoreTextType = typeof Core;

export interface TextType extends CoreTextType {
  Centered: typeof Centered;
}

export const Text: TextType = Object.assign(Core, { Centered });
