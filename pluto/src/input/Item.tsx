// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { direction } from "@synnaxlabs/x";

import { Align } from "@/align";
import { CSS } from "@/css";
import { HelpText } from "@/input/HelpText";
import { Label } from "@/input/Label";

import "@/input/Item.css";

export interface ItemProps extends Align.SpaceProps {
  label?: string;
  showLabel?: boolean;
  helpText?: string;
  padHelpText?: boolean;
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
  align,
  size = "small",
  padHelpText = false,
  ...props
}: ItemProps): ReactElement => {
  let inputAndHelp: ReactElement;
  if (direction === "x")
    inputAndHelp = (
      <Align.Space direction="y" size="small">
        {children}
        {(padHelpText || (helpText != null && helpText.length > 0)) && (
          <HelpText>{helpText}</HelpText>
        )}
      </Align.Space>
    );
  else
    inputAndHelp = (
      <Align.Space direction="y" size={1 / 3}>
        {children}
        {(padHelpText || (helpText != null && helpText.length > 0)) && (
          <HelpText>{helpText}</HelpText>
        )}
      </Align.Space>
    );

  return (
    <Align.Space
      className={CSS(CSS.B("input-item"), className)}
      direction={direction}
      size={size}
      align={maybeDefaultAlignment(align, direction)}
      {...props}
    >
      {showLabel && <Label>{label}</Label>}
      {inputAndHelp}
    </Align.Space>
  );
};
