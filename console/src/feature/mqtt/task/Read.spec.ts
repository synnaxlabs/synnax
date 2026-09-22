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
  deployAndAwaitTask,
  renderTaskFormTab,
  selectFromDropdown,
} from "@/platform/task/testutil";
import { getHeaderIconButton, getSwitchInput, uniqueName } from "@/testutil";

const client = createTestClient();

const createReadField = (
  key: string,
  pointer: string,
  overrides: Partial<MQTT.Task.ReadField> = {},
): MQTT.Task.ReadField => ({
  ...mqtt.readFieldZ.parse({}),
  key,
  pointer,
  ...overrides,
});

const createReadEntry = (
  key: string,
  topic: string,
  overrides: Partial<MQTT.Task.PlainReadEntry> = {},
): MQTT.Task.PlainReadEntry => ({
  ...mqtt.plainReadEntryZ.parse({ type: "plain" }),
  key,
  topic,
  ...overrides,
});

const createTagEntry = (
  key: string,
  tag: string,
  overrides: Partial<MQTT.Task.SparkplugReadEntry> = {},
): MQTT.Task.SparkplugReadEntry => ({
  ...mqtt.sparkplugReadEntryZ.parse({ type: "sparkplug" }),
  key,
  group: "plant",
  edgeNode: "line1",
  tag,
  ...overrides,
});

const retrieveEntries = async (key: task.Key) =>
  (await client.tasks.retrieve({ key, schemas: MQTT.Task.READ_SCHEMAS })).config
    .entries;

const createReadConfig = (
  device: string,
  entries: MQTT.Task.ReadEntry[],
): MQTT.Task.ReadPayload["config"] => ({
  ...MQTT.Task.READ_SCHEMAS.config.parse({}),
  device,
  entries,
});

// Drafts carry no key; the created task mints its own.
const ZERO_DRAFT: task.New<MQTT.Task.ReadSchemas> = {
  name: "MQTT read task",
  type: MQTT.Task.READ_TYPE,
  config: MQTT.Task.READ_SCHEMAS.config.parse({}),
};

const createDraft = async (client: Synnax, config: MQTT.Task.ReadPayload["config"]) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, MQTT.Task.READ_SCHEMAS);

// The form body renders against a configured broker only.
const renderRead = async (entries: MQTT.Task.ReadEntry[] = []) => {
  const dev = await createBroker(client);
  const draft = await createDraft(client, createReadConfig(dev.key, entries));
  const result = await renderTaskFormTab(MQTT.Task.Read, {
    client,
    taskKey: draft.key,
  });
  return { ...result, dev, draft };
};

const addEntry = async (): Promise<void> => {
  fireEvent.click(await screen.findByText("Add entry"));
  await screen.findByText("Timestamp source");
};

describe("MQTT Read form", () => {
  it("should ask for a broker before it shows the form body", async () => {
    await renderTaskFormTab(MQTT.Task.Read, { task: ZERO_DRAFT });
    await screen.findByText("No device selected");
    expect(screen.queryByText("Entries")).toBeNull();
  });

  it("should show the empty state and add + select an entry", async () => {
    await renderRead();
    await screen.findByText("Select an entry to configure");
    await screen.findByText("No entries");
    await addEntry();
    expect(screen.getByPlaceholderText("plant/line1/temperature")).toBeTruthy();
    expect(screen.getByText("At most once (0)")).toBeTruthy();
    expect(getSwitchInput("Ignore retained messages").checked).toBe(false);
    expect(screen.getByText("No fields")).toBeTruthy();
    expect(screen.queryByText("Select an entry to configure")).toBeNull();
  });

  it("should show the browser beside the entries", async () => {
    await renderRead();
    await screen.findByText("Entries");
    expect(screen.getByRole("button", { name: "Browse" })).toBeTruthy();
  });

  it("should select a quality of service", async () => {
    await renderRead();
    await addEntry();
    await selectFromDropdown("At most once (0)", "Exactly once (2)");
    await screen.findByText("Exactly once (2)");
  });

  it("should add a timestamp field on payload timing that stays out of the fields list", async () => {
    await renderRead();
    await addEntry();
    fireEvent.click(screen.getByRole("button", { name: "Payload" }));
    await screen.findByText("Timestamp pointer");
    expect(screen.getByText("Format")).toBeTruthy();
    expect(screen.getByText("No fields")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Arrival time" }));
    await waitFor(() => expect(screen.queryByText("Timestamp pointer")).toBeNull());
  });

  it("should add a field, select it, and show the enum mapping editor", async () => {
    await renderRead();
    await addEntry();
    fireEvent.click(screen.getByText("Add field"));
    await screen.findByPlaceholderText("/temperature");
    await screen.findByText("Enum mapping");
  });

  it("should copy the previous field's settings when adding another field", async () => {
    await renderRead();
    await addEntry();
    fireEvent.click(screen.getByText("Add field"));
    const pointer = await screen.findByPlaceholderText("/temperature");
    fireEvent.change(pointer, { target: { value: "/a" } });
    fireEvent.blur(pointer);
    fireEvent.click(getHeaderIconButton("Fields"));
    await waitFor(() => expect(screen.getAllByDisplayValue("/a")).toHaveLength(2));
  });

  it("should duplicate, disable, and remove entries through the context menu", async () => {
    const { draft } = await renderRead([createReadEntry("e1", "plant/oven")]);
    const item = await screen.findByText(/plant\/oven/);
    fireEvent.contextMenu(item);
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() => expect(screen.getAllByText(/plant\/oven/)).toHaveLength(2));
    fireEvent.contextMenu(screen.getAllByText(/plant\/oven/)[0]);
    expect(screen.queryByText("Enable")).toBeNull();
    fireEvent.click(await screen.findByText("Disable"));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: MQTT.Task.READ_SCHEMAS,
      });
      expect(saved.config.entries.map(({ disabled }) => disabled)).toEqual([
        true,
        false,
      ]);
    });
    fireEvent.contextMenu(screen.getAllByText(/plant\/oven/)[0]);
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(() => expect(screen.getAllByText(/plant\/oven/)).toHaveLength(1));
  });

  describe("Sparkplug B entries", () => {
    it("should list a tag with its edge node beside a plain entry", async () => {
      await renderRead([
        createReadEntry("e1", "plant/oven"),
        createTagEntry("s1", "flow", { device: "pumpA" }),
        createTagEntry("s2", "Node Control/Rebirth"),
      ]);
      await screen.findByText(/plant\/oven/);
      expect(screen.getByText(/flow/)).toBeTruthy();
      expect(screen.getByText("plant/line1/pumpA")).toBeTruthy();
      expect(screen.getByText("plant/line1")).toBeTruthy();
    });

    it("should add a tag from the header and show its fields", async () => {
      const { draft } = await renderRead();
      await screen.findByText("No entries");
      fireEvent.click(getHeaderIconButton("Entries", "variable"));
      await screen.findByText("Edge node");
      expect(screen.getByText("Group")).toBeTruthy();
      expect(screen.getByPlaceholderText("Optional")).toBeTruthy();
      expect(screen.getByText("No tag")).toBeTruthy();
      expect(screen.getByText("Data type")).toBeTruthy();
      expect(screen.queryByText("Timestamp source")).toBeNull();
      const tag = screen.getByPlaceholderText("oven/temperature");
      fireEvent.change(tag, { target: { value: "zone 1/temperature" } });
      await waitFor(async () =>
        expect(await retrieveEntries(draft.key)).toMatchObject([
          { type: "sparkplug", tag: "zone 1/temperature", dataType: "float64" },
        ]),
      );
    });

    it("should hide the data type of a tag that has a channel", async () => {
      const ch = await client.channels.create({
        name: uniqueName("mqtt_tag"),
        dataType: "string",
        virtual: true,
      });
      await renderRead([createTagEntry("s1", "mode", { channel: ch.key })]);
      await screen.findByText(ch.name);
      expect(screen.queryByText("Data type")).toBeNull();
    });

    it("should duplicate a tag with no channels", async () => {
      const { draft } = await renderRead([
        createTagEntry("s1", "flow", { channel: 12, index: 11 }),
      ]);
      fireEvent.contextMenu(await screen.findByText(/flow/));
      fireEvent.click(await screen.findByText("Duplicate"));
      await waitFor(async () => {
        const entries = await retrieveEntries(draft.key);
        expect(entries).toMatchObject([
          { key: "s1", channel: 12, index: 11 },
          { tag: "flow", channel: 0, index: 0 },
        ]);
        expect(entries[1].key).not.toBe("s1");
      });
    });
  });

  describe("deploying against a live Core", () => {
    it("should put the deploy errors on the fields they belong to", async () => {
      const { container } = await renderRead([createReadEntry("e1", "plant/+/oven")]);
      await screen.findByText(/plant\/\+\/oven/);
      await clickDeploy(container);
      await screen.findByText("Topic must not hold the wildcards + or #");
    });

    it("should put the deploy errors of a tag on its fields", async () => {
      const { container } = await renderRead([
        createTagEntry("s1", "", { group: "plant/a", edgeNode: "" }),
      ]);
      await screen.findByText("Edge node");
      await clickDeploy(container);
      await screen.findByText("Group must not hold /, +, or #");
      expect(screen.getByText("Edge node is required")).toBeTruthy();
      expect(screen.getByText("Tag is required")).toBeTruthy();
    });

    it("should create an index and a data channel for each tag", async () => {
      const { container, dev, draft } = await renderRead([
        createTagEntry("s1", "zone 1/temperature", { device: "ovenA" }),
        createTagEntry("s2", "flow", { dataType: "float32" }),
      ]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.READ_SCHEMAS,
      );
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      const [deviceTag, nodeTag] = created.config.entries;
      if (deviceTag.type !== "sparkplug" || nodeTag.type !== "sparkplug")
        throw new Error("expected Sparkplug B entries");
      expect(nodeTag.index).not.toBe(deviceTag.index);
      expect(updated.properties.read).toEqual({
        "spBv1.0/plant/line1/ovenA/zone 1/temperature": {
          index: deviceTag.index,
          channels: { "": deviceTag.channel },
        },
        "spBv1.0/plant/line1//flow": {
          index: nodeTag.index,
          channels: { "": nodeTag.channel },
        },
      });
      const prefix = `${dev.name}_plant_line1_ovenA_zone_1_temperature`;
      const dataCh = await client.channels.retrieve(deviceTag.channel);
      expect(dataCh.name).toBe(prefix);
      expect(dataCh.index).toBe(deviceTag.index);
      const indexCh = await client.channels.retrieve(deviceTag.index);
      expect(indexCh.name).toBe(`${prefix}_time`);
      const flowCh = await client.channels.retrieve(nodeTag.channel);
      expect(flowCh.name).toBe(`${dev.name}_plant_line1_flow`);
      expect(flowCh.dataType.toString()).toBe("float32");
    });

    it("should give the channels the name of the entry", async () => {
      const name = uniqueName("oven_temperature");
      const { container, draft } = await renderRead([
        createTagEntry("s1", "temperature", { name }),
      ]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.READ_SCHEMAS,
      );
      const [entry] = created.config.entries;
      if (entry.type !== "sparkplug") throw new Error("expected a Sparkplug B entry");
      expect((await client.channels.retrieve(entry.channel)).name).toBe(name);
      expect((await client.channels.retrieve(entry.index)).name).toBe(`${name}_time`);
    });

    it("should create a virtual channel with no index for a string tag", async () => {
      const { container, dev, draft } = await renderRead([
        createTagEntry("s1", "mode", { dataType: "string", index: 7 }),
      ]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.READ_SCHEMAS,
      );
      const [entry] = created.config.entries;
      if (entry.type !== "sparkplug") throw new Error("expected a Sparkplug B entry");
      expect(entry.index).toBe(0);
      expect((await client.channels.retrieve(entry.channel)).virtual).toBe(true);
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      expect(updated.properties.read["spBv1.0/plant/line1//mode"].index).toBe(0);
    });

    it("should reuse the channels of a tag that the device stores", async () => {
      const { container, draft } = await renderRead([createTagEntry("s1", "flow")]);
      const first = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.READ_SCHEMAS,
      );
      const [stored] = first.config.entries;
      if (stored.type !== "sparkplug") throw new Error("expected a Sparkplug B entry");
      const config = createReadConfig(first.config.device, [
        createTagEntry("s2", "flow"),
      ]);
      const second = await createDraft(client, config);
      const rendered = await renderTaskFormTab(MQTT.Task.Read, {
        client,
        taskKey: second.key,
      });
      const created = await deployAndAwaitTask(
        client,
        rendered.container,
        second.key,
        MQTT.Task.READ_SCHEMAS,
      );
      expect(created.config.entries).toMatchObject([
        { key: "s2", channel: stored.channel, index: stored.index },
      ]);
    });

    it("should take the index of a tag from its live channel", async () => {
      const idxCh = await client.channels.create({
        name: uniqueName("mqtt_idx"),
        dataType: "timestamp",
        isIndex: true,
      });
      const dataCh = await client.channels.create({
        name: uniqueName("mqtt_data"),
        dataType: "float64",
        index: idxCh.key,
      });
      const { container, dev, draft } = await renderRead([
        createTagEntry("s1", "flow", { channel: dataCh.key }),
      ]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.READ_SCHEMAS,
      );
      expect(created.config.entries).toMatchObject([
        { channel: dataCh.key, index: idxCh.key },
      ]);
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      expect(updated.properties.read).toEqual({});
    });

    it("should create index and data channels and persist them to the device", async () => {
      const { container, dev, draft } = await renderRead([
        createReadEntry("e1", "plant/oven", {
          index: "tf",
          fields: [
            createReadField("f1", "/temperature"),
            createReadField("tf", "/ts", {
              dataType: "timestamp",
              timeFormat: "unix_sec",
            }),
          ],
        }),
      ]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.READ_SCHEMAS,
      );

      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      const topicProps = updated.properties.read["plant/oven"];
      expect(topicProps.index).toBeGreaterThan(0);
      const indexCh = await client.channels.retrieve(topicProps.index);
      expect(indexCh.name).toBe(`${dev.name}_plant_oven_time`);

      const dataKey = topicProps.channels["/temperature"];
      const dataCh = await client.channels.retrieve(dataKey);
      expect(dataCh.index).toBe(topicProps.index);
      expect(dataCh.name).toBe(`${dev.name}_plant_oven_temperature`);

      const [entry] = created.config.entries;
      if (entry.type !== "plain") throw new Error("expected a plain entry");
      expect(entry.fields.find((f) => f.key === "f1")?.channel).toBe(dataKey);
      expect(entry.fields.find((f) => f.key === "tf")?.channel).toBe(topicProps.index);
    });

    it("should create virtual channels with no index for string fields", async () => {
      const { container, dev, draft } = await renderRead([
        createReadEntry("e1", "plant/mode", {
          fields: [createReadField("f1", "", { dataType: "string" })],
        }),
      ]);
      await deployAndAwaitTask(client, container, draft.key, MQTT.Task.READ_SCHEMAS);
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      const topicProps = updated.properties.read["plant/mode"];
      expect(topicProps.index).toBe(0);
      const virtualCh = await client.channels.retrieve(topicProps.channels[""]);
      expect(virtualCh.virtual).toBe(true);
      expect(virtualCh.name).toBe(`${dev.name}_plant_mode`);
    });

    it("should keep the case of a topic and a pointer in the device properties", async () => {
      const { container, dev, draft } = await renderRead([
        createReadEntry("e1", "plant/line_a/ovenTemp", {
          fields: [createReadField("f1", "/outlet_temp/degC")],
        }),
      ]);
      await deployAndAwaitTask(client, container, draft.key, MQTT.Task.READ_SCHEMAS);
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      const topicProps = updated.properties.read["plant/line_a/ovenTemp"];
      expect(topicProps.channels["/outlet_temp/degC"]).toBeGreaterThan(0);
    });

    it("should skip a disabled entry", async () => {
      const { container, dev, draft } = await renderRead([
        createReadEntry("e1", "plant/oven", {
          fields: [createReadField("f1", "/temperature")],
        }),
        createReadEntry("e2", "plant/spare", {
          disabled: true,
          fields: [createReadField("f2", "/temperature")],
        }),
      ]);
      await deployAndAwaitTask(client, container, draft.key, MQTT.Task.READ_SCHEMAS);
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      expect(Object.keys(updated.properties.read)).toEqual(["plant/oven"]);
    });

    it("should reuse channels already stored on the device instead of creating new ones", async () => {
      const dev = await createBroker(client);
      const idxCh = await client.channels.create({
        name: uniqueName("mqtt_idx"),
        dataType: "timestamp",
        isIndex: true,
      });
      const dataCh = await client.channels.create({
        name: uniqueName("mqtt_data"),
        dataType: "float64",
        index: idxCh.key,
      });
      dev.properties = {
        ...MQTT.Device.ZERO_PROPERTIES,
        read: {
          "plant/oven": { index: idxCh.key, channels: { "/temperature": dataCh.key } },
        },
      };
      await client.devices.create(dev, MQTT.Device.SCHEMAS);
      const config = createReadConfig(dev.key, [
        createReadEntry("e1", "plant/oven", {
          fields: [createReadField("f1", "/temperature")],
        }),
      ]);
      const draft = await createDraft(client, config);
      const { container } = await renderTaskFormTab(MQTT.Task.Read, {
        client,
        taskKey: draft.key,
      });
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.READ_SCHEMAS,
      );
      const [entry] = created.config.entries;
      if (entry.type !== "plain") throw new Error("expected a plain entry");
      expect(entry.fields[0].channel).toBe(dataCh.key);
    });

    it("should recover the index from an existing data channel when the stored index is gone", async () => {
      const dev = await createBroker(client);
      const idxCh = await client.channels.create({
        name: uniqueName("mqtt_idx"),
        dataType: "timestamp",
        isIndex: true,
      });
      const dataCh = await client.channels.create({
        name: uniqueName("mqtt_data"),
        dataType: "float64",
        index: idxCh.key,
      });
      dev.properties = {
        ...MQTT.Device.ZERO_PROPERTIES,
        read: { "plant/oven": { index: 0, channels: { "/existing": dataCh.key } } },
      };
      await client.devices.create(dev, MQTT.Device.SCHEMAS);
      const config = createReadConfig(dev.key, [
        createReadEntry("e1", "plant/oven", {
          fields: [createReadField("f1", "/temperature")],
        }),
      ]);
      const draft = await createDraft(client, config);
      const { container } = await renderTaskFormTab(MQTT.Task.Read, {
        client,
        taskKey: draft.key,
      });
      await deployAndAwaitTask(client, container, draft.key, MQTT.Task.READ_SCHEMAS);
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      expect(updated.properties.read["plant/oven"].index).toBe(idxCh.key);
    });
  });
});
