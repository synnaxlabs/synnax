// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel as channelAPI, type device } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/hardware/opc/device/types/v0";

const VERSION = "1.0.0";
type Version = typeof VERSION;

export const propertiesZ = z.object({
  version: z.literal(VERSION),
  connection: v0.connectionConfigZ,
  read: z.object({
    indexes: z.array(channelAPI.keyZ),
    channels: z.record(z.string(), channelAPI.keyZ),
  }),
  write: z.object({
    channels: z.record(z.string(), channelAPI.keyZ),
  }),
});

export type Properties = z.infer<typeof propertiesZ>;

export const ZERO_PROPERTIES: Properties = {
  ...v0.ZERO_PROPERTIES,
  read: { indexes: [], channels: {} },
  version: VERSION,
};

export const PROPERTIES_MIGRATION_NAME = "hardware.opc.device.properties";

export const propertiesMigration = migrate.createMigration<v0.Properties, Properties>({
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

export interface Device extends device.Device<typeof propertiesZ, typeof v0.makeZ> {}
export interface New extends device.New<typeof propertiesZ, typeof v0.makeZ> {}
