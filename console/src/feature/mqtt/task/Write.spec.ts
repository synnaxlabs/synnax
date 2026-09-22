// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mqtt, type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker } from "@/feature/mqtt/testutil";
import {
  clickDeploy,
  createChannelReadOnlyClient,
  deployAndAwaitTask,
  findDialogTriggerByText,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  selectFromDropdown,
} from "@/platform/task/testutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  getHeaderIconButton,
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

const addTarget = async (): Promise<void> => {
  fireEvent.click(await screen.findByText("Add target"));
  await screen.findByText("JSON pointer");
};

describe("MQTT Write form", () => {
  it("should show the empty state and add + select a target", async () => {
    await renderWrite();
    await screen.findByText("Select a target to configure");
    await addTarget();
    expect(screen.getByPlaceholderText("plant/line1/valve/set")).toBeTruthy();
    expect(screen.getByText("At least once (1)")).toBeTruthy();
    expect(getSwitchInput("Retain message").checked).toBe(false);
    expect(screen.getByText("Synnax data type")).toBeTruthy();
    expect(screen.getByText("No additional fields")).toBeTruthy();
  });

  it("should show the enum mapping editor when the channel JSON type is string", async () => {
    await renderWrite();
    await addTarget();
    expect(screen.queryByText("Enum mappings")).toBeNull();
    await selectFromDropdown("Number", "String");
    await screen.findByText("Enum mappings");
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
    fireEvent.click(getHeaderIconButton("Additional fields", "add"));
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
    fireEvent.click(getHeaderIconButton("Additional fields", "time"));
    await screen.findByText("generated");
    await selectFromDropdown("UUID", "Timestamp (s)");
    await findDialogTriggerByText("Timestamp (s)");
  });

  it("should remove additional fields through a context menu without a duplicate option", async () => {
    await renderWrite();
    await addTarget();
    fireEvent.click(getHeaderIconButton("Additional fields", "add"));
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
    await screen.findByText("JSON pointer");
    await screen.findByText(ch.name);
  };

  it("should show the time format field for a timestamp command channel", async () => {
    await renderWithCommandChannel("timestamp");
    await screen.findByText("Time format");
  });

  it("should hide the time format field for a numeric command channel", async () => {
    await renderWithCommandChannel("float64");
    expect(screen.queryByText("Time format")).toBeNull();
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
      await screen.findByText(/plant\/valve\/set/);
      expect(screen.getByText(/setpoint/)).toBeTruthy();
      expect(screen.getByText("plant/line1/ovenA")).toBeTruthy();
      await screen.findByText(ch.name);
    });

    it("should add a tag from the header and select its Sparkplug B type", async () => {
      const { draft } = await renderWrite();
      await screen.findByText("No targets");
      fireEvent.click(getHeaderIconButton("Targets", "variable"));
      await screen.findByText("Edge node");
      expect(screen.getByPlaceholderText("Optional")).toBeTruthy();
      expect(screen.queryByText("JSON pointer")).toBeNull();
      await selectFromDropdown("Double", "DateTime");
      await waitFor(async () =>
        expect(await retrieveTargets(draft.key)).toMatchObject([
          { type: "sparkplug", sparkplugType: "date_time", channel: 0 },
        ]),
      );
    });

    it("should duplicate a tag with no command channel", async () => {
      const { draft } = await renderWrite([
        createTagTarget("s1", "setpoint", { channel: 12, sparkplugType: "float" }),
      ]);
      fireEvent.contextMenu(await screen.findByText(/setpoint/));
      fireEvent.click(await screen.findByText("Duplicate"));
      await waitFor(async () => {
        const targets = await retrieveTargets(draft.key);
        expect(targets).toMatchObject([
          { key: "s1", channel: 12 },
          { tag: "setpoint", sparkplugType: "float", channel: 0 },
        ]);
        expect(targets[1].key).not.toBe("s1");
      });
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
          sparkplugType: "float",
        }),
        createTagTarget("s2", "running", { sparkplugType: "boolean" }),
        createTagTarget("s3", "mode", { sparkplugType: "string" }),
        createTagTarget("s4", "started", { sparkplugType: "date_time" }),
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
      await screen.findByText("JSON pointer");
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
