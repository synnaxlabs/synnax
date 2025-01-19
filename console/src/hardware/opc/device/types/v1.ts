// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type device } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/hardware/opc/device/types/v0";

const VERSION = "1.0.0";

export const propertiesZ = v0.propertiesZ.omit({ read: true }).extend({
  read: z.object({
    indexes: channel.keyZ.array(),
    channels: z.record(z.string(), channel.keyZ),
  }),
  version: z.literal(VERSION),
});
export interface Properties extends z.infer<typeof propertiesZ> {}
export const ZERO_PROPERTIES: Properties = {
  connection: v0.ZERO_CONNECTION_CONFIG,
  read: { indexes: [], channels: {} },
  write: { channels: {} },
  version: VERSION,
};

export interface Device extends device.Device<Properties> {}

export const propertiesMigration = migrate.createMigration<v0.Properties, Properties>({
  name: "hardware.opc.device.properties",
  migrate: (p) => {
    const read = p.read;
    return {
      ...p,
      version: VERSION,
      read: { indexes: read.index === 0 ? [] : [read.index], channels: read.channels },
    };
  },
});
