// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/component/nav";
import { type Palette } from "@/component/palette";
import { TOOLBAR } from "@/service/device/Toolbar";
import { HTTP } from "@/service/http";
import { Modbus } from "@/service/modbus";
import { OPC } from "@/service/opc";

export * from "@/service/device/ChangeIdentifierMenuItem";
export * from "@/service/device/ConfigureMenuItem";
export * from "@/service/device/EditConnectionMenuItem";
export * from "@/service/device/link";
export * from "@/service/device/notifications";
export * from "@/service/device/ontology";
export * from "@/service/device/TaskContextMenuItems";
export * from "@/service/device/Toolbar";
export * from "@/service/device/useListenForChanges";

export const COMMANDS: Palette.Command[] = [
  ...HTTP.Device.COMMANDS,
  ...Modbus.Device.COMMANDS,
  ...OPC.Device.COMMANDS,
];

export const NAV_DRAWER_ITEMS: Nav.Item[] = [TOOLBAR];
