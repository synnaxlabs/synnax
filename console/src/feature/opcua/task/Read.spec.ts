// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OPCUA } from "@/feature/opcua";
import { createOPCDevice } from "@/feature/opcua/testutil";
import {
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  reportTaskStopped,
} from "@/platform/task/testutil";
import { getLabeledInput, uniqueName } from "@/testutil";

const client = createTestClient();

const renderRead = async (options: RenderTaskFormTabOptions = {}) =>
  await renderTaskFormTab(OPCUA.Task.Read, options);

interface CreateReadChannelOverrides extends Partial<OPCUA.Task.ReadChannel> {}

const createReadChannel = (
  overrides: CreateReadChannelOverrides = {},
): OPCUA.Task.ReadChannel => {
  // Underscore-free so the device-properties record keys survive the server's
  // snake-to-camel decode untouched.
  const nodeName = uniqueName("node").replace(/_/g, "");
  return {
    key: `ns=1;s=${nodeName}`,
    nodeId: `ns=1;s=${nodeName}`,
    nodeName,
    channel: 0,
    disabled: false,
    isIndex: false,
    dataType: "float32",
    name: "",
    ...overrides,
  };
};

const createReadConfig = (
  device: string,
  channels: OPCUA.Task.ReadChannel[],
): OPCUA.Task.ReadPayload["config"] => ({
  ...OPCUA.Task.READ_SCHEMAS.config.parse({}),
  device,
  channels,
});

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<OPCUA.Task.ReadSchemas> = {
  name: "OPC UA read task",
  type: OPCUA.Task.READ_TYPE,
  config: OPCUA.Task.READ_SCHEMAS.config.parse({}),
};

const createDraft = async (client: Synnax, config: OPCUA.Task.ReadPayload["config"]) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, OPCUA.Task.READ_SCHEMAS);

describe("OPCUA.Read", () => {
  it("should create channels under a new index on deploy", async () => {
    const dev = await createOPCDevice(client);
    const chA = createReadChannel();
    const chB = createReadChannel();
    const draft = await createDraft(client, createReadConfig(dev.key, [chA, chB]));
    const { container } = await renderRead({ client, taskKey: draft.key });
    await screen.findByText(new RegExp(chA.nodeName));
    await screen.findByText(new RegExp(chB.nodeName));

    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      OPCUA.Task.READ_SCHEMAS,
    );
    expect(created.rack).toBe(dev.rack);
    const { config } = created;
    expect(config.channels).toHaveLength(2);
    config.channels.forEach(({ channel }) => expect(channel).not.toBe(0));

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: OPCUA.Device.SCHEMAS,
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
    const tsChannel = createReadChannel({ isIndex: true, dataType: "timestamp" });
    const dataChannel = createReadChannel();
    const draft = await createDraft(
      client,
      createReadConfig(dev.key, [tsChannel, dataChannel]),
    );
    const first = await renderRead({ client, taskKey: draft.key });
    await screen.findByText(new RegExp(tsChannel.nodeName));
    expect(screen.getAllByText("Use as Index")).toHaveLength(1);

    const deployed = await deployAndAwaitTask(
      client,
      first.container,
      draft.key,
      OPCUA.Task.READ_SCHEMAS,
    );

    const afterFirst = await client.devices.retrieve({
      key: dev.key,
      schemas: OPCUA.Device.SCHEMAS,
    });
    const indexKey = afterFirst.properties.read.channels[tsChannel.nodeId];
    expect(afterFirst.properties.read.indexes).toContain(indexKey);
    const index = await client.channels.retrieve(indexKey);
    expect(index.isIndex).toBe(true);
    const dataKey = afterFirst.properties.read.channels[dataChannel.nodeId];
    const created = await client.channels.retrieve(dataKey);
    expect(created.index).toBe(indexKey);
    await reportTaskStopped(client, deployed.payload);
    first.unmount();

    const second = await renderRead({ client, taskKey: draft.key });
    // The index channel exists by now, so the node id and the resolved channel name
    // both match.
    await screen.findAllByText(new RegExp(tsChannel.nodeName));
    await deployAndAwaitTask(
      client,
      second.container,
      draft.key,
      OPCUA.Task.READ_SCHEMAS,
    );
    const afterSecond = await client.devices.retrieve({
      key: dev.key,
      schemas: OPCUA.Device.SCHEMAS,
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
    const ch = createReadChannel();
    const draft = await createDraft(client, createReadConfig(dev.key, [ch]));
    await renderRead({ client, taskKey: draft.key });
    // The seeded channel appearing means the task row's config has loaded.
    await screen.findByText(new RegExp(ch.nodeName));
    await screen.findByText("Stream rate");
    expect(screen.queryByText("Array size")).toBeNull();

    const arrayModeSwitch = getLabeledInput("Array sampling");
    fireEvent.click(arrayModeSwitch);
    await screen.findByText("Array size");
    expect(screen.queryByText("Stream rate")).toBeNull();

    fireEvent.click(arrayModeSwitch);
    await screen.findByText("Stream rate");
    expect(screen.queryByText("Array size")).toBeNull();
  });
});
