// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OPC } from "@/feature/opc";
import { createOPCDevice } from "@/feature/opc/testutil";
import {
  awaitCommand,
  clickDeploy,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";
import { getLabeledInput, uniqueName } from "@/testutil";

const client = createTestClient();

const renderRead = async (options: RenderTaskFormTabOptions = {}) =>
  await renderTaskFormTab(OPC.Task.Read, options);

interface CreateInputChannelOverrides extends Partial<OPC.Task.InputChannel> {}

const createInputChannel = (
  overrides: CreateInputChannelOverrides = {},
): OPC.Task.InputChannel => {
  // Underscore-free so the device-properties record keys survive the server's
  // snake-to-camel decode untouched.
  const nodeName = uniqueName("node").replace(/_/g, "");
  return {
    key: `ns=1;s=${nodeName}`,
    nodeId: `ns=1;s=${nodeName}`,
    nodeName,
    channel: 0,
    enabled: true,
    useAsIndex: false,
    dataType: "float32",
    name: "",
    ...overrides,
  };
};

const createReadConfig = (
  device: string,
  channels: OPC.Task.InputChannel[],
): OPC.Task.ReadPayload["config"] => ({
  ...OPC.Task.ZERO_READ_PAYLOAD.config,
  device,
  channels,
});

// Draft creates mint their own key; the zero payload's empty key must not be sent.
const { key: _key, ...ZERO_DRAFT } = OPC.Task.ZERO_READ_PAYLOAD;

const createDraft = async (client: Synnax, config: OPC.Task.ReadPayload["config"]) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, OPC.Task.READ_SCHEMAS);

const deployAndAwaitTask = async (
  client: Synnax,
  container: ParentNode,
  key: task.Key,
) => {
  const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
  try {
    await clickDeploy(container);
    await awaitCommand(streamer, key);
  } finally {
    streamer.close();
  }
  return await client.tasks.retrieve({ key, schemas: OPC.Task.READ_SCHEMAS });
};

describe("OPC.Read", () => {
  it("should create channels under a new index on deploy", async () => {
    const dev = await createOPCDevice(client);
    const chA = createInputChannel();
    const chB = createInputChannel();
    const draft = await createDraft(client, createReadConfig(dev.key, [chA, chB]));
    const { container } = await renderRead({ client, taskKey: draft.key });
    await screen.findByText(new RegExp(chA.nodeName));
    await screen.findByText(new RegExp(chB.nodeName));

    const created = await deployAndAwaitTask(client, container, draft.key);
    expect(created.rack).toBe(dev.rack);
    const { config } = created;
    expect(config.channels).toHaveLength(2);
    config.channels.forEach(({ channel }) => expect(channel).not.toBe(0));

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: OPC.Device.SCHEMAS,
    });
    expect(updated.properties.read.indexes).toHaveLength(1);
    const indexKey = updated.properties.read.indexes[0];
    const index = await client.channels.retrieve(indexKey);
    expect(index.isIndex).toBe(true);

    for (const ch of config.channels) {
      const created = await client.channels.retrieve(ch.channel);
      expect(created.index).toBe(indexKey);
      expect(created.dataType.toString()).toBe("float32");
    }
    expect(updated.properties.read.channels[chA.nodeId]).toBe(
      config.channels[0].channel,
    );
    expect(updated.properties.read.channels[chB.nodeId]).toBe(
      config.channels[1].channel,
    );
  });

  it("should use the flagged timestamp channel as the index and reuse it on redeploy", async () => {
    const dev = await createOPCDevice(client);
    const tsChannel = createInputChannel({ useAsIndex: true, dataType: "timestamp" });
    const dataChannel = createInputChannel();
    const draft = await createDraft(
      client,
      createReadConfig(dev.key, [tsChannel, dataChannel]),
    );
    const first = await renderRead({ client, taskKey: draft.key });
    await screen.findByText(new RegExp(tsChannel.nodeName));
    expect(screen.getAllByText("Use as Index")).toHaveLength(1);

    await deployAndAwaitTask(client, first.container, draft.key);

    const afterFirst = await client.devices.retrieve({
      key: dev.key,
      schemas: OPC.Device.SCHEMAS,
    });
    const indexKey = afterFirst.properties.read.channels[tsChannel.nodeId];
    expect(afterFirst.properties.read.indexes).toContain(indexKey);
    const index = await client.channels.retrieve(indexKey);
    expect(index.isIndex).toBe(true);
    const dataKey = afterFirst.properties.read.channels[dataChannel.nodeId];
    const created = await client.channels.retrieve(dataKey);
    expect(created.index).toBe(indexKey);
    first.unmount();

    const second = await renderRead({ client, taskKey: draft.key });
    await screen.findByText(new RegExp(tsChannel.nodeName));
    await deployAndAwaitTask(client, second.container, draft.key);
    const afterSecond = await client.devices.retrieve({
      key: dev.key,
      schemas: OPC.Device.SCHEMAS,
    });
    expect(afterSecond.properties.read.indexes).toEqual(
      afterFirst.properties.read.indexes,
    );
    expect(afterSecond.properties.read.channels).toEqual(
      afterFirst.properties.read.channels,
    );
  });

  it("should swap the stream rate field for an array size field in array mode", async () => {
    const dev = await createOPCDevice(client);
    const ch = createInputChannel();
    const draft = await createDraft(client, createReadConfig(dev.key, [ch]));
    await renderRead({ client, taskKey: draft.key });
    // The seeded channel appearing means the task row's config has loaded.
    await screen.findByText(new RegExp(ch.nodeName));
    await screen.findByText("Stream Rate");
    expect(screen.queryByText("Array Size")).toBeNull();

    const arrayModeSwitch = getLabeledInput("Array Sampling");
    fireEvent.click(arrayModeSwitch);
    await screen.findByText("Array Size");
    expect(screen.queryByText("Stream Rate")).toBeNull();

    fireEvent.click(arrayModeSwitch);
    await screen.findByText("Stream Rate");
    expect(screen.queryByText("Array Size")).toBeNull();
  });
});
