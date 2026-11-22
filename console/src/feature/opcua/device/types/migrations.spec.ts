// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { OPCUA } from "@/feature/opcua";
import { v0, v1, v2 } from "@/feature/opcua/device/types/versions";

const CLIENT_CERTIFICATE_KEY = uuid.create();
const SERVER_CERTIFICATE_KEY = uuid.create();

interface TestCase {
  description: string;
  input: v0.Properties | v1.Properties | v2.Properties;
  output: v2.Properties;
}

const TEST_RECORD: TestCase[] = [
  {
    description: "migrates from v0 to v2",
    input: {
      connection: v0.ZERO_CONNECTION_CONFIG,
      read: { index: 0, channels: {} },
      version: "0.0.0",
      write: { channels: {} },
    },
    output: v2.ZERO_PROPERTIES,
  },
  {
    description: "migrates from v1 to v2",
    input: v1.ZERO_PROPERTIES,
    output: v2.ZERO_PROPERTIES,
  },
  {
    description: "does not migrate v2",
    input: v2.ZERO_PROPERTIES,
    output: v2.ZERO_PROPERTIES,
  },
  {
    description: "generic case",
    input: {
      connection: {
        endpoint: "opc.tcp://localhost:48",
        securityMode: "Sign",
        securityPolicy: "Basic256",
      },
      read: { index: 34, channels: { test: 34 } },
      version: "0.0.0",
      write: { channels: { "4132": 4132 } },
    },
    output: {
      connection: {
        endpoint: "opc.tcp://localhost:48",
        securityMode: "Sign",
        securityPolicy: "Basic256",
        password: "",
        clientCertificate: "",
        serverCertificate: "",
        certificateValidationDisabled: false,
        clientTimeoutMs: 5000,
        secureChannelLifetimeMs: 600_000,
        sessionTimeoutMs: 1_200_000,
      },
      read: { indexes: [34], channels: { test: 34 }, fields: [] },
      version: "2.0.0",
      write: { channels: { "4132": 4132 } },
    },
  },
  {
    description: "keeps v1 certificate keys and drops the private key path",
    input: {
      ...v1.ZERO_PROPERTIES,
      connection: {
        ...v0.ZERO_CONNECTION_CONFIG,
        clientCertificate: CLIENT_CERTIFICATE_KEY,
        clientPrivateKey: "/etc/opcua/client.key",
        serverCertificate: SERVER_CERTIFICATE_KEY,
      },
    },
    output: {
      ...v2.ZERO_PROPERTIES,
      connection: {
        ...v2.ZERO_PROPERTIES.connection,
        clientCertificate: CLIENT_CERTIFICATE_KEY,
        serverCertificate: SERVER_CERTIFICATE_KEY,
      },
    },
  },
];

describe("migrations", () =>
  TEST_RECORD.forEach(({ description, input, output }) =>
    it(description, () =>
      expect(OPCUA.Device.propertiesZ.parse(input)).toEqual(output),
    ),
  ));
