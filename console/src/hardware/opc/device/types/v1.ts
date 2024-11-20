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

const { keyZ } = channel;

export const propertiesZ = v0.propertiesZ.omit({ read: true }).extend({
  read: z.object({ indexes: keyZ.array(), channels: z.record(z.string(), keyZ) }),
  version: z.literal("1.0.0"),
});
export type Properties = z.infer<typeof propertiesZ>;
export const ZERO_PROPERTIES: Properties = {
  connection: v0.ZERO_CONNECTION_CONFIG,
  read: { indexes: [], channels: {} },
  write: { channels: {} },
  version: "1.0.0",
};

export type Device = device.Device<Properties>;

export const propertiesMigration = migrate.createMigration<v0.Properties, Properties>({
  name: "hardware.opc.device.properties",
  migrate: (p) => ({
    ...p,
    version: "1.0.0",
    read: { indexes: [p.read.index], channels: p.read.channels },
  }),
});
