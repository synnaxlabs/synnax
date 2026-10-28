// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
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
  ZERO_PROPERTIES as BASE_ZERO_PROPERTIES,
} from "@/feature/opcua/device/types/v1";

export const VERSION = "2.0.0";

const PEM_CERTIFICATE_HEADER = "-----BEGIN CERTIFICATE-----";

export const connectionConfigZ = baseConnectionConfigZ.extend({
  certificateValidationDisabled: z.boolean().default(false),
});
export type ConnectionConfig = z.infer<typeof connectionConfigZ>;
export const ZERO_CONNECTION_CONFIG: ConnectionConfig = {
  ...BASE_ZERO_CONNECTION_CONFIG,
  certificateValidationDisabled: false,
};

export const propertiesZ = basePropertiesZ.extend({
  version: z.literal(VERSION),
  connection: connectionConfigZ.default(() => ({ ...ZERO_CONNECTION_CONFIG })),
});
export type Properties = z.infer<typeof propertiesZ>;

export const ZERO_PROPERTIES: Properties = {
  ...BASE_ZERO_PROPERTIES,
  version: VERSION,
  connection: ZERO_CONNECTION_CONFIG,
};

/** Clears a server certificate that is not PEM, since v1 stored a file path there. */
export const propertiesMigration = migrate.createMigration<BaseProperties, Properties>({
  name: PROPERTIES_MIGRATION_NAME,
  migrate: (p) => {
    const serverCertificate = p.connection.serverCertificate ?? "";
    return {
      ...p,
      version: VERSION,
      connection: {
        ...p.connection,
        serverCertificate: serverCertificate.includes(PEM_CERTIFICATE_HEADER)
          ? serverCertificate
          : "",
        certificateValidationDisabled: false,
      },
    };
  },
});

export interface Device extends device.Device<typeof propertiesZ, typeof makeZ> {}
