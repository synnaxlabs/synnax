// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TOOLBAR } from "@/hardware/device/Toolbar";
import { LabJack } from "@/hardware/labjack";
import { Modbus } from "@/hardware/modbus";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Sift } from "@/hardware/sift";
import { type Layout } from "@/layout";
import { type Palette } from "@/palette";

export * from "@/hardware/device/notifications";
export * from "@/hardware/device/ontology";
export * from "@/hardware/device/Toolbar";
export * from "@/hardware/device/useListenForChanges";

export const COMMANDS: Palette.Command[] = [
  ...Modbus.Device.COMMANDS,
  ...OPC.Device.COMMANDS,
  ...Sift.Device.COMMANDS,
];

export const LAYOUTS: Record<string, Layout.Renderer> = {
  ...LabJack.Device.LAYOUTS,
  ...Modbus.Device.LAYOUTS,
  ...NI.Device.LAYOUTS,
  ...OPC.Device.LAYOUTS,
  ...Sift.Device.LAYOUTS,
};

export const NAV_DRAWER_ITEMS: Layout.NavDrawerItem[] = [TOOLBAR];
