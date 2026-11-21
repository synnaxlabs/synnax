// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/device/LocationBadge.css";

import { Badge } from "@synnaxlabs/lyra/badge";
import { Device } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface LocationBadgeProps extends Omit<Device.LocationProps, "className"> {}

/**
 * Shows where a device is, such as "USB" or "cDAQ1Mod1", as the devices toolbar does.
 * It gives up its width before the text beside it does.
 */
export const LocationBadge = (props: LocationBadgeProps): ReactElement => (
  <Badge.Badge className={CSS.B("location-badge")}>
    <Device.Location {...props} className={CSS.BE("location-badge", "text")} />
  </Badge.Badge>
);
