// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/os/Controls/Windows.css";

import { Icon } from "@synnaxlabs/media";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { Button as CoreButton } from "@/button";
import { CSS } from "@/css";
import { type InternalControlsProps } from "@/os/Controls/types";

export const Windows = ({
  disabled = [],
  onMinimize,
  onMaximize,
  onClose,
  // no-ops on windows
  onFullscreen: _,
  focused: __,
  shade = 2,
  ...rest
}: InternalControlsProps): ReactElement => (
  <Align.Pack {...rest}>
    <Button
      className={CSS.BM("windows-control", "minimize")}
      onClick={onMinimize}
      disabled={disabled.includes("minimize")}
      shade={shade}
    >
      <Icon.Subtract />
    </Button>
    <Button
      className={CSS.BM("windows-control", "maximize")}
      onClick={onMaximize}
      disabled={disabled.includes("maximize")}
      shade={shade}
    >
      <Icon.Box />
    </Button>
    <Button
      onClick={onClose}
      className={CSS.BM("windows-control", "close")}
      disabled={disabled.includes("close")}
      shade={shade}
    >
      <Icon.Close />
    </Button>
  </Align.Pack>
);

const Button = ({
  disabled = false,
  className,
  ...rest
}: CoreButton.IconProps): ReactElement | null =>
  !disabled ? (
    <CoreButton.Icon
      className={CSS(CSS.B("windows-control"), className)}
      tabIndex={-1}
      {...rest}
    />
  ) : null;
