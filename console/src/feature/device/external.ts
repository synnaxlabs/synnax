// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TOOLBAR } from "@/feature/device/Toolbar";
import { HTTP } from "@/feature/http";
import { Modbus } from "@/feature/modbus";
import { OPC } from "@/feature/opc";
import { type Nav } from "@/primitive/nav";
import { type Palette } from "@/primitive/palette";

export * from "@/feature/device/ChangeIdentifierMenuItem";
export * from "@/feature/device/ConfigureMenuItem";
export * from "@/feature/device/EditConnectionMenuItem";
export * from "@/feature/device/link";
export * from "@/feature/device/notifications";
export * from "@/feature/device/ontology";
export * from "@/feature/device/TaskContextMenuItems";
export * from "@/feature/device/Toolbar";
export * from "@/feature/device/useListenForChanges";

export const COMMANDS: Palette.Command[] = [
  ...HTTP.Device.COMMANDS,
  ...Modbus.Device.COMMANDS,
  ...OPC.Device.COMMANDS,
];

export const NAV_DRAWER_ITEMS: Nav.Item[] = [TOOLBAR];
