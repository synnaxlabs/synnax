// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type device } from "@synnaxlabs/client";
import { caseconv, migrate } from "@synnaxlabs/x";
import { z } from "zod";

import {
  connectionConfigZ as baseConnectionConfigZ,
  type makeZ,
  ZERO_CONNECTION_CONFIG as BASE_ZERO_CONNECTION_CONFIG,
} from "@/feature/opcua/device/types/v0";
import {
  type Properties as BaseProperties,
  PROPERTIES_MIGRATION_NAME,
  propertiesZ as basePropertiesZ,
  readZ as baseReadZ,
  ZERO_PROPERTIES as BASE_ZERO_PROPERTIES,
} from "@/feature/opcua/device/types/v1";

const VERSION = "2.0.0";

const PEM_CERTIFICATE_HEADER = "-----BEGIN CERTIFICATE-----";

const DEFAULT_CLIENT_TIMEOUT_MS = 5000;
const DEFAULT_SECURE_CHANNEL_LIFETIME_MS = 600_000;
const DEFAULT_SESSION_TIMEOUT_MS = 1_200_000;

const TIMEOUTS = {
  clientTimeoutMs: DEFAULT_CLIENT_TIMEOUT_MS,
  secureChannelLifetimeMs: DEFAULT_SECURE_CHANNEL_LIFETIME_MS,
  sessionTimeoutMs: DEFAULT_SESSION_TIMEOUT_MS,
};

export const connectionConfigZ = baseConnectionConfigZ.extend({
  certificateValidationDisabled: z.boolean().default(false),
  clientTimeoutMs: z.int().positive().default(DEFAULT_CLIENT_TIMEOUT_MS),
  secureChannelLifetimeMs: z
    .int()
    .positive()
    .default(DEFAULT_SECURE_CHANNEL_LIFETIME_MS),
  sessionTimeoutMs: z.int().positive().default(DEFAULT_SESSION_TIMEOUT_MS),
});
export type ConnectionConfig = z.infer<typeof connectionConfigZ>;
const ZERO_CONNECTION_CONFIG: ConnectionConfig = {
  ...BASE_ZERO_CONNECTION_CONFIG,
  certificateValidationDisabled: false,
  ...TIMEOUTS,
};

const readZ = baseReadZ.extend({
  // The channel of each structure field, by node ID and then by field. Both keys are
  // server text, so they keep their case on the wire.
  fields: caseconv.preserveCase(
    z.record(z.string(), z.record(z.string(), channel.keyZ)).default(() => ({})),
  ),
});

export const propertiesZ = basePropertiesZ.extend({
  version: z.literal(VERSION),
  connection: connectionConfigZ.default(() => ({ ...ZERO_CONNECTION_CONFIG })),
  read: readZ.default(() => ({ indexes: [], channels: {}, fields: {} })),
});
export type Properties = z.infer<typeof propertiesZ>;

export const ZERO_PROPERTIES: Properties = {
  ...BASE_ZERO_PROPERTIES,
  version: VERSION,
  connection: ZERO_CONNECTION_CONFIG,
  read: { ...BASE_ZERO_PROPERTIES.read, fields: {} },
};

/** Clears a server certificate that is not PEM, since v1 stored a file path there. */
export const propertiesMigration = migrate.createMigration<BaseProperties, Properties>({
  name: PROPERTIES_MIGRATION_NAME,
  migrate: (p) => {
    const serverCertificate = p.connection.serverCertificate ?? "";
    return {
      ...p,
      version: VERSION,
      read: { ...p.read, fields: {} },
      connection: {
        ...p.connection,
        serverCertificate: serverCertificate.includes(PEM_CERTIFICATE_HEADER)
          ? serverCertificate
          : "",
        certificateValidationDisabled: false,
        ...TIMEOUTS,
      },
    };
  },
});

export interface Device extends device.Device<typeof propertiesZ, typeof makeZ> {}
