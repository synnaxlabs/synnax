// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/dialog/Trigger.css";

import { type ReactElement } from "react";

import { Button } from "@/button";
import { Caret } from "@/caret";
import { CSS } from "@/css";
import { useContext } from "@/dialog/Frame";
import { type Icon } from "@/icon";

export interface TriggerProps extends Button.ButtonProps {
  hideCaret?: boolean;
  iconOnly?: boolean;
}

export const Trigger = ({
  onClick,
  className,
  hideCaret = false,
  startIcon,
  iconOnly,
  ...rest
}: TriggerProps): ReactElement => {
  const { toggle, visible, variant } = useContext();
  let endIcon: Icon.ReactElement | undefined;
  if (variant !== "modal" && !hideCaret)
    endIcon = (
      <Caret.Animated enabled={visible} enabledLoc="bottom" disabledLoc="left" />
    );
  if (iconOnly && startIcon != null)
    return (
      <Button.Icon
        className={CSS(CSS.BE("dialog", "trigger"), className)}
        onClick={(e) => {
          onClick?.(e);
          toggle();
        }}
        {...rest}
      >
        {startIcon as Icon.ReactElement}
      </Button.Icon>
    );

  return (
    <Button.Button
      className={CSS(CSS.BE("dialog", "trigger"), className)}
      onClick={(e) => {
        onClick?.(e);
        toggle();
      }}
      startIcon={startIcon}
      endIcon={endIcon}
      {...rest}
    />
  );
};
