// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Item.css";

import { direction, type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { HelpText } from "@/input/HelpText";
import { Label } from "@/input/Label";

export interface ItemProps extends Flex.BoxProps {
  label?: string;
  required?: boolean;
  showLabel?: boolean;
  helpText?: string;
  padHelpText?: boolean;
  helpTextVariant?: status.Variant;
  showHelpText?: boolean;
}

const maybeDefaultAlignment = (
  align?: Flex.Alignment,
  dir: direction.Crude = "x",
): Flex.Alignment => {
  if (align != null) return align;
  return direction.construct(dir) === "y" ? "stretch" : "center";
};

export const Item = ({
  label,
  showLabel = true,
  helpText,
  direction,
  x,
  y,
  className,
  children,
  required,
  align,
  gap: size = "small",
  padHelpText = false,
  helpTextVariant,
  showHelpText = true,
  ...rest
}: ItemProps): ReactElement => {
  const dir = Flex.parseDirection(direction, x, y, false);
  let inputAndHelp: ReactElement;
  const showHelpText_ = showHelpText && helpText != null && helpText.length > 0;
  const showLabel_ = showLabel && label != null && label.length > 0;
  if (!showHelpText_ && !showLabel_) return <>{children}</>;
  if (dir === "x")
    inputAndHelp = (
      <Flex.Box y gap="small">
        {children}
        {showHelpText && (padHelpText || (helpText != null && helpText.length > 0)) && (
          <HelpText variant={helpTextVariant}>{helpText}</HelpText>
        )}
      </Flex.Box>
    );
  else
    inputAndHelp = (
      <Flex.Box y gap={1 / 3}>
        {children}
        {(padHelpText || (helpText != null && helpText.length > 0)) && (
          <HelpText variant={helpTextVariant}>{helpText}</HelpText>
        )}
      </Flex.Box>
    );

  return (
    <Flex.Box
      className={CSS(CSS.B("input-item"), className)}
      direction={dir}
      gap={size}
      align={maybeDefaultAlignment(align, dir)}
      {...rest}
    >
      {showLabel_ && <Label required={required}>{label}</Label>}
      {inputAndHelp}
    </Flex.Box>
  );
};
