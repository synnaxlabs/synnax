// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OPC } from "@/feature/opc";
import { createOPCDevice } from "@/feature/opc/testutil";
import { awaitTaskKey, renderTaskFormTab } from "@/platform/task/testutil";
import { getLabeledInput, stubGeometry, uniqueName } from "@/testutil";

const client = createTestClient();

stubGeometry();

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
): OPC.Task.ReadConfig => ({
  device,
  sampleRate: 50,
  streamRate: 25,
  arrayMode: false,
  dataSaving: true,
  autoStart: false,
  channels,
});

describe("OPC.Read", () => {
  it("should create channels under a new index on configure", async () => {
    const dev = await createOPCDevice(client);
    const chA = createInputChannel();
    const chB = createInputChannel();
    const rendered = await renderTaskFormTab(OPC.Task.Read, OPC.Task.READ_TYPE, {
      client,
      args: { deviceKey: dev.key, config: createReadConfig(dev.key, [chA, chB]) },
    });
    await screen.findByText(new RegExp(chA.nodeName));
    await screen.findByText(new RegExp(chB.nodeName));

    fireEvent.click(await screen.findByRole("button", { name: /Configure/ }));
    const taskKey = await awaitTaskKey(rendered);

    const tsk = await client.tasks.retrieve({ key: taskKey });
    expect(tsk.rack).toBe(dev.rack);
    const config = OPC.Task.READ_SCHEMAS.config.parse(tsk.config);
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

  it("should use the flagged timestamp channel as the index and reuse it on reconfigure", async () => {
    const dev = await createOPCDevice(client);
    const tsChannel = createInputChannel({ useAsIndex: true, dataType: "timestamp" });
    const dataChannel = createInputChannel();
    const first = await renderTaskFormTab(OPC.Task.Read, OPC.Task.READ_TYPE, {
      client,
      args: {
        deviceKey: dev.key,
        config: createReadConfig(dev.key, [tsChannel, dataChannel]),
      },
    });
    await screen.findByText(new RegExp(tsChannel.nodeName));
    expect(screen.getAllByText("Use as Index")).toHaveLength(1);

    fireEvent.click(await screen.findByRole("button", { name: /Configure/ }));
    const taskKey = await awaitTaskKey(first);

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

    await renderTaskFormTab(OPC.Task.Read, OPC.Task.READ_TYPE, {
      client,
      args: { deviceKey: dev.key, taskKey },
    });
    await screen.findByText(new RegExp(tsChannel.nodeName));
    fireEvent.click(screen.getByRole("button", { name: /Configure/ }));
    await waitFor(async () => {
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
  });

  it("should swap the stream rate field for an array size field in array mode", async () => {
    const dev = await createOPCDevice(client);
    await renderTaskFormTab(OPC.Task.Read, OPC.Task.READ_TYPE, {
      client,
      args: { deviceKey: dev.key },
    });
    await screen.findByRole("button", { name: /Configure/ });
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
