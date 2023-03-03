// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { OSControlsProps } from "./types";

import { Button, ButtonIconProps, Pack } from "@/core";

import "./WindowsControls.css";

export const WindowsControls = ({
  onMinimize,
  onMaximize,
  onClose,
  // no-op on windows
  onFullscreen: _,
  ...props
}: OSControlsProps): JSX.Element => {
  return (
    <Pack {...props}>
      <WindowsControlButton onClick={onMinimize}>
        <Icon.Subtract />
      </WindowsControlButton>
      <WindowsControlButton onClick={onMaximize}>
        <Icon.Box />
      </WindowsControlButton>
      <WindowsControlButton onClick={onClose} className="pluto-windows-control--close">
        <Icon.Close />
      </WindowsControlButton>
    </Pack>
  );
};

interface WindowsControlButtonProps extends ButtonIconProps {
  disabled?: boolean;
}

const WindowsControlButton = (props: WindowsControlButtonProps): JSX.Element => (
  <Button.Icon {...props} />
);
