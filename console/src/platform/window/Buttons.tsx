// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/window/Buttons.css";

import { Button as BaseButton } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

type Action = "close" | "minimize" | "maximize";

export interface ButtonsProps extends Flex.BoxProps {
  /** The buttons to leave out. */
  disabled?: Action[];
  maximized?: boolean;
  onMinimize?: () => void;
  onMaximize?: () => void;
  onClose?: () => void;
}

/** The minimize, maximize, and close buttons a window draws for itself on Windows. */
export const Buttons = ({
  disabled = [],
  maximized = false,
  onMinimize,
  onMaximize,
  onClose,
  ...rest
}: ButtonsProps): ReactElement => (
  <Flex.Box pack {...rest}>
    <Button
      action="minimize"
      aria-label="Minimize"
      onClick={onMinimize}
      disabled={disabled.includes("minimize")}
    >
      <Icon.Subtract />
    </Button>
    <Button
      action="maximize"
      aria-label={maximized ? "Restore" : "Maximize"}
      onClick={onMaximize}
      disabled={disabled.includes("maximize")}
    >
      {maximized ? <Icon.Boxes /> : <Icon.Box />}
    </Button>
    <Button
      action="close"
      aria-label="Close"
      onClick={onClose}
      disabled={disabled.includes("close")}
    >
      <Icon.CloseThin />
    </Button>
  </Flex.Box>
);

interface ButtonProps extends BaseButton.ButtonProps {
  action: Action;
}

const Button = ({
  action,
  disabled = false,
  ...rest
}: ButtonProps): ReactElement | null =>
  !disabled ? (
    <BaseButton.Button
      className={CSS.cls(
        CSS.BE("window", "button"),
        CSS.BEM("window", "button", action),
      )}
      tabIndex={-1}
      variant="text"
      {...rest}
    />
  ) : null;
