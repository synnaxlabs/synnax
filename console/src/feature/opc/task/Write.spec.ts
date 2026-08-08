// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OPC } from "@/feature/opc";
import { createOPCDevice } from "@/feature/opc/testutil";
import { awaitTaskKey, renderTaskFormTab } from "@/platform/task/testutil";
import { awaitTextEditingElement, commitTextEdit, uniqueName } from "@/testutil";

const client = createTestClient();

const createOutputChannel = (): OPC.Task.OutputChannel => {
  // Underscore-free so the device-properties record keys survive the server's
  // snake-to-camel decode untouched.
  const nodeName = uniqueName("node").replace(/_/g, "");
  return {
    key: `ns=1;s=${nodeName}`,
    nodeId: `ns=1;s=${nodeName}`,
    nodeName,
    cmdChannel: 0,
    enabled: true,
    dataType: "float32",
    name: "",
  };
};

const createWriteConfig = (
  device: string,
  channels: OPC.Task.OutputChannel[],
): { device: string; autoStart: boolean; channels: OPC.Task.OutputChannel[] } => ({
  device,
  autoStart: false,
  channels,
});

describe("OPC.Write", () => {
  it("should create command and index channels on configure", async () => {
    const dev = await createOPCDevice(client);
    const chA = createOutputChannel();
    const chB = createOutputChannel();
    const rendered = await renderTaskFormTab(OPC.Task.Write, OPC.Task.WRITE_TYPE, {
      client,
      params: { deviceKey: dev.key, config: createWriteConfig(dev.key, [chA, chB]) },
    });
    await screen.findByText(new RegExp(chA.nodeName));
    await screen.findByText(new RegExp(chB.nodeName));

    fireEvent.click(await screen.findByRole("button", { name: /Configure/ }));
    const taskKey = await awaitTaskKey(rendered);

    const tsk = await client.tasks.retrieve(taskKey);
    expect(task.rackKey(tsk.key)).toBe(dev.rack);
    const config = OPC.Task.WRITE_SCHEMAS.config.parse(tsk.config);
    expect(config.channels).toHaveLength(2);
    config.channels.forEach(({ cmdChannel }) => expect(cmdChannel).not.toBe(0));

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: OPC.Device.SCHEMAS,
    });
    expect(updated.properties.write.channels[chA.nodeId]).toBe(
      config.channels[0].cmdChannel,
    );

    const cmd = await client.channels.retrieve(config.channels[0].cmdChannel);
    expect(cmd.name).toBe(`${chA.nodeName}_cmd`);
    expect(cmd.dataType.toString()).toBe("float32");
    const index = await client.channels.retrieve(cmd.index);
    expect(index.name).toBe(`${chA.nodeName}_cmd_time`);
    expect(index.isIndex).toBe(true);
  });

  it("should reuse existing command channels when reconfiguring", async () => {
    const dev = await createOPCDevice(client);
    const ch = createOutputChannel();
    const first = await renderTaskFormTab(OPC.Task.Write, OPC.Task.WRITE_TYPE, {
      client,
      params: { deviceKey: dev.key, config: createWriteConfig(dev.key, [ch]) },
    });
    await screen.findByText(new RegExp(ch.nodeName));
    fireEvent.click(await screen.findByRole("button", { name: /Configure/ }));
    const taskKey = await awaitTaskKey(first);
    const afterFirst = await client.devices.retrieve({
      key: dev.key,
      schemas: OPC.Device.SCHEMAS,
    });
    first.unmount();

    await renderTaskFormTab(OPC.Task.Write, OPC.Task.WRITE_TYPE, {
      client,
      params: { deviceKey: dev.key, taskKey },
    });
    await screen.findByText(new RegExp(ch.nodeName));
    fireEvent.click(screen.getByRole("button", { name: /Configure/ }));
    await waitFor(async () => {
      const afterSecond = await client.devices.retrieve({
        key: dev.key,
        schemas: OPC.Device.SCHEMAS,
      });
      expect(afterSecond.properties.write.channels).toEqual(
        afterFirst.properties.write.channels,
      );
      const matches = await client.channels.retrieve([`${ch.nodeName}_cmd`]);
      expect(matches).toHaveLength(1);
    });
  });

  it("should rename and remove a channel through the context menu", async () => {
    const dev = await createOPCDevice(client);
    const ch = createOutputChannel();
    await renderTaskFormTab(OPC.Task.Write, OPC.Task.WRITE_TYPE, {
      client,
      params: { deviceKey: dev.key, config: createWriteConfig(dev.key, [ch]) },
    });
    fireEvent.contextMenu(await screen.findByText(new RegExp(ch.nodeName)));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = await awaitTextEditingElement();
    commitTextEdit(editable, "my_cmd_channel");
    await screen.findByText("my_cmd_channel");
    fireEvent.contextMenu(screen.getByText("my_cmd_channel"));
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(() => expect(screen.queryByText(new RegExp(ch.nodeName))).toBeNull());
  });
});
