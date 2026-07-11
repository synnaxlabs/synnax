// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MouseEventHandler, type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { useContext as useFrameContext } from "@/tabs/Frame";
import { useTabContext } from "@/tabs/Tab";

export interface CloseProps extends Button.ButtonProps {}

/**
 * Close renders a close button inside a Tab, wired to the enclosing Frame's
 * onClose handler. Clicking it closes the tab without selecting it. Children
 * override the default close icon.
 */
export const Close = ({
  className,
  onClick,
  children,
  ...rest
}: CloseProps): ReactElement => {
  const { onClose } = useFrameContext("Tabs.Close");
  const { itemKey } = useTabContext("Tabs.Close");
  const handleClick: MouseEventHandler<HTMLButtonElement> = (e) => {
    e.preventDefault();
    onClick?.(e);
    onClose?.(itemKey);
  };
  return (
    <Button.Button
      aria-label="pluto-tabs__close"
      className={CSS(CSS.BE("tabs", "close"), className)}
      variant="text"
      sharp
      onClick={handleClick}
      {...rest}
    >
      {children ?? <Icon.Close />}
    </Button.Button>
  );
};
