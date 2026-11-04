// Copyright 2026 Synnax Labs, Inc.
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

const wholeNumberZ = (name: string) => z.int(`${name} must be a whole number`);

export const connectionConfigZ = baseConnectionConfigZ.extend({
  certificateValidationDisabled: z.boolean().default(false),
  clientTimeoutMs: wholeNumberZ("Request timeout")
    .positive()
    .default(DEFAULT_CLIENT_TIMEOUT_MS),
  secureChannelLifetimeMs: wholeNumberZ("Channel lifetime")
    .positive()
    .default(DEFAULT_SECURE_CHANNEL_LIFETIME_MS),
  sessionTimeoutMs: wholeNumberZ("Session timeout")
    .positive()
    .default(DEFAULT_SESSION_TIMEOUT_MS),
});
export type ConnectionConfig = z.infer<typeof connectionConfigZ>;
const ZERO_CONNECTION_CONFIG: ConnectionConfig = {
  ...BASE_ZERO_CONNECTION_CONFIG,
  certificateValidationDisabled: false,
  ...TIMEOUTS,
};

const fieldZ = z.object({
  nodeId: z.string(),
  field: z.string(),
  channel: channel.keyZ,
});

const readZ = baseReadZ.extend({
  // The channel of each structure field. Node IDs and field names are values here, not
  // keys, because a device decode without this schema converts the case of keys.
  fields: z.array(fieldZ).default(() => []),
});

export const propertiesZ = basePropertiesZ.extend({
  version: z.literal(VERSION),
  connection: connectionConfigZ.default(() => ({ ...ZERO_CONNECTION_CONFIG })),
  read: readZ.default(() => ({ indexes: [], channels: {}, fields: [] })),
});
export type Properties = z.infer<typeof propertiesZ>;

export const ZERO_PROPERTIES: Properties = {
  ...BASE_ZERO_PROPERTIES,
  version: VERSION,
  connection: ZERO_CONNECTION_CONFIG,
  read: { ...BASE_ZERO_PROPERTIES.read, fields: [] },
};

/** Clears a server certificate that is not PEM, since v1 stored a file path there. */
export const propertiesMigration = migrate.createMigration<BaseProperties, Properties>({
  name: PROPERTIES_MIGRATION_NAME,
  migrate: (p) => {
    const serverCertificate = p.connection.serverCertificate ?? "";
    return {
      ...p,
      version: VERSION,
      read: { ...p.read, fields: [] },
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
