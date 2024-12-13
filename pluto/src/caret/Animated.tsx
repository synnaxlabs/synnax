// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/caret/Animated.css";

import { Icon } from "@synnaxlabs/media";
import { type location } from "@synnaxlabs/x";

import { CSS } from "@/css";

export interface AnimatedProps {
  enabledLoc: location.Location;
  disabledLoc: location.Location;
  enabled: boolean;
}

export const Animated = ({ enabledLoc, disabledLoc, enabled }: AnimatedProps) => (
  <Icon.Caret.Up
    className={CSS(
      CSS.B("caret-animated"),
      CSS.loc(enabled ? enabledLoc : disabledLoc),
    )}
  />
);
