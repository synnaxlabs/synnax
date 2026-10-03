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
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker, openScanner } from "@/feature/mqtt/testutil";
import {
  clickDeploy,
  deployAndAwaitTask,
  renderTaskFormTab,
  selectFromDropdown,
} from "@/platform/task/testutil";
import { getSwitchInput, uniqueName } from "@/testutil";

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
  fireEvent.click(await screen.findByRole("button", { name: "Add topic" }));
  await screen.findByText("Subscription");
};

const addField = async (): Promise<void> => {
  fireEvent.click(screen.getAllByRole("button", { name: "Add field" })[0]);
  await screen.findByRole("treeitem", { name: /Whole payload/ });
};

describe("MQTT Read form", () => {
  it("should ask for a broker before it shows the form body", async () => {
    await renderTaskFormTab(MQTT.Task.Read, { task: ZERO_DRAFT });
    await screen.findByText("No device selected");
    expect(screen.queryByText("Entries")).toBeNull();
  });

  it("should show the empty state and add + select an entry", async () => {
    await renderRead();
    await screen.findByText("Select an entry or field to configure");
    await screen.findByText("No entries");
    await addEntry();
    expect(screen.getByPlaceholderText("plant/line1/temperature")).toBeTruthy();
    expect(screen.getByText("At most once (0)")).toBeTruthy();
    expect(getSwitchInput("Ignore retained").checked).toBe(false);
    expect(screen.getByRole("button", { name: "Arrival time" })).toBeTruthy();
    expect(screen.getAllByText("New topic")).toHaveLength(2);
    expect(screen.queryByText("Select an entry or field to configure")).toBeNull();
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

  it("should add a timestamp field on payload timing that stays out of the tree", async () => {
    await renderRead();
    await addEntry();
    fireEvent.click(screen.getByRole("button", { name: "Payload value" }));
    await screen.findByPlaceholderText("/timestamp");
    expect(screen.getByText("Format")).toBeTruthy();
    expect(screen.getAllByRole("treeitem")).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: "Arrival time" }));
    await waitFor(() => expect(screen.queryByText("Format")).toBeNull());
  });

  it("should add a field beneath its entry and show its details", async () => {
    await renderRead();
    await addEntry();
    await addField();
    expect(screen.getByPlaceholderText("/temperature")).toBeTruthy();
    expect(screen.getByText("Data type")).toBeTruthy();
    expect(screen.getByText("Enum mapping")).toBeTruthy();
  });

  it("should name a new field's channel after the broker, topic, and pointer", async () => {
    const { dev } = await renderRead([createReadEntry("e1", "plant/oven")]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /plant\/oven/ }));
    fireEvent.click(await screen.findByText("Add field"));
    const pointer = await screen.findByPlaceholderText("/temperature");
    fireEvent.change(pointer, { target: { value: "/temp" } });
    fireEvent.blur(pointer);
    const escape = channel.escapeInvalidName;
    await screen.findByPlaceholderText(
      `${escape(dev.name)}_${escape("plant/oven")}${escape("/temp")}`,
    );
  });

  it("should copy the previous field's settings when adding another field", async () => {
    await renderRead();
    await addEntry();
    await addField();
    const pointer = screen.getByPlaceholderText("/temperature");
    fireEvent.change(pointer, { target: { value: "/a" } });
    fireEvent.blur(pointer);
    await screen.findByRole("treeitem", { name: /\/a/ });
    fireEvent.click(screen.getAllByRole("button", { name: "Add field" })[0]);
    await waitFor(() =>
      expect(within(screen.getByRole("tree")).getAllByText("/a")).toHaveLength(2),
    );
  });

  it("should disable and enable a field through its checkbox and context menu", async () => {
    const { draft } = await renderRead([
      createReadEntry("e1", "plant/oven", { fields: [createReadField("f1", "/t")] }),
    ]);
    const field = await screen.findByRole("treeitem", { name: /\/t/ });
    fireEvent.click(within(field).getByRole("checkbox", { name: "Enabled" }));
    await waitFor(async () =>
      expect(await retrieveEntries(draft.key)).toMatchObject([
        { fields: [{ key: "f1", disabled: true }] },
      ]),
    );
    fireEvent.contextMenu(field);
    expect(screen.queryByText("Disable")).toBeNull();
    fireEvent.click(await screen.findByText("Enable"));
    await waitFor(async () =>
      expect(await retrieveEntries(draft.key)).toMatchObject([
        { fields: [{ key: "f1", disabled: false }] },
      ]),
    );
  });

  it("should duplicate, disable, and remove entries through the context menu", async () => {
    const { draft } = await renderRead([createReadEntry("e1", "plant/oven")]);
    const entryItems = () => screen.getAllByRole("treeitem", { name: /plant\/oven/ });
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /plant\/oven/ }));
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() => expect(entryItems()).toHaveLength(2));
    fireEvent.contextMenu(entryItems()[0]);
    expect(screen.queryByText("Enable")).toBeNull();
    fireEvent.click(await screen.findByText("Disable"));
    await waitFor(async () =>
      expect(
        (await retrieveEntries(draft.key)).map(({ disabled }) => disabled),
      ).toEqual([true, false]),
    );
    fireEvent.contextMenu(entryItems()[0]);
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(() => expect(entryItems()).toHaveLength(1));
  });

  it("should offer no add field action on a Sparkplug B entry", async () => {
    await renderRead([createTagEntry("s1", "flow")]);
    const item = await screen.findByRole("treeitem", { name: /flow/ });
    expect(within(item).queryByRole("button", { name: "Add field" })).toBeNull();
    fireEvent.contextMenu(item);
    await screen.findByText("Duplicate");
    expect(screen.queryByText("Add field")).toBeNull();
  });

  describe("dropping browsed items on the entries", () => {
    const TOPIC = "plant/line1/temperature";

    const scoped = (container: ParentNode, selector: string) => {
      const el = container.querySelector<HTMLElement>(selector);
      if (el == null) throw new Error(`${selector} is not rendered`);
      return within(el);
    };

    const renderPanes = async () => {
      const { container, dev, draft } = await renderRead();
      // The browser mounts once the form leaves its loading preview.
      await screen.findByRole("button", { name: "Browse" });
      return {
        dev,
        draft,
        browser: scoped(container, ".console-mqtt-browser"),
        entries: scoped(container, ".console-panes__list"),
      };
    };

    it("should build an entry from a dropped topic once", async () => {
      const { dev, draft, browser, entries } = await renderPanes();
      const scanner = await openScanner(client, dev.rack);
      fireEvent.click(browser.getByRole("button", { name: "Browse" }));
      const topics = [
        {
          topic: TOPIC,
          payload: '{"temperature": 21.5, "label": "ok"}',
          retained: true,
        },
      ];
      await scanner.answer({ data: { topics, truncated: false } });
      scanner.close();
      fireEvent.dragStart(await browser.findByText(TOPIC));
      fireEvent.drop(entries.getByText("No entries"));
      await screen.findByText("Subscription");
      await waitFor(async () =>
        expect(await retrieveEntries(draft.key)).toMatchObject([
          {
            type: "plain",
            topic: TOPIC,
            fields: [
              { pointer: "/temperature", dataType: "float64", disabled: false },
              { pointer: "/label", dataType: "string", disabled: true },
            ],
          },
        ]),
      );
      await entries.findByRole("treeitem", { name: /\/label/ });
      fireEvent.dragStart(browser.getByText(TOPIC));
      fireEvent.drop(entries.getByRole("treeitem", { name: /^\/temperature/ }));
      await waitFor(async () =>
        expect(await retrieveEntries(draft.key)).toHaveLength(1),
      );
    });

    it("should build an entry from a dropped Sparkplug B tag once", async () => {
      const { dev, draft, browser, entries } = await renderPanes();
      const scanner = await openScanner(client, dev.rack);
      fireEvent.click(browser.getByRole("button", { name: "Sparkplug B" }));
      fireEvent.click(await browser.findByRole("button", { name: "Browse" }));
      await scanner.answer({
        data: { nodes: [{ group: "plant", edgeNode: "line1", devices: [] }], tags: [] },
      });
      fireEvent.click(await browser.findByText("plant/line1"));
      const tag = {
        device: "",
        name: "flow",
        dataType: "float",
        value: "1",
        supported: true,
      };
      await scanner.answer({ data: { nodes: [], tags: [tag] } });
      scanner.close();
      fireEvent.dragStart(await browser.findByText("flow"));
      fireEvent.drop(entries.getByText("No entries"));
      await screen.findByText("Edge node");
      await waitFor(async () =>
        expect(await retrieveEntries(draft.key)).toMatchObject([
          {
            type: "sparkplug",
            group: "plant",
            edgeNode: "line1",
            tag: "flow",
            dataType: "float32",
          },
        ]),
      );
      fireEvent.dragStart(browser.getByText("flow"));
      fireEvent.drop(entries.getByRole("treeitem", { name: /plant\/line1\/flow/ }));
      await waitFor(async () =>
        expect(await retrieveEntries(draft.key)).toHaveLength(1),
      );
    });
  });

  describe("Sparkplug B entries", () => {
    it("should list a tag by its full path beside a plain entry", async () => {
      await renderRead([
        createReadEntry("e1", "plant/oven"),
        createTagEntry("s1", "flow", { device: "pumpA" }),
        createTagEntry("s2", "Node Control/Rebirth"),
      ]);
      await screen.findByRole("treeitem", { name: /JSON.*plant\/oven/ });
      expect(
        screen.getByRole("treeitem", { name: /SpB.*plant\/line1\/pumpA\/flow/ }),
      ).toBeTruthy();
      expect(
        screen.getByRole("treeitem", {
          name: /SpB.*plant\/line1\/Node Control\/Rebirth/,
        }),
      ).toBeTruthy();
    });

    it("should add a tag from the list footer and show its fields", async () => {
      const { draft } = await renderRead();
      await screen.findByText("No entries");
      fireEvent.click(screen.getByRole("button", { name: "Add Sparkplug B tag" }));
      await screen.findByText("Edge node");
      expect(screen.getByText("Group")).toBeTruthy();
      expect(screen.getByPlaceholderText("Optional")).toBeTruthy();
      expect(screen.getAllByText("New tag")).toHaveLength(2);
      expect(screen.getByText("Data type")).toBeTruthy();
      expect(screen.queryByText("Timestamp")).toBeNull();
      const tag = screen.getByPlaceholderText("oven/temperature");
      fireEvent.change(tag, { target: { value: "zone 1/temperature" } });
      await waitFor(async () =>
        expect(await retrieveEntries(draft.key)).toMatchObject([
          { type: "sparkplug", tag: "zone 1/temperature", dataType: "float64" },
        ]),
      );
    });

    it("should lock the data type of a tag that has a channel", async () => {
      const ch = await client.channels.create({
        name: uniqueName("mqtt_tag"),
        dataType: "string",
        virtual: true,
      });
      await renderRead([createTagEntry("s1", "mode", { channel: ch.key })]);
      fireEvent.click(await screen.findByRole("treeitem", { name: /mode/ }));
      await screen.findByDisplayValue(ch.name);
      expect(screen.getByText("Set on the channel")).toBeTruthy();
    });

    it("should duplicate a tag with no channels", async () => {
      const { draft } = await renderRead([
        createTagEntry("s1", "flow", { channel: 12, index: 11 }),
      ]);
      fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /flow/ }));
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

  describe("binding to the channels the broker maps", () => {
    const createIndexed = async (prefix: string) => {
      const index = await client.channels.create({
        name: uniqueName(`${prefix}_time`),
        dataType: "timestamp",
        isIndex: true,
      });
      const data = await client.channels.create({
        name: uniqueName(prefix),
        dataType: "float64",
        index: index.key,
      });
      return { index, data };
    };

    const renderOnBroker = async (
      read: MQTT.Device.Properties["read"],
      entries: MQTT.Task.ReadEntry[],
    ) => {
      const dev = await createBroker(client, { properties: { read } });
      const draft = await createDraft(client, createReadConfig(dev.key, entries));
      await renderTaskFormTab(MQTT.Task.Read, { client, taskKey: draft.key });
      return draft;
    };

    it("should bind the fields of every plain entry to the channels of its topic", async () => {
      const [oven, line] = await Promise.all([
        createIndexed("mqtt_oven"),
        createIndexed("mqtt_line"),
      ]);
      const draft = await renderOnBroker(
        {
          "plant/oven": { index: oven.index.key, channels: { "/temp": oven.data.key } },
          "plant/line": { index: line.index.key, channels: { "/flow": line.data.key } },
        },
        [
          createReadEntry("e1", "plant/oven", {
            index: "t1",
            fields: [
              createReadField("f1", "/temp"),
              createReadField("t1", "/ts", { timeFormat: "unix_sec" }),
            ],
          }),
          createReadEntry("e2", "plant/line", {
            fields: [createReadField("f2", "/flow")],
          }),
        ],
      );
      await waitFor(async () =>
        expect(await retrieveEntries(draft.key)).toMatchObject([
          {
            fields: [
              { key: "f1", channel: oven.data.key },
              { key: "t1", channel: oven.index.key },
            ],
          },
          { fields: [{ key: "f2", channel: line.data.key }] },
        ]),
      );
    });

    it("should bind a Sparkplug entry to the channel of its tag and follow a tag edit", async () => {
      const { index, data } = await createIndexed("mqtt_tag");
      const entry = createTagEntry("s1", "flow");
      const draft = await renderOnBroker(
        {
          [MQTT.Task.sparkplugPropertiesKey(entry)]: {
            index: index.key,
            channels: { "": data.key },
          },
        },
        [entry],
      );
      await waitFor(async () =>
        expect(await retrieveEntries(draft.key)).toMatchObject([
          { channel: data.key, index: index.key },
        ]),
      );
      fireEvent.click(await screen.findByRole("treeitem", { name: /flow/ }));
      fireEvent.change(await screen.findByPlaceholderText("oven/temperature"), {
        target: { value: "level" },
      });
      await waitFor(async () =>
        expect(await retrieveEntries(draft.key)).toMatchObject([
          { tag: "level", channel: 0, index: 0 },
        ]),
      );
    });
  });

  describe("deploying against a live Core", () => {
    it("should put the deploy errors on the fields they belong to", async () => {
      const { container } = await renderRead([createReadEntry("e1", "plant/+/oven")]);
      await screen.findByRole("treeitem", { name: /plant\/\+\/oven/ });
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

    it("should map the payload timestamp typed in the form to the index channel", async () => {
      const { container, dev, draft } = await renderRead([
        createReadEntry("e1", "plant/oven", {
          fields: [createReadField("f1", "/temperature")],
        }),
      ]);
      fireEvent.click(await screen.findByRole("button", { name: "Payload value" }));
      const pointer = await screen.findByPlaceholderText("/timestamp");
      fireEvent.change(pointer, { target: { value: "/ts" } });
      fireEvent.blur(pointer);
      await selectFromDropdown("Unix (s)", "Unix (ms)");
      await screen.findByText("Unix (ms)");
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.READ_SCHEMAS,
      );
      const [entry] = created.config.entries;
      if (entry.type !== "plain") throw new Error("expected a plain entry");
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      const { index } = updated.properties.read["plant/oven"];
      expect(entry.fields).toMatchObject([
        { key: "f1", pointer: "/temperature" },
        { key: entry.index, pointer: "/ts", timeFormat: "unix_ms", channel: index },
      ]);
      expect(index).toBeGreaterThan(0);
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
