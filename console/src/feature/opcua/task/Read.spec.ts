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
import { type Status } from "@synnaxlabs/lyra/status";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
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
  answerCommand,
  answerNextCommand,
  awaitCommand,
  awaitEditableForm,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  reportTaskStopped,
} from "@/platform/task/testutil";
import {
  captureBrowserDownloads,
  fakePickedFile,
  findDialogTriggerByText,
  getLabeledInput,
  interceptFilePicker,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderRead = async (options: RenderTaskFormTabOptions = {}) => {
  const rendered = await renderTaskFormTab(OPCUA.Task.Read, options);
  await awaitEditableForm();
  return rendered;
};

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
    field: "",
    ...overrides,
  };
};

const createReadConfig = (
  device: string,
  channels: OPCUA.Task.ReadChannel[],
): z.infer<OPCUA.Task.ReadSchemas["config"]> => ({
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

const createDraft = async (
  client: Synnax,
  config: z.infer<OPCUA.Task.ReadSchemas["config"]>,
) => await client.tasks.create({ ...ZERO_DRAFT, config }, OPCUA.Task.READ_SCHEMAS);

const retrieveChannels = async (key: string) =>
  (await client.tasks.retrieve({ key, schemas: OPCUA.Task.READ_SCHEMAS })).config
    .channels;

const createFolder = (): OPCUA.Task.ScannedNode => {
  const name = uniqueName("Line").replace(/_/g, "");
  return {
    key: `ns=1;s=${name}`,
    nodeId: `ns=1;s=${name}`,
    name,
    nodeClass: "Object",
    dataType: "",
    isArray: false,
    field: "",
  };
};

// Renders a draft whose device sits on a running rack, with a browser that lists only
// the node.
const renderWithNode = async (
  node: OPCUA.Task.ScannedNode,
  channels: OPCUA.Task.ReadChannel[],
) => {
  const { scan, device } = await createScannedDevice(client);
  const draft = await createDraft(client, createReadConfig(device.key, channels));
  const root = await answerNextCommand(client, scan, {
    type: OPCUA.Task.BROWSE_COMMAND_TYPE,
    data: { channels: [node] },
  });
  const statuses: Status.NotificationSpec[] = [];
  await renderRead({
    client,
    taskKey: draft.key,
    onStatuses: (next) => statuses.splice(0, statuses.length, ...next),
  });
  await root.answered;
  return { scan, draft, statuses };
};

// Moves the task from the device named from to the device named to.
const selectDevice = async (from: string, to: string) => {
  fireEvent.click(await findDialogTriggerByText(from));
  const search = await waitFor(() => {
    const dialog = screen.getAllByRole("dialog").at(-1);
    if (dialog == null) throw new Error("the device list did not open");
    return within(dialog).getByRole("textbox");
  });
  fireEvent.change(search, { target: { value: to } });
  fireEvent.click(await screen.findByText(to));
};

const importCSV = async (rows: string[]) => {
  const picker = interceptFilePicker();
  fireEvent.click(
    await screen.findByRole("button", { name: "Import channels from CSV" }),
  );
  picker.selectFiles([fakePickedFile("channels.csv", rows.join("\n"))]);
};

describe("Read", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

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

  it("should map structure field channels by node and field on deploy", async () => {
    const dev = await createOPCDevice(client);
    // Underscores pin that the field map keeps its keys' case on the wire.
    const nodeId = `ns=1;s=${uniqueName("motor")}`;
    // Channel names come from node names, so a fixed name collides across runs.
    const motor = uniqueName("Motor").replace(/_/g, "");
    const speed = createReadChannel({
      key: `${nodeId}/motor_speed`,
      nodeId,
      field: "motor_speed",
      nodeName: `${motor}.motor_speed`,
    });
    const y = createReadChannel({
      key: `${nodeId}/Position.Y`,
      nodeId,
      field: "Position.Y",
      nodeName: `${motor}.Position.Y`,
    });
    const draft = await createDraft(client, createReadConfig(dev.key, [speed, y]));
    const { container } = await renderRead({ client, taskKey: draft.key });
    await screen.findByText(new RegExp(y.nodeName));

    const { config } = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      OPCUA.Task.READ_SCHEMAS,
    );
    expect(config.channels.map(({ field }) => field)).toEqual([
      "motor_speed",
      "Position.Y",
    ]);
    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: OPCUA.Device.SCHEMAS,
    });
    expect(updated.properties.read.fields).toEqual({
      [nodeId]: {
        motor_speed: config.channels[0].channel,
        "Position.Y": config.channels[1].channel,
      },
    });
    expect(updated.properties.read.channels).toEqual({});
  });

  it("should bind a new entry to the channel the device already maps", async () => {
    const dev = await createOPCDevice(client);
    const ch = createReadChannel();
    const firstDraft = await createDraft(client, createReadConfig(dev.key, [ch]));
    const first = await renderRead({ client, taskKey: firstDraft.key });
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      firstDraft.key,
      OPCUA.Task.READ_SCHEMAS,
    );
    first.unmount();
    const existing = await client.channels.retrieve(
      firstTask.config.channels[0].channel,
    );
    const secondDraft = await createDraft(client, createReadConfig(dev.key, [ch]));
    await renderRead({ client, taskKey: secondDraft.key });
    await screen.findByText(existing.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: secondDraft.key,
        schemas: OPCUA.Task.READ_SCHEMAS,
      });
      expect(saved.config.channels[0].channel).toBe(existing.key);
    });
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
    expect(screen.getAllByText("Use as index")).toHaveLength(1);

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

  it("should save NaN fill when its switch is turned on", async () => {
    const dev = await createOPCDevice(client);
    const ch = createReadChannel();
    const draft = await createDraft(client, createReadConfig(dev.key, [ch]));
    await renderRead({ client, taskKey: draft.key });
    await screen.findByText(new RegExp(ch.nodeName));
    const toggle = getLabeledInput("NaN fill");
    expect(toggle.checked).toBe(false);
    fireEvent.click(toggle);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: OPCUA.Task.READ_SCHEMAS,
      });
      expect(saved.config.nanFillEnabled).toBe(true);
    });
  });

  it("should export the channel list to a CSV named after the task", async () => {
    const downloads = captureBrowserDownloads();
    const dev = await createOPCDevice(client);
    const pump = createReadChannel({ name: "pump, inlet" });
    const speed = createReadChannel({ field: "Speed", disabled: true });
    const draft = await createDraft(client, createReadConfig(dev.key, [pump, speed]));
    await renderRead({ client, taskKey: draft.key });
    await screen.findByText(new RegExp(speed.nodeName));
    fireEvent.click(screen.getByRole("button", { name: "Export channels to CSV" }));
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toBe("OPC UA read task_channels.csv");
    expect(new TextDecoder().decode(await downloads.blobs[0].arrayBuffer())).toBe(
      [
        "name,node_name,node_id,field,data_type,enabled,index",
        `"pump, inlet",${pump.nodeName},${pump.nodeId},,float32,true,false`,
        `,${speed.nodeName},${speed.nodeId},Speed,float32,false,false`,
        "",
      ].join("\n"),
    );
  });

  it("should add the channels of an imported CSV that the task does not hold", async () => {
    const dev = await createOPCDevice(client);
    const held = createReadChannel();
    const draft = await createDraft(client, createReadConfig(dev.key, [held]));
    await renderRead({ client, taskKey: draft.key });
    await screen.findByText(new RegExp(held.nodeName));
    const motor = uniqueName("motor").replace(/_/g, "");
    const nodeId = `ns=1;s=${motor}`;
    await importCSV([
      "name,node_id,field,data_type,enabled,index",
      `,${held.nodeId},,float32,true,false`,
      `motor_time,${nodeId},,timestamp,true,true`,
      `,${nodeId},Speed,float64,false,false`,
    ]);
    await waitFor(async () =>
      expect(await retrieveChannels(draft.key)).toEqual([
        held,
        {
          key: nodeId,
          nodeName: motor,
          nodeId,
          field: "",
          channel: 0,
          disabled: false,
          isIndex: true,
          dataType: "timestamp",
          name: "motor_time",
        },
        {
          key: OPCUA.Task.nodeIdentity({ nodeId, field: "Speed" }),
          nodeName: `${motor}.Speed`,
          nodeId,
          field: "Speed",
          channel: 0,
          disabled: true,
          isIndex: false,
          dataType: "float64",
          name: "",
        },
      ]),
    );
  });

  it("should add the variables under a dropped folder, named from their path", async () => {
    const folder = createFolder();
    const held = createReadChannel();
    const { scan, draft, statuses } = await renderWithNode(folder, [held]);
    const pressure = `${folder.nodeId}.Tank.Pressure`;
    const search = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_VARIABLES_COMMAND_TYPE,
      data: {
        channels: [
          {
            nodeId: held.nodeId,
            name: held.nodeName,
            nodeClass: "Variable",
            dataType: "float32",
            isArray: false,
            field: "",
            path: [held.nodeName],
          },
          {
            nodeId: pressure,
            name: "Pressure",
            nodeClass: "Variable",
            dataType: "float64",
            isArray: false,
            field: "",
            path: ["Tank", "Pressure"],
          },
        ],
        truncated: false,
        skipped: [],
      },
    });
    await dragNode(folder.name, held.nodeName);
    expect((await search.answered).args).toMatchObject({ nodeId: folder.nodeId });
    await screen.findByText(`Add 1 variable from ${folder.name}?`);
    expect(statuses).toContainEqual(
      expect.objectContaining({
        variant: "success",
        name: folder.name,
        message: "Found 1 new variable",
      }),
    );
    expect(statuses).not.toContainEqual(
      expect.objectContaining({ variant: "loading", name: folder.name }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    await waitFor(async () =>
      expect(await retrieveChannels(draft.key)).toEqual([
        held,
        {
          key: pressure,
          nodeName: `${folder.name}_Tank_Pressure`,
          nodeId: pressure,
          field: "",
          channel: 0,
          disabled: false,
          isIndex: false,
          dataType: "float64",
          name: "",
        },
      ]),
    );
  });

  it("should number a folder variable whose name the task already has", async () => {
    const folder = createFolder();
    const held = createReadChannel({ nodeName: `${folder.name}_Tank_Pressure` });
    const { scan, draft } = await renderWithNode(folder, [held]);
    const variable = (path: string[]) => ({
      nodeId: `${folder.nodeId}.${path.join(".")}`,
      name: path[path.length - 1],
      nodeClass: "Variable",
      dataType: "float64",
      isArray: false,
      field: "",
      path,
    });
    await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_VARIABLES_COMMAND_TYPE,
      data: {
        channels: [variable(["Tank", "Pressure"]), variable(["Tank_Pressure"])],
        truncated: false,
        skipped: [],
      },
    });
    await dragNode(folder.name, held.nodeName);
    fireEvent.click(await screen.findByRole("button", { name: "Add" }));
    await waitFor(async () =>
      expect((await retrieveChannels(draft.key)).map((ch) => ch.nodeName)).toEqual([
        held.nodeName,
        `${folder.name}_Tank_Pressure_2`,
        `${folder.name}_Tank_Pressure_3`,
      ]),
    );
  });

  it("should browse the server of a device the task moves to", async () => {
    const first = await createScannedDevice(client);
    const second = await createScannedDevice(client);
    const firstNode = createFolder();
    const secondNode = createFolder();
    const draft = await createDraft(client, createReadConfig(first.device.key, []));
    const firstRoot = await answerNextCommand(client, first.scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [firstNode] },
    });
    await renderRead({ client, taskKey: draft.key });
    await firstRoot.answered;
    await screen.findByText(firstNode.name);
    const secondRoot = await answerNextCommand(client, second.scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [secondNode] },
    });
    await selectDevice(first.device.name, second.device.name);
    await secondRoot.answered;
    await screen.findByText(secondNode.name);
    expect(screen.queryByText(firstNode.name)).toBeNull();
  });

  it("should add nothing when the task moves to another server during a search", async () => {
    const first = await createScannedDevice(client);
    const second = await createScannedDevice(client);
    const folder = createFolder();
    const held = createReadChannel();
    const draft = await createDraft(client, createReadConfig(first.device.key, [held]));
    const root = await answerNextCommand(client, first.scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [folder] },
    });
    const statuses: Status.NotificationSpec[] = [];
    await renderRead({
      client,
      taskKey: draft.key,
      onStatuses: (next) => statuses.splice(0, statuses.length, ...next),
    });
    await root.answered;
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    let search: task.Command;
    try {
      await dragNode(folder.name, held.nodeName);
      search = await awaitCommand(
        streamer,
        first.scan.key,
        OPCUA.Task.BROWSE_VARIABLES_COMMAND_TYPE,
      );
    } finally {
      streamer.close();
    }
    const secondRoot = await answerNextCommand(client, second.scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [] },
    });
    await selectDevice(first.device.name, second.device.name);
    await secondRoot.answered;
    await answerCommand(client, first.scan, search, {
      data: {
        channels: [
          {
            nodeId: `${folder.nodeId}.Pressure`,
            name: "Pressure",
            nodeClass: "Variable",
            dataType: "float64",
            isArray: false,
            field: "",
            path: ["Pressure"],
          },
        ],
        truncated: false,
        skipped: [],
      },
    });
    await waitFor(() =>
      expect(statuses).toContainEqual(
        expect.objectContaining({
          variant: "warning",
          name: folder.name,
          message: "Added no variables",
          description: "The task moved to another server during the search.",
        }),
      ),
    );
    expect(screen.queryByRole("button", { name: "Add" })).toBeNull();
    expect(await retrieveChannels(draft.key)).toEqual([held]);
  });

  it("should warn and add nothing when a dropped folder holds no variables", async () => {
    const folder = createFolder();
    const held = createReadChannel();
    const { scan, draft, statuses } = await renderWithNode(folder, [held]);
    const search = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_VARIABLES_COMMAND_TYPE,
      data: { channels: [], truncated: false, skipped: [] },
    });
    await dragNode(folder.name, held.nodeName);
    await search.answered;
    await waitFor(() =>
      expect(statuses).toContainEqual(
        expect.objectContaining({
          variant: "warning",
          name: folder.name,
          message: "Found no new variables",
          description: "No variables are under the folder.",
        }),
      ),
    );
    expect(await retrieveChannels(draft.key)).toEqual([held]);
  });

  it("should say the search stopped early when the reply is truncated", async () => {
    const folder = createFolder();
    const held = createReadChannel();
    const { scan } = await renderWithNode(folder, [held]);
    const search = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_VARIABLES_COMMAND_TYPE,
      data: {
        channels: [{ ...createVariableNode("float64"), path: ["Pressure"] }],
        truncated: true,
        skipped: [],
      },
    });
    await dragNode(folder.name, held.nodeName);
    await search.answered;
    await screen.findByText(`Add 1 variable from ${folder.name}?`);
    expect(
      screen.getByText(
        "The search stopped at a limit before it found every variable. Drag folders further down to add the rest.",
      ),
    ).toBeTruthy();
  });

  it("should say why the server did not browse some objects", async () => {
    const folder = createFolder();
    const held = createReadChannel();
    const { scan } = await renderWithNode(folder, [held]);
    const search = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_VARIABLES_COMMAND_TYPE,
      data: {
        channels: [{ ...createVariableNode("float64"), path: ["Pressure"] }],
        truncated: false,
        skipped: ["ns=1;s=A: Access denied.", "ns=1;s=B: Access denied."],
      },
    });
    await dragNode(folder.name, held.nodeName);
    await search.answered;
    await screen.findByText(`Add 1 variable from ${folder.name}?`);
    expect(
      screen.getByText(
        "The server did not browse 2 objects, so their variables are missing. First failure: ns=1;s=A: Access denied.",
      ),
    ).toBeTruthy();
  });

  it("should replace the search status with the failure when the search fails", async () => {
    const folder = createFolder();
    const held = createReadChannel();
    const { scan, draft, statuses } = await renderWithNode(folder, [held]);
    const search = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_VARIABLES_COMMAND_TYPE,
      variant: "error",
      message: "Failed to browse",
    });
    await dragNode(folder.name, held.nodeName);
    await search.answered;
    await waitFor(() =>
      expect(statuses).toContainEqual(
        expect.objectContaining({
          variant: "error",
          name: folder.name,
          message: "Failed to search for variables",
        }),
      ),
    );
    expect(statuses).not.toContainEqual(
      expect.objectContaining({ variant: "loading", name: folder.name }),
    );
    expect(await retrieveChannels(draft.key)).toEqual([held]);
  });

  it("should add a node dragged from the details", async () => {
    const node = createVariableNode("float64");
    const held = createReadChannel();
    const { scan, draft } = await renderWithNode(node, [held]);
    const read = await answerNextCommand(client, scan, {
      type: OPCUA.Task.READ_NODE_COMMAND_TYPE,
      data: { node, nodeClass: "Variable", browseName: node.name, readable: true },
    });
    fireEvent.click(await screen.findByText(node.name));
    await read.answered;
    // The details header repeats the name below the tree.
    await waitFor(() => expect(screen.getAllByText(node.name)).toHaveLength(2));
    fireEvent.dragStart(screen.getAllByText(node.name)[1]);
    fireEvent.drop(screen.getByText(new RegExp(held.nodeName)));
    await waitFor(async () =>
      expect(
        (await retrieveChannels(draft.key)).map(({ nodeId, dataType }) => ({
          nodeId,
          dataType,
        })),
      ).toEqual([
        { nodeId: held.nodeId, dataType: held.dataType },
        { nodeId: node.nodeId, dataType: "float64" },
      ]),
    );
  });

  it("should report an imported CSV row without a node ID and add nothing", async () => {
    const statuses: Status.NotificationSpec[] = [];
    const dev = await createOPCDevice(client);
    const held = createReadChannel();
    const draft = await createDraft(client, createReadConfig(dev.key, [held]));
    await renderRead({
      client,
      taskKey: draft.key,
      onStatuses: (next) => statuses.splice(0, statuses.length, ...next),
    });
    await screen.findByText(new RegExp(held.nodeName));
    await importCSV(["name,node_id", `pump,ns=1;s=${uniqueName("pump")}`, "tank,"]);
    await waitFor(() =>
      expect(statuses).toContainEqual(
        expect.objectContaining({
          variant: "error",
          message: "Failed to import channels",
          description: "Row 3 has no node_id.",
        }),
      ),
    );
    expect(await retrieveChannels(draft.key)).toEqual([held]);
  });
});

describe("Read structure members", () => {
  it("should add each dropped member of a node once", async () => {
    const { scan, device } = await createScannedDevice(client);
    const held = createReadChannel();
    const draft = await createDraft(client, createReadConfig(device.key, [held]));
    const motor = createVariableNode();
    const root = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [motor] },
    });
    await renderRead({ client, taskKey: draft.key });
    await root.answered;
    const members = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: {
        channels: [
          createMember(motor, "Speed", "float64"),
          createMember(motor, "Torque", "float32"),
        ],
      },
    });
    fireEvent.click(await screen.findByText(motor.name));
    await members.answered;
    await dragNode(`${motor.name}.Speed`, held.nodeName);
    await dragNode(`${motor.name}.Torque`, held.nodeName);
    await dragNode(`${motor.name}.Speed`, held.nodeName);
    await waitFor(async () => {
      const { config } = await client.tasks.retrieve({
        key: draft.key,
        schemas: OPCUA.Task.READ_SCHEMAS,
      });
      expect(config.channels.map(({ nodeId, field }) => ({ nodeId, field }))).toEqual([
        { nodeId: held.nodeId, field: "" },
        { nodeId: motor.nodeId, field: "Speed" },
        { nodeId: motor.nodeId, field: "Torque" },
      ]);
    });
  });
});
