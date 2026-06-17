// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TOOLBAR } from "@/hardware/device/Toolbar";
import { EtherCAT } from "@/hardware/ethercat";
import { HTTP } from "@/hardware/http";
import { LabJack } from "@/hardware/labjack";
import { Modbus } from "@/hardware/modbus";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { type Layout } from "@/layout";
import { type Modals } from "@/modals";
import { type Palette } from "@/palette";

export * from "@/hardware/device/link";
export * from "@/hardware/device/notifications";
export * from "@/hardware/device/ontology";
export * from "@/hardware/device/Toolbar";
export * from "@/hardware/device/useListenForChanges";

export const COMMANDS: Palette.Command[] = [
  ...HTTP.Device.COMMANDS,
  ...Modbus.Device.COMMANDS,
  ...OPC.Device.COMMANDS,
];

export const MODALS: Record<string, Modals.Renderer> = {
  ...EtherCAT.Device.MODALS,
  ...HTTP.Device.MODALS,
  ...LabJack.Device.MODALS,
  ...Modbus.Device.MODALS,
  ...NI.Device.MODALS,
  ...OPC.Device.MODALS,
};

export const NAV_DRAWER_ITEMS: Layout.NavDrawerItem[] = [TOOLBAR];
