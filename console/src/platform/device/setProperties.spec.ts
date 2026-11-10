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
