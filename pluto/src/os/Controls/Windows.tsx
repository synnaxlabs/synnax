// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";

import { Align } from "@/align";
import { Button as CoreButton } from "@/button";
import { CSS } from "@/css";
import { type InternalControlsProps } from "@/os/Controls/types";

import "@/os/Controls/Windows.css";

export const Windows = ({
  disabled = [],
  onMinimize,
  onMaximize,
  onClose,
  // no-op on windows
  onFullscreen: _,
  ...props
}: InternalControlsProps): ReactElement => (
  <Align.Pack {...props}>
    <Button 
      className={CSS.BM("windows-control", "minimize")}
      onClick={onMinimize} 
      disabled={disabled.includes("minimize")}
    >
      <Icon.Subtract />
    </Button>
    <Button 
      className={CSS.BM("windows-control", "maximize")}
      onClick={onMaximize} disabled={disabled.includes("maximize")}
    >
      <Icon.Box />
    </Button>
    <Button
      onClick={onClose}
      className={CSS.BM("windows-control", "close")}
      disabled={disabled.includes("close")}
    >
      <Icon.Close />
    </Button>
  </Align.Pack>
);

const Button = ({
  disabled = false,
  className,
  ...props
}: CoreButton.IconProps): ReactElement | null =>
  !disabled ? (
    <CoreButton.Icon
      className={CSS(CSS.B("windows-control"), className)}
      tabIndex={-1}
      {...props}
    />
  ) : null;
