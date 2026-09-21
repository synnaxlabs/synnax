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
  overrides: Partial<MQTT.Task.ReadEntry> = {},
): MQTT.Task.ReadEntry => ({
  ...mqtt.plainReadEntryZ.parse({ type: "plain" }),
  key,
  topic,
  ...overrides,
});

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

  it("should keep a Sparkplug B entry out of the list", async () => {
    const dev = await createBroker(client);
    const draft = await createDraft(client, {
      ...createReadConfig(dev.key, []),
      entries: [
        createReadEntry("e1", "plant/oven"),
        mqtt.sparkplugReadEntryZ.parse({ type: "sparkplug", key: "s1", tag: "flow" }),
      ],
    });
    await renderTaskFormTab(MQTT.Task.Read, { client, taskKey: draft.key });
    await screen.findByText(/plant\/oven/);
    expect(screen.queryByText("No topic")).toBeNull();
  });

  describe("deploying against a live Core", () => {
    it("should put the deploy errors on the fields they belong to", async () => {
      const { container } = await renderRead([createReadEntry("e1", "plant/+/oven")]);
      await screen.findByText(/plant\/\+\/oven/);
      await clickDeploy(container);
      await screen.findByText("Topic must not hold the wildcards + or #");
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
