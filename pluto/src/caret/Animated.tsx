// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/caret/Animated.css";

import { Icon, type IconProps } from "@synnaxlabs/media";
import { type location } from "@synnaxlabs/x";

import { CSS } from "@/css";

export interface AnimatedProps extends IconProps {
  enabledLoc: location.Location;
  disabledLoc: location.Location;
  enabled: boolean;
}

export const Animated = ({
  className,
  enabledLoc,
  disabledLoc,
  enabled,
  ...rest
}: AnimatedProps) => (
  <Icon.Caret.Up
    className={CSS(
      CSS.B("caret-animated"),
      CSS.loc(enabled ? enabledLoc : disabledLoc),
      className,
    )}
    {...rest}
  />
);
