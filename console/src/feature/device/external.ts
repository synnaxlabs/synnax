// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { HTTP } from "@/feature/http";
import { Modbus } from "@/feature/modbus";
import { OPC } from "@/feature/opc";
import { Slack } from "@/feature/slack";
import { type Command } from "@/platform/command";

export * from "@/feature/device/link";
export * from "@/feature/device/notifications";
export * from "@/feature/device/Toolbar";
export * from "@/feature/device/tree";
export * from "@/feature/device/useListenForChanges";
export * from "@/platform/device/external";

export const COMMANDS: Command.Command[] = [
  ...HTTP.Device.COMMANDS,
  ...Modbus.Device.COMMANDS,
  ...OPC.Device.COMMANDS,
  ...Slack.Device.COMMANDS,
];
