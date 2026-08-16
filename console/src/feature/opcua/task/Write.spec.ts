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
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OPCUA } from "@/feature/opcua";
import { createOPCDevice } from "@/feature/opcua/testutil";
import {
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  reportTaskStopped,
} from "@/platform/task/testutil";
import { awaitTextEditingElement, commitTextEdit, uniqueName } from "@/testutil";

const client = createTestClient();

const renderWrite = async (options: RenderTaskFormTabOptions = {}) =>
  await renderTaskFormTab(OPCUA.Task.Write, options);

const createWriteChannel = (): OPCUA.Task.WriteChannel => {
  // Underscore-free so the device-properties record keys survive the server's
  // snake-to-camel decode untouched.
  const nodeName = uniqueName("node").replace(/_/g, "");
  return {
    key: `ns=1;s=${nodeName}`,
    nodeId: `ns=1;s=${nodeName}`,
    nodeName,
    cmdChannel: 0,
    disabled: false,
    dataType: "float32",
    name: "",
  };
};

const createWriteConfig = (
  device: string,
  channels: OPCUA.Task.WriteChannel[],
): OPCUA.Task.WritePayload["config"] => ({
  ...OPCUA.Task.WRITE_SCHEMAS.config.parse({}),
  device,
  channels,
});

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<OPCUA.Task.WriteSchemas> = {
  name: "OPC UA Write Task",
  type: OPCUA.Task.WRITE_TYPE,
  config: OPCUA.Task.WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (client: Synnax, config: OPCUA.Task.WritePayload["config"]) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, OPCUA.Task.WRITE_SCHEMAS);

describe("OPCUA.Write", () => {
  it("should create command and index channels on deploy", async () => {
    const dev = await createOPCDevice(client);
    const chA = createWriteChannel();
    const chB = createWriteChannel();
    const draft = await createDraft(client, createWriteConfig(dev.key, [chA, chB]));
    const { container } = await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(new RegExp(chA.nodeName));
    await screen.findByText(new RegExp(chB.nodeName));

    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      OPCUA.Task.WRITE_SCHEMAS,
    );
    expect(created.rack).toBe(dev.rack);
    const { config } = created;
    expect(config.channels).toHaveLength(2);
    config.channels.forEach(({ cmdChannel }) => expect(cmdChannel).not.toBe(0));

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: OPCUA.Device.SCHEMAS,
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

  it("should reuse existing command channels when redeploying", async () => {
    const dev = await createOPCDevice(client);
    const ch = createWriteChannel();
    const draft = await createDraft(client, createWriteConfig(dev.key, [ch]));
    const first = await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(new RegExp(ch.nodeName));
    const deployed = await deployAndAwaitTask(
      client,
      first.container,
      draft.key,
      OPCUA.Task.WRITE_SCHEMAS,
    );
    const afterFirst = await client.devices.retrieve({
      key: dev.key,
      schemas: OPCUA.Device.SCHEMAS,
    });
    await reportTaskStopped(client, deployed.payload);
    first.unmount();

    const second = await renderWrite({ client, taskKey: draft.key });
    // The command channel exists by now, so the node id and the resolved channel name
    // both match.
    await screen.findAllByText(new RegExp(ch.nodeName));
    await deployAndAwaitTask(
      client,
      second.container,
      draft.key,
      OPCUA.Task.WRITE_SCHEMAS,
    );
    const afterSecond = await client.devices.retrieve({
      key: dev.key,
      schemas: OPCUA.Device.SCHEMAS,
    });
    expect(afterSecond.properties.write.channels).toEqual(
      afterFirst.properties.write.channels,
    );
    const matches = await client.channels.retrieve([`${ch.nodeName}_cmd`]);
    expect(matches).toHaveLength(1);
  });

  it("should rename and remove a channel through the context menu", async () => {
    const dev = await createOPCDevice(client);
    const ch = createWriteChannel();
    const draft = await createDraft(client, createWriteConfig(dev.key, [ch]));
    await renderWrite({ client, taskKey: draft.key });
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
