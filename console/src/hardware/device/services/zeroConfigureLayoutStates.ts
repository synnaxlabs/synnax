// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Make } from "@/hardware/device/services/make";
import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { type Layout } from "@/layout";

export const ZERO_CONFIGURE_LAYOUTS: Record<Make, Layout.BaseState> = {
  [LabJack.Device.MAKE]: LabJack.Device.ZERO_CONFIGURE_LAYOUT,
  [NI.Device.MAKE]: NI.Device.ZERO_CONFIGURE_LAYOUT,
  [OPC.Device.MAKE]: OPC.Device.ZERO_CONFIGURE_LAYOUT,
};
