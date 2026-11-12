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
import { caseconv, deep, id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Device } from "@/platform/device";
import { createTestDevice } from "@/platform/device/testutil";

const client = createTestClient();

// A retrieve with schemas reads the Core instead of the cache.
const schemas = {
  properties: z.object({
    readIndex: z.number(),
    channels: caseconv.preserveCase(z.record(z.string(), z.number())),
  }),
  make: z.string(),
  model: z.string(),
};

const createDevice = async () => {
  const dev = await createTestDevice(
    client,
    { properties: { readIndex: 0, channels: {} } },
    schemas,
  );
  return await client.devices.retrieve({ key: dev.key, schemas });
};

describe("setProperties", () => {
  it("should keep a property that another write changed", async () => {
    const stale = await createDevice();
    const addChannel = async (name: string, key: number) => {
      const properties = deep.copy(stale.properties);
      properties.channels[name] = key;
      await Device.setProperties(
        client,
        { key: stale.key, properties },
        stale.properties,
        schemas,
      );
    };
    await addChannel("myTopic", 7);
    await addChannel("otherTopic", 8);
    const stored = await client.devices.retrieve({ key: stale.key, schemas });
    expect(stored.properties).toEqual({
      readIndex: 0,
      channels: { myTopic: 7, otherTopic: 8 },
    });
  });

  it("should remove a property that the new properties omit", async () => {
    const dev = await createDevice();
    const before = { readIndex: 3, channels: { myTopic: 7, otherTopic: 8 } };
    await Device.setProperties(
      client,
      { ...dev, properties: before },
      dev.properties,
      schemas,
    );
    const properties = { readIndex: 3, channels: { myTopic: 7 } };
    await Device.setProperties(client, { ...dev, properties }, before, schemas);
    const stored = await client.devices.retrieve({ key: dev.key, schemas });
    expect(stored.properties).toEqual(properties);
  });

  it("should write every property of a versioned device", async () => {
    const v1 = z.object({
      version: z.literal(1),
      channels: caseconv.preserveCase(z.record(z.string(), z.number())),
    });
    const v2 = v1.extend({ version: z.literal(2), fields: z.array(z.string()) });
    const migrated = {
      ...schemas,
      properties: v2.or(
        v1.transform((p) => ({ ...p, version: 2 as const, fields: [] })),
      ),
    };
    const created = await createTestDevice(
      client,
      { properties: { version: 1, channels: {} } },
      { ...schemas, properties: v1 },
    );
    const dev = await client.devices.retrieve({ key: created.key, schemas: migrated });
    const properties = { ...deep.copy(dev.properties), fields: ["temperature"] };
    properties.channels.myTopic = 7;
    await Device.setProperties(
      client,
      { ...dev, properties },
      dev.properties,
      migrated,
    );
    const stored = await client.devices.retrieve({ key: dev.key, schemas: migrated });
    expect(stored.properties).toEqual({
      version: 2,
      channels: { myTopic: 7 },
      fields: ["temperature"],
    });
  });

  it("should send nothing when no property differs", async () => {
    const properties = { readIndex: 1, channels: {} };
    await expect(
      Device.setProperties(
        client,
        { key: id.create(), properties },
        properties,
        schemas,
      ),
    ).resolves.toBeUndefined();
  });

  it("should throw a NotFoundError when the device does not exist", async () => {
    await expect(
      Device.setProperties(
        client,
        { key: id.create(), properties: { readIndex: 1, channels: {} } },
        { readIndex: 0, channels: {} },
        schemas,
      ),
    ).rejects.toThrow(NotFoundError);
  });
});
