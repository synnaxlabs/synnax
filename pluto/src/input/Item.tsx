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
import { type ReactElement, type ReactNode } from "react";

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
  status?: status.Variant;
  showHelpText?: boolean;
}

const maybeDefaultAlignment = (
  align?: Flex.Alignment,
  dir: direction.Crude = "y",
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
  status,
  showHelpText = true,
  ...rest
}: ItemProps): ReactNode => {
  const dir = Flex.parseDirection(direction, x, y, false);
  let inputAndHelp: ReactElement;
  const actuallyShowHelpText = showHelpText && helpText != null && helpText.length > 0;
  const actuallyShowLabel = showLabel && label != null && label.length > 0;
  if (!actuallyShowHelpText && !actuallyShowLabel) return children;
  if (dir === "x")
    inputAndHelp = (
      <Flex.Box y gap="small">
        {children}
        {showHelpText && (padHelpText || (helpText != null && helpText.length > 0)) && (
          <HelpText variant={status}>{helpText}</HelpText>
        )}
      </Flex.Box>
    );
  else
    inputAndHelp = (
      <Flex.Box y gap={1 / 3} align="stretch">
        {children}
        {(padHelpText || (helpText != null && helpText.length > 0)) && (
          <HelpText variant={status}>{helpText}</HelpText>
        )}
      </Flex.Box>
    );

  return (
    <Flex.Box
      className={CSS(CSS.BE("input", "item"), className)}
      direction={dir}
      gap={size}
      align={maybeDefaultAlignment(align, dir)}
      {...rest}
    >
      {actuallyShowLabel && <Label required={required}>{label}</Label>}
      {inputAndHelp}
    </Flex.Box>
  );
};
