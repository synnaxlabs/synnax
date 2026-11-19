// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, icd, mqtt, type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker, openScanner } from "@/feature/mqtt/testutil";
import { Device } from "@/platform/device";
import {
  clickDeploy,
  createChannelReadOnlyClient,
  deployAndAwaitTask,
  findChannelListItem,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  selectFromDropdown,
} from "@/platform/task/testutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  getSwitchInput,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

// Drafts carry no key; the created task mints its own.
const ZERO_DRAFT: task.New<MQTT.Task.WriteSchemas> = {
  name: "MQTT write task",
  type: MQTT.Task.WRITE_TYPE,
  config: MQTT.Task.WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (client: Synnax, config: mqtt.WriteConfig) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, MQTT.Task.WRITE_SCHEMAS);

/**
 * Creates an ICD that holds one message with a float field at /value, and a plain
 * target that sends the commands of a new channel to it.
 */
const createPlainTarget = async (
  key: string,
  topic: string,
  overrides: Partial<MQTT.Task.PlainWriteTarget> = {},
): Promise<MQTT.Task.PlainWriteTarget> => {
  const message = icd.messageZ.parse({
    name: "Command",
    payload: {
      format: "json",
      fields: [{ name: "value", pointer: "/value", type: "float" }],
    },
  });
  const [ic, cmd] = await Promise.all([
    client.icds.create({ name: uniqueName("format"), messages: [message] }),
    client.channels.create({
      name: uniqueName("mqtt_cmd"),
      dataType: "float64",
      virtual: true,
    }),
  ]);
  if (message.payload.format !== "json") throw new Error("expected a JSON message");
  const [field] = message.payload.fields;
  return {
    ...mqtt.plainWriteTargetZ.parse({
      type: "plain",
      icd: ic.key,
      message: {
        message: message.key,
        bindings: [{ field: field.key, channel: cmd.key }],
      },
    }),
    key,
    topic,
    ...overrides,
  };
};

const createTagTarget = (
  key: string,
  tag: string,
  overrides: Partial<MQTT.Task.SparkplugWriteTarget> = {},
): MQTT.Task.SparkplugWriteTarget => ({
  ...mqtt.sparkplugWriteTargetZ.parse({ type: "sparkplug" }),
  key,
  group: "plant",
  edgeNode: "line1",
  tag,
  ...overrides,
});

const retrieveTargets = async (key: task.Key) =>
  (await client.tasks.retrieve({ key, schemas: MQTT.Task.WRITE_SCHEMAS })).config
    .targets;

const createWriteConfig = (
  device: string,
  targets: MQTT.Task.WriteTarget[],
): mqtt.WriteConfig => ({
  ...MQTT.Task.WRITE_SCHEMAS.config.parse({}),
  device,
  targets,
});

// The form body renders against a configured broker only.
const renderWrite = async (
  targets: MQTT.Task.WriteTarget[] = [],
  options: RenderTaskFormTabOptions = {},
) => {
  const dev = await createBroker(client);
  const draft = await createDraft(client, createWriteConfig(dev.key, targets));
  const result = await renderTaskFormTab(MQTT.Task.Write, {
    client,
    taskKey: draft.key,
    ...options,
  });
  return { ...result, dev, draft };
};

// The add button renders once the form leaves its loading preview.
const openAddTargets = async () => {
  fireEvent.click(await screen.findByRole("button", { name: "Add targets" }));
  return within(await screen.findByRole("dialog"));
};

const addTag = async (): Promise<void> => {
  const dialog = await openAddTargets();
  fireEvent.click(await dialog.findByRole("button", { name: "Add a tag by name" }));
  await screen.findByText("Edge node");
};

// The name deploying gives the command channel of the tag of createTagTarget.
const tagCommandName = (deviceName: string, tag: string): string =>
  `${channel.escapeInvalidName(`${deviceName}_plant_line1_${tag}`)}_cmd`;

describe("MQTT Write form", () => {
  it("should show the empty state and add + select a target", async () => {
    await renderWrite();
    await screen.findByText("Select a target to configure");
    await addTag();
    expect(screen.getByText("Sparkplug B type")).toBeTruthy();
  });

  it("should show the topic settings of a plain target and no payload", async () => {
    await renderWrite([
      await createPlainTarget("t1", "plant/valve/set", { retained: true }),
    ]);
    await screen.findByText("Publish");
    expect(screen.getByDisplayValue("plant/valve/set")).toBeTruthy();
    expect(screen.getByText("At least once (1)")).toBeTruthy();
    expect(getSwitchInput("Retain").checked).toBe(true);
    expect(screen.queryByText("JSON type")).toBeNull();
  });

  it("should duplicate a plain target with its message and bindings", async () => {
    const plain = await createPlainTarget("t1", "plant/valve/set");
    const { draft } = await renderWrite([plain]);
    fireEvent.contextMenu(await screen.findByRole("option", { name: /plant\/valve/ }));
    expect(await screen.findByText("Duplicate")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    fireEvent.click(screen.getByText("Duplicate"));
    await waitFor(async () => {
      const targets = await retrieveTargets(draft.key);
      expect(targets).toMatchObject([
        { key: "t1" },
        { topic: plain.topic, icd: plain.icd, message: plain.message },
      ]);
      expect(targets[1].key).not.toBe("t1");
    });
  });

  it("should rename the command channel of a tag via the context menu", async () => {
    const { dev } = await renderWrite([createTagTarget("s1", "setpoint")]);
    const name = tagCommandName(dev.name, "setpoint");
    fireEvent.contextMenu(within(await screen.findByRole("option")).getByText(name));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = await awaitTextEditingElement();
    commitTextEdit(editable, "my_cmd_channel");
    await screen.findByText("my_cmd_channel");
  });

  it("should mark a new command channel, unless its target is disabled", async () => {
    const { dev } = await renderWrite([
      createTagTarget("s1", "setpoint"),
      createTagTarget("s2", "spare", { disabled: true }),
    ]);
    const on = await findChannelListItem(tagCommandName(dev.name, "setpoint"));
    expect(within(on).getByRole("img", { name: "New channel" })).toBeTruthy();
    const off = await findChannelListItem(tagCommandName(dev.name, "spare"));
    expect(within(off).queryByRole("img", { name: "New channel" })).toBeNull();
  });

  it("should withhold rename from a subject who cannot update channels", async () => {
    const { dev } = await renderWrite([createTagTarget("s1", "setpoint")], {
      as: await createChannelReadOnlyClient(client),
    });
    const name = tagCommandName(dev.name, "setpoint");
    fireEvent.contextMenu(within(await screen.findByRole("option")).getByText(name));
    // Duplicate is ungated, so its presence proves the menu resolved before the
    // absence below is read.
    expect(await screen.findByText("Duplicate")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
  });

  it("should bind a tag to the command channel the broker stores, and leave a plain target as it is", async () => {
    const tagCh = await client.channels.create({
      name: uniqueName("mqtt_tag_cmd"),
      dataType: "string",
      virtual: true,
    });
    const dev = await createBroker(client, {
      properties: { write: { "spBv1.0/plant/line1//setpoint": tagCh.key } },
    });
    const plain = await createPlainTarget("t1", "plant/cmd");
    const draft = await createDraft(
      client,
      createWriteConfig(dev.key, [plain, createTagTarget("s1", "setpoint")]),
    );
    await renderTaskFormTab(MQTT.Task.Write, { client, taskKey: draft.key });
    await waitFor(async () =>
      expect(await retrieveTargets(draft.key)).toMatchObject([
        { message: plain.message },
        { channel: tagCh.key },
      ]),
    );
  });

  it("should disable a target through its checkbox", async () => {
    const { draft } = await renderWrite([
      await createPlainTarget("t1", "plant/valve/set"),
    ]);
    const item = await screen.findByRole("option", { name: /plant\/valve\/set/ });
    fireEvent.click(within(item).getByRole("checkbox", { name: "Enabled" }));
    await waitFor(async () =>
      expect(await retrieveTargets(draft.key)).toMatchObject([{ disabled: true }]),
    );
  });

  describe("Sparkplug B targets", () => {
    it("should list a tag with its edge node and its command channel", async () => {
      const ch = await client.channels.create({
        name: uniqueName("mqtt_cmd"),
        dataType: "float64",
        virtual: true,
      });
      await renderWrite([
        await createPlainTarget("t1", "plant/valve/set"),
        createTagTarget("s1", "setpoint", { device: "ovenA", channel: ch.key }),
      ]);
      await screen.findByRole("option", { name: /JSON.*plant\/valve\/set/ });
      expect(
        screen.getByRole("option", { name: /SpB.*plant\/line1\/ovenA\/setpoint/ }),
      ).toBeTruthy();
      await screen.findByText(ch.name);
    });

    it("should focus the group of a tag added by name", async () => {
      await renderWrite();
      const dialog = await openAddTargets();
      fireEvent.click(await dialog.findByRole("button", { name: "Add a tag by name" }));
      const group = await screen.findByPlaceholderText("plant");
      await waitFor(() => expect(document.activeElement).toBe(group));
    });

    it("should add a tag by name and select its Sparkplug B type", async () => {
      const { draft } = await renderWrite();
      await screen.findByText("No targets");
      await addTag();
      expect(screen.getByPlaceholderText("Optional")).toBeTruthy();
      expect(screen.queryByText("JSON type")).toBeNull();
      await selectFromDropdown("Double", "DateTime");
      await waitFor(async () =>
        expect(await retrieveTargets(draft.key)).toMatchObject([
          { type: "sparkplug", tagType: "date_time", channel: 0 },
        ]),
      );
    });

    it("should duplicate a tag with no command channel", async () => {
      const ch = await client.channels.create({
        name: uniqueName("mqtt_cmd"),
        dataType: "float32",
        virtual: true,
      });
      const { draft } = await renderWrite([
        createTagTarget("s1", "setpoint", { channel: ch.key, tagType: "float" }),
      ]);
      fireEvent.contextMenu(await screen.findByRole("option", { name: /setpoint/ }));
      fireEvent.click(await screen.findByText("Duplicate"));
      await waitFor(async () => {
        const targets = await retrieveTargets(draft.key);
        expect(targets).toMatchObject([
          { key: "s1", channel: ch.key },
          { tag: "setpoint", tagType: "float", channel: 0 },
        ]);
        expect(targets[1].key).not.toBe("s1");
      });
    });
  });

  describe("picking targets in the add targets dialog", () => {
    const NODES = [{ group: "plant", edgeNode: "line1", devices: ["ovenA"] }];

    // Opens the dialog over a broker whose scan task answers the edge nodes, then the
    // tags of the first.
    const openOnTags = async (reply: { tags?: object[]; message?: string }) => {
      const { dev, draft } = await renderWrite();
      const scanner = await openScanner(client, dev.rack);
      const dialog = await openAddTargets();
      await scanner.answer({ data: { nodes: NODES, tags: [] } });
      if (reply.message != null)
        await scanner.answer({ variant: "error", message: reply.message });
      else await scanner.answer({ data: { nodes: [], tags: reply.tags } });
      return { dialog, draft, scanner };
    };

    it("should list only Sparkplug B tags", async () => {
      const { dialog, scanner } = await openOnTags({ tags: [] });
      scanner.close();
      await dialog.findByText("line1");
      expect(dialog.queryByRole("button", { name: "JSON topics" })).toBeNull();
      expect(dialog.queryByRole("button", { name: "Add a topic by name" })).toBeNull();
    });

    it("should show a failed listen in place of the list, with no notification", async () => {
      let statuses: Status.NotificationSpec[] = [];
      await renderWrite([], { onStatuses: (s) => (statuses = s) });
      const dialog = await openAddTargets();
      await dialog.findByText(/Found no task where rack = \d+ and type = "mqtt_scan"/);
      expect(statuses).toHaveLength(0);
    });

    it("should add a picked tag as a target of its Sparkplug B type, once", async () => {
      const tags = [
        { device: "", name: "flow", dataType: "float", value: "1", supported: true },
      ];
      const { dialog, draft, scanner } = await openOnTags({ tags });
      fireEvent.click(await dialog.findByRole("checkbox", { name: "Record flow" }));
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
      await screen.findByText("Sparkplug B type");
      await waitFor(async () =>
        expect(await retrieveTargets(draft.key)).toMatchObject([
          {
            type: "sparkplug",
            group: "plant",
            edgeNode: "line1",
            device: "",
            tag: "flow",
            tagType: "float",
            channel: 0,
          },
        ]),
      );
      const again = await openAddTargets();
      await scanner.answer({ data: { nodes: NODES, tags: [] } });
      await scanner.answer({ data: { nodes: [], tags } });
      scanner.close();
      const flow = await again.findByRole<HTMLInputElement>("checkbox", {
        name: "Record flow",
      });
      expect(flow.disabled).toBe(true);
      expect(again.getByText("In task")).toBeTruthy();
    });

    it("should list the tags of an edge node under their devices and lock an unsupported one", async () => {
      const tags = [
        {
          device: "",
          name: "Node Control/Rebirth",
          dataType: "boolean",
          value: "false",
        },
        { device: "ovenA", name: "zoneTemp", dataType: "double", value: "21.5" },
        { device: "ovenA", name: "recipe", dataType: "DataSet", value: "" },
      ].map((t) => ({ ...t, supported: t.dataType !== "DataSet" }));
      const { dialog, scanner } = await openOnTags({ tags });
      scanner.close();
      const zoneTemp = await dialog.findByRole<HTMLInputElement>("checkbox", {
        name: "Record zoneTemp",
      });
      expect(zoneTemp.disabled).toBe(false);
      expect(dialog.getByText("Edge node")).toBeTruthy();
      expect(dialog.getByText("ovenA")).toBeTruthy();
      const recipe = dialog.getByRole<HTMLInputElement>("checkbox", {
        name: "Record recipe",
      });
      expect(recipe.disabled).toBe(true);
      expect(dialog.getByText("Not supported")).toBeTruthy();
    });

    it("should keep the edge nodes when one does not answer", async () => {
      const message = "edge node plant/line1 did not answer the rebirth request";
      const { dialog, scanner } = await openOnTags({ message });
      scanner.close();
      await dialog.findByText(message);
      expect(dialog.getByText("line1")).toBeTruthy();
    });
  });

  describe("deploying against a live Core", () => {
    it("should put the deploy errors of a tag on its fields", async () => {
      const { container } = await renderWrite([
        createTagTarget("s1", "", { device: "oven#1" }),
      ]);
      await screen.findByText("Edge node");
      await clickDeploy(container);
      await screen.findByText("Device must not hold /, +, or #");
      expect(screen.getByText("Tag is required")).toBeTruthy();
    });

    it("should create a command channel of the type of each tag", async () => {
      const { container, dev, draft } = await renderWrite([
        createTagTarget("s1", "zone 1/setpoint", {
          device: "ovenA",
          tagType: "float",
        }),
        createTagTarget("s2", "running", { tagType: "boolean" }),
        createTagTarget("s3", "mode", { tagType: "string" }),
        createTagTarget("s4", "started", { tagType: "date_time" }),
      ]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.WRITE_SCHEMAS,
      );
      const keys = created.config.targets.map((t) =>
        t.type === "sparkplug" ? t.channel : 0,
      );
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      expect(updated.properties.write).toEqual({
        "spBv1.0/plant/line1/ovenA/zone 1/setpoint": keys[0],
        "spBv1.0/plant/line1//running": keys[1],
        "spBv1.0/plant/line1//mode": keys[2],
        "spBv1.0/plant/line1//started": keys[3],
      });
      const channels = await client.channels.retrieve(keys);
      expect(channels.map((ch) => ch.dataType.toString())).toEqual([
        "float32",
        "uint8",
        "string",
        "timestamp",
      ]);
      const cmdName = `${dev.name}_plant_line1_ovenA_zone_1_setpoint_cmd`;
      expect(channels[0].name).toBe(cmdName);
      const indexCh = await client.channels.retrieve(channels[0].index);
      expect(indexCh.name).toBe(`${cmdName}_time`);
      expect(channels[2].virtual).toBe(true);
      expect(channels[3].isIndex).toBe(false);
    });

    it("should give the command channel the name of the target", async () => {
      const name = uniqueName("oven_setpoint_cmd");
      const { container, draft } = await renderWrite([
        createTagTarget("s1", "setpoint", { name }),
      ]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.WRITE_SCHEMAS,
      );
      const [target] = created.config.targets;
      if (target.type !== "sparkplug") throw new Error("expected a Sparkplug B target");
      const cmdCh = await client.channels.retrieve(target.channel);
      expect(cmdCh.name).toBe(name);
      expect((await client.channels.retrieve(cmdCh.index)).name).toBe(`${name}_time`);
    });

    it("should reuse the command channel of a tag that the device stores", async () => {
      const dev = await createBroker(client);
      const storedCh = await client.channels.create({
        name: uniqueName("mqtt_cmd"),
        dataType: "float64",
        virtual: true,
      });
      const properties = {
        ...MQTT.Device.ZERO_PROPERTIES,
        write: { "spBv1.0/plant/line1//setpoint": storedCh.key },
      };
      await Device.setProperties(
        client,
        { key: dev.key, properties },
        MQTT.Device.ZERO_PROPERTIES,
        MQTT.Device.SCHEMAS,
      );
      const config = createWriteConfig(dev.key, [createTagTarget("s1", "setpoint")]);
      const draft = await createDraft(client, config);
      const { container } = await renderTaskFormTab(MQTT.Task.Write, {
        client,
        taskKey: draft.key,
      });
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.WRITE_SCHEMAS,
      );
      expect(created.config.targets).toMatchObject([{ channel: storedCh.key }]);
    });

    it("should put the deploy errors on the fields they belong to", async () => {
      const { container } = await renderWrite([
        await createPlainTarget("t1", "plant/#"),
      ]);
      await screen.findByText("Publish");
      await clickDeploy(container);
      await screen.findByText("Topic must not hold the wildcards + or #");
    });

    it("should deploy a plain target as it is stored", async () => {
      const plain = await createPlainTarget("t1", "plant/valve/set");
      const { container, draft } = await renderWrite([plain]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.WRITE_SCHEMAS,
      );
      expect(created.config.targets).toMatchObject([
        { topic: plain.topic, icd: plain.icd, message: plain.message },
      ]);
    });
  });
});
