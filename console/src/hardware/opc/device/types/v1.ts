// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type device } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/hardware/opc/device/types/v0";

const VERSION = "1.0.0";
type Version = typeof VERSION;

export type Properties = Omit<v0.Properties, "read" | "version"> & {
  read: { indexes: channel.Key[]; channels: Record<string, channel.Key> };
  version: Version;
};
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

export interface Device extends device.Device<Properties, v0.Make> {}
export interface New extends device.New<Properties, v0.Make> {}
