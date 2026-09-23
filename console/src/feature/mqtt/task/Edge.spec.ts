// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, mqtt, type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker } from "@/feature/mqtt/testutil";
import {
  clickDeploy,
  createChannelReadOnlyClient,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  findDialogTriggerByText,
  getHeaderIconButton,
  getLabeledInput,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

// Drafts carry no key; the created task mints its own.
const ZERO_DRAFT: task.New<MQTT.Task.EdgeSchemas> = {
  name: "Sparkplug edge node",
  type: MQTT.Task.EDGE_TYPE,
  config: MQTT.Task.EDGE_SCHEMAS.config.parse({}),
};

const createDraft = async (client: Synnax, config: MQTT.Task.EdgePayload["config"]) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, MQTT.Task.EDGE_SCHEMAS);

const createTag = (
  key: string,
  overrides: Partial<MQTT.Task.EdgeTag> = {},
): MQTT.Task.EdgeTag => ({ ...mqtt.edgeTagZ.parse({}), key, ...overrides });

const createDataChannel = async (dataType: string = "float64") =>
  await client.channels.create({
    name: uniqueName("mqtt_tag"),
    dataType,
    virtual: true,
  });

const retrieveTags = async (key: task.Key) =>
  (await client.tasks.retrieve({ key, schemas: MQTT.Task.EDGE_SCHEMAS })).config.tags;

const createEdgeConfig = (
  device: string,
  tags: MQTT.Task.EdgeTag[],
): MQTT.Task.EdgePayload["config"] => ({
  ...MQTT.Task.EDGE_SCHEMAS.config.parse({}),
  device,
  group: "plant",
  edgeNode: "line1",
  tags,
});

// The form body renders against a configured broker only.
const renderEdge = async (
  tags: MQTT.Task.EdgeTag[] = [],
  options: Omit<RenderTaskFormTabOptions, "onStatuses"> = {},
) => {
  const dev = await createBroker(client);
  const draft = await createDraft(client, createEdgeConfig(dev.key, tags));
  const statuses: Status.NotificationSpec[] = [];
  const result = await renderTaskFormTab(MQTT.Task.Edge, {
    client,
    taskKey: draft.key,
    onStatuses: (next) => {
      statuses.length = 0;
      statuses.push(...next);
    },
    ...options,
  });
  return { ...result, dev, draft, statuses };
};

const openTagMenu = async (): Promise<void> => {
  fireEvent.contextMenu(await screen.findByPlaceholderText("oven/temperature"));
};

const addCommandChannel = async (): Promise<void> => {
  await openTagMenu();
  fireEvent.click(await screen.findByText("Add command channel"));
};

const awaitCommandChannel = async (taskKey: task.Key): Promise<channel.Channel> => {
  let key: channel.Key = 0;
  await waitFor(async () => {
    [{ commandChannel: key }] = await retrieveTags(taskKey);
    expect(key).not.toBe(0);
  });
  return await client.channels.retrieve(key);
};

describe("MQTT Edge form", () => {
  it("should ask for a broker before it shows the form body", async () => {
    await renderTaskFormTab(MQTT.Task.Edge, { task: ZERO_DRAFT });
    await screen.findByText("No device selected");
    expect(screen.queryByText("Channels")).toBeNull();
  });

  it("should show the node fields, the authority, and the empty state", async () => {
    await renderEdge();
    await screen.findByText("No channels in task");
    await screen.findByPlaceholderText("plant");
    expect(screen.getByPlaceholderText("line1")).toBeTruthy();
    expect(getLabeledInput("Authority").value).toBe("0");
    expect(screen.getByText("Auto start")).toBeTruthy();
  });

  it("should add a tag with no channels and the double type", async () => {
    const { draft } = await renderEdge();
    fireEvent.click(await screen.findByText("Add channel"));
    await screen.findByPlaceholderText("oven/temperature");
    await findDialogTriggerByText("Select channel");
    await findDialogTriggerByText("Double");
    expect(screen.getByText("No command channel")).toBeTruthy();
    await waitFor(async () =>
      expect(await retrieveTags(draft.key)).toMatchObject([
        { name: "", channel: 0, sparkplugType: "double", commandChannel: 0 },
      ]),
    );
  });

  it("should copy the type of the last tag into a new one", async () => {
    const { draft } = await renderEdge([createTag("t1", { sparkplugType: "int16" })]);
    await screen.findByPlaceholderText("oven/temperature");
    fireEvent.click(getHeaderIconButton("Channels"));
    await waitFor(async () =>
      expect(await retrieveTags(draft.key)).toMatchObject([
        { key: "t1" },
        { sparkplugType: "int16" },
      ]),
    );
  });

  const selectChannel = async (name: string): Promise<void> => {
    fireEvent.click(await findDialogTriggerByText("Select channel"));
    fireEvent.change(await screen.findByPlaceholderText("Search channels..."), {
      target: { value: name },
    });
    fireEvent.click(await screen.findByText(name));
  };

  it("should name an unnamed tag after the channel it picks", async () => {
    const ch = await createDataChannel();
    const { draft } = await renderEdge([createTag("t1")]);
    await selectChannel(ch.name);
    await waitFor(async () =>
      expect(await retrieveTags(draft.key)).toMatchObject([
        { channel: ch.key, name: ch.name },
      ]),
    );
  });

  it("should keep the name of a named tag when it picks a channel", async () => {
    const ch = await createDataChannel();
    const { draft } = await renderEdge([createTag("t1", { name: "oven/temp" })]);
    await selectChannel(ch.name);
    await waitFor(async () =>
      expect(await retrieveTags(draft.key)).toMatchObject([
        { channel: ch.key, name: "oven/temp" },
      ]),
    );
  });

  it("should show the channel and the command channel of a tag", async () => {
    const ch = await createDataChannel();
    const cmd = await createDataChannel();
    await renderEdge([createTag("t1", { channel: ch.key, commandChannel: cmd.key })]);
    await findDialogTriggerByText(ch.name);
    await screen.findByText(cmd.name);
    expect(screen.queryByText("No command channel")).toBeNull();
  });

  it("should create a command channel of the type of the tag", async () => {
    const name = uniqueName("zone 1/setpoint");
    const { draft } = await renderEdge([
      createTag("t1", { name, channel: 1, sparkplugType: "float" }),
    ]);
    await addCommandChannel();
    const cmd = await awaitCommandChannel(draft.key);
    const cmdName = `${name.replace(/[ /]/g, "_")}_cmd`;
    expect(cmd.name).toBe(cmdName);
    expect(cmd.dataType.toString()).toBe("float32");
    expect((await client.channels.retrieve(cmd.index)).name).toBe(`${cmdName}_time`);
    await screen.findByText(cmdName);
  });

  it("should create a virtual command channel for a string tag", async () => {
    const name = uniqueName("mode");
    const { draft } = await renderEdge([
      createTag("t1", { name, channel: 1, sparkplugType: "string" }),
    ]);
    await addCommandChannel();
    const cmd = await awaitCommandChannel(draft.key);
    expect(cmd.virtual).toBe(true);
    expect(cmd.dataType.toString()).toBe("string");
  });

  it("should refuse a command channel for a tag with no name", async () => {
    const { draft, statuses } = await renderEdge([createTag("t1", { channel: 1 })]);
    await addCommandChannel();
    await waitFor(() =>
      expect(
        statuses.some(({ description }) =>
          description?.includes("Name the tag before adding a command channel"),
        ),
      ).toBe(true),
    );
    expect(await retrieveTags(draft.key)).toMatchObject([{ commandChannel: 0 }]);
  });

  it("should remove the command channel of a tag without deleting it", async () => {
    const cmd = await createDataChannel();
    const { draft } = await renderEdge([
      createTag("t1", { name: "setpoint", channel: 1, commandChannel: cmd.key }),
    ]);
    await screen.findByText(cmd.name);
    await openTagMenu();
    expect(screen.queryByText("Add command channel")).toBeNull();
    fireEvent.click(await screen.findByText("Remove command channel"));
    await screen.findByText("No command channel");
    await waitFor(async () =>
      expect(await retrieveTags(draft.key)).toMatchObject([{ commandChannel: 0 }]),
    );
    expect((await client.channels.retrieve(cmd.key)).key).toBe(cmd.key);
  });

  it("should rename the command channel through the context menu", async () => {
    const cmd = await createDataChannel();
    await renderEdge([
      createTag("t1", { name: "setpoint", channel: 1, commandChannel: cmd.key }),
    ]);
    await screen.findByText(cmd.name);
    await openTagMenu();
    fireEvent.click(await screen.findByText("Rename"));
    const editable = await awaitTextEditingElement();
    const renamed = uniqueName("renamed_cmd");
    commitTextEdit(editable, renamed);
    await waitFor(async () =>
      expect((await client.channels.retrieve(cmd.key)).name).toBe(renamed),
    );
  });

  it("should withhold rename from a subject who cannot update channels", async () => {
    const cmd = await createDataChannel();
    await renderEdge(
      [createTag("t1", { name: "setpoint", channel: 1, commandChannel: cmd.key })],
      { as: await createChannelReadOnlyClient(client) },
    );
    await screen.findByText(cmd.name);
    await openTagMenu();
    // Remove is ungated, so its presence proves the menu resolved before the
    // absence below is read.
    expect(await screen.findByText("Remove")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
  });

  describe("deploying against a live Core", () => {
    it("should put the deploy errors on the fields they belong to", async () => {
      const dev = await createBroker(client);
      const config = { ...createEdgeConfig(dev.key, [createTag("t1")]), group: "a/b" };
      const draft = await createDraft(client, config);
      const { container } = await renderTaskFormTab(MQTT.Task.Edge, {
        client,
        taskKey: draft.key,
      });
      await screen.findByPlaceholderText("oven/temperature");
      await clickDeploy(container);
      await screen.findByText("Group must not hold /, +, or #");
      expect(screen.getByText("Name is required")).toBeTruthy();
      expect(screen.getByText("A channel is required")).toBeTruthy();
    });

    it("should keep the config and take the rack of the broker", async () => {
      const ch = await createDataChannel();
      const cmd = await createDataChannel();
      const tag = createTag("t1", {
        name: "setpoint",
        channel: ch.key,
        commandChannel: cmd.key,
        sparkplugType: "float",
      });
      const { container, dev, draft } = await renderEdge([tag]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.EDGE_SCHEMAS,
      );
      expect(created.rack).toBe(dev.rack);
      expect(created.config).toMatchObject({
        device: dev.key,
        group: "plant",
        edgeNode: "line1",
        tags: [tag],
      });
    });

    it("should drop a command channel that no longer exists", async () => {
      const ch = await createDataChannel();
      const kept = await createDataChannel();
      const deleted = await createDataChannel();
      await client.channels.delete(deleted.key);
      const { container, draft } = await renderEdge([
        createTag("t1", { name: "a", channel: ch.key, commandChannel: kept.key }),
        createTag("t2", { name: "b", channel: ch.key, commandChannel: deleted.key }),
      ]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.EDGE_SCHEMAS,
      );
      expect(created.config.tags.map(({ commandChannel }) => commandChannel)).toEqual([
        kept.key,
        0,
      ]);
    });
  });
});
