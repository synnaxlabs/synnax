// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CAN } from "@/feature/can";
import { CONNECT_COMMANDS } from "@/feature/device/commands";
import { HTTP } from "@/feature/http";
import { Kafka } from "@/feature/kafka";
import { Modbus } from "@/feature/modbus";
import { MQTT } from "@/feature/mqtt";
import { OPCUA } from "@/feature/opcua";
import { Serial } from "@/feature/serial";
import { TCP } from "@/feature/tcp";
import { UDP } from "@/feature/udp";
import { FLAGS } from "@/flags";
import { type Command } from "@/platform/command";

export * from "@/feature/device/Connect";
export { INTEGRATIONS } from "@/feature/device/integrations";
export * from "@/feature/device/link";
export * from "@/feature/device/notifications";
export * from "@/feature/device/Toolbar";
export * from "@/feature/device/tree";
export * from "@/feature/device/useListenForChanges";
export * from "@/platform/device/external";

export const COMMANDS: Command.Command[] = [
  ...CONNECT_COMMANDS,
  ...HTTP.Device.COMMANDS,
  ...Kafka.Device.COMMANDS,
  ...Modbus.Device.COMMANDS,
  ...MQTT.Device.COMMANDS,
  ...OPCUA.Device.COMMANDS,
  ...(FLAGS.can
    ? [
        ...CAN.Device.COMMANDS,
        ...Serial.Device.COMMANDS,
        ...TCP.Device.COMMANDS,
        ...UDP.Device.COMMANDS,
      ]
    : []),
];
