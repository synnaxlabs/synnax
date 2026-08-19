// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Item.css";

import { type status } from "@synnaxlabs/client";
import { direction } from "@synnaxlabs/x";
import {
  createContext,
  type ReactElement,
  type ReactNode,
  useContext,
  useId,
} from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { HelpText } from "@/input/HelpText";
import { Label } from "@/input/Label";

const LabelIDContext = createContext<string | undefined>(undefined);
LabelIDContext.displayName = "Input.LabelIDContext";

/**
 * Resolves a control's aria-labelledby: an explicit value wins, then the enclosing
 * {@link Item}'s label id, unless the control carries its own aria-label.
 */
export const useLabelledBy = (props: {
  "aria-label"?: string;
  "aria-labelledby"?: string;
}): string | undefined => {
  const labelID = useContext(LabelIDContext);
  return (
    props["aria-labelledby"] ?? (props["aria-label"] == null ? labelID : undefined)
  );
};

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
  const labelID = useId();
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
    <LabelIDContext value={actuallyShowLabel ? labelID : undefined}>
      <Flex.Box
        className={CSS.cls(CSS.BE("input", "item"), className)}
        direction={dir}
        gap={size}
        align={maybeDefaultAlignment(align, dir)}
        {...rest}
      >
        {actuallyShowLabel && (
          <Label id={labelID} required={required}>
            {label}
          </Label>
        )}
        {inputAndHelp}
      </Flex.Box>
    </LabelIDContext>
  );
};
