// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/caret/Animated.css";

import { type location } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { Icon } from "@/icon";

/** Props for {@link Animated}. */
export interface AnimatedProps extends Icon.IconProps {
  /** Where the caret points while enabled. */
  enabledLoc: location.Location;
  /** Where it points otherwise. */
  disabledLoc: location.Location;
  enabled: boolean;
}

/** A caret that turns between two directions. Use it on anything that opens. */
export const Animated = ({
  className,
  enabledLoc,
  disabledLoc,
  enabled,
  ...rest
}: AnimatedProps) => (
  <Icon.Caret.Up
    className={CSS.cls(
      CSS.B("caret-animated"),
      CSS.loc(enabled ? enabledLoc : disabledLoc),
      className,
    )}
    {...rest}
  />
);
