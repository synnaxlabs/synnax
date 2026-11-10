// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import {
  connectionConfigZ,
  type Properties as BaseProperties,
  ZERO_CONNECTION_CONFIG,
  ZERO_PROPERTIES as BASE_ZERO_PROPERTIES,
} from "@/feature/opcua/device/types/v0";

export const VERSION = "1.0.0";

export const readZ = z.object({
  indexes: z.array(channel.keyZ).default(() => []),
  channels: z.record(z.string(), channel.keyZ).default(() => ({})),
});

export const propertiesZ = z.object({
  version: z.literal(VERSION),
  connection: connectionConfigZ.default(() => ({ ...ZERO_CONNECTION_CONFIG })),
  read: readZ.default(() => ({ indexes: [], channels: {} })),
  write: z
    .object({ channels: z.record(z.string(), channel.keyZ).default(() => ({})) })
    .default(() => ({ channels: {} })),
});
export type Properties = z.infer<typeof propertiesZ>;

export const ZERO_PROPERTIES: Properties = {
  ...BASE_ZERO_PROPERTIES,
  read: { indexes: [], channels: {} },
  version: VERSION,
};

export const PROPERTIES_MIGRATION_NAME = "hardware.opc.device.properties";

export const propertiesMigration = migrate.createMigration<BaseProperties, Properties>({
  name: PROPERTIES_MIGRATION_NAME,
  migrate: (p) => {
    const read = p.read;
    return {
      ...p,
      version: VERSION,
      read: { indexes: read.index === 0 ? [] : [read.index], channels: read.channels },
    };
  },
});
