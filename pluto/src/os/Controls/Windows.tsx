// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/os/Controls/Windows.css";

import { type ReactElement } from "react";

import { Button as CoreButton } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { type InternalControlsProps } from "@/os/Controls/types";

export const Windows = ({
  disabled = [],
  onMinimize,
  onMaximize,
  onClose,
  // no-ops on windows
  onFullscreen: _,
  focused: __,
  contrast = 2,
  ...rest
}: InternalControlsProps): ReactElement => (
  <Flex.Box pack {...rest}>
    <Button
      className={CSS.BM("windows-control", "minimize")}
      onClick={onMinimize}
      disabled={disabled.includes("minimize")}
      contrast={contrast}
    >
      <Icon.Subtract />
    </Button>
    <Button
      className={CSS.BM("windows-control", "maximize")}
      onClick={onMaximize}
      disabled={disabled.includes("maximize")}
      contrast={contrast}
    >
      <Icon.Box />
    </Button>
    <Button
      onClick={onClose}
      className={CSS.BM("windows-control", "close")}
      disabled={disabled.includes("close")}
      contrast={contrast}
    >
      <Icon.Close />
    </Button>
  </Flex.Box>
);

const Button = ({
  disabled = false,
  className,
  ...rest
}: CoreButton.ButtonProps): ReactElement | null =>
  !disabled ? (
    <CoreButton.Button
      className={CSS(CSS.B("windows-control"), className)}
      tabIndex={-1}
      {...rest}
    />
  ) : null;
