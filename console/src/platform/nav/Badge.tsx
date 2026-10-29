// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface BadgeProps {
  count: number;
  variant: Status.Variant;
}

/** A count on the nav icon of a toolbar, in the color of its variant. */
export const Badge = ({ count, variant }: BadgeProps): ReactElement => (
  <Text.Text
    level="small"
    weight={500}
    className={CSS.cls(CSS.BE("main-nav", "badge"), CSS.M(variant))}
  >
    {count}
  </Text.Text>
);
