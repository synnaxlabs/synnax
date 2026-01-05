// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type AnyProperties, migrateProperties } from "@/hardware/opc/device/types";
import * as v0 from "@/hardware/opc/device/types/v0";
import * as v1 from "@/hardware/opc/device/types/v1";

interface TestCase {
  description: string;
  input: AnyProperties;
  output: v1.Properties;
}

const TEST_RECORD: TestCase[] = [
  {
    description: "migrates from v0 to v1",
    input: {
      connection: v0.ZERO_CONNECTION_CONFIG,
      read: { index: 0, channels: {} },
      version: "0.0.0",
      write: { channels: {} },
    },
    output: v1.ZERO_PROPERTIES,
  },
  {
    description: "does not migrate v1",
    input: v1.ZERO_PROPERTIES,
    output: v1.ZERO_PROPERTIES,
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
      },
      read: { indexes: [34], channels: { test: 34 } },
      version: "1.0.0",
      write: { channels: { "4132": 4132 } },
    },
  },
  {
    description: "edge case with an empty read index",
    input: {
      connection: { ...v0.ZERO_CONNECTION_CONFIG },
      read: { indexes: [0], channels: {} },
      version: "1.0.0",
      write: { channels: {} },
    },
    output: {
      connection: { ...v0.ZERO_CONNECTION_CONFIG },
      read: { indexes: [0], channels: {} },
      version: "1.0.0",
      write: { channels: {} },
    },
  },
];

describe("migrations", () =>
  TEST_RECORD.forEach(({ description, input, output }) =>
    it(description, () => expect(migrateProperties(input)).toEqual(output)),
  ));
