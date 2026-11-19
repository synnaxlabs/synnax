// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, icd, mqtt, type rack } from "@synnaxlabs/client";
import { createTestClient, rejectRequests } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { DataType, type payload, TimeStamp, uuid } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker, openAddData, openScanner } from "@/feature/mqtt/testutil";
import {
  clickDeploy,
  clickFailedDeploy,
  deployAndAwaitTask,
  renderTaskFormTab,
  START_FAILURE,
} from "@/platform/task/testutil";
import { findDialogTriggerByText } from "@/testutil";

const client = createTestClient();

const TOPIC = "plant/oven";

const createField = (name: string, kind: payload.Kind): icd.JSONField =>
  icd.jsonFieldZ.parse({ name, pointer: `/${name}`, type: kind });

const createTimeField = (): icd.JSONField =>
  icd.jsonFieldZ.parse({
    name: "time",
    pointer: "/time",
    type: "timestamp",
    timeFormat: "unix_ms",
  });

const createTagEntry = (
  tag: string,
  overrides: Partial<MQTT.Task.SparkplugReadEntry> = {},
): MQTT.Task.SparkplugReadEntry =>
  mqtt.sparkplugReadEntryZ.parse({
    type: "sparkplug",
    key: uuid.create(),
    group: "plant",
    edgeNode: "line1",
    tag,
    ...overrides,
  });

interface EntryParams {
  /** The fields of the topic's message format. */
  fields: icd.JSONField[];
  topic?: string;
  index?: string;
  /** The channel of each field the task reads, by field key. All fields by default. */
  bound?: Record<string, number>;
}

interface RenderParams extends Partial<Omit<EntryParams, "topic">> {
  properties?: Partial<MQTT.Device.Properties>;
  /** The entries the task reads after the one that the fields make. */
  entries?: MQTT.Task.ReadEntry[];
  /**
   * Runs after the task is saved, before the form opens. The Core saves no task that an
   * ICD edit here would make invalid.
   */
  beforeRender?: (dev: device.Device) => Promise<unknown>;
}

/** Creates an ICD that holds one message, and an entry that reads it. */
const createEntry = async ({
  fields,
  topic = TOPIC,
  index,
  bound = Object.fromEntries(fields.map(({ key }) => [key, 0])),
}: EntryParams): Promise<MQTT.Task.PlainReadEntry> => {
  const message: icd.Message = {
    key: uuid.create(),
    name: topic,
    payload: { format: "json", fields },
  };
  const format = await client.icds.create({
    name: uniqueName("format"),
    messages: [message],
  });
  const bindings = Object.entries(bound).map(([field, channel]) => ({
    field,
    channel,
    dataType: "",
  }));
  return mqtt.plainReadEntryZ.parse({
    type: "plain",
    icd: format.key,
    topic,
    messages: [{ message: message.key, index, bindings }],
  });
};

// The form body renders against a configured broker only.
const renderRead = async ({
  fields,
  index,
  bound,
  properties,
  entries = [],
  beforeRender,
}: RenderParams = {}) => {
  const dev = await createBroker(client, { properties });
  const first = fields == null ? [] : [await createEntry({ fields, index, bound })];
  const config = MQTT.Task.READ_SCHEMAS.config.parse({
    device: dev.key,
    entries: [...first, ...entries],
  });
  const draft = await client.tasks.create(
    { name: "MQTT read task", type: MQTT.Task.READ_TYPE, config },
    MQTT.Task.READ_SCHEMAS,
  );
  await beforeRender?.(dev);
  const statuses: Status.NotificationSpec[] = [];
  const result = await renderTaskFormTab(MQTT.Task.Read, {
    client,
    taskKey: draft.key,
    onStatuses: (next) => {
      statuses.length = 0;
      statuses.push(...next);
    },
  });
  return { ...result, dev, draft, statuses };
};

const retrieveEntries = async (key: string) =>
  (await client.tasks.retrieve({ key, schemas: MQTT.Task.READ_SCHEMAS })).config
    .entries;

const plainOf = (entry?: MQTT.Task.ReadEntry): MQTT.Task.PlainReadEntry => {
  if (entry?.type !== "plain") throw new Error("expected a plain entry");
  return entry;
};

const sparkplugOf = (entry?: MQTT.Task.ReadEntry): MQTT.Task.SparkplugReadEntry => {
  if (entry?.type !== "sparkplug") throw new Error("expected a Sparkplug B entry");
  return entry;
};

/** @returns the bindings of the first message of the first entry. */
const retrieveBindings = async (key: string) =>
  plainOf((await retrieveEntries(key))[0]).messages[0].bindings;

const deploy = async (container: HTMLElement, key: string) =>
  await deployAndAwaitTask(client, container, key, MQTT.Task.READ_SCHEMAS);

/** Removes the message of the entry from its ICD. */
const removeMessage = async ({ icd: key, messages }: MQTT.Task.PlainReadEntry) =>
  await client.icds.dispatch(
    key,
    icd.remove({ path: ["messages"], key: messages[0].message }),
  );

/** Gives a field of the entry's message a new kind, in the ICD. */
const retype = async (
  { icd: key, messages }: MQTT.Task.PlainReadEntry,
  field: icd.JSONField,
  kind: payload.Kind,
  after?: string,
) => {
  const path = ["messages", messages[0].message, "payload", "fields"];
  await client.icds.dispatch(key, [
    icd.remove({ path, key: field.key }),
    icd.insert({ path, value: { ...field, type: kind }, after }),
  ]);
};

/** @returns the fields of the first message of an ICD. */
const jsonFieldsOf = ({ messages: [first] }: icd.ICD): icd.JSONField[] =>
  first.payload.format === "json" ? first.payload.fields : [];

/** Removes a field from the entry's message, in the ICD. */
const removeField = async (
  { icd: key, messages }: MQTT.Task.PlainReadEntry,
  field: icd.JSONField,
) =>
  await client.icds.dispatch(
    key,
    icd.remove({
      path: ["messages", messages[0].message, "payload", "fields"],
      key: field.key,
    }),
  );

/** Adds the number at /name in the topic through the Add data dialog. */
const addField = async (rackKey: rack.Key, topic: string, name: string) => {
  const scanner = await openScanner(client, rackKey);
  const dialog = await openAddData();
  const scanned = {
    topic,
    payload: `{"${name}": 1}`,
    fields: [{ pointer: `/${name}`, type: "number", sample: "1" }],
    retained: true,
  };
  await scanner.answer({ data: { topics: [scanned], truncated: false } });
  scanner.close();
  fireEvent.click(await dialog.findByRole("checkbox", { name: `Record ${name}` }));
  fireEvent.click(dialog.getByRole("button", { name: "Add 1 field" }));
};

/** Moves a field of the entry's message to the pointer /name, in the ICD. */
const repoint = async (
  { icd: key, messages }: MQTT.Task.PlainReadEntry,
  field: icd.JSONField,
  name: string,
) => {
  const path = ["messages", messages[0].message, "payload", "fields"];
  await client.icds.dispatch(key, [
    icd.remove({ path, key: field.key }),
    icd.insert({ path, value: { ...field, name, pointer: `/${name}` } }),
  ]);
};

// The details of a field show once its item in the tree is selected.
const selectField = async (name: string) =>
  fireEvent.click(
    await screen.findByRole("treeitem", { name: new RegExp(`^${name}`) }),
  );

const retrieveRead = async (key: string) =>
  (await client.devices.retrieve({ key, schemas: MQTT.Device.SCHEMAS })).properties
    .read;

/** @returns the channels that the device stores for the message of the topic. */
const retrieveStored = async (key: string, message: string) =>
  (await retrieveRead(key))[`${TOPIC}#${message}`];

/** @returns the key of the first message that the task reads. */
const firstMessage = ({ config }: { config: { entries: MQTT.Task.ReadEntry[] } }) =>
  plainOf(config.entries[0]).messages[0].message;

/** Adds a message to the ICD of the entry after its last, and reads each field. */
const addMessage = async (
  entry: MQTT.Task.PlainReadEntry,
  name: string,
  fields: icd.JSONField[],
  match?: icd.JSONMatch,
): Promise<icd.Message> => {
  const message: icd.Message = {
    key: uuid.create(),
    name,
    payload: { format: "json", match, fields },
  };
  await client.icds.dispatch(
    entry.icd,
    icd.insert({
      path: ["messages"],
      value: message,
      after: entry.messages.at(-1)?.message,
    }),
  );
  const bindings = fields.map(({ key }) => ({ field: key, channel: 0, dataType: "" }));
  entry.messages.push(icd.readMessageZ.parse({ message: message.key, bindings }));
  return message;
};

const KIND_MATCH: icd.JSONMatch = { type: "string", pointer: "/kind", value: "status" };

const createTimeChannel = async () =>
  await client.channels.create({
    name: uniqueName("time"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });

describe("MQTT read form", () => {
  it("should create a format named after the task and add later picks to it", async () => {
    const TOPICS = [
      {
        topic: TOPIC,
        payload: '{"temperature": 21.5, "label": "ok"}',
        fields: [
          { pointer: "/temperature", type: "number", sample: "21.5" },
          { pointer: "/label", type: "string", sample: '"ok"' },
        ],
        retained: true,
      },
    ];
    const { dev, draft } = await renderRead();
    await screen.findByText("Add data to read a topic or a tag");
    const scanner = await openScanner(client, dev.rack);
    let dialog = await openAddData();
    await scanner.answer({ data: { topics: TOPICS, truncated: false } });
    fireEvent.click(
      await dialog.findByRole("checkbox", { name: "Record temperature" }),
    );
    fireEvent.click(dialog.getByRole("button", { name: "Add 1 field" }));
    const [entry] = await waitFor(async () => {
      const entries = await retrieveEntries(draft.key);
      expect(entries).toHaveLength(1);
      return entries.map(plainOf);
    });
    dialog = await openAddData();
    await scanner.answer({ data: { topics: TOPICS, truncated: false } });
    scanner.close();
    fireEvent.click(await dialog.findByRole("checkbox", { name: "Record label" }));
    fireEvent.click(dialog.getByRole("button", { name: "Add 1 field" }));
    const [temperature, label] = await waitFor(async () => {
      const format = await client.icds.retrieve(entry.icd);
      expect(format.name).toBe("MQTT read task");
      expect(format.messages).toMatchObject([
        {
          key: entry.messages[0].message,
          name: TOPIC,
          payload: {
            format: "json",
            fields: [
              { name: "temperature", pointer: "/temperature", type: "float" },
              { name: "label", pointer: "/label", type: "string" },
            ],
          },
        },
      ]);
      return format.messages[0].payload.format === "json"
        ? format.messages[0].payload.fields
        : [];
    });
    await waitFor(async () =>
      expect(await retrieveBindings(draft.key)).toEqual([
        { field: temperature.key, channel: 0, dataType: "" },
        { field: label.key, channel: 0, dataType: "" },
      ]),
    );
    expect(await retrieveEntries(draft.key)).toHaveLength(1);
  });

  it("should pick and read the string field of a topic with its numbers", async () => {
    const { dev, draft } = await renderRead();
    const scanner = await openScanner(client, dev.rack);
    const dialog = await openAddData();
    await scanner.answer({
      data: {
        topics: [
          {
            topic: TOPIC,
            payload: '{"temperature": 21.5, "label": "ok"}',
            fields: [
              { pointer: "/temperature", type: "number", sample: "21.5" },
              { pointer: "/label", type: "string", sample: '"ok"' },
            ],
            retained: true,
          },
        ],
        truncated: false,
      },
    });
    scanner.close();
    fireEvent.click(await dialog.findByRole("checkbox", { name: "Pick oven" }));
    fireEvent.click(dialog.getByRole("button", { name: "Add 2 fields" }));
    await waitFor(async () =>
      expect(await retrieveBindings(draft.key)).toHaveLength(2),
    );
  });

  it("should focus the topic of a topic added by name", async () => {
    await renderRead();
    const dialog = await openAddData();
    fireEvent.click(dialog.getByRole("button", { name: "Add a topic by name" }));
    const topic = await screen.findByPlaceholderText("plant/line1/telemetry");
    await waitFor(() => expect(document.activeElement).toBe(topic));
  });

  it("should add a topic by name", async () => {
    const { draft } = await renderRead();
    const dialog = await openAddData();
    fireEvent.click(dialog.getByRole("button", { name: "Add a topic by name" }));
    const topic = await screen.findByPlaceholderText("plant/line1/telemetry");
    await waitFor(async () =>
      expect(await retrieveEntries(draft.key)).toMatchObject([
        { type: "plain", topic: "" },
      ]),
    );
    fireEvent.change(topic, { target: { value: "plant/kiln" } });
    fireEvent.blur(topic);
    await waitFor(async () =>
      expect(await retrieveEntries(draft.key)).toMatchObject([
        { type: "plain", topic: "plant/kiln" },
      ]),
    );
  });

  it("should open one Add data dialog for a double click", async () => {
    await renderRead();
    const button = await screen.findByRole("button", { name: "Add data" });
    fireEvent.click(button);
    fireEvent.click(button);
    await screen.findByRole("dialog");
    expect(screen.getAllByRole("dialog")).toHaveLength(1);
  });

  it("should give the ICD a default name while the task has none", async () => {
    const { dev, draft } = await renderRead();
    const name = await screen.findByLabelText("Name");
    fireEvent.change(name, { target: { value: "" } });
    fireEvent.blur(name);
    await addField(dev.rack, TOPIC, "temperature");
    const [entry] = await waitFor(async () => {
      const entries = await retrieveEntries(draft.key);
      expect(entries).toHaveLength(1);
      return entries;
    });
    expect((await client.icds.retrieve(plainOf(entry).icd)).name).toBe(
      "MQTT read task",
    );
  });

  it("should show the format of a field from the ICD of its entry", async () => {
    const kiln = await createEntry({
      topic: "plant/kiln",
      fields: [createField("pressure", "float")],
    });
    await renderRead({
      fields: [createField("temperature", "float")],
      entries: [kiln],
    });
    await selectField("temperature");
    await screen.findByDisplayValue("/temperature");
    await selectField("pressure");
    await screen.findByDisplayValue("/pressure");
    expect(screen.queryByDisplayValue("/temperature")).toBeNull();
  });

  it("should add a field to the message of a topic, read it, and select it", async () => {
    const temperature = createField("temperature", "float");
    const { draft } = await renderRead({ fields: [temperature] });
    fireEvent.click(await screen.findByRole("treeitem", { name: /JSON.*plant\/oven/ }));
    fireEvent.click(await screen.findByRole("button", { name: "Add field" }));
    await screen.findByDisplayValue("Field 2");
    const { icd: icdKey, messages } = plainOf((await retrieveEntries(draft.key))[0]);
    await waitFor(async () =>
      expect((await client.icds.retrieve(icdKey)).messages[0].payload).toMatchObject({
        fields: [{ key: temperature.key }, { name: "Field 2", pointer: "" }],
      }),
    );
    const [, added] = jsonFieldsOf(await client.icds.retrieve(icdKey));
    await waitFor(async () =>
      expect(await retrieveBindings(draft.key)).toEqual([
        messages[0].bindings[0],
        { field: added.key, channel: 0, dataType: "" },
      ]),
    );
  });

  it("should remove a field from the ICD and from each topic that reads it", async () => {
    const temperature = createField("temperature", "float");
    const pressure = createField("pressure", "float");
    const first = await createEntry({ fields: [temperature, pressure] });
    const second = mqtt.plainReadEntryZ.parse({
      ...first,
      key: uuid.create(),
      topic: `${TOPIC}/2`,
    });
    const { draft } = await renderRead({ entries: [first, second] });
    const [item] = await screen.findAllByRole("treeitem", { name: /^pressure/ });
    fireEvent.contextMenu(item);
    fireEvent.click(await screen.findByRole("menuitem", { name: "Remove" }));
    await waitFor(async () =>
      expect(
        jsonFieldsOf(await client.icds.retrieve(first.icd)).map(({ key }) => key),
      ).toEqual([temperature.key]),
    );
    expect(screen.queryByRole("treeitem", { name: /^pressure/ })).toBeNull();
    await waitFor(async () => {
      const entries = (await retrieveEntries(draft.key)).map(plainOf);
      const reads = entries.map(({ messages: [m] }) => m.bindings.map((b) => b.field));
      expect(reads).toEqual([[temperature.key], [temperature.key]]);
    });
  });

  it("should clear the index of a topic whose index field is removed", async () => {
    const time = createTimeField();
    const temperature = createField("temperature", "float");
    const first = await createEntry({ fields: [time, temperature] });
    const second = mqtt.plainReadEntryZ.parse({
      ...first,
      key: uuid.create(),
      topic: `${TOPIC}/2`,
      messages: [{ ...first.messages[0], index: time.key }],
    });
    const { draft } = await renderRead({ entries: [first, second] });
    const [item] = await screen.findAllByRole("treeitem", { name: /^time/ });
    fireEvent.contextMenu(item);
    fireEvent.click(await screen.findByRole("menuitem", { name: "Remove" }));
    await waitFor(async () => {
      const [, indexed] = (await retrieveEntries(draft.key)).map(plainOf);
      expect(indexed.messages[0].index).toBeUndefined();
      expect(indexed.messages[0].bindings.map(({ field }) => field)).toEqual([
        temperature.key,
      ]);
    });
  });

  it("should store the default and mapping set in the details of a field", async () => {
    const { draft } = await renderRead({ fields: [createField("state", "integer")] });
    await selectField("state");
    fireEvent.click(await screen.findByRole("checkbox", { name: "Optional" }));
    fireEvent.click(await screen.findByRole("button", { name: "Add mapping" }));
    for (const [label, value] of [
      ["Label 1", "RUN"],
      ["Value 1", "1"],
    ]) {
      const input = await screen.findByLabelText(label);
      fireEvent.change(input, { target: { value } });
      fireEvent.blur(input);
    }
    const { icd: icdKey } = plainOf((await retrieveEntries(draft.key))[0]);
    await waitFor(async () =>
      expect(jsonFieldsOf(await client.icds.retrieve(icdKey))[0]).toMatchObject({
        default: "0",
        enumValues: [{ label: "RUN", value: "1" }],
      }),
    );
  });

  it("should remove the default of a timestamp field in its details", async () => {
    const at = TimeStamp.milliseconds(1_700_000_000_000);
    const time = icd.jsonFieldZ.parse({
      ...createTimeField(),
      default: at,
    });
    const { draft } = await renderRead({ fields: [time], bound: { [time.key]: 0 } });
    await selectField("time");
    expect(await screen.findByText(at.toString("preciseDate", "local"))).toBeTruthy();
    fireEvent.click(await screen.findByRole("checkbox", { name: "Optional" }));
    const { icd: icdKey } = plainOf((await retrieveEntries(draft.key))[0]);
    await waitFor(async () =>
      expect(jsonFieldsOf(await client.icds.retrieve(icdKey))[0]).not.toHaveProperty(
        "default",
      ),
    );
    expect(screen.queryByRole("checkbox", { name: "Optional" })).toBeNull();
  });

  it("should add fields to a topic in the ICD of its entry and read them", async () => {
    const kiln = await createEntry({
      topic: "plant/kiln",
      fields: [createField("pressure", "float")],
    });
    const { dev, draft } = await renderRead({
      fields: [createField("temperature", "float")],
      entries: [kiln],
    });
    await screen.findByRole("treeitem", { name: /^temperature/ });
    await addField(dev.rack, "plant/kiln", "flow");
    await waitFor(async () =>
      expect((await client.icds.retrieve(kiln.icd)).messages).toMatchObject([
        {
          key: kiln.messages[0].message,
          payload: { fields: [{ pointer: "/pressure" }, { pointer: "/flow" }] },
        },
      ]),
    );
    await waitFor(async () =>
      expect(
        plainOf((await retrieveEntries(draft.key))[1]).messages[0].bindings,
      ).toEqual([
        kiln.messages[0].bindings[0],
        { field: expect.any(String), channel: 0, dataType: "" },
      ]),
    );
  });

  it("should number a picked field whose name a field of its message holds", async () => {
    const rate = icd.jsonFieldZ.parse({
      name: "flow",
      pointer: "/rate",
      type: "float",
    });
    const kiln = await createEntry({ topic: "plant/kiln", fields: [rate] });
    const { dev } = await renderRead({
      fields: [createField("temperature", "float")],
      entries: [kiln],
    });
    await screen.findByRole("treeitem", { name: /^temperature/ });
    await addField(dev.rack, "plant/kiln", "flow");
    await waitFor(async () =>
      expect((await client.icds.retrieve(kiln.icd)).messages).toMatchObject([
        {
          payload: {
            fields: [
              { name: "flow", pointer: "/rate" },
              { name: "flow_2", pointer: "/flow" },
            ],
          },
        },
      ]),
    );
  });

  it("should reuse a channel stored under another message of a topic it adds", async () => {
    const time = await createTimeChannel();
    const flow = await client.channels.create({
      name: uniqueName("flow"),
      dataType: DataType.FLOAT64,
      index: time.key,
    });
    const { dev, draft } = await renderRead({
      fields: [createField("temperature", "float")],
      properties: {
        read: {
          [`plant/kiln#${uuid.create()}`]: {
            index: time.key,
            channels: { "/flow": flow.key },
          },
        },
      },
    });
    await screen.findByRole("treeitem", { name: /^temperature/ });
    await addField(dev.rack, "plant/kiln", "flow");
    await waitFor(async () =>
      expect(
        plainOf((await retrieveEntries(draft.key))[1]).messages[0].bindings,
      ).toEqual([{ field: expect.any(String), channel: flow.key, dataType: "" }]),
    );
  });

  it("should not reuse a channel stored for another message of the topic", async () => {
    const time = await createTimeChannel();
    const flow = await client.channels.create({
      name: uniqueName("flow"),
      dataType: DataType.FLOAT64,
      index: time.key,
    });
    const entry = await createEntry({ fields: [createField("temperature", "float")] });
    const status = await addMessage(entry, "status", [], KIND_MATCH);
    const { dev, draft } = await renderRead({
      entries: [entry],
      properties: {
        read: {
          [`${TOPIC}#${status.key}`]: {
            index: time.key,
            channels: { "/flow": flow.key },
          },
        },
      },
    });
    await screen.findByRole("treeitem", { name: /^temperature/ });
    await addField(dev.rack, TOPIC, "flow");
    await waitFor(async () =>
      expect((await retrieveBindings(draft.key)).map(({ channel }) => channel)).toEqual(
        [0, 0],
      ),
    );
  });

  it("should show a note for a topic whose message is gone", async () => {
    const entry = await createEntry({ fields: [createField("temperature", "float")] });
    await renderRead({ entries: [entry], beforeRender: () => removeMessage(entry) });
    await screen.findByText("Message not found");
  });

  it.each([
    { name: "gone", note: "Message not found", change: removeMessage },
    {
      name: "not JSON",
      note: "The format of this message is not JSON. Open it to change its format.",
      change: async ({ icd: key, messages }: MQTT.Task.PlainReadEntry) =>
        await client.icds.dispatch(
          key,
          icd.set({
            path: ["messages", messages[0].message, "payload"],
            value: { format: "text", delimiter: ",", prefix: "", fields: [] },
          }),
        ),
    },
  ])(
    "should not offer Add field for a topic whose message is $name",
    async ({ note, change }) => {
      const entry = await createEntry({
        fields: [createField("temperature", "float")],
      });
      await renderRead({ entries: [entry], beforeRender: () => change(entry) });
      await screen.findByText(note);
      expect(screen.queryByRole("button", { name: "Add field" })).toBeNull();
      fireEvent.contextMenu(
        await screen.findByRole("treeitem", { name: /plant\/oven/ }),
      );
      await screen.findByRole("menuitem", { name: "Add message" });
      expect(screen.queryByRole("menuitem", { name: "Add field" })).toBeNull();
    },
  );

  it("should point a topic whose message is gone at a new message", async () => {
    const time = createTimeField();
    const timeCh = await createTimeChannel();
    const entry = await createEntry({
      fields: [time, createField("temperature", "float")],
      index: time.key,
      bound: { [time.key]: timeCh.key },
    });
    const { dev, draft } = await renderRead({
      entries: [entry],
      beforeRender: () => removeMessage(entry),
    });
    await screen.findByText("Message not found");
    await addField(dev.rack, TOPIC, "temperature");
    await waitFor(async () => {
      const { messages } = plainOf((await retrieveEntries(draft.key))[0]);
      expect(messages).toEqual([
        {
          message: expect.any(String),
          bindings: [{ field: expect.any(String), channel: 0, dataType: "" }],
        },
      ]);
      expect((await client.icds.retrieve(entry.icd)).messages).toMatchObject([
        {
          key: messages[0].message,
          name: TOPIC,
          payload: { fields: [{ pointer: "/temperature" }] },
        },
      ]);
    });
  });

  it("should disable and enable a topic through the context menu", async () => {
    const { draft } = await renderRead({
      fields: [createField("temperature", "float")],
    });
    const item = () => screen.findByRole("treeitem", { name: /plant\/oven/ });
    fireEvent.contextMenu(await item());
    const disable = await screen.findByText("Disable");
    expect(screen.queryByText("Enable")).toBeNull();
    fireEvent.click(disable);
    await waitFor(async () =>
      expect((await retrieveEntries(draft.key))[0].disabled).toBe(true),
    );
    fireEvent.contextMenu(await item());
    fireEvent.click(await screen.findByText("Enable"));
    await waitFor(async () =>
      expect((await retrieveEntries(draft.key))[0].disabled).toBe(false),
    );
  });

  it("should read the time from the first timestamp field on payload timing", async () => {
    const time = createTimeField();
    const temperature = createField("temperature", "float");
    const { dev, draft } = await renderRead({
      fields: [temperature, time],
      bound: { [temperature.key]: 0 },
    });
    fireEvent.click(await screen.findByRole("button", { name: "Payload value" }));
    await waitFor(async () =>
      expect(plainOf((await retrieveEntries(draft.key))[0]).messages[0]).toEqual({
        message: expect.any(String),
        index: time.key,
        bindings: [
          { field: temperature.key, channel: 0, dataType: "" },
          { field: time.key, channel: 0, dataType: "" },
        ],
      }),
    );
    await screen.findByDisplayValue(`${dev.name}_plant_oven_time`);
    expect(screen.queryByRole("treeitem", { name: /^time/ })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Arrival time" }));
    await waitFor(async () =>
      expect(plainOf((await retrieveEntries(draft.key))[0]).messages[0]).toEqual({
        message: expect.any(String),
        bindings: [{ field: temperature.key, channel: 0, dataType: "" }],
      }),
    );
    const box = await screen.findByRole("checkbox", { name: "Read time" });
    expect((box as HTMLInputElement).checked).toBe(false);
  });

  it("should read the time only from a timestamp field with no default", async () => {
    const started = icd.jsonFieldZ.parse({
      name: "started",
      pointer: "/started",
      type: "timestamp",
      timeFormat: "unix_ms",
      default: TimeStamp.milliseconds(1_700_000_000_000),
    });
    const time = createTimeField();
    const { draft } = await renderRead({ fields: [started, time] });
    fireEvent.click(await screen.findByRole("button", { name: "Payload value" }));
    await waitFor(async () =>
      expect(plainOf((await retrieveEntries(draft.key))[0]).messages[0].index).toBe(
        time.key,
      ),
    );
    fireEvent.click(await screen.findByRole("button", { name: /^Field/ }));
    const option = await screen.findByRole("option", { name: /^started/ });
    expect(option.getAttribute("aria-disabled")).toBe("true");
    expect(option.textContent).toContain(
      "It has a default, so it cannot be the index.",
    );
  });

  describe("messages", () => {
    it("should list each message of a topic that reads several, with its fields", async () => {
      const entry = await createEntry({
        fields: [createField("temperature", "float")],
      });
      await addMessage(entry, "status", [createField("mode", "string")], KIND_MATCH);
      await renderRead({ entries: [entry] });
      await screen.findByRole("treeitem", { name: TOPIC });
      expect(screen.getByRole("treeitem", { name: "status" })).toBeTruthy();
      expect(screen.getByRole("checkbox", { name: "Read temperature" })).toBeTruthy();
      expect(screen.getByRole("checkbox", { name: "Read mode" })).toBeTruthy();
    });

    it("should add a message that matches at the pointer of the others", async () => {
      const entry = await createEntry({
        fields: [createField("temperature", "float")],
      });
      await addMessage(entry, "status", [createField("mode", "string")], KIND_MATCH);
      const { draft } = await renderRead({ entries: [entry] });
      fireEvent.click(
        await screen.findByRole("treeitem", { name: /JSON.*plant\/oven/ }),
      );
      fireEvent.click(await screen.findByRole("button", { name: "Add message" }));
      await screen.findByDisplayValue(`${TOPIC} 3`);
      expect(screen.getByDisplayValue("/kind")).toBeTruthy();
      await waitFor(async () => {
        const { messages } = await client.icds.retrieve({ key: entry.icd });
        expect(messages).toHaveLength(3);
        expect(messages[2]).toMatchObject({
          name: `${TOPIC} 3`,
          payload: { format: "json", match: { ...KIND_MATCH, value: "" }, fields: [] },
        });
        const reads = plainOf((await retrieveEntries(draft.key))[0]).messages;
        expect(reads.map(({ message }) => message)).toEqual([
          entry.messages[0].message,
          entry.messages[1].message,
          messages[2].key,
        ]);
      });
    });

    it("should add a message from the context menu of a topic", async () => {
      const entry = await createEntry({
        fields: [createField("temperature", "float")],
      });
      const { draft } = await renderRead({ entries: [entry] });
      fireEvent.contextMenu(
        await screen.findByRole("treeitem", { name: /JSON.*plant\/oven/ }),
      );
      fireEvent.click(await screen.findByRole("menuitem", { name: "Add message" }));
      await screen.findByRole("treeitem", { name: `${TOPIC} 2` });
      await waitFor(async () =>
        expect(plainOf((await retrieveEntries(draft.key))[0]).messages).toHaveLength(2),
      );
    });

    it("should stop reading a message of a topic, keeping it in the ICD", async () => {
      const entry = await createEntry({
        fields: [createField("temperature", "float")],
      });
      const status = await addMessage(
        entry,
        "status",
        [createField("mode", "string")],
        KIND_MATCH,
      );
      const { draft } = await renderRead({ entries: [entry] });
      fireEvent.contextMenu(await screen.findByRole("treeitem", { name: "status" }));
      fireEvent.click(await screen.findByText("Remove"));
      await waitFor(async () =>
        expect(plainOf((await retrieveEntries(draft.key))[0]).messages).toEqual([
          entry.messages[0],
        ]),
      );
      expect(screen.queryByRole("treeitem", { name: "status" })).toBeNull();
      const { messages } = await client.icds.retrieve({ key: entry.icd });
      expect(messages.map(({ key }) => key)).toContain(status.key);
    });

    it("should not offer to remove every message of a topic", async () => {
      const entry = await createEntry({
        fields: [createField("temperature", "float")],
      });
      await addMessage(entry, "status", [createField("mode", "string")], KIND_MATCH);
      await renderRead({ entries: [entry] });
      fireEvent.click(await screen.findByRole("treeitem", { name: TOPIC }));
      fireEvent.keyDown(window, { code: "Shift" });
      fireEvent.click(screen.getByRole("treeitem", { name: "status" }), {
        shiftKey: true,
      });
      fireEvent.keyUp(window, { code: "Shift" });
      fireEvent.contextMenu(screen.getByRole("treeitem", { name: "status" }));
      await screen.findByText("Reload Console");
      expect(screen.queryByText("Remove")).toBeNull();
    });
  });

  it("should not rename the message of a topic when the topic changes", async () => {
    const { draft } = await renderRead({
      fields: [createField("temperature", "float")],
    });
    const input = await screen.findByDisplayValue(TOPIC);
    fireEvent.change(input, { target: { value: "plant/kiln" } });
    fireEvent.blur(input);
    const entry = await waitFor(async () => {
      const [first] = await retrieveEntries(draft.key);
      expect(plainOf(first).topic).toBe("plant/kiln");
      return plainOf(first);
    });
    expect((await client.icds.retrieve(entry.icd)).messages[0].name).toBe(TOPIC);
  });

  it("should number the name of a new message that the ICD already holds", async () => {
    const { dev, draft } = await renderRead({
      fields: [createField("temperature", "float")],
    });
    const input = await screen.findByDisplayValue(TOPIC);
    fireEvent.change(input, { target: { value: "plant/kiln" } });
    fireEvent.blur(input);
    const entry = await waitFor(async () => {
      const [first] = await retrieveEntries(draft.key);
      expect(plainOf(first).topic).toBe("plant/kiln");
      return plainOf(first);
    });
    await addField(dev.rack, TOPIC, "flow");
    await waitFor(async () =>
      expect(
        (await client.icds.retrieve(entry.icd)).messages.map(({ name }) => name),
      ).toEqual([TOPIC, `${TOPIC} 2`]),
    );
  });

  describe("field details", () => {
    it("should show the scale, offset, and units only for a number type", async () => {
      await renderRead({
        fields: [createField("temperature", "float"), createField("mode", "string")],
      });
      await selectField("temperature");
      await screen.findByText("Units");
      expect(screen.getByText("Scale")).toBeTruthy();
      expect(screen.getByText("Offset")).toBeTruthy();
      await selectField("mode");
      await waitFor(() => expect(screen.queryByText("Units")).toBeNull());
      expect(screen.queryByText("Scale")).toBeNull();
      expect(screen.getByText("Enum mapping")).toBeTruthy();
    });
  });

  describe("channels", () => {
    it("should stop and start reading a field, keeping the channel of its pointer", async () => {
      const ch = await client.channels.create({
        name: uniqueName("temperature"),
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const temperature = createField("temperature", "float");
      const { draft } = await renderRead({
        fields: [temperature, createField("pressure", "float")],
        bound: { [temperature.key]: ch.key },
        properties: {
          read: { [TOPIC]: { index: 0, channels: { "/temperature": ch.key } } },
        },
      });
      await selectField("temperature");
      const box = await screen.findByRole("checkbox", { name: "Read temperature" });
      await screen.findByDisplayValue(ch.name);
      fireEvent.click(box);
      await waitFor(async () => expect(await retrieveBindings(draft.key)).toEqual([]));
      expect(screen.queryByDisplayValue(ch.name)).toBeNull();
      fireEvent.click(box);
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toEqual([
          { field: temperature.key, channel: ch.key, dataType: "" },
        ]),
      );
      await screen.findByDisplayValue(ch.name);
    });

    it("should start reading a field again with the channel that a deploy in the open form stored", async () => {
      const time = createTimeField();
      const temperature = createField("temperature", "float");
      const { container, dev, draft } = await renderRead({
        fields: [time, temperature],
        index: time.key,
      });
      const created = await deploy(container, draft.key);
      const { channels } = await retrieveStored(dev.key, firstMessage(created));
      const ch = channels["/temperature"];
      expect(ch).not.toBe(0);
      await selectField("temperature");
      const box = await screen.findByRole("checkbox", { name: "Read temperature" });
      fireEvent.click(box);
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).not.toContainEqual(
          expect.objectContaining({ field: temperature.key }),
        ),
      );
      fireEvent.click(box);
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toContainEqual({
          field: temperature.key,
          channel: ch,
          dataType: "",
        }),
      );
      const again = await deploy(container, draft.key);
      expect(plainOf(again.config.entries[0]).messages[0].bindings).toContainEqual({
        field: temperature.key,
        channel: ch,
        dataType: "",
      });
      expect((await retrieveStored(dev.key, firstMessage(again))).channels).toEqual(
        channels,
      );
    });

    it("should stop and start reading a field through the context menu", async () => {
      const temperature = createField("temperature", "float");
      const { draft } = await renderRead({ fields: [temperature] });
      const item = () => screen.findByRole("treeitem", { name: /^temperature/ });
      fireEvent.contextMenu(await item());
      const disable = await screen.findByText("Disable");
      expect(screen.queryByText("Enable")).toBeNull();
      fireEvent.click(disable);
      await waitFor(async () => expect(await retrieveBindings(draft.key)).toEqual([]));
      fireEvent.contextMenu(await item());
      fireEvent.click(await screen.findByText("Enable"));
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toEqual([
          { field: temperature.key, channel: 0, dataType: "" },
        ]),
      );
    });

    it("should name the channel of a new field after the broker, topic, and field", async () => {
      const { dev } = await renderRead({
        fields: [createField("temperature", "float")],
      });
      await selectField("temperature");
      const name = await screen.findByDisplayValue(
        `${dev.name}_plant_oven_temperature`,
      );
      expect((name as HTMLInputElement).disabled).toBe(true);
      const item = screen.getByRole("treeitem", { name: /^temperature/ });
      expect(within(item).getByRole("img", { name: "New channel" })).toBeTruthy();
    });

    it("should offer only the data types that the kind of a field converts to", async () => {
      const count = createField("count", "integer");
      const { container, dev, draft } = await renderRead({ fields: [count] });
      await selectField("count");
      fireEvent.click(await findDialogTriggerByText("int64"));
      const dialogs = await screen.findAllByRole("dialog");
      const options = within(dialogs[dialogs.length - 1]);
      expect(options.getByText("float64")).toBeTruthy();
      expect(options.queryByText("uint8")).toBeNull();
      expect(options.queryByText("String")).toBeNull();
      fireEvent.click(options.getByText("int32"));
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toEqual([
          { field: count.key, channel: 0, dataType: "int32" },
        ]),
      );
      await deploy(container, draft.key);
      const { channels } = await retrieveStored(dev.key, firstMessage(draft));
      const ch = await client.channels.retrieve(channels["/count"]);
      expect(ch.dataType.toString()).toBe("int32");
    });

    it("should lock the data type of a field to that of its channel", async () => {
      const ch = await client.channels.create({
        name: uniqueName("count"),
        dataType: DataType.INT16,
        virtual: true,
      });
      const count = createField("count", "integer");
      await renderRead({ fields: [count], bound: { [count.key]: ch.key } });
      await selectField("count");
      await screen.findByDisplayValue(ch.name);
      expect(screen.getByText("Set on the channel")).toBeTruthy();
      await findDialogTriggerByText("int16");
    });

    it("should rename the live channel of a field", async () => {
      const ch = await client.channels.create({
        name: uniqueName("temperature"),
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const temperature = createField("temperature", "float");
      await renderRead({
        fields: [temperature],
        bound: { [temperature.key]: ch.key },
      });
      await selectField("temperature");
      const input = await screen.findByDisplayValue(ch.name);
      const renamed = uniqueName("oven_temperature");
      fireEvent.change(input, { target: { value: renamed } });
      fireEvent.blur(input);
      await waitFor(async () =>
        expect((await client.channels.retrieve(ch.key)).name).toBe(renamed),
      );
    });

    it("should unbind a channel that the kind of its field no longer converts to", async () => {
      const ch = await client.channels.create({
        name: uniqueName("count"),
        dataType: DataType.INT64,
        virtual: true,
      });
      const count = createField("count", "integer");
      const entry = await createEntry({
        fields: [count],
        bound: { [count.key]: ch.key },
      });
      const { draft } = await renderRead({
        entries: [entry],
        beforeRender: () => retype(entry, count, "float"),
      });
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toEqual([
          { field: count.key, channel: 0, dataType: "" },
        ]),
      );
    });

    it("should stop reading the time of a field that is no longer a timestamp", async () => {
      const timeCh = await createTimeChannel();
      const time = createTimeField();
      const temperature = createField("temperature", "float");
      const entry = await createEntry({
        fields: [time, temperature],
        index: time.key,
        bound: { [time.key]: timeCh.key, [temperature.key]: 0 },
      });
      const { draft } = await renderRead({
        entries: [entry],
        beforeRender: () =>
          retype(entry, { ...createField("time", "float"), key: time.key }, "float"),
      });
      await waitFor(async () =>
        expect(plainOf((await retrieveEntries(draft.key))[0]).messages[0]).toEqual({
          message: entry.messages[0].message,
          bindings: [{ field: temperature.key, channel: 0, dataType: "" }],
        }),
      );
    });

    it("should stop reading a field that leaves the message format", async () => {
      const temperature = createField("temperature", "float");
      const pressure = createField("pressure", "float");
      const entry = await createEntry({ fields: [temperature, pressure] });
      const { draft } = await renderRead({ entries: [entry] });
      await screen.findByRole("checkbox", { name: "Read pressure" });
      await removeField(entry, pressure);
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toEqual([
          { field: temperature.key, channel: 0, dataType: "" },
        ]),
      );
    });
  });

  describe("channel name preview", () => {
    /** Renders a task whose first topic reads temperature, then selects the field. */
    const renderTemperature = async (beforeRender?: RenderParams["beforeRender"]) => {
      const time = createTimeField();
      const { dev } = await renderRead({
        fields: [time, createField("temperature", "float")],
        index: time.key,
        beforeRender,
      });
      await selectField("temperature");
      return `${dev.name}_plant_oven_temperature`;
    };

    it("should preview a numbered name for a new channel whose name is taken", async () => {
      const name = await renderTemperature(
        async (dev) =>
          await client.channels.create({
            name: `${dev.name}_plant_oven_temperature`,
            dataType: DataType.FLOAT64,
            virtual: true,
          }),
      );
      await screen.findByDisplayValue(`${name}_2`);
    });

    it("should preview the plain name for a new channel whose name is free", async () => {
      const name = await renderTemperature();
      await screen.findByDisplayValue(name);
      expect(screen.queryByDisplayValue(`${name}_2`)).toBeNull();
    });

    it("should preview the plain name for a channel that replaces the one holding it", async () => {
      const time = await createTimeChannel();
      const countCh = await client.channels.create({
        name: uniqueName("count"),
        dataType: DataType.INT64,
        index: time.key,
      });
      const timeField = createTimeField();
      const temperature = createField("temperature", "float");
      const count = createField("count", "integer");
      const entry = await createEntry({
        fields: [timeField, temperature, count],
        index: timeField.key,
        bound: {
          [timeField.key]: time.key,
          [temperature.key]: 0,
          [count.key]: countCh.key,
        },
      });
      const { dev } = await renderRead({
        entries: [entry],
        beforeRender: async (dev) => {
          await retype(entry, count, "float", temperature.key);
          await client.channels.rename(countCh.key, `${dev.name}_plant_oven_count`);
          await client.channels.create({
            name: `${dev.name}_plant_oven_temperature`,
            dataType: DataType.FLOAT64,
            virtual: true,
          });
        },
        properties: {
          read: { [TOPIC]: { index: time.key, channels: { "/count": countCh.key } } },
        },
      });
      // The numbered name of temperature shows that the plan of the message is done.
      await selectField("temperature");
      await screen.findByDisplayValue(`${dev.name}_plant_oven_temperature_2`);
      await selectField("count");
      await screen.findByDisplayValue(`${dev.name}_plant_oven_count`);
      expect(screen.queryByDisplayValue(`${dev.name}_plant_oven_count_2`)).toBeNull();
    });

    it("should preview a numbered name for a new index whose name is taken", async () => {
      const time = createTimeField();
      const { dev } = await renderRead({
        fields: [time, createField("temperature", "float")],
        index: time.key,
        beforeRender: async (dev) =>
          await client.channels.create({
            name: `${dev.name}_plant_oven_time`,
            dataType: DataType.TIMESTAMP,
            isIndex: true,
          }),
      });
      fireEvent.click(
        await screen.findByRole("treeitem", { name: /JSON.*plant\/oven/ }),
      );
      await screen.findByDisplayValue(`${dev.name}_plant_oven_time_2`);
    });

    it("should show the details of a topic that reads no messages", async () => {
      const format = await client.icds.create({
        name: uniqueName("format"),
        messages: [],
      });
      await renderRead({
        entries: [
          mqtt.plainReadEntryZ.parse({
            type: "plain",
            icd: format.key,
            topic: "plant/empty",
            messages: [],
          }),
        ],
      });
      fireEvent.click(await screen.findByRole("treeitem", { name: /plant\/empty/ }));
      await screen.findByDisplayValue("plant/empty");
    });

    it("should preview a numbered name for the new channel of a tag whose name is taken", async () => {
      const { dev } = await renderRead({
        entries: [createTagEntry("flow")],
        beforeRender: async (dev) =>
          await client.channels.create({
            name: `${dev.name}_plant_line1_flow`,
            dataType: DataType.FLOAT64,
            virtual: true,
          }),
      });
      fireEvent.click(
        await screen.findByRole("treeitem", { name: /SpB.*plant\/line1\/flow/ }),
      );
      await screen.findByPlaceholderText(`${dev.name}_plant_line1_flow_2`);
    });
  });

  it("should say the numbered name that a typed tag channel name saves as", async () => {
    const typed = uniqueName("flow");
    const { dev } = await renderRead({
      entries: [createTagEntry("flow")],
      beforeRender: async () =>
        await client.channels.create({
          name: typed,
          dataType: DataType.FLOAT64,
          virtual: true,
        }),
    });
    fireEvent.click(
      await screen.findByRole("treeitem", { name: /SpB.*plant\/line1\/flow/ }),
    );
    const input = await screen.findByPlaceholderText(`${dev.name}_plant_line1_flow`);
    fireEvent.change(input, { target: { value: typed } });
    await screen.findByText(`Saves as ${typed}_2`);
  });

  it("should say the numbered name that a name typed in an earlier tag saves as", async () => {
    const typed = uniqueName("flow");
    const { dev } = await renderRead({
      entries: [createTagEntry("flow", { name: typed }), createTagEntry("level")],
    });
    fireEvent.click(
      await screen.findByRole("treeitem", { name: /SpB.*plant\/line1\/level/ }),
    );
    const input = await screen.findByPlaceholderText(`${dev.name}_plant_line1_level`);
    fireEvent.change(input, { target: { value: typed } });
    await screen.findByText(`Saves as ${typed}_2`);
  });

  it("should leave a disabled tag out of the numbering of typed names", async () => {
    const typed = uniqueName("flow");
    const { dev } = await renderRead({
      entries: [
        createTagEntry("flow", { name: typed, disabled: true }),
        createTagEntry("level", { name: typed }),
        createTagEntry("pressure"),
      ],
    });
    fireEvent.click(
      await screen.findByRole("treeitem", { name: /SpB.*plant\/line1\/pressure/ }),
    );
    const input = await screen.findByPlaceholderText(
      `${dev.name}_plant_line1_pressure`,
    );
    fireEvent.change(input, { target: { value: typed } });
    await screen.findByText(`Saves as ${typed}_2`);
  });

  it("should number the planned name of a field renamed in its ICD", async () => {
    const temperature = createField("temperature", "float");
    const later = await createEntry({ fields: [temperature], topic: "plant_oven" });
    const { dev } = await renderRead({
      entries: [
        await createEntry({ fields: [createField("pressure", "float")] }),
        later,
      ],
    });
    fireEvent.click(await screen.findByRole("treeitem", { name: /^temperature/ }));
    await screen.findByDisplayValue(`${dev.name}_plant_oven_temperature`);
    const { message } = later.messages[0];
    await client.icds.dispatch(
      later.icd,
      icd.set({
        path: ["messages", message, "payload", "fields", temperature.key, "name"],
        value: "pressure",
      }),
    );
    await screen.findByDisplayValue(`${dev.name}_plant_oven_pressure_2`);
  });

  it("should preview its own name for each of two entries that read one message", async () => {
    const entry = await createEntry({ fields: [createField("temperature", "float")] });
    const { dev } = await renderRead({
      entries: [entry, { ...entry, key: uuid.create() }],
      beforeRender: async (dev) =>
        await client.channels.create({
          name: `${dev.name}_plant_oven_temperature`,
          dataType: DataType.FLOAT64,
          virtual: true,
        }),
    });
    const items = await screen.findAllByRole("treeitem", { name: /^temperature/ });
    fireEvent.click(items[1]);
    await screen.findByDisplayValue(`${dev.name}_plant_oven_temperature_3`);
    fireEvent.click(items[0]);
    await screen.findByDisplayValue(`${dev.name}_plant_oven_temperature_2`);
  });

  it("should preview a numbered name for a field whose name an earlier topic takes", async () => {
    const fields = () => [createField("temperature", "float")];
    const { dev } = await renderRead({
      entries: [
        await createEntry({ fields: fields(), topic: "plant/oven" }),
        await createEntry({ fields: fields(), topic: "plant_oven" }),
      ],
    });
    const items = await screen.findAllByRole("treeitem", { name: /^temperature/ });
    fireEvent.click(items[1]);
    await screen.findByDisplayValue(`${dev.name}_plant_oven_temperature_2`);
  });

  describe("Sparkplug B entries", () => {
    it("should list a tag by its full path beside a plain entry", async () => {
      await renderRead({
        fields: [createField("temperature", "float")],
        entries: [
          createTagEntry("flow", { device: "pumpA" }),
          createTagEntry("Node Control/Rebirth"),
        ],
      });
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

    it("should focus the group of a tag added by name", async () => {
      await renderRead();
      const dialog = await openAddData();
      fireEvent.click(dialog.getByRole("button", { name: "Sparkplug B" }));
      fireEvent.click(await dialog.findByRole("button", { name: "Add a tag by name" }));
      const group = await screen.findByPlaceholderText("plant");
      await waitFor(() => expect(document.activeElement).toBe(group));
    });

    it("should add a tag by name and show its fields", async () => {
      const { draft } = await renderRead();
      const dialog = await openAddData();
      fireEvent.click(dialog.getByRole("button", { name: "Sparkplug B" }));
      fireEvent.click(await dialog.findByRole("button", { name: "Add a tag by name" }));
      await screen.findByText("Edge node");
      expect(screen.getByText("Group")).toBeTruthy();
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

    it("should add a picked tag once", async () => {
      const NODES = [{ group: "plant", edgeNode: "line1", devices: [] }];
      const TAG = {
        device: "",
        name: "flow",
        dataType: "float",
        value: "1",
        supported: true,
      };
      const { dev, draft } = await renderRead();
      const scanner = await openScanner(client, dev.rack);
      const browseTag = async () => {
        const dialog = await openAddData();
        await scanner.answer({ data: { topics: [], truncated: false } });
        fireEvent.click(dialog.getByRole("button", { name: "Sparkplug B" }));
        await scanner.answer({ data: { nodes: NODES, tags: [] } });
        await scanner.answer({ data: { nodes: [], tags: [TAG] } });
        await dialog.findByRole("checkbox", { name: "Record flow" });
        return dialog;
      };
      let dialog = await browseTag();
      fireEvent.click(dialog.getByRole("checkbox", { name: "Record flow" }));
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 tag" }));
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
      dialog = await browseTag();
      scanner.close();
      expect(dialog.getByText("In task")).toBeTruthy();
      fireEvent.click(dialog.getByRole("button", { name: "Cancel" }));
      await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
      await act(async () => {});
      expect(screen.getAllByRole("treeitem", { name: /flow/ })).toHaveLength(1);
    });

    it("should count picked fields and tags apart on the add button", async () => {
      const { dev } = await renderRead();
      const scanner = await openScanner(client, dev.rack);
      const dialog = await openAddData();
      const topic = {
        topic: TOPIC,
        payload: '{"temperature": 1}',
        fields: [{ pointer: "/temperature", type: "number", sample: "1" }],
        retained: true,
      };
      await scanner.answer({ data: { topics: [topic], truncated: false } });
      fireEvent.click(
        await dialog.findByRole("checkbox", { name: "Record temperature" }),
      );
      fireEvent.click(dialog.getByRole("button", { name: "Sparkplug B" }));
      expect(dialog.getByRole("button", { name: "Add 1 field" })).toBeTruthy();
      await scanner.answer({
        data: { nodes: [{ group: "plant", edgeNode: "line1", devices: [] }], tags: [] },
      });
      const tag = { device: "", name: "flow", dataType: "float", value: "1" };
      await scanner.answer({
        data: { nodes: [], tags: [{ ...tag, supported: true }] },
      });
      fireEvent.click(await dialog.findByRole("checkbox", { name: "Record flow" }));
      expect(
        dialog.getByRole("button", { name: "Add 1 field and 1 tag" }),
      ).toBeTruthy();
      scanner.close();
    });

    it("should lock the data type of a tag to that of its channel", async () => {
      const ch = await client.channels.create({
        name: uniqueName("mqtt_tag"),
        dataType: "string",
        virtual: true,
      });
      await renderRead({
        entries: [createTagEntry("mode", { channel: ch.key, dataType: "float64" })],
      });
      await screen.findByDisplayValue(ch.name);
      expect(screen.getByText("Set on the channel")).toBeTruthy();
      await findDialogTriggerByText("String");
    });
  });

  describe("deploying against a live Core", () => {
    it("should not deploy while the message format holds an invalid edit", async () => {
      const { container, dev } = await renderRead({
        fields: [createField("temperature", "float")],
      });
      await selectField("temperature");
      const name = await screen.findByDisplayValue("temperature");
      fireEvent.change(name, { target: { value: "" } });
      fireEvent.blur(name);
      await clickFailedDeploy(container);
      expect(await retrieveRead(dev.key)).toEqual({});
    });

    it.each([
      ["", "Topic is required"],
      ["plant/+", "Topic must not hold the wildcards + or #"],
    ])("should save the topic %j and refuse it on deploy", async (value, error) => {
      const { container, draft } = await renderRead({
        fields: [createField("temperature", "float")],
      });
      const topic = await screen.findByDisplayValue(TOPIC);
      fireEvent.change(topic, { target: { value } });
      fireEvent.blur(topic);
      await waitFor(async () =>
        expect(plainOf((await retrieveEntries(draft.key))[0]).topic).toBe(value),
      );
      await clickDeploy(container);
      const help = await screen.findByText(error);
      expect(help.closest(".lyra-input-help-text")).not.toBeNull();
      expect(help.closest(".lyra-input__item")).toBe(
        topic.closest(".lyra-input__item"),
      );
    });

    it("should put the deploy errors of a tag on its fields", async () => {
      const { container } = await renderRead({
        entries: [createTagEntry("", { group: "plant/a", edgeNode: "" })],
      });
      await screen.findByText("Edge node");
      await clickDeploy(container);
      await screen.findByText("Group must not hold /, +, or #");
      expect(screen.getByText("Edge node is required")).toBeTruthy();
      expect(screen.getByText("Tag is required")).toBeTruthy();
    });

    it("should create and bind an index channel and a data channel", async () => {
      const time = createTimeField();
      const temperature = createField("temperature", "float");
      const { container, dev, draft } = await renderRead({
        fields: [time, temperature],
        index: time.key,
      });
      const created = await deploy(container, draft.key);
      const { index, channels } = await retrieveStored(dev.key, firstMessage(draft));
      const indexCh = await client.channels.retrieve(index);
      expect(indexCh.isIndex).toBe(true);
      expect(indexCh.name).toBe(`${dev.name}_plant_oven_time`);
      const dataCh = await client.channels.retrieve(channels["/temperature"]);
      expect(dataCh.index).toBe(index);
      expect(dataCh.dataType.equals(DataType.FLOAT64)).toBe(true);
      expect(dataCh.name).toBe(`${dev.name}_plant_oven_temperature`);
      expect(plainOf(created.config.entries[0]).messages).toEqual([
        {
          message: expect.any(String),
          index: time.key,
          bindings: [
            { field: time.key, channel: index, dataType: "" },
            { field: temperature.key, channel: dataCh.key, dataType: "" },
          ],
        },
      ]);
    });

    it("should give a channel that two messages store to the first message alone", async () => {
      const time = await createTimeChannel();
      const level = await client.channels.create({
        name: uniqueName("level"),
        dataType: DataType.FLOAT64,
        index: time.key,
      });
      const entry = await createEntry({ fields: [createField("level", "float")] });
      const status = await addMessage(
        entry,
        "status",
        [createField("level", "float")],
        KIND_MATCH,
      );
      const stored = { index: time.key, channels: { "/level": level.key } };
      const { container, draft } = await renderRead({
        entries: [entry],
        properties: {
          read: {
            [`${TOPIC}#${entry.messages[0].message}`]: stored,
            [`${TOPIC}#${status.key}`]: stored,
          },
        },
      });
      const created = await deploy(container, draft.key);
      const [first, second] = plainOf(created.config.entries[0]).messages.map(
        ({ bindings }) => bindings[0].channel,
      );
      expect(first).toBe(level.key);
      expect(second).not.toBe(level.key);
      expect(second).toBeGreaterThan(0);
    });

    it("should create channels for the fields the task reads only", async () => {
      const temperature = createField("temperature", "float");
      const { container, dev, draft } = await renderRead({
        fields: [temperature, createField("pressure", "float")],
        bound: { [temperature.key]: 0 },
      });
      await deploy(container, draft.key);
      const { channels } = await retrieveStored(dev.key, firstMessage(draft));
      expect(Object.keys(channels)).toEqual(["/temperature"]);
    });

    it("should create virtual channels with no index for string fields", async () => {
      const mode = createField("mode", "string");
      const { container, dev, draft } = await renderRead({ fields: [mode] });
      await deploy(container, draft.key);
      const { index, channels } = await retrieveStored(dev.key, firstMessage(draft));
      expect(index).toBe(0);
      const ch = await client.channels.retrieve(channels["/mode"]);
      expect(ch.virtual).toBe(true);
      expect(ch.dataType.equals(DataType.STRING)).toBe(true);
    });

    it("should store a string field on the index of a number field", async () => {
      const { container, dev, draft } = await renderRead({
        fields: [createField("mode", "string"), createField("temperature", "float")],
      });
      await deploy(container, draft.key);
      const { index, channels } = await retrieveStored(dev.key, firstMessage(draft));
      const mode = await client.channels.retrieve(channels["/mode"]);
      expect(mode.virtual).toBe(false);
      expect(mode.index).toBe(index);
      expect(mode.dataType.equals(DataType.STRING)).toBe(true);
      expect((await client.channels.retrieve(channels["/temperature"])).index).toBe(
        index,
      );
    });

    it("should store a string field on the time of the payload", async () => {
      const time = createTimeField();
      const { container, dev, draft } = await renderRead({
        fields: [time, createField("mode", "string")],
        index: time.key,
      });
      await deploy(container, draft.key);
      const { index, channels } = await retrieveStored(dev.key, firstMessage(draft));
      const mode = await client.channels.retrieve(channels["/mode"]);
      expect(mode.virtual).toBe(false);
      expect(mode.index).toBe(index);
    });

    it("should store a streamed string field once its message reads a number", async () => {
      const mode = createField("mode", "string");
      const temperature = createField("temperature", "float");
      const { container, dev, draft } = await renderRead({
        fields: [mode, temperature],
        bound: { [mode.key]: 0 },
      });
      await deploy(container, draft.key);
      const [{ channel: streamed }] = await retrieveBindings(draft.key);
      expect((await client.channels.retrieve(streamed)).virtual).toBe(true);
      fireEvent.click(
        await screen.findByRole("checkbox", { name: "Read temperature" }),
      );
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toHaveLength(2),
      );
      await deploy(container, draft.key);
      const { index, channels } = await retrieveStored(dev.key, firstMessage(draft));
      const stored = await client.channels.retrieve(channels["/mode"]);
      expect(stored.virtual).toBe(false);
      expect(stored.index).toBe(index);
      const name = `${dev.name}_plant_oven_mode`;
      expect(stored.name).toBe(name);
      expect((await client.channels.retrieve(streamed)).name).toBe(`${name}_2`);
    });

    it("should show a new channel for a streamed field while its message reads a number", async () => {
      const mode = createField("mode", "string");
      const { container, dev, draft } = await renderRead({
        fields: [
          mode,
          createField("label", "string"),
          createField("temperature", "float"),
        ],
        bound: { [mode.key]: 0 },
        beforeRender: async (dev) =>
          await client.channels.create({
            name: `${dev.name}_plant_oven_label`,
            dataType: DataType.STRING,
            virtual: true,
          }),
      });
      await deploy(container, draft.key);
      const isNew = () =>
        within(screen.getByRole("treeitem", { name: /^mode/ })).queryByRole("img", {
          name: "New channel",
        });
      // The form holds the deployed channel.
      await waitFor(() => expect(isNew()).toBeNull());
      const temperature = await screen.findByRole("checkbox", {
        name: "Read temperature",
      });
      fireEvent.click(temperature);
      await waitFor(() => expect(isNew()).not.toBeNull());
      await selectField("mode");
      const name = await screen.findByDisplayValue(`${dev.name}_plant_oven_mode`);
      await waitFor(() => expect((name as HTMLInputElement).disabled).toBe(true));
      fireEvent.click(temperature);
      fireEvent.click(await screen.findByRole("checkbox", { name: "Read label" }));
      await selectField("label");
      // Only a finished plan numbers the taken name.
      await screen.findByDisplayValue(`${dev.name}_plant_oven_label_2`);
      expect(isNew()).toBeNull();
    });

    it("should keep a streamed channel when its message stops reading a number before a deploy", async () => {
      const mode = createField("mode", "string");
      const { container, dev, draft } = await renderRead({
        fields: [mode, createField("temperature", "float")],
        bound: { [mode.key]: 0 },
      });
      await deploy(container, draft.key);
      const [{ channel: streamed }] = await retrieveBindings(draft.key);
      const temperature = await screen.findByRole("checkbox", {
        name: "Read temperature",
      });
      fireEvent.click(temperature);
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toHaveLength(2),
      );
      fireEvent.click(temperature);
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toHaveLength(1),
      );
      await deploy(container, draft.key);
      expect((await retrieveBindings(draft.key))[0].channel).toBe(streamed);
      const ch = await client.channels.retrieve(streamed);
      expect(ch.virtual).toBe(true);
      expect(ch.name).toBe(`${dev.name}_plant_oven_mode`);
    });

    it("should keep storing a message that comes to read only strings", async () => {
      const temperature = createField("temperature", "float");
      const label = createField("label", "string");
      const { container, dev, draft } = await renderRead({
        fields: [createField("mode", "string"), temperature, label],
        bound: { [temperature.key]: 0 },
      });
      fireEvent.click(await screen.findByRole("checkbox", { name: "Read mode" }));
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toHaveLength(2),
      );
      await deploy(container, draft.key);
      const { index } = await retrieveStored(dev.key, firstMessage(draft));
      fireEvent.click(
        await screen.findByRole("checkbox", { name: "Read temperature" }),
      );
      fireEvent.click(await screen.findByRole("checkbox", { name: "Read label" }));
      await waitFor(async () =>
        expect((await retrieveBindings(draft.key)).map(({ field }) => field)).toEqual([
          expect.any(String),
          label.key,
        ]),
      );
      await deploy(container, draft.key);
      const { channels } = await retrieveStored(dev.key, firstMessage(draft));
      const stored = await client.channels.retrieve(channels["/label"]);
      expect(stored.virtual).toBe(false);
      expect(stored.index).toBe(index);
    });

    it("should not fall back to a stored streamed channel once its message reads a number", async () => {
      const [boundCh, storedCh] = await client.channels.create([
        { name: uniqueName("mode"), dataType: DataType.STRING, virtual: true },
        { name: uniqueName("mode"), dataType: DataType.STRING, virtual: true },
      ]);
      const mode = createField("mode", "string");
      const temperature = createField("temperature", "float");
      const { container, dev, draft } = await renderRead({
        fields: [mode, temperature],
        bound: { [mode.key]: boundCh.key, [temperature.key]: 0 },
        properties: {
          read: { [TOPIC]: { index: 0, channels: { "/mode": storedCh.key } } },
        },
      });
      await deploy(container, draft.key);
      const { index, channels } = await retrieveStored(dev.key, firstMessage(draft));
      const stored = await client.channels.retrieve(channels["/mode"]);
      expect(stored.virtual).toBe(false);
      expect(stored.index).toBe(index);
      expect((await retrieveBindings(draft.key))[0].channel).toBe(stored.key);
    });

    it("should keep a number field bound to a virtual channel", async () => {
      const temperature = createField("temperature", "float");
      const streamed = await client.channels.create({
        name: uniqueName("temperature"),
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const { container, draft } = await renderRead({
        fields: [temperature],
        bound: { [temperature.key]: streamed.key },
      });
      await deploy(container, draft.key);
      expect((await retrieveBindings(draft.key))[0].channel).toBe(streamed.key);
    });

    it("should store a streamed field whose kind changes to a number", async () => {
      const streamed = await client.channels.create({
        name: uniqueName("mode"),
        dataType: DataType.STRING,
        virtual: true,
      });
      const mode = createField("mode", "string");
      const entry = await createEntry({
        fields: [mode],
        bound: { [mode.key]: streamed.key },
      });
      const { container, draft } = await renderRead({
        entries: [entry],
        beforeRender: () => retype(entry, mode, "float"),
        properties: {
          read: { [TOPIC]: { index: 0, channels: { "/mode": streamed.key } } },
        },
      });
      const created = await deploy(container, draft.key);
      const [{ channel }] = plainOf(created.config.entries[0]).messages[0].bindings;
      const stored = await client.channels.retrieve(channel);
      expect(stored.virtual).toBe(false);
      expect(stored.dataType.equals(DataType.FLOAT64)).toBe(true);
    });

    it("should refuse a topic whose message is gone", async () => {
      const entry = await createEntry({
        fields: [createField("temperature", "float")],
      });
      const { container, dev, statuses } = await renderRead({
        entries: [entry],
        beforeRender: () => removeMessage(entry),
      });
      await screen.findByText("Message not found");
      await clickDeploy(container);
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            message: "Failed to start task",
            description: `The message format of ${TOPIC} is missing or not JSON`,
          }),
        ),
      );
      expect(await retrieveRead(dev.key)).toEqual({});
    });

    it("should refuse a topic that reads no fields", async () => {
      const { container, statuses } = await renderRead({
        fields: [createField("temperature", "float")],
        bound: {},
      });
      await clickDeploy(container);
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            message: "Failed to start task",
            description: `The task reads no fields of ${TOPIC}`,
          }),
        ),
      );
    });

    it("should time and deploy each message of a topic on its own", async () => {
      const time = createTimeField();
      const entry = await createEntry({
        fields: [createField("temperature", "float"), time],
      });
      const status = await addMessage(
        entry,
        "status",
        [createField("mode", "string")],
        KIND_MATCH,
      );
      const later = entry.messages[1];
      const { container, dev, draft } = await renderRead({ entries: [entry] });
      fireEvent.click(await screen.findByRole("treeitem", { name: TOPIC }));
      fireEvent.click(await screen.findByRole("button", { name: "Payload value" }));
      await waitFor(async () =>
        expect(plainOf((await retrieveEntries(draft.key))[0]).messages).toEqual([
          { ...entry.messages[0], index: time.key },
          later,
        ]),
      );
      const created = await deploy(container, draft.key);
      const [, deployed] = plainOf(created.config.entries[0]).messages;
      const { index, channels } = await retrieveStored(dev.key, status.key);
      expect(index).toBe(0);
      expect(deployed.index).toBeUndefined();
      expect(deployed.bindings[0].channel).toBe(channels["/mode"]);
      const ch = await client.channels.retrieve(channels["/mode"]);
      expect(ch.name).toBe(`${dev.name}_plant_oven_status_mode`);
    });

    it("should replace the stored channel of a field whose kind changed and keep its name", async () => {
      const time = await createTimeChannel();
      const [temperatureCh, countCh] = await client.channels.create([
        {
          name: uniqueName("temperature"),
          dataType: DataType.FLOAT64,
          index: time.key,
        },
        { name: uniqueName("count"), dataType: DataType.INT64, index: time.key },
      ]);
      const timeField = createTimeField();
      const temperature = createField("temperature", "float");
      const count = createField("count", "integer");
      const entry = await createEntry({
        fields: [timeField, temperature, count],
        index: timeField.key,
        bound: {
          [timeField.key]: time.key,
          [temperature.key]: temperatureCh.key,
          [count.key]: countCh.key,
        },
      });
      const { container, dev, draft } = await renderRead({
        entries: [entry],
        beforeRender: () => retype(entry, count, "float", temperature.key),
        properties: {
          read: {
            [TOPIC]: {
              index: time.key,
              channels: { "/temperature": temperatureCh.key, "/count": countCh.key },
            },
          },
        },
      });
      const name = `${dev.name}_plant_oven_count`;
      await client.channels.rename(countCh.key, name);
      const created = await deploy(container, draft.key);
      const { channels } = await retrieveStored(dev.key, firstMessage(draft));
      expect(channels["/temperature"]).toBe(temperatureCh.key);
      expect(channels["/count"]).not.toBe(countCh.key);
      const replaced = await client.channels.retrieve(channels["/count"]);
      expect(replaced.dataType.equals(DataType.FLOAT64)).toBe(true);
      expect(replaced.index).toBe(time.key);
      expect(replaced.name).toBe(name);
      expect((await client.channels.retrieve(countCh.key)).name).toBe(`${name}_2`);
      expect(plainOf(created.config.entries[0]).messages[0].bindings).toEqual([
        { field: timeField.key, channel: time.key, dataType: "" },
        { field: temperature.key, channel: temperatureCh.key, dataType: "" },
        { field: count.key, channel: replaced.key, dataType: "" },
      ]);
    });

    it("should not reuse a stored channel that another field reads", async () => {
      const time = await createTimeChannel();
      const ch = await client.channels.create({
        name: uniqueName("temperature"),
        dataType: DataType.FLOAT64,
        index: time.key,
      });
      const temperature = createField("temperature", "float");
      const entry = await createEntry({
        fields: [temperature],
        bound: { [temperature.key]: ch.key },
      });
      const { container, dev, draft, statuses } = await renderRead({
        entries: [entry],
        beforeRender: () => repoint(entry, temperature, "oven"),
        properties: {
          read: { [TOPIC]: { index: time.key, channels: { "/temperature": ch.key } } },
        },
      });
      await screen.findByRole("checkbox", { name: "Read oven" });
      await addField(dev.rack, TOPIC, "temperature");
      const added = await waitFor(async () => {
        const bindings = await retrieveBindings(draft.key);
        expect(bindings).toEqual([
          { field: temperature.key, channel: ch.key, dataType: "" },
          { field: expect.any(String), channel: 0, dataType: "" },
        ]);
        return bindings[1].field;
      });
      expect(statuses).toEqual([]);
      const created = await deploy(container, draft.key);
      const { channels } = await retrieveStored(dev.key, firstMessage(draft));
      expect(channels["/temperature"]).not.toBe(ch.key);
      expect(plainOf(created.config.entries[0]).messages[0].bindings).toEqual([
        { field: temperature.key, channel: ch.key, dataType: "" },
        { field: added, channel: channels["/temperature"], dataType: "" },
      ]);
    });

    it("should keep the stored index of a message after the message before it is removed", async () => {
      const [before, own] = await Promise.all([
        createTimeChannel(),
        createTimeChannel(),
      ]);
      const valueCh = await client.channels.create({
        name: uniqueName("value"),
        dataType: DataType.FLOAT64,
        index: own.key,
      });
      const entry = await createEntry({
        fields: [createField("temperature", "float")],
      });
      const time = createTimeField();
      const value = createField("value", "float");
      const status = await addMessage(entry, "status", [time, value], KIND_MATCH);
      const bindings = [
        { field: time.key, channel: own.key, dataType: "" },
        { field: value.key, channel: valueCh.key, dataType: "" },
      ];
      entry.messages = [{ message: status.key, index: time.key, bindings }];
      const { container, draft } = await renderRead({
        entries: [entry],
        properties: {
          read: {
            [TOPIC]: { index: before.key, channels: {} },
            [`${TOPIC}#${status.key}`]: {
              index: own.key,
              channels: { "/value": valueCh.key },
            },
          },
        },
      });
      const created = await deploy(container, draft.key);
      expect(plainOf(created.config.entries[0]).messages[0].bindings).toEqual(bindings);
    });

    it("should move the channels stored under a topic to the message that binds them", async () => {
      const time = await createTimeChannel();
      const ch = await client.channels.create({
        name: uniqueName("temperature"),
        dataType: DataType.FLOAT64,
        index: time.key,
      });
      const temperature = createField("temperature", "float");
      const entry = await createEntry({
        fields: [temperature],
        bound: { [temperature.key]: ch.key },
      });
      const stored = { index: time.key, channels: { "/temperature": ch.key } };
      const { container, dev, draft } = await renderRead({
        entries: [entry],
        properties: { read: { [TOPIC]: stored } },
      });
      await deploy(container, draft.key);
      expect(await retrieveRead(dev.key)).toEqual({
        [`${TOPIC}#${entry.messages[0].message}`]: stored,
      });
    });

    it("should create no channel when a later message fails to plan", async () => {
      const entry = await createEntry({
        fields: [createField("temperature", "float")],
      });
      await addMessage(entry, "later", [createField("rejected", "float")], {
        ...KIND_MATCH,
        value: "later",
      });
      vi.spyOn(console, "error").mockImplementation(() => {});
      const { container, dev, statuses } = await renderRead({ entries: [entry] });
      // Only the plan of the later message names a channel after its field.
      const reject = rejectRequests(
        client,
        (target, req) =>
          target === "/channel/retrieve" && JSON.stringify(req).includes("rejected"),
        new Error("plan failed"),
      );
      try {
        await clickFailedDeploy(container);
      } finally {
        reject.mockRestore();
      }
      expect(statuses).toContainEqual(START_FAILURE);
      const made = await client.channels.retrieve({ searchTerm: dev.name });
      expect(
        made.map(({ name }) => name).filter((n) => n.startsWith(dev.name)),
      ).toEqual([]);
    });

    it("should keep the channels stored under a topic when an earlier message fails", async () => {
      const time = await createTimeChannel();
      const createFloat = async (name: string) =>
        await client.channels.create({
          name: uniqueName(name),
          dataType: DataType.FLOAT64,
          index: time.key,
        });
      const value = await createFloat("value");
      const notIndex = await createFloat("not_index");
      const entry = await createEntry({
        fields: [createField("temperature", "float")],
      });
      const match = (value: string): icd.JSONMatch => ({ ...KIND_MATCH, value });
      const failing = await addMessage(
        entry,
        "failing",
        [createField("pressure", "float")],
        match("failing"),
      );
      const valueField = createField("value", "float");
      await addMessage(entry, "owner", [valueField], match("owner"));
      entry.messages[2].bindings = [
        { field: valueField.key, channel: value.key, dataType: "" },
      ];
      const legacy = { index: time.key, channels: { "/value": value.key } };
      const { container, dev, statuses } = await renderRead({
        entries: [entry],
        properties: {
          read: {
            [TOPIC]: legacy,
            [`${TOPIC}#${failing.key}`]: { index: notIndex.key, channels: {} },
          },
        },
      });
      await clickDeploy(container);
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({ message: "Failed to start task" }),
        ),
      );
      expect((await retrieveRead(dev.key))[TOPIC]).toEqual(legacy);
    });

    it("should give a new entry the stored channel that only a disabled entry binds", async () => {
      const time = await createTimeChannel();
      const ch = await client.channels.create({
        name: uniqueName("temperature"),
        dataType: DataType.FLOAT64,
        index: time.key,
      });
      const temperature = createField("temperature", "float");
      const disabled = await createEntry({
        fields: [temperature],
        bound: { [temperature.key]: ch.key },
      });
      disabled.disabled = true;
      const entry = {
        ...disabled,
        key: uuid.create(),
        disabled: false,
        messages: disabled.messages.map((read) => ({
          ...read,
          bindings: read.bindings.map((b) => ({ ...b, channel: 0 })),
        })),
      };
      const { message } = entry.messages[0];
      const { container, draft } = await renderRead({
        entries: [disabled, entry],
        properties: {
          read: {
            [`${TOPIC}#${message}`]: {
              index: time.key,
              channels: { "/temperature": ch.key },
            },
          },
        },
      });
      const created = await deploy(container, draft.key);
      expect(plainOf(created.config.entries[1]).messages[0].bindings[0].channel).toBe(
        ch.key,
      );
    });

    it("should create an index and a data channel for each tag", async () => {
      const { container, dev, draft } = await renderRead({
        entries: [
          createTagEntry("zone 1/temperature", { device: "ovenA" }),
          createTagEntry("flow", { dataType: "float32" }),
        ],
      });
      const created = await deploy(container, draft.key);
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      const [deviceTag, nodeTag] = created.config.entries.map(sparkplugOf);
      const dataCh = await client.channels.retrieve(deviceTag.channel);
      const flowCh = await client.channels.retrieve(nodeTag.channel);
      expect(flowCh.index).not.toBe(dataCh.index);
      expect(updated.properties.read).toEqual({
        "spBv1.0/plant/line1/ovenA/zone 1/temperature": {
          index: dataCh.index,
          channels: { "": deviceTag.channel },
        },
        "spBv1.0/plant/line1//flow": {
          index: flowCh.index,
          channels: { "": nodeTag.channel },
        },
      });
      const prefix = `${dev.name}_plant_line1_ovenA_zone_1_temperature`;
      expect(dataCh.name).toBe(prefix);
      expect((await client.channels.retrieve(dataCh.index)).name).toBe(
        `${prefix}_time`,
      );
      expect(flowCh.name).toBe(`${dev.name}_plant_line1_flow`);
      expect(flowCh.dataType.toString()).toBe("float32");
    });

    it("should number a typed name that an earlier tag takes", async () => {
      const name = uniqueName("temperature");
      const { container, draft } = await renderRead({
        entries: [createTagEntry("flow", { name }), createTagEntry("level", { name })],
      });
      const created = await deploy(container, draft.key);
      const names = await Promise.all(
        created.config.entries.map(
          async (e) => (await client.channels.retrieve(sparkplugOf(e).channel)).name,
        ),
      );
      expect(names).toEqual([name, `${name}_2`]);
    });

    it("should reuse the channel of a tag that the device stores", async () => {
      const first = await renderRead({ entries: [createTagEntry("flow")] });
      const deployed = await deploy(first.container, first.draft.key);
      const [stored] = deployed.config.entries.map(sparkplugOf);
      first.unmount();
      const config = MQTT.Task.READ_SCHEMAS.config.parse({
        device: deployed.config.device,
        entries: [createTagEntry("flow")],
      });
      const second = await client.tasks.create(
        { name: "MQTT read task", type: MQTT.Task.READ_TYPE, config },
        MQTT.Task.READ_SCHEMAS,
      );
      const rendered = await renderTaskFormTab(MQTT.Task.Read, {
        client,
        taskKey: second.key,
      });
      const created = await deploy(rendered.container, second.key);
      expect(sparkplugOf(created.config.entries[0]).channel).toBe(stored.channel);
    });
  });
});

describe("plotChannels", () => {
  it("should find the bound channels of enabled plain entries and enabled Sparkplug tags", () => {
    const plotChannels = MQTT.Task.INTEGRATION.plotChannels?.[MQTT.Task.READ_TYPE];
    assertDefined(plotChannels, "read tasks have no channels to plot");
    const plain = (disabled: boolean, ...channels: number[][]) => ({
      type: "plain",
      disabled,
      icd: uuid.create(),
      messages: channels.map((keys) => ({
        message: uuid.create(),
        bindings: keys.map((channel) => ({ field: uuid.create(), channel })),
      })),
    });
    const config = {
      entries: [
        plain(false, [1, 0], [2]),
        plain(true, [3]),
        { type: "sparkplug", channel: 4 },
        { type: "sparkplug", channel: 5, disabled: true },
      ],
    };
    expect(plotChannels(config)).toEqual([1, 2, 4]);
  });
});
