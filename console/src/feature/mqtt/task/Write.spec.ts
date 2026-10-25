// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, mqtt, type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker, openScanner } from "@/feature/mqtt/testutil";
import {
  clickDeploy,
  createChannelReadOnlyClient,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  selectFromDropdown,
} from "@/platform/task/testutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  findDialogTriggerByText,
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

const createDraft = async (client: Synnax, config: MQTT.Task.WritePayload["config"]) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, MQTT.Task.WRITE_SCHEMAS);

const createWriteTarget = (
  key: string,
  topic: string,
  channel: Partial<MQTT.Task.ChannelField> = {},
): MQTT.Task.PlainWriteTarget => ({
  ...mqtt.plainWriteTargetZ.parse({ type: "plain" }),
  key,
  topic,
  channel: { ...mqtt.channelFieldZ.parse({}), pointer: "/value", ...channel },
});

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
): MQTT.Task.WritePayload["config"] => ({
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

const addTarget = async (): Promise<void> => {
  const dialog = await openAddTargets();
  fireEvent.click(dialog.getByRole("button", { name: "Add a topic by name" }));
  await screen.findByText("Publish");
};

describe("MQTT Write form", () => {
  it("should show the empty state and add + select a target", async () => {
    await renderWrite();
    await screen.findByText("Select a target to configure");
    await addTarget();
    expect(screen.getByPlaceholderText("plant/line1/valve/set")).toBeTruthy();
    expect(screen.getByText("At least once (1)")).toBeTruthy();
    expect(getSwitchInput("Retain").checked).toBe(false);
    expect(screen.getByText("Data type")).toBeTruthy();
    expect(screen.getAllByText("New topic")).toHaveLength(2);
    expect(screen.getByText("No additional fields")).toBeTruthy();
  });

  it("should show the enum mapping editor when the channel JSON type is string", async () => {
    await renderWrite();
    await addTarget();
    expect(screen.queryByText("Enum mapping")).toBeNull();
    await selectFromDropdown("Number", "String");
    await screen.findByText("Enum mapping");
  });

  it("should drop the enum values when the JSON type leaves string", async () => {
    const { draft } = await renderWrite([
      createWriteTarget("t1", "plant/valve/set", {
        jsonType: "string",
        enumValues: [{ label: "OPEN", value: 1 }],
      }),
    ]);
    await screen.findByDisplayValue("OPEN");
    await selectFromDropdown("String", "Number");
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: MQTT.Task.WRITE_SCHEMAS,
      });
      const [target] = saved.config.targets;
      if (target.type !== "plain") throw new Error("expected a plain target");
      expect(target.channel).toMatchObject({ jsonType: "number", enumValues: [] });
    });
  });

  it("should add a static field and reset its value when the JSON type changes", async () => {
    await renderWrite();
    await addTarget();
    fireEvent.click(screen.getByRole("button", { name: "Static" }));
    await screen.findByText("static");
    const value = screen.getByPlaceholderText<HTMLInputElement>("value");
    fireEvent.change(value, { target: { value: "on" } });
    fireEvent.blur(value);
    await screen.findByDisplayValue("on");
    await selectFromDropdown("String", "Number");
    await waitFor(() => expect(screen.queryByDisplayValue("on")).toBeNull());
  });

  it("should add a generated field and switch its generator to a timestamp", async () => {
    await renderWrite();
    await addTarget();
    fireEvent.click(screen.getByRole("button", { name: "Generated" }));
    await screen.findByText("generated");
    await selectFromDropdown("UUID", "Timestamp (s)");
    await findDialogTriggerByText("Timestamp (s)");
  });

  it("should remove additional fields through a context menu without a duplicate option", async () => {
    await renderWrite();
    await addTarget();
    fireEvent.click(screen.getByRole("button", { name: "Static" }));
    const pointer = await screen.findByPlaceholderText("/field");
    fireEvent.contextMenu(pointer);
    await screen.findByText("Remove");
    expect(screen.queryByText("Duplicate")).toBeNull();
    expect(screen.queryByText("Disable")).toBeNull();
    fireEvent.click(screen.getByText("Remove"));
    await waitFor(() => expect(screen.queryByText("static")).toBeNull());
  });

  it("should duplicate targets and rename the command channel via the context menu", async () => {
    await renderWrite();
    await addTarget();
    const item = await screen.findByText("No channel");
    fireEvent.contextMenu(item);
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() => expect(screen.getAllByText("No channel")).toHaveLength(2));
    fireEvent.contextMenu(screen.getAllByText("No channel")[0]);
    fireEvent.click(await screen.findByText("Rename"));
    const editable = await awaitTextEditingElement();
    commitTextEdit(editable, "my_cmd_channel");
    await screen.findByText("my_cmd_channel");
  });

  it("should withhold rename from a subject who cannot update channels", async () => {
    await renderWrite([], { as: await createChannelReadOnlyClient(client) });
    await addTarget();
    fireEvent.contextMenu(await screen.findByText("No channel"));
    // Duplicate is ungated, so its presence proves the menu resolved before the
    // absence below is read.
    expect(await screen.findByText("Duplicate")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
  });

  // Waits on the channel name so the negative case asserts against a settled lookup
  // rather than one still in flight.
  const renderWithCommandChannel = async (dataType: string) => {
    const ch = await client.channels.create({
      name: uniqueName("mqtt_cmd"),
      dataType,
      virtual: true,
    });
    await renderWrite([createWriteTarget("t1", "plant/cmd", { channel: ch.key })]);
    await screen.findByText("JSON type");
    await screen.findByText(ch.name);
  };

  it("should show the time format field for a timestamp command channel", async () => {
    await renderWithCommandChannel("timestamp");
    await screen.findByText("Time format");
  });

  it("should bind plain and Sparkplug targets to the command channels the broker stores", async () => {
    const [plainCh, tagCh] = await Promise.all(
      ["mqtt_cmd", "mqtt_tag_cmd"].map(
        async (prefix) =>
          await client.channels.create({
            name: uniqueName(prefix),
            dataType: "string",
            virtual: true,
          }),
      ),
    );
    const tag = createTagTarget("s1", "setpoint");
    const dev = await createBroker(client, {
      properties: {
        write: {
          "plant/cmd": plainCh.key,
          "spBv1.0/plant/line1//setpoint": tagCh.key,
        },
      },
    });
    const draft = await createDraft(
      client,
      createWriteConfig(dev.key, [createWriteTarget("t1", "plant/cmd"), tag]),
    );
    await renderTaskFormTab(MQTT.Task.Write, { client, taskKey: draft.key });
    await waitFor(async () =>
      expect(await retrieveTargets(draft.key)).toMatchObject([
        { channel: { channel: plainCh.key } },
        { channel: tagCh.key },
      ]),
    );
  });

  it("should hide the time format field for a numeric command channel", async () => {
    await renderWithCommandChannel("float64");
    expect(screen.queryByText("Time format")).toBeNull();
  });

  it("should name the command channel after the broker and topic", async () => {
    const { dev } = await renderWrite([createWriteTarget("t1", "plant/valve/set")]);
    const escape = channel.escapeInvalidName;
    await screen.findByPlaceholderText(
      `${escape(dev.name)}_${escape("plant/valve/set")}_cmd`,
    );
  });

  it("should disable a target through its checkbox", async () => {
    const { draft } = await renderWrite([createWriteTarget("t1", "plant/valve/set")]);
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
        createWriteTarget("t1", "plant/valve/set"),
        createTagTarget("s1", "setpoint", { device: "ovenA", channel: ch.key }),
      ]);
      await screen.findByRole("option", { name: /JSON.*plant\/valve\/set/ });
      expect(
        screen.getByRole("option", { name: /SpB.*plant\/line1\/ovenA\/setpoint/ }),
      ).toBeTruthy();
      await screen.findByText(ch.name);
    });

    it("should add a tag by name and select its Sparkplug B type", async () => {
      const { draft } = await renderWrite();
      await screen.findByText("No targets");
      const dialog = await openAddTargets();
      fireEvent.click(dialog.getByRole("button", { name: "Sparkplug B" }));
      fireEvent.click(await dialog.findByRole("button", { name: "Add a tag by name" }));
      await screen.findByText("Edge node");
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
    // Opens the dialog over a broker whose scan task answers the first browse.
    const openOnTopics = async (
      topics: object[],
      targets: MQTT.Task.WriteTarget[] = [],
      truncated = false,
    ) => {
      const { dev, draft } = await renderWrite(targets);
      const scanner = await openScanner(client, dev.rack);
      const dialog = await openAddTargets();
      await scanner.answer({ data: { topics, truncated } });
      return { dialog, draft, scanner };
    };

    const NODES = [{ group: "plant", edgeNode: "line1", devices: ["ovenA"] }];

    // Answers an empty topic browse, then the edge nodes and the tags of the first.
    const openOnTags = async (reply: { tags?: object[]; message?: string }) => {
      const { dialog, draft, scanner } = await openOnTopics([]);
      fireEvent.click(dialog.getByRole("button", { name: "Sparkplug B" }));
      await scanner.answer({ data: { nodes: NODES, tags: [] } });
      if (reply.message != null)
        await scanner.answer({ variant: "error", message: reply.message });
      else await scanner.answer({ data: { nodes: [], tags: reply.tags } });
      return { dialog, draft, scanner };
    };

    const byIdentity = (targets: MQTT.Task.WriteTarget[]) =>
      Object.fromEntries(targets.map((t) => [t.type === "plain" ? t.topic : t.tag, t]));

    it("should add a target at the first value of each topic under a picked level", async () => {
      const { dialog, draft, scanner } = await openOnTopics([
        {
          topic: "plant/valve",
          payload: '{"open": true}',
          fields: [{ pointer: "/open", type: "boolean", sample: "true" }],
          retained: false,
        },
        {
          topic: "plant/speed",
          payload: '{"unit": "rpm", "rpm": 1200}',
          fields: [
            { pointer: "/unit", type: "string", sample: '"rpm"' },
            { pointer: "/rpm", type: "number", sample: "1200" },
          ],
          retained: false,
        },
        { topic: "plant/mode", payload: "AUTO", fields: [], retained: false },
      ]);
      scanner.close();
      fireEvent.click(await dialog.findByRole("checkbox", { name: "Pick plant" }));
      fireEvent.click(dialog.getByRole("button", { name: "Add 3 targets" }));
      await waitFor(async () => {
        const targets = await retrieveTargets(draft.key);
        expect(targets).toHaveLength(3);
        expect(byIdentity(targets)).toMatchObject({
          "plant/valve": {
            channel: { pointer: "/open", jsonType: "boolean", dataType: "uint8" },
          },
          "plant/speed": {
            channel: { pointer: "/rpm", jsonType: "number", dataType: "float64" },
          },
          "plant/mode": {
            channel: { pointer: "", jsonType: "string", dataType: "string" },
          },
        });
      });
    });

    it("should replace the value picked from a topic with the next one", async () => {
      const { dialog, draft, scanner } = await openOnTopics([
        {
          topic: "plant/oven",
          payload: '{"setpoint": 21.5, "mode": "AUTO"}',
          fields: [
            { pointer: "/setpoint", type: "number", sample: "21.5" },
            { pointer: "/mode", type: "string", sample: '"AUTO"' },
          ],
          retained: false,
        },
      ]);
      scanner.close();
      const setpoint = await dialog.findByRole("checkbox", { name: "Record setpoint" });
      expect(dialog.queryByRole("button", { name: "Select all" })).toBeNull();
      fireEvent.click(setpoint);
      fireEvent.click(dialog.getByRole("checkbox", { name: "Record mode" }));
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
      await waitFor(async () =>
        expect(await retrieveTargets(draft.key)).toMatchObject([
          {
            topic: "plant/oven",
            channel: { pointer: "/mode", jsonType: "string", dataType: "string" },
          },
        ]),
      );
    });

    it("should offer no value of a topic that the task writes to", async () => {
      const { dialog, scanner } = await openOnTopics(
        [
          {
            topic: "plant/oven",
            payload: '{"setpoint": 21.5}',
            fields: [{ pointer: "/setpoint", type: "number", sample: "21.5" }],
            retained: false,
          },
          {
            topic: "plant/pump",
            payload: '{"speed": 3}',
            fields: [{ pointer: "/speed", type: "number", sample: "3" }],
            retained: false,
          },
        ],
        [createWriteTarget("t1", "plant/oven")],
      );
      scanner.close();
      const pump = await dialog.findByRole<HTMLInputElement>("checkbox", {
        name: "Pick pump",
      });
      expect(pump.disabled).toBe(false);
      const oven = dialog.getByRole<HTMLInputElement>("checkbox", {
        name: "Pick oven",
      });
      expect(oven.disabled).toBe(true);
      const setpoint = dialog.getByRole<HTMLInputElement>("checkbox", {
        name: "Record setpoint",
      });
      expect(setpoint.disabled).toBe(true);
    });

    it("should mark a retained topic and say when the list is full", async () => {
      const { dialog, scanner } = await openOnTopics(
        [
          {
            topic: "plant/temperature",
            payload: '{"value": 21.5}',
            fields: [{ pointer: "/value", type: "number", sample: "21.5" }],
            retained: true,
          },
        ],
        [],
        true,
      );
      scanner.close();
      await dialog.findByText("Retained");
      expect(dialog.getByText(/list is full/)).toBeTruthy();
    });

    it("should show a failed listen in place of each list, with no notification", async () => {
      let statuses: Status.NotificationSpec[] = [];
      await renderWrite([], { onStatuses: (s) => (statuses = s) });
      const dialog = await openAddTargets();
      await dialog.findByText(/not found/i);
      fireEvent.click(dialog.getByRole("button", { name: "Sparkplug B" }));
      await waitFor(() => expect(dialog.getAllByText(/not found/i)).toHaveLength(2));
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
      await scanner.answer({ data: { topics: [], truncated: false } });
      fireEvent.click(again.getByRole("button", { name: "Sparkplug B" }));
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
      dev.properties = {
        ...MQTT.Device.ZERO_PROPERTIES,
        write: { "spBv1.0/plant/line1//setpoint": storedCh.key },
      };
      await client.devices.create(dev, MQTT.Device.SCHEMAS);
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
      const { container } = await renderWrite([createWriteTarget("t1", "plant/#")]);
      await screen.findByText("JSON type");
      await clickDeploy(container);
      await screen.findByText("Topic must not hold the wildcards + or #");
    });

    it("should create command channels, virtual when variable, and persist them to the device", async () => {
      const virtualName = uniqueName("mqtt_cmd");
      const { container, dev, draft } = await renderWrite([
        createWriteTarget("t1", "plant/valve/set", { dataType: "uint8" }),
        createWriteTarget("t2", "plant/mode/set", {
          dataType: "string",
          name: virtualName,
        }),
      ]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.WRITE_SCHEMAS,
      );

      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      const cmdKey = updated.properties.write["plant/valve/set"];
      const cmdCh = await client.channels.retrieve(cmdKey);
      expect(cmdCh.name).toBe(`${dev.name}_plant_valve_set_cmd`);
      expect(cmdCh.dataType.toString()).toBe("uint8");
      const indexCh = await client.channels.retrieve(cmdCh.index);
      expect(indexCh.name).toBe(`${dev.name}_plant_valve_set_cmd_time`);

      const virtualKey = updated.properties.write["plant/mode/set"];
      const virtualCh = await client.channels.retrieve(virtualKey);
      expect(virtualCh.virtual).toBe(true);
      expect(virtualCh.name).toBe(virtualName);

      const channels = created.config.targets.map((t) =>
        t.type === "plain" ? t.channel.channel : 0,
      );
      expect(channels).toEqual([cmdKey, virtualKey]);
    });

    it("should adopt existing channels from the config and the device instead of creating new ones", async () => {
      const dev = await createBroker(client);
      const configuredCh = await client.channels.create({
        name: uniqueName("mqtt_cmd"),
        dataType: "string",
        virtual: true,
      });
      const storedCh = await client.channels.create({
        name: uniqueName("mqtt_cmd"),
        dataType: "string",
        virtual: true,
      });
      dev.properties = {
        ...MQTT.Device.ZERO_PROPERTIES,
        write: { "plant/stored": storedCh.key },
      };
      await client.devices.create(dev, MQTT.Device.SCHEMAS);
      const config = createWriteConfig(dev.key, [
        createWriteTarget("t1", "plant/cmd", { channel: configuredCh.key }),
        createWriteTarget("t2", "plant/stored"),
      ]);
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
      const channels = created.config.targets.map((t) =>
        t.type === "plain" ? t.channel.channel : 0,
      );
      expect(channels).toEqual([configuredCh.key, storedCh.key]);
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      expect(updated.properties.write["plant/cmd"]).toBe(configuredCh.key);
      expect(updated.properties.write["plant/stored"]).toBe(storedCh.key);
    });
  });
});
