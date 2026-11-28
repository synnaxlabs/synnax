// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ni, NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { describe, expect, it, vi } from "vitest";

import { NI } from "@/feature/ni";

vi.mock("@/flags", async (importOriginal) => {
  const { FLAGS } = await importOriginal<{ FLAGS: object }>();
  return { FLAGS: { ...FLAGS, typedNIDevices: true } };
});

const client = createTestClient();

const EMPTY: NI.Device.ReadChannels = { index: 0, channels: {} };

const createDevice = async () => {
  const rack = await client.racks.create({ name: uniqueName("ni_rack") });
  return await client.ni.devices.create({
    rack: rack.key,
    name: uniqueName("ni_dev"),
    resourceName: id.create(),
    model: "PXIe-4300",
    identifier: "dev1",
  });
};

const createIndex = async (): Promise<number> =>
  (
    await client.channels.create({
      name: uniqueName("ni_time"),
      dataType: "timestamp",
      isIndex: true,
    })
  ).key;

const createData = async (index: number): Promise<number> =>
  (
    await client.channels.create({
      name: uniqueName("ni_data"),
      dataType: "float32",
      index,
    })
  ).key;

describe("NI.Device read channels with typed devices", () => {
  it("should store analog input channels and the built-in sensor on the device", async () => {
    const dev = await createDevice();
    const index = await createIndex();
    const [ai0, ai3, temp] = await Promise.all([1, 2, 3].map(() => createData(index)));
    const channels = { "0": ai0, "3": ai3, [NI.Device.BUILT_IN_TEMP_KEY]: temp };
    await NI.Device.writeReadChannels(client, dev.key, "analogInput", EMPTY, {
      index,
      channels,
    });
    const stored = await client.ni.devices.retrieve({ key: dev.key });
    expect(stored.analogInput).toEqual({
      index,
      temperature: temp,
      channels: { 0: ai0, 3: ai3 },
    });
    const [target] = await NI.Device.retrieveReadTargets(
      client,
      [dev.key],
      "analogInput",
    );
    expect(target.configured).toBe(true);
    expect(target.channels).toEqual({ index, channels });
  });

  it("should store each counter's channel with the shared index", async () => {
    const dev = await createDevice();
    const index = await createIndex();
    const ctr1 = await createData(index);
    await NI.Device.writeReadChannels(client, dev.key, "counterInput", EMPTY, {
      index,
      channels: { "1": ctr1 },
    });
    const stored = await client.ni.devices.retrieve({ key: dev.key });
    expect(stored.counterInput).toEqual({ 1: { index, channel: ctr1 } });
    const [target] = await NI.Device.retrieveReadTargets(
      client,
      [dev.key],
      "counterInput",
    );
    expect(target.channels).toEqual({ index, channels: { "1": ctr1 } });
  });

  it("should keep the other configuration fields of the device", async () => {
    const dev = await createDevice();
    const index = await createIndex();
    const [ctr0, ai0] = await Promise.all([1, 2].map(() => createData(index)));
    await NI.Device.writeReadChannels(client, dev.key, "counterInput", EMPTY, {
      index,
      channels: { "0": ctr0 },
    });
    await NI.Device.writeReadChannels(client, dev.key, "analogInput", EMPTY, {
      index,
      channels: { "0": ai0 },
    });
    const stored = await client.ni.devices.retrieve({ key: dev.key });
    expect(stored.name).toBe(dev.name);
    expect(stored.identifier).toBe("dev1");
    expect(stored.counterInput).toEqual({ 0: { index, channel: ctr0 } });
  });

  it("should report a device without an identifier as unconfigured", async () => {
    const dev = await createDevice();
    await client.ni.devices.update(dev.key, {
      ...ni.device.configurationZ.parse(dev),
      identifier: "",
      group: "configuration",
    });
    const [target] = await NI.Device.retrieveReadTargets(
      client,
      [dev.key],
      "analogInput",
    );
    expect(target.configured).toBe(false);
  });

  it("should keep the channels another write stored since the snapshot", async () => {
    const dev = await createDevice();
    const index = await createIndex();
    const [ai0, ai1] = await Promise.all([1, 2].map(() => createData(index)));
    await NI.Device.writeReadChannels(client, dev.key, "analogInput", EMPTY, {
      index,
      channels: { "0": ai0 },
    });
    await NI.Device.writeReadChannels(
      client,
      dev.key,
      "analogInput",
      { index, channels: {} },
      { index, channels: { "1": ai1 } },
    );
    const { analogInput } = await client.ni.devices.retrieve({ key: dev.key });
    expect(analogInput.channels).toEqual({ 0: ai0, 1: ai1 });
  });

  it("should delete a counter index that has no counters to hold it", async () => {
    const dev = await createDevice();
    const index = await createIndex();
    await NI.Device.writeReadChannels(client, dev.key, "counterInput", EMPTY, {
      index,
      channels: {},
    });
    await expect(client.channels.retrieve(index)).rejects.toThrow(NotFoundError);
  });
});
