// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type device } from "@synnaxlabs/client";
import { z } from "zod";

import {
  connectionConfigZ as baseConnectionConfigZ,
  type makeZ,
  ZERO_CONNECTION_CONFIG as BASE_ZERO_CONNECTION_CONFIG,
} from "@/feature/opcua/device/types/v0";
import {
  type Properties as BaseProperties,
  propertiesZ as basePropertiesZ,
  readZ as baseReadZ,
  ZERO_PROPERTIES as BASE_ZERO_PROPERTIES,
} from "@/feature/opcua/device/types/v1";

const VERSION = "2.0.0";

const DEFAULT_CLIENT_TIMEOUT_MS = 5000;
const DEFAULT_SECURE_CHANNEL_LIFETIME_MS = 600_000;
const DEFAULT_SESSION_TIMEOUT_MS = 1_200_000;

const TIMEOUTS = {
  clientTimeoutMs: DEFAULT_CLIENT_TIMEOUT_MS,
  secureChannelLifetimeMs: DEFAULT_SECURE_CHANNEL_LIFETIME_MS,
  sessionTimeoutMs: DEFAULT_SESSION_TIMEOUT_MS,
};

const wholeNumberZ = (name: string) => z.int(`${name} must be a whole number`);

const itemKeyZ = z.union([z.uuid(), z.literal("")]).default("");

export const connectionConfigZ = baseConnectionConfigZ
  .omit({ clientPrivateKey: true })
  .extend({
    /** The key of the secret that holds the password, or empty for none. */
    password: itemKeyZ,
    /**
     * The key of the certificate that the Driver presents, or empty for the Driver
     * certificate.
     */
    clientCertificate: itemKeyZ,
    /**
     * The key of the trusted server certificate, or empty to trust the first one that
     * the server presents.
     */
    serverCertificate: itemKeyZ,
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
const { clientPrivateKey: _, ...BASE_ZERO_CONNECTION } = BASE_ZERO_CONNECTION_CONFIG;
const ZERO_CONNECTION_CONFIG: ConnectionConfig = {
  ...BASE_ZERO_CONNECTION,
  password: "",
  clientCertificate: "",
  serverCertificate: "",
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

export const propertiesMigration = ({
  connection: { clientPrivateKey: _, ...connection },
  ...p
}: BaseProperties): Properties => ({
  ...p,
  version: VERSION,
  read: { ...p.read, fields: [] },
  connection: {
    ...connection,
    password: connection.password ?? "",
    clientCertificate: connection.clientCertificate ?? "",
    serverCertificate: connection.serverCertificate ?? "",
    certificateValidationDisabled: false,
    ...TIMEOUTS,
  },
});

export interface Device extends device.Device<typeof propertiesZ, typeof makeZ> {}
