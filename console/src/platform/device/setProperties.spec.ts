// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { caseconv, id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Device } from "@/platform/device";
import { createTestDevice } from "@/platform/device/testutil";

const client = createTestClient();

// A retrieve with schemas reads the Core instead of the cache.
const genericSchemas = {
  properties: z.record(z.string(), z.unknown()),
  make: z.string(),
  model: z.string(),
};

const schemas = {
  ...genericSchemas,
  properties: z.object({
    readIndex: z.number(),
    channels: caseconv.preserveCase(z.record(z.string(), z.number())),
  }),
};

describe("setProperties", () => {
  it("should replace the properties of a device", async () => {
    const dev = await createTestDevice(client, {
      properties: { identifier: "dev", readIndex: 1 },
    });
    await Device.setProperties(client, { ...dev, properties: { readIndex: 2 } });
    const stored = await client.devices.retrieve({
      key: dev.key,
      schemas: genericSchemas,
    });
    expect(stored.properties).toEqual({ readIndex: 2 });
    expect(stored.name).toEqual(dev.name);
  });

  it("should keep the case of the keys that the schemas preserve", async () => {
    const dev = await createTestDevice(
      client,
      { properties: { readIndex: 0, channels: {} } },
      schemas,
    );
    const properties = { readIndex: 3, channels: { myTopic: 7 } };
    await Device.setProperties(client, { ...dev, properties }, schemas);
    const stored = await client.devices.retrieve({ key: dev.key, schemas });
    expect(stored.properties).toEqual(properties);
  });

  it("should throw a NotFoundError when the device does not exist", async () => {
    await expect(
      Device.setProperties(client, { key: id.create(), properties: {} }),
    ).rejects.toThrow(NotFoundError);
  });
});
