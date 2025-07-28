// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { VARIANT_COLORS } from "@/status/colors";
import { Text as BaseText } from "@/text";

export interface TextProps
  extends Omit<BaseText.WithIconProps, "level" | "wrap">,
    Partial<Omit<status.Status, "key">> {
  level?: BaseText.Level;
  hideIcon?: boolean;
}

const Core = ({
  variant = "info",
  level = "p",
  description,
  hideIcon = false,
  className,
  ...rest
}: TextProps): ReactElement => {
  let icon: Icon.ReactElement | undefined;
  if (!hideIcon) icon = variant === "loading" ? <Icon.Loading /> : <Icon.Circle />;
  const baseText = (
    <BaseText.WithIcon
      color={VARIANT_COLORS[variant]}
      className={CSS(className, CSS.B("status-text"))}
      level={level}
      startIcon={icon}
      {...rest}
    />
  );
  if (description == null) return baseText;
  const descriptionText = (
    <BaseText.Text level="small" {...rest} shade={8} style={{ maxWidth: 150 }}>
      {description}
    </BaseText.Text>
  );
  return (
    <Align.Space y align="start" gap="small">
      {baseText}
      {descriptionText}
    </Align.Space>
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
