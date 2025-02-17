// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Item.css";

import { direction } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { HelpText } from "@/input/HelpText";
import { Label } from "@/input/Label";
import { type status } from "@/status/aether";

export interface ItemProps extends Align.SpaceProps {
  label?: string;
  required?: boolean;
  showLabel?: boolean;
  helpText?: string;
  padHelpText?: boolean;
  helpTextVariant?: status.Variant;
  showHelpText?: boolean;
}

const maybeDefaultAlignment = (
  align?: Align.Alignment,
  dir: direction.Crude = "x",
): Align.Alignment => {
  if (align != null) return align;
  return direction.construct(dir) === "y" ? "stretch" : "center";
};

export const Item = ({
  label,
  showLabel = true,
  helpText,
  direction = "y",
  className,
  children,
  required,
  align,
  size = "small",
  padHelpText = false,
  helpTextVariant,
  showHelpText = true,
  ...rest
}: ItemProps): ReactElement => {
  let inputAndHelp: ReactElement;
  const showHelpText_ = showHelpText && helpText != null && helpText.length > 0;
  const showLabel_ = showLabel && label != null && label.length > 0;
  if (!showHelpText_ && !showLabel_) return <>{children}</>;
  if (direction === "x")
    inputAndHelp = (
      <Align.Space direction="y" size="small">
        {children}
        {(padHelpText || (helpText != null && helpText.length > 0)) && (
          <HelpText variant={helpTextVariant}>{helpText}</HelpText>
        )}
      </Align.Space>
    );
  else
    inputAndHelp = (
      <Align.Space direction="y" size={1 / 3}>
        {children}
        {(padHelpText || (helpText != null && helpText.length > 0)) && (
          <HelpText variant={helpTextVariant}>{helpText}</HelpText>
        )}
      </Align.Space>
    );

  return (
    <Align.Space
      className={CSS(CSS.B("input-item"), className)}
      direction={direction}
      size={size}
      align={maybeDefaultAlignment(align, direction)}
      {...rest}
    >
      {showLabel_ && <Label required={required}>{label}</Label>}
      {inputAndHelp}
    </Align.Space>
  );
};
