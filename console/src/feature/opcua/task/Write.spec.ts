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
import { afterEach, describe, expect, it, vi } from "vitest";
import { type z } from "zod";

import { OPCUA } from "@/feature/opcua";
import {
  createMember,
  createOPCDevice,
  createScannedDevice,
  createVariableNode,
  dragNode,
} from "@/feature/opcua/testutil";
import {
  answerNextCommand,
  awaitEditableForm,
  createChannelReadOnlyClient,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  reportTaskStopped,
} from "@/platform/task/testutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  fakePickedFile,
  interceptFilePicker,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderWrite = async (options: RenderTaskFormTabOptions = {}) => {
  const rendered = await renderTaskFormTab(OPCUA.Task.Write, options);
  await awaitEditableForm();
  return rendered;
};

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
    type: "node",
    objectId: "",
    arguments: [],
  };
};

// The Core holds one channel per name, so each method takes its own name.
const createMethodName = (): string => uniqueName("SetSpeed").replace(/_/g, "");

const createMethodChannel = (
  nodeName: string,
  args: OPCUA.Task.MethodArgument[],
): OPCUA.Task.WriteChannel => ({
  key: uniqueName("method"),
  nodeId: `ns=1;s=Pump.${nodeName}`,
  nodeName,
  cmdChannel: 0,
  disabled: false,
  dataType: args.find(({ source }) => source === "command")?.dataType ?? "uint8",
  name: "",
  type: "method",
  objectId: "ns=1;s=Pump",
  arguments: args,
});

const createWriteConfig = (
  device: string,
  channels: OPCUA.Task.WriteChannel[],
): z.infer<OPCUA.Task.WriteSchemas["config"]> => ({
  ...OPCUA.Task.WRITE_SCHEMAS.config.parse({}),
  device,
  channels,
});

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<OPCUA.Task.WriteSchemas> = {
  name: "OPC UA write task",
  type: OPCUA.Task.WRITE_TYPE,
  config: OPCUA.Task.WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (
  client: Synnax,
  config: z.infer<OPCUA.Task.WriteSchemas["config"]>,
) => await client.tasks.create({ ...ZERO_DRAFT, config }, OPCUA.Task.WRITE_SCHEMAS);

describe("Write", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

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

  it("should bind a new entry to the channel the device already maps", async () => {
    const dev = await createOPCDevice(client);
    const ch = createWriteChannel();
    const firstDraft = await createDraft(client, createWriteConfig(dev.key, [ch]));
    const first = await renderWrite({ client, taskKey: firstDraft.key });
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      firstDraft.key,
      OPCUA.Task.WRITE_SCHEMAS,
    );
    first.unmount();
    const cmd = await client.channels.retrieve(firstTask.config.channels[0].cmdChannel);
    const secondDraft = await createDraft(client, createWriteConfig(dev.key, [ch]));
    await renderWrite({ client, taskKey: secondDraft.key });
    await screen.findByText(cmd.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: secondDraft.key,
        schemas: OPCUA.Task.WRITE_SCHEMAS,
      });
      expect(saved.config.channels[0].cmdChannel).toBe(cmd.key);
    });
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

  it("should withhold rename from a subject who cannot update channels", async () => {
    const dev = await createOPCDevice(client);
    const ch = createWriteChannel();
    const draft = await createDraft(client, createWriteConfig(dev.key, [ch]));
    await renderWrite({
      client,
      taskKey: draft.key,
      as: await createChannelReadOnlyClient(client),
    });
    fireEvent.contextMenu(await screen.findByText(new RegExp(ch.nodeName)));
    // Remove is ungated, so its presence proves the menu resolved before the absence
    // below is read.
    expect(await screen.findByText("Remove")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
  });

  it("should skip the structure field rows of an imported CSV", async () => {
    const picker = interceptFilePicker();
    const dev = await createOPCDevice(client);
    const draft = await createDraft(client, createWriteConfig(dev.key, []));
    await renderWrite({ client, taskKey: draft.key });
    const valve = uniqueName("valve").replace(/_/g, "");
    const nodeId = `ns=1;s=${valve}`;
    fireEvent.click(
      await screen.findByRole("button", { name: "Import channels from CSV" }),
    );
    picker.selectFiles([
      fakePickedFile(
        "channels.csv",
        [
          "node_id,field,name,enabled",
          `${nodeId},Open,open,true`,
          `${nodeId},,,0`,
        ].join("\n"),
      ),
    ]);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: OPCUA.Task.WRITE_SCHEMAS,
      });
      expect(saved.config.channels).toEqual([
        {
          key: nodeId,
          nodeName: valve,
          nodeId,
          name: "",
          cmdChannel: 0,
          disabled: true,
          dataType: "float32",
          type: "node",
          objectId: "",
          arguments: [],
        },
      ]);
    });
  });

  describe("methods", () => {
    const SPEED: OPCUA.Task.MethodArgument = {
      source: "command",
      value: "",
      name: "speed",
      dataType: "float64",
    };
    const MODE: OPCUA.Task.MethodArgument = {
      source: "value",
      value: "auto",
      name: "mode",
      dataType: "string",
    };

    it("should give a method channel a command channel of the command argument's type", async () => {
      const dev = await createOPCDevice(client);
      const ch = createMethodChannel(createMethodName(), [MODE, SPEED]);
      const draft = await createDraft(client, createWriteConfig(dev.key, [ch]));
      const { container } = await renderWrite({ client, taskKey: draft.key });
      await screen.findByText(`Call ${ch.nodeName}`);
      const { config } = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        OPCUA.Task.WRITE_SCHEMAS,
      );
      const cmd = await client.channels.retrieve(config.channels[0].cmdChannel);
      expect(cmd.name).toBe(`${ch.nodeName}_cmd`);
      expect(cmd.dataType.toString()).toBe("float64");
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: OPCUA.Device.SCHEMAS,
      });
      expect(updated.properties.write.channels[ch.nodeId]).toBeUndefined();
    });

    it("should give each channel of one method its own command channel, with a free name", async () => {
      const dev = await createOPCDevice(client);
      const nodeName = createMethodName();
      const draft = await createDraft(
        client,
        createWriteConfig(dev.key, [
          createMethodChannel(nodeName, [MODE, SPEED]),
          createMethodChannel(nodeName, [MODE, SPEED]),
        ]),
      );
      const { container } = await renderWrite({ client, taskKey: draft.key });
      await screen.findByText(`Call ${nodeName}`);
      const { config } = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        OPCUA.Task.WRITE_SCHEMAS,
      );
      const commands = await client.channels.retrieve(
        config.channels.map(({ cmdChannel }) => cmdChannel),
      );
      expect(commands.map(({ name }) => name).sort()).toEqual([
        `${nodeName}_cmd`,
        `${nodeName}_cmd_2`,
      ]);
    });

    it("should replace the command channel when the command moves to an argument of another type", async () => {
      const dev = await createOPCDevice(client);
      const ch = createMethodChannel(createMethodName(), [MODE, SPEED]);
      const draft = await createDraft(client, createWriteConfig(dev.key, [ch]));
      const first = await renderWrite({ client, taskKey: draft.key });
      await screen.findByText(`Call ${ch.nodeName}`);
      const deployed = await deployAndAwaitTask(
        client,
        first.container,
        draft.key,
        OPCUA.Task.WRITE_SCHEMAS,
      );
      const before = deployed.config.channels[0].cmdChannel;
      await reportTaskStopped(client, deployed.payload);
      first.unmount();
      const second = await renderWrite({ client, taskKey: draft.key });
      await screen.findByText(`Call ${ch.nodeName}`);
      const [modeSwitch] = screen.getAllByRole("checkbox", { name: "Command" });
      fireEvent.click(modeSwitch);
      await waitFor(async () => {
        const saved = await client.tasks.retrieve({
          key: draft.key,
          schemas: OPCUA.Task.WRITE_SCHEMAS,
        });
        const [savedCh] = saved.config.channels;
        expect(savedCh.dataType).toBe("string");
        expect(savedCh.arguments.map(({ source }) => source)).toEqual([
          "command",
          "value",
        ]);
      });
      const { config } = await deployAndAwaitTask(
        client,
        second.container,
        draft.key,
        OPCUA.Task.WRITE_SCHEMAS,
      );
      const after = config.channels[0].cmdChannel;
      expect(after).not.toBe(before);
      const cmd = await client.channels.retrieve(after);
      expect(cmd.name).toBe(`${ch.nodeName}_cmd_2`);
      expect(cmd.dataType.toString()).toBe("string");
    });

    it("should add a dropped method with its arguments, the first taking the command", async () => {
      const { scan, device } = await createScannedDevice(client);
      const held = createWriteChannel();
      const draft = await createDraft(client, createWriteConfig(device.key, [held]));
      const method: OPCUA.Task.ScannedNode = {
        key: "ns=1;s=Pump.Configure",
        nodeId: "ns=1;s=Pump.Configure",
        name: "Configure",
        nodeClass: "Method",
        dataType: "",
        isArray: false,
        field: "",
      };
      const root = await answerNextCommand(client, scan, {
        type: OPCUA.Task.BROWSE_COMMAND_TYPE,
        data: { channels: [method] },
      });
      await renderWrite({ client, taskKey: draft.key });
      await root.answered;
      const read = await answerNextCommand(client, scan, {
        type: OPCUA.Task.READ_NODE_COMMAND_TYPE,
        data: {
          node: method,
          nodeClass: "Method",
          browseName: "Configure",
          inputArguments: [
            { name: "mode", typeName: "String", dataType: "string" },
            { name: "speed", typeName: "Double", dataType: "float64" },
          ],
        },
      });
      fireEvent.dragStart(await screen.findByText("Configure"));
      fireEvent.drop(await screen.findByText(new RegExp(held.nodeName)));
      expect((await read.answered).args).toMatchObject({ nodeId: method.nodeId });
      await waitFor(async () => {
        const saved = await client.tasks.retrieve({
          key: draft.key,
          schemas: OPCUA.Task.WRITE_SCHEMAS,
        });
        expect(saved.config.channels).toEqual([
          held,
          {
            key: expect.any(String),
            nodeName: "Configure",
            nodeId: method.nodeId,
            name: "",
            cmdChannel: 0,
            disabled: false,
            type: "method",
            objectId: "",
            dataType: "string",
            arguments: [
              { source: "command", value: "", name: "mode", dataType: "string" },
              { source: "value", value: "", name: "speed", dataType: "float64" },
            ],
          },
        ]);
      });
    });
  });
});

describe("Write structure members", () => {
  it("should add a dropped variable but not a dropped structure member", async () => {
    const { scan, device } = await createScannedDevice(client);
    const held = createWriteChannel();
    const draft = await createDraft(client, createWriteConfig(device.key, [held]));
    const motor = createVariableNode();
    const valve = createVariableNode("float32");
    const root = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [motor, valve] },
    });
    await renderWrite({ client, taskKey: draft.key });
    await root.answered;
    const members = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [createMember(motor, "Speed", "float64")] },
    });
    fireEvent.click(await screen.findByText(motor.name));
    await members.answered;
    await dragNode(`${motor.name}.Speed`, held.nodeName);
    await dragNode(valve.name, held.nodeName);
    await waitFor(async () => {
      const { config } = await client.tasks.retrieve({
        key: draft.key,
        schemas: OPCUA.Task.WRITE_SCHEMAS,
      });
      expect(config.channels.map(({ nodeId }) => nodeId)).toEqual([
        held.nodeId,
        valve.nodeId,
      ]);
    });
  });
});
