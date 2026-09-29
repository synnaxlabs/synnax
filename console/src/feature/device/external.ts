// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ARINC429 } from "@/feature/arinc429";
import { CAN } from "@/feature/can";
import { HTTP } from "@/feature/http";
import { MIL1553 } from "@/feature/mil1553";
import { Modbus } from "@/feature/modbus";
import { OPCUA } from "@/feature/opcua";
import { Serial } from "@/feature/serial";
import { TCP } from "@/feature/tcp";
import { UDP } from "@/feature/udp";
import { FLAGS } from "@/flags";
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
  ...OPCUA.Device.COMMANDS,
  ...(FLAGS.can
    ? [
        ...CAN.Device.COMMANDS,
        ...Serial.Device.COMMANDS,
        ...TCP.Device.COMMANDS,
        ...UDP.Device.COMMANDS,
        ...ARINC429.Device.COMMANDS,
        ...MIL1553.Device.COMMANDS,
      ]
    : []),
];
