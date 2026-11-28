// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";

import type { channel } from "@/channel";
import { createTestClient, waitForStreamLive } from "@/testutil";

const client = createTestClient();

const createChannel = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: `used_${id.create()}`,
    dataType: "float32",
    virtual: true,
  });

const createSchematic = async (ch: channel.Channel) => {
  const proj = await client.projects.create({ name: `uses-${id.create()}` });
  return await client.schematics.create({
    project: proj.key,
    name: `uses-${id.create()}`,
    nodes: [
      {
        key: "value",
        position: { x: 0, y: 0 },
        config: { variant: "value", channel: ch.key },
      },
    ],
  });
};

describe("Uses", () => {
  it("should list a schematic that reads the channel", async () => {
    const ch = await createChannel();
    const sch = await createSchematic(ch);
    const [retrieved] = await client.channels.retrieve({
      where: { key: { eq: ch.key } },
      includeUses: true,
    });
    expect(retrieved.uses).toEqual([
      { id: { type: "schematic", key: sch.key }, direction: "read" },
    ]);
  });

  it("should return empty uses for a channel that nothing uses", async () => {
    const ch = await createChannel();
    const [retrieved] = await client.channels.retrieve({
      where: { key: { eq: ch.key } },
      includeUses: true,
    });
    expect(retrieved.uses).toEqual([]);
  });

  it("should leave uses out when the request does not ask for them", async () => {
    const ch = await createChannel();
    await createSchematic(ch);
    const retrieved = await client.channels.retrieve(ch.key);
    expect(retrieved.uses).toBeUndefined();
  });

  it("should read the uses again after a schematic starts to use the channel", async () => {
    const ch = await createChannel();
    const request = { where: { key: { eq: ch.key } }, includeUses: true };
    await client.channels.retrieve(request);
    const sch = await createSchematic(ch);
    await expect
      .poll(async () => (await client.channels.retrieve(request))[0].uses)
      .toEqual([{ id: { type: "schematic", key: sch.key }, direction: "read" }]);
  });

  it("should read the uses again after an NI device starts to use the channel", async () => {
    await client.connect();
    await waitForStreamLive(client.connection);
    const ch = await createChannel();
    const sch = await createSchematic(ch);
    const live = client.channels.watch({
      where: { key: { eq: ch.key } },
      includeUses: true,
    });
    onTestFinished(live.onChange(() => {}));
    await live.retrieve();
    const cachedUses = () => live.getCached()?.records[0]?.uses;
    const schematicUse = { id: { type: "schematic", key: sch.key }, direction: "read" };
    expect(cachedUses()).toEqual([schematicUse]);
    const rack = await client.racks.create({ name: `uses-${id.create()}` });
    const dev = await client.ni.devices.create({
      rack: rack.key,
      name: `uses-${id.create()}`,
      resourceName: id.create(),
      model: "PXIe-4300",
    });
    await client.ni.devices.update(dev.key, {
      group: "configuration",
      name: dev.name,
      identifier: "",
      analogInput: { index: 0, temperature: ch.key, channels: {} },
      analogOutput: {},
      counterInput: {},
      digitalInput: {},
      digitalOutput: {},
    });
    await expect
      .poll(cachedUses)
      .toEqual([
        { id: { type: "ni_device", key: dev.key }, direction: "write" },
        schematicUse,
      ]);
  });
});

describe("Built-in channels", () => {
  it("should match the built-in channels with the built-in line", async () => {
    const cmd = await client.channels.retrieve("sy_task_cmd");
    const user = await createChannel();
    const keys = [cmd.key, user.key];
    const builtIn = await client.channels.retrieve({
      where: { key: { in: keys }, builtIn: { eq: true } },
    });
    expect(builtIn.map((ch) => ch.key)).toEqual([cmd.key]);
    const other = await client.channels.retrieve({
      where: { key: { in: keys }, builtIn: { eq: false } },
    });
    expect(other.map((ch) => ch.key)).toEqual([user.key]);
  });
});
