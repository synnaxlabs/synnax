// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";

import { CSS } from "@/css";
import { OSControlsProps } from "@/os/Controls/types";
import { Button, ButtonIconProps, Pack } from "@/core/std";

import "@/core/os/Controls/WindowsControls.css";

export const WindowsControls = ({
  disabled = [],
  onMinimize,
  onMaximize,
  onClose,
  // no-op on windows
  onFullscreen: _,
  ...props
}: OSControlsProps): ReactElement => (
  <Pack {...props}>
    <WindowsControlButton onClick={onMinimize} disabled={disabled.includes("minimize")}>
      <Icon.Subtract />
    </WindowsControlButton>
    <WindowsControlButton onClick={onMaximize} disabled={disabled.includes("maximize")}>
      <Icon.Box />
    </WindowsControlButton>
    <WindowsControlButton
      onClick={onClose}
      className={CSS.BM("windows-control", "close")}
      disabled={disabled.includes("close")}
    >
      <Icon.Close />
    </WindowsControlButton>
  </Pack>
);

interface WindowsControlButtonProps extends ButtonIconProps {
  disabled?: boolean;
}

const WindowsControlButton = ({
  disabled,
  className,
  ...props
}: WindowsControlButtonProps): ReactElement | null =>
  !disabled ? (
    <Button.Icon
      className={CSS(CSS.B("windows-control"), className)}
      tabIndex={-1}
      {...props}
    />
  ) : null;
