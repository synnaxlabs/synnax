// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type device,
  icd,
  kafka,
  type resource,
  type Synnax,
  task,
} from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { array, DataType, deep, type payload, uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { Kafka } from "@/feature/kafka";
import { createKafkaDevice } from "@/feature/kafka/testutil";
import { Device } from "@/platform/device";
import { startRackHeartbeat } from "@/platform/device/testutil";
import {
  answerCommand,
  answerNextCommand,
  awaitCommand,
  clickDeploy,
  type CommandAnswer,
  deployAndAwaitTask,
  renderTaskFormTab,
} from "@/platform/task/testutil";
import { assertDefined, findDialogTriggerByText, uniqueName } from "@/testutil";

const client = createTestClient();

const TOPIC = "oven";

const createField = (name: string, kind: payload.Kind): icd.JSONField =>
  icd.jsonFieldZ.parse({ name, pointer: `/${name}`, type: kind });

const createTimeField = (): icd.JSONField =>
  icd.jsonFieldZ.parse({
    name: "time",
    pointer: "/time",
    type: "timestamp",
    timeFormat: "unix_ms",
  });

interface RecordParams {
  /** The fields of the record's message format. */
  fields: icd.JSONField[];
  recordKey?: string;
  index?: string;
  /** The channel of each field the task reads, by field key. All fields by default. */
  bound?: Record<string, number>;
}

interface RenderParams extends Partial<RecordParams> {
  /** The records of TOPIC that the task reads after the one that the fields make. */
  records?: Kafka.Task.ReadRecord[];
  /** The topics that the task reads after TOPIC. */
  topics?: Kafka.Task.ReadTopic[];
  /** The task holds no topics, as a new task does. */
  empty?: boolean;
  /**
   * Runs after the task is saved, before the form opens. The Core saves no task that an
   * ICD edit here would make invalid.
   */
  beforeRender?: (dev: device.Device) => Promise<unknown>;
  /** The client the form runs as. */
  as?: Synnax;
}

/** Creates an ICD that holds one message, and a record that reads it. */
const createRecord = async ({
  fields,
  recordKey = "",
  index,
  bound = Object.fromEntries(fields.map(({ key }) => [key, 0])),
}: RecordParams): Promise<Kafka.Task.ReadRecord> => {
  const message: icd.Message = {
    key: uuid.create(),
    name: recordKey === "" ? "Any key" : recordKey,
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
  return kafka.readRecordZ.parse({
    icd: format.key,
    recordKey,
    messages: [{ message: message.key, index, bindings }],
  });
};

// The form body renders against a configured cluster only.
const renderRead = async ({
  fields,
  recordKey,
  index,
  bound,
  records = [],
  topics = [],
  empty = false,
  beforeRender,
  as,
}: RenderParams = {}) => {
  const dev = await createKafkaDevice(client);
  const first =
    fields == null ? [] : [await createRecord({ fields, recordKey, index, bound })];
  const config = Kafka.Task.READ_SCHEMAS.config.parse({
    device: dev.key,
    topics: empty ? [] : [{ name: TOPIC, records: [...first, ...records] }, ...topics],
  });
  const draft = await client.tasks.create(
    { name: "Kafka read task", type: Kafka.Task.READ_TYPE, config },
    Kafka.Task.READ_SCHEMAS,
  );
  await beforeRender?.(dev);
  const statuses: Status.NotificationSpec[] = [];
  const result = await renderTaskFormTab(Kafka.Task.Read, {
    client,
    as,
    taskKey: draft.key,
    onStatuses: (next) => {
      statuses.length = 0;
      statuses.push(...next);
    },
  });
  return { ...result, dev, draft, statuses };
};

/** @returns the records of every topic of the task, in order. */
const retrieveRecords = async (key: string) =>
  (
    await client.tasks.retrieve({ key, schemas: Kafka.Task.READ_SCHEMAS })
  ).config.topics.flatMap(({ records }) => records);

/** @returns the bindings of the first message of the first record. */
const retrieveBindings = async (key: string) =>
  (await retrieveRecords(key))[0].messages[0].bindings;

const deploy = async (container: HTMLElement, key: string) =>
  await deployAndAwaitTask(client, container, key, Kafka.Task.READ_SCHEMAS);

/** Removes the message of the record from its ICD. */
const removeMessage = async ({ icd: key, messages }: Kafka.Task.ReadRecord) =>
  await client.icds.dispatch(
    key,
    icd.remove({ path: ["messages"], key: messages[0].message }),
  );

/** Gives a field of the record's message a new kind, in the ICD. */
const retype = async (
  { icd: key, messages }: Kafka.Task.ReadRecord,
  field: icd.JSONField,
  kind: payload.Kind,
) => {
  const path = ["messages", messages[0].message, "payload", "fields"];
  await client.icds.dispatch(key, [
    icd.remove({ path, key: field.key }),
    icd.insert({ path, value: { ...field, type: kind } }),
  ]);
};

/** @returns a client with the policies of the Owner role, which reads ICDs only. */
const createICDReaderClient = async (): Promise<Synnax> => {
  const owner = (await client.access.roles.retrieve({})).find(
    ({ name }) => name === "Owner",
  );
  assertDefined(owner, "no built-in role named Owner");
  const policies = await client.access.policies.retrieve({ roles: [owner.key] });
  const isICD = ({ type }: resource.ID) => type === icd.TYPE_RESOURCE_ID.type;
  const reader = await createTestClientWithPolicy(client, [
    ...policies.map(({ objects, actions }) => ({
      name: uniqueName("policy"),
      objects: objects.filter((o) => !isICD(o)),
      actions,
    })),
    {
      name: uniqueName("policy"),
      objects: [icd.TYPE_RESOURCE_ID],
      actions: ["retrieve"],
    },
  ]);
  onTestFinished(async () => await reader.close());
  return reader;
};

/**
 * Adds a field to the message of a record, in a form whose client reads ICDs only.
 * @param onReader - Runs on the client before the form renders.
 */
const addFieldAsICDReader = async () => {
  const temperature = createField("temperature", "float");
  const reader = await createICDReaderClient();
  const { draft, statuses } = await renderRead({ fields: [temperature], as: reader });
  const [{ icd: icdKey, messages }] = await retrieveRecords(draft.key);
  fireEvent.click(await screen.findByRole("treeitem", { name: /JSON.*Any key/ }));
  fireEvent.click(await screen.findByRole("button", { name: "Add field" }));
  return { draft, statuses, temperature, icdKey, bindings: messages[0].bindings };
};

/** Expects the ICD and the read of an added field to hold what the Core holds. */
const expectFieldUndone = async ({
  draft,
  temperature,
  icdKey,
  bindings,
}: Awaited<ReturnType<typeof addFieldAsICDReader>>) => {
  await waitFor(() =>
    expect(screen.queryByRole("treeitem", { name: /^Field 2/ })).toBeNull(),
  );
  expect(jsonFieldsOf(await client.icds.retrieve(icdKey))).toEqual([temperature]);
  expect(await retrieveBindings(draft.key)).toEqual(bindings);
  // A read that still held the refused field would be refused again.
  fireEvent.click(await screen.findByRole("checkbox", { name: "Read temperature" }));
  await waitFor(async () => expect(await retrieveBindings(draft.key)).toEqual([]));
};

/** @returns the fields of the first message of an ICD. */
const jsonFieldsOf = ({ messages: [first] }: icd.ICD): icd.JSONField[] =>
  first.payload.format === "json" ? first.payload.fields : [];

// The details of a field show once its item in the tree is selected.
const selectField = async (name: string) =>
  fireEvent.click(
    await screen.findByRole("treeitem", { name: new RegExp(`^${name}`) }),
  );

/** Adds a message to the ICD of the record after its last, and reads each field. */
const addMessage = async (
  rec: Kafka.Task.ReadRecord,
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
    rec.icd,
    icd.insert({
      path: ["messages"],
      value: message,
      after: rec.messages.at(-1)?.message,
    }),
  );
  const bindings = fields.map(({ key }) => ({ field: key, channel: 0, dataType: "" }));
  rec.messages.push(kafka.readMessageZ.parse({ message: message.key, bindings }));
  return message;
};

const KIND_MATCH: icd.JSONMatch = { type: "string", pointer: "/kind", value: "status" };

/** @returns the key under which the device stores the first message of rec. */
const storedKeyOf = ({ recordKey, messages }: Kafka.Task.ReadRecord) =>
  `${TOPIC}#${recordKey}#${messages[0].message}`;

/** @returns the channels that the device stores for the first message of rec. */
const retrieveStored = async (key: string, rec: Kafka.Task.ReadRecord) =>
  (await client.devices.retrieve({ key, schemas: Kafka.Device.SCHEMAS })).properties
    .read[storedKeyOf(rec)];

/** Stores channels on the device for the first message of rec, as a deploy does. */
const storeChannels = async (
  key: string,
  rec: Kafka.Task.ReadRecord,
  props: { index: number; channels: Record<string, number> },
) => {
  const dev = await client.devices.retrieve({ key, schemas: Kafka.Device.SCHEMAS });
  const before = deep.copy(dev.properties);
  dev.properties.read[storedKeyOf(rec)] = props;
  await Device.setProperties(client, dev, before, Kafka.Device.SCHEMAS);
};

const createTimeChannel = async () =>
  await client.channels.create({
    name: uniqueName("time"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });

const createInferredField = (
  pointer: string,
  dataType: string,
  type: "number" | "boolean" | "string",
  timeFormat?: payload.TimeFormat,
) => ({ pointer, dataType, type, timeFormat, enum: [], count: 2, mismatched: 0 });

/** Two keys of values that hold a time, a temperature, and a state. */
const OVEN_SHAPE = {
  sources: ["oven-1", "oven-2"],
  fields: [
    createInferredField("/time", "timestamp", "number", "unix_ms"),
    createInferredField("/temperature", "float64", "number"),
    createInferredField("/state", "string", "string"),
  ],
  index: 0,
  count: 2,
};

/** Answers the next command of the scan task of the rack of dev. */
const answerScan = async (
  dev: device.Device,
  answer: Omit<CommandAnswer, "type"> = {},
) => {
  const rck = await client.racks.retrieve({ key: dev.rack });
  await startRackHeartbeat(client, rck);
  const scan = await rck.createTask({
    name: uniqueName("kafka_scan"),
    type: Kafka.Task.SCAN_TYPE,
    config: {},
  });
  return await answerNextCommand(client, scan, answer);
};

/**
 * Opens the Add data dialog of the task on dev, and answers its sample command.
 * @param topic - The topic that the dialog samples first.
 * @returns queries scoped to the dialog.
 */
const openAddData = async (
  dev: device.Device,
  answer: Omit<CommandAnswer, "type"> = {},
  topic = TOPIC,
) => {
  const { answered } = await answerScan(dev, answer);
  fireEvent.click(await screen.findByRole("button", { name: "Add data" }));
  const dialog = within(await screen.findByRole("dialog"));
  const cmd = await answered;
  expect(cmd.args).toEqual({ device: dev.key, topics: [topic] });
  return dialog;
};

/** @returns the data of a sample of a topic that read values of the shapes. */
const sampleOf = (...shapes: object[]) => sampleOfTopic(TOPIC, ...shapes);

const sampleOfTopic = (topic: string, ...shapes: object[]) => ({
  topics: [{ topic, records: [], shapes, error: "" }],
  truncated: false,
});

/** @returns the topics of the task. */
const retrieveTopics = async (key: string) =>
  (await client.tasks.retrieve({ key, schemas: Kafka.Task.READ_SCHEMAS })).config
    .topics;

/** @returns a topic named name that reads one record of fields. */
const createTopic = async (name: string, fields: icd.JSONField[]) =>
  kafka.readTopicZ.parse({ name, records: [await createRecord({ fields })] });

describe("Kafka read form", () => {
  describe("topics", () => {
    it("should show each topic above its records", async () => {
      await renderRead({
        fields: [createField("temperature", "float")],
        topics: [await createTopic("plant", [createField("pressure", "float")])],
      });
      const level = async (name: RegExp) =>
        (await screen.findByRole("treeitem", { name })).getAttribute("aria-level");
      expect(await level(/Topic.*oven/)).toBe("1");
      expect(await level(/Topic.*plant/)).toBe("1");
      expect(await level(/^temperature/)).toBe("3");
      expect(await level(/^pressure/)).toBe("3");
      expect(screen.getAllByRole("treeitem", { name: /JSON.*Any key/ })).toHaveLength(
        2,
      );
    });

    it("should show no topics for a new task", async () => {
      await renderRead({ empty: true });
      await screen.findByText("No topics");
      await screen.findByText("Add data to read the records of a topic");
    });

    it("should add the picks of an entered topic as a new topic", async () => {
      const { draft, dev } = await renderRead({ empty: true });
      await screen.findByText("No topics");
      const { answered } = await answerScan(dev, {
        data: sampleOfTopic("plant", OVEN_SHAPE),
      });
      fireEvent.click(await screen.findByRole("button", { name: "Add data" }));
      const dialog = within(await screen.findByRole("dialog"));
      await dialog.findByText("Enter a topic to read its records.");
      expect(dialog.queryByRole("button", { name: "Add an empty record" })).toBeNull();
      const input = dialog.getByRole("textbox", { name: "Topic" });
      fireEvent.change(input, { target: { value: "plant" } });
      fireEvent.blur(input);
      expect((await answered).args).toEqual({ device: dev.key, topics: ["plant"] });
      fireEvent.click(await dialog.findByRole("button", { name: "Select all" }));
      fireEvent.click(dialog.getByRole("button", { name: "Add 2 fields" }));
      const topics = await waitFor(async () => {
        const next = await retrieveTopics(draft.key);
        expect(next).toHaveLength(1);
        return next;
      });
      expect(topics[0].name).toBe("plant");
      expect(topics[0].records.map(({ recordKey }) => recordKey)).toEqual([
        "oven-1",
        "oven-2",
      ]);
      await screen.findByRole("treeitem", { name: /Topic.*plant/ });
    });

    it("should hide the last topic's sample while it reads a new one", async () => {
      const { dev } = await renderRead({ empty: true });
      await screen.findByText("No topics");
      const { answered } = await answerScan(dev, {
        data: sampleOfTopic("plant", OVEN_SHAPE),
      });
      fireEvent.click(await screen.findByRole("button", { name: "Add data" }));
      const dialog = within(await screen.findByRole("dialog"));
      const input = dialog.getByRole("textbox", { name: "Topic" });
      fireEvent.change(input, { target: { value: "plant" } });
      fireEvent.blur(input);
      await answered;
      await dialog.findByRole("button", { name: "Select all" });
      const scan = await client.tasks.retrieve({
        type: Kafka.Task.SCAN_TYPE,
        rack: dev.rack,
        one: true,
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      fireEvent.change(input, { target: { value: "boiler" } });
      fireEvent.blur(input);
      const cmd = await awaitCommand(streamer, scan.key).finally(() =>
        streamer.close(),
      );
      expect(dialog.queryByRole("button", { name: "Select all" })).toBeNull();
      expect(dialog.queryAllByText(/oven-1/)).toHaveLength(0);
      const boiler = { ...OVEN_SHAPE, sources: ["boiler-1"] };
      await answerCommand(client, scan, cmd, {
        data: sampleOfTopic("boiler", boiler),
      });
      await dialog.findByRole("button", { name: "Select all" });
      await dialog.findAllByText(/boiler-1/);
    });

    it("should open Add data on the topic of the selected record", async () => {
      const { dev } = await renderRead({
        fields: [createField("temperature", "float")],
        topics: [await createTopic("plant", [createField("pressure", "float")])],
      });
      fireEvent.click(await screen.findByRole("treeitem", { name: /^pressure/ }));
      const dialog = await openAddData(dev, { data: sampleOfTopic("plant") }, "plant");
      await dialog.findByText(/holds no JSON records/);
    });

    it("should rename a topic in its details", async () => {
      const { draft } = await renderRead({
        fields: [createField("temperature", "float")],
      });
      fireEvent.click(await screen.findByRole("treeitem", { name: /Topic.*oven/ }));
      const input = await screen.findByPlaceholderText("sensor-readings");
      fireEvent.change(input, { target: { value: "kiln" } });
      fireEvent.blur(input);
      await waitFor(async () =>
        expect((await retrieveTopics(draft.key))[0].name).toBe("kiln"),
      );
      await screen.findByRole("treeitem", { name: /Topic.*kiln/ });
    });

    it("should remove a topic with its records", async () => {
      const { draft } = await renderRead({
        fields: [createField("temperature", "float")],
        topics: [await createTopic("plant", [createField("pressure", "float")])],
      });
      fireEvent.contextMenu(
        await screen.findByRole("treeitem", { name: /Topic.*oven/ }),
      );
      fireEvent.click(await screen.findByRole("menuitem", { name: "Remove" }));
      await waitFor(async () =>
        expect((await retrieveTopics(draft.key)).map(({ name }) => name)).toEqual([
          "plant",
        ]),
      );
      expect(screen.queryByRole("treeitem", { name: /^temperature/ })).toBeNull();
    });

    it("should not offer to remove a topic chosen with a record", async () => {
      await renderRead({ fields: [createField("temperature", "float")] });
      fireEvent.click(await screen.findByRole("treeitem", { name: /Topic.*oven/ }));
      fireEvent.keyDown(window, { code: "Shift" });
      fireEvent.click(screen.getByRole("treeitem", { name: /JSON.*Any key/ }), {
        shiftKey: true,
      });
      fireEvent.keyUp(window, { code: "Shift" });
      fireEvent.contextMenu(screen.getByRole("treeitem", { name: /JSON.*Any key/ }));
      await screen.findByText("Reload Console");
      expect(screen.queryByText("Remove")).toBeNull();
    });
  });

  describe("records", () => {
    it("should create an ICD named after the task for the first record", async () => {
      const { draft, dev } = await renderRead();
      await screen.findByRole("treeitem", { name: /Topic.*oven/ });
      const dialog = await openAddData(dev, { data: sampleOf() });
      await dialog.findByText(/holds no JSON records/);
      fireEvent.click(dialog.getByRole("button", { name: "Add an empty record" }));
      const [rec] = await waitFor(async () => {
        const records = await retrieveRecords(draft.key);
        expect(records).toHaveLength(1);
        return records;
      });
      expect(rec).toMatchObject({ recordKey: "", disabled: false });
      const format = await client.icds.retrieve(rec.icd);
      expect(format.name).toBe("Kafka read task");
      expect(format.messages).toMatchObject([
        {
          key: rec.messages[0].message,
          name: "Any key",
          payload: { format: "json", fields: [] },
        },
      ]);
      await screen.findByPlaceholderText("Any key");
    });

    it("should add a later record to the ICD of the first", async () => {
      const { draft, dev } = await renderRead({
        fields: [createField("temperature", "float")],
      });
      await screen.findByRole("treeitem", { name: /^temperature/ });
      const dialog = await openAddData(dev, { data: sampleOf() });
      fireEvent.click(dialog.getByRole("button", { name: "Add an empty record" }));
      const records = await waitFor(async () => {
        const next = await retrieveRecords(draft.key);
        expect(next).toHaveLength(2);
        return next;
      });
      expect(records[1].icd).toBe(records[0].icd);
      const { messages } = await client.icds.retrieve(records[0].icd);
      expect(messages.map(({ name }) => name)).toEqual(["Any key", "Any key 2"]);
      expect(records[1].messages[0].message).toBe(messages[1].key);
    });

    it("should open one Add data dialog for a double click", async () => {
      await renderRead();
      const button = await screen.findByRole("button", { name: "Add data" });
      fireEvent.click(button);
      fireEvent.click(button);
      await screen.findByRole("dialog");
      expect(screen.getAllByRole("dialog")).toHaveLength(1);
    });

    it("should name a record by its record key", async () => {
      const { draft } = await renderRead({
        fields: [createField("temperature", "float")],
      });
      fireEvent.click(await screen.findByRole("treeitem", { name: /JSON.*Any key/ }));
      const input = await screen.findByPlaceholderText("Any key");
      fireEvent.change(input, { target: { value: "pump" } });
      fireEvent.blur(input);
      await waitFor(async () =>
        expect((await retrieveRecords(draft.key))[0].recordKey).toBe("pump"),
      );
      await screen.findByRole("treeitem", { name: /JSON.*pump/ });
    });

    it("should disable and enable a record through the context menu", async () => {
      const { draft } = await renderRead({
        fields: [createField("temperature", "float")],
      });
      const item = () => screen.findByRole("treeitem", { name: /JSON.*Any key/ });
      fireEvent.contextMenu(await item());
      const disable = await screen.findByText("Disable");
      expect(screen.queryByText("Enable")).toBeNull();
      fireEvent.click(disable);
      await waitFor(async () =>
        expect((await retrieveRecords(draft.key))[0].disabled).toBe(true),
      );
      fireEvent.contextMenu(await item());
      fireEvent.click(await screen.findByText("Enable"));
      await waitFor(async () =>
        expect((await retrieveRecords(draft.key))[0].disabled).toBe(false),
      );
    });

    it.each([
      { name: "gone", note: "Message not found", change: removeMessage },
      {
        name: "not JSON",
        note: "The format of this message is not JSON. Open it to change its format.",
        change: async ({ icd: key, messages }: Kafka.Task.ReadRecord) =>
          await client.icds.dispatch(
            key,
            icd.set({
              path: ["messages", messages[0].message, "payload"],
              value: { format: "text", delimiter: ",", prefix: "", fields: [] },
            }),
          ),
      },
    ])(
      "should not offer Add field for a record whose message is $name",
      async ({ note, change }) => {
        const rec = await createRecord({
          fields: [createField("temperature", "float")],
        });
        await renderRead({ records: [rec], beforeRender: () => change(rec) });
        await screen.findByText(note);
        expect(screen.queryByRole("button", { name: "Add field" })).toBeNull();
        fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /Any key/ }));
        await screen.findByRole("menuitem", { name: "Add message" });
        expect(screen.queryByRole("menuitem", { name: "Add field" })).toBeNull();
      },
    );
  });

  describe("adding data", () => {
    it("should add a message of the picked fields and a record for each key", async () => {
      const { draft, dev } = await renderRead();
      const dialog = await openAddData(dev, { data: sampleOf(OVEN_SHAPE) });
      fireEvent.click(
        await dialog.findByRole("checkbox", { name: "Record /temperature" }),
      );
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 field" }));
      const records = await waitFor(async () => {
        const next = await retrieveRecords(draft.key);
        expect(next).toHaveLength(2);
        return next;
      });
      expect(records.map(({ recordKey }) => recordKey)).toEqual(["oven-1", "oven-2"]);
      const format = await client.icds.retrieve(records[0].icd);
      expect(format.name).toBe("Kafka read task");
      expect(format.messages).toHaveLength(1);
      const [message] = format.messages;
      expect(message).toMatchObject({
        name: "oven-1",
        payload: {
          format: "json",
          fields: [
            {
              name: "time",
              pointer: "/time",
              type: "timestamp",
              timeFormat: "unix_ms",
            },
            { name: "temperature", pointer: "/temperature", type: "float" },
          ],
        },
      });
      const fields = message.payload.format === "json" ? message.payload.fields : [];
      for (const rec of records)
        expect(rec).toMatchObject({
          icd: format.key,
          messages: [
            {
              message: message.key,
              index: fields[0].key,
              bindings: fields.map(({ key }) => ({ field: key, channel: 0 })),
            },
          ],
        });
    });

    it("should show the hint above the fields and count the picks below", async () => {
      const { dev } = await renderRead();
      const dialog = await openAddData(dev, { data: sampleOf(OVEN_SHAPE) });
      await dialog.findByText(
        "Select the fields to read. Records that hold the same fields share a message.",
      );
      expect(dialog.queryByText(/selected$/)).toBeNull();
      fireEvent.click(
        await dialog.findByRole("checkbox", { name: "Record /temperature" }),
      );
      await dialog.findByText("1 field selected");
    });

    it("should pick a string field with the numbers of its record", async () => {
      const { dev } = await renderRead();
      const dialog = await openAddData(dev, { data: sampleOf(OVEN_SHAPE) });
      const state = await dialog.findByRole("checkbox", { name: "Record /state" });
      expect(state).toHaveProperty("disabled", false);
      fireEvent.click(dialog.getByRole("button", { name: "Select all" }));
      expect(state).toHaveProperty("checked", true);
      dialog.getByRole("button", { name: "Add 2 fields" });
    });

    it("should give each number field the type of its inferred data type", async () => {
      const { draft, dev } = await renderRead();
      const pump = {
        sources: ["pump"],
        fields: [
          createInferredField("/count", "int64", "number"),
          createInferredField("/id", "uint64", "number"),
          createInferredField("/level", "float64", "number"),
        ],
        index: -1,
        count: 2,
      };
      const dialog = await openAddData(dev, { data: sampleOf(pump) });
      fireEvent.click(await dialog.findByRole("button", { name: "Select all" }));
      fireEvent.click(dialog.getByRole("button", { name: "Add 3 fields" }));
      const [rec] = await waitFor(async () => {
        const next = await retrieveRecords(draft.key);
        expect(next).toHaveLength(1);
        return next;
      });
      const { messages } = await client.icds.retrieve(rec.icd);
      expect(messages[0].payload).toMatchObject({
        fields: [
          { pointer: "/count", type: "integer" },
          { pointer: "/id", type: "unsigned" },
          { pointer: "/level", type: "float" },
        ],
      });
    });

    it("should name fields apart when their pointers give one name", async () => {
      const { draft, dev } = await renderRead();
      const pump = {
        sources: ["pump"],
        fields: [
          createInferredField("/a/b", "float64", "number"),
          createInferredField("/a_b", "float64", "number"),
        ],
        index: -1,
        count: 2,
      };
      const dialog = await openAddData(dev, { data: sampleOf(pump) });
      fireEvent.click(await dialog.findByRole("button", { name: "Select all" }));
      fireEvent.click(dialog.getByRole("button", { name: "Add 2 fields" }));
      const [rec] = await waitFor(async () => {
        const next = await retrieveRecords(draft.key);
        expect(next).toHaveLength(1);
        return next;
      });
      const { messages } = await client.icds.retrieve(rec.icd);
      expect(messages[0].payload).toMatchObject({
        fields: [{ name: "a_b" }, { name: "a_b_2" }],
      });
    });

    it("should move the picks onto a new sample on Listen again", async () => {
      const { draft, dev } = await renderRead();
      const pump = {
        sources: ["pump-1"],
        fields: [createInferredField("/pressure", "int64", "number")],
        index: -1,
        count: 2,
      };
      const dialog = await openAddData(dev, { data: sampleOf(OVEN_SHAPE, pump) });
      fireEvent.click(await dialog.findByText("pump-1"));
      fireEvent.click(await dialog.findByRole("button", { name: "Select all" }));
      const scan = await client.tasks.retrieve({
        type: Kafka.Task.SCAN_TYPE,
        rack: dev.rack,
        one: true,
      });
      const grown = { ...pump, sources: ["pump-1", "pump-2"] };
      const again = await answerNextCommand(client, scan, {
        data: sampleOf(grown, OVEN_SHAPE),
      });
      fireEvent.click(dialog.getByRole("button", { name: "Listen again" }));
      await again.answered;
      await dialog.findAllByText("pump-1, pump-2");
      dialog.getByText("1 field selected");
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 field" }));
      const records = await waitFor(async () => {
        const next = await retrieveRecords(draft.key);
        expect(next).toHaveLength(2);
        return next;
      });
      expect(records.map(({ recordKey }) => recordKey)).toEqual(["pump-1", "pump-2"]);
    });

    it("should show each field's value from the newest record that holds it", async () => {
      const { dev } = await renderRead();
      const record = (offset: number, value: object) => ({
        partition: 0,
        offset,
        key: "oven-1",
        time: offset,
        headers: [],
        format: "json",
        payload: JSON.stringify(value),
      });
      const records = [record(1, { state: "idle" }), record(2, { temperature: 61.5 })];
      const dialog = await openAddData(dev, {
        data: {
          topics: [{ topic: TOPIC, records, shapes: [OVEN_SHAPE], error: "" }],
          truncated: false,
        },
      });
      await dialog.findByText("61.5");
      dialog.getByText("idle");
    });

    it("should add later messages to the ICD of the first record", async () => {
      const { draft, dev } = await renderRead({
        fields: [createField("temperature", "float")],
        recordKey: "oven-1",
      });
      await screen.findByRole("treeitem", { name: /^temperature/ });
      const pump = {
        sources: ["pump"],
        fields: [createInferredField("/pressure", "int64", "number")],
        index: -1,
        count: 1,
      };
      const dialog = await openAddData(dev, { data: sampleOf(OVEN_SHAPE, pump) });
      expect(
        await dialog.findByRole("checkbox", { name: "Record /temperature" }),
      ).toHaveProperty("disabled", false);
      fireEvent.click(dialog.getByText("pump"));
      fireEvent.click(
        await dialog.findByRole("checkbox", { name: "Record /pressure" }),
      );
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 field" }));
      const records = await waitFor(async () => {
        const next = await retrieveRecords(draft.key);
        expect(next).toHaveLength(2);
        return next;
      });
      expect(records[1]).toMatchObject({ icd: records[0].icd, recordKey: "pump" });
      expect(records[1].messages[0].index).toBeUndefined();
      const { messages } = await client.icds.retrieve(records[0].icd);
      expect(messages.map(({ name }) => name)).toEqual(["oven-1", "pump"]);
      expect(messages[1].payload).toMatchObject({
        fields: [{ name: "pressure", pointer: "/pressure", type: "integer" }],
      });
    });

    it("should not count the picks of a shape that a new sample lacks", async () => {
      const { dev } = await renderRead();
      const pump = {
        sources: ["pump"],
        fields: [createInferredField("/pressure", "int64", "number")],
        index: -1,
        count: 2,
      };
      const dialog = await openAddData(dev, { data: sampleOf(pump) });
      fireEvent.click(await dialog.findByRole("button", { name: "Select all" }));
      await dialog.findByText("1 field selected");
      const scan = await client.tasks.retrieve({
        type: Kafka.Task.SCAN_TYPE,
        rack: dev.rack,
        one: true,
      });
      const grown = {
        ...pump,
        fields: [...pump.fields, createInferredField("/flow", "float64", "number")],
      };
      const again = await answerNextCommand(client, scan, { data: sampleOf(grown) });
      fireEvent.click(dialog.getByRole("button", { name: "Listen again" }));
      await again.answered;
      await dialog.findByRole("checkbox", { name: "Record /flow" });
      expect(dialog.queryByText("1 field selected")).toBeNull();
    });

    it("should say why a sample read no records", async () => {
      const { dev } = await renderRead();
      const data = {
        topics: [
          { topic: TOPIC, records: [], shapes: [], error: "The topic does not exist." },
        ],
        truncated: false,
      };
      const dialog = await openAddData(dev, { data });
      await dialog.findByText("Read no records");
      dialog.getByText("The topic does not exist.");
      expect(dialog.queryByText(/some unread/)).toBeNull();
    });

    it("should show why the sample failed", async () => {
      const { dev } = await renderRead();
      const dialog = await openAddData(dev, {
        variant: "error",
        message: "Failed to reach the cluster",
      });
      await dialog.findByText("Failed to read records");
      dialog.getByText(/Failed to reach the cluster/);
    });
  });

  describe("fields", () => {
    it("should add a field to the message of a record, read it, and select it", async () => {
      const temperature = createField("temperature", "float");
      const { draft } = await renderRead({ fields: [temperature] });
      fireEvent.click(await screen.findByRole("treeitem", { name: /JSON.*Any key/ }));
      fireEvent.click(await screen.findByRole("button", { name: "Add field" }));
      await screen.findByDisplayValue("Field 2");
      const { icd: icdKey, messages } = (await retrieveRecords(draft.key))[0];
      await waitFor(async () =>
        expect(jsonFieldsOf(await client.icds.retrieve(icdKey))).toMatchObject([
          { key: temperature.key },
          { name: "Field 2", pointer: "" },
        ]),
      );
      const [, added] = jsonFieldsOf(await client.icds.retrieve(icdKey));
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toEqual([
          messages[0].bindings[0],
          { field: added.key, channel: 0, dataType: "" },
        ]),
      );
    });

    it("should drop the read of a field whose ICD edit fails", async () => {
      const added = await addFieldAsICDReader();
      await waitFor(() =>
        expect(added.statuses).toContainEqual(
          expect.objectContaining({ message: "Failed to save ICD" }),
        ),
      );
      await expectFieldUndone(added);
    });

    it("should undo a read that the Core refuses after a slow ICD edit", async () => {
      const temperature = createField("temperature", "float");
      const reader = await createICDReaderClient();
      const held = Promise.withResolvers<void>();
      const begin = reader.icds.beginTransaction.bind(reader.icds);
      vi.spyOn(reader.icds, "beginTransaction").mockImplementation((key) => {
        const tx = begin(key);
        return {
          add: (actions) => tx.add(actions),
          abort: () => tx.abort(),
          commit: async () => {
            await held.promise;
            return await tx.commit();
          },
        };
      });
      const { draft, statuses } = await renderRead({
        fields: [temperature],
        as: reader,
      });
      const [{ icd: icdKey, messages }] = await retrieveRecords(draft.key);
      fireEvent.click(await screen.findByRole("treeitem", { name: /JSON.*Any key/ }));
      const add = await screen.findByRole("button", { name: "Add field" });
      // The ICD edit fails only after the read's edit leaves the window that holds it.
      vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
      onTestFinished(() => {
        vi.useRealTimers();
      });
      fireEvent.click(add);
      await vi.runOnlyPendingTimersAsync();
      vi.useRealTimers();
      held.resolve();
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            message: "Failed to save task",
            description: expect.stringContaining("does not exist in message"),
          }),
        ),
      );
      await expectFieldUndone({
        draft,
        statuses,
        temperature,
        icdKey,
        bindings: messages[0].bindings,
      });
    });

    it("should remove a field from the ICD through the context menu", async () => {
      const temperature = createField("temperature", "float");
      const pressure = createField("pressure", "float");
      const { draft } = await renderRead({ fields: [temperature, pressure] });
      fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /^pressure/ }));
      fireEvent.click(await screen.findByRole("menuitem", { name: "Remove" }));
      const { icd: icdKey } = (await retrieveRecords(draft.key))[0];
      await waitFor(async () =>
        expect(
          jsonFieldsOf(await client.icds.retrieve(icdKey)).map(({ key }) => key),
        ).toEqual([temperature.key]),
      );
    });

    it("should stop and start reading a field", async () => {
      const temperature = createField("temperature", "float");
      const { draft } = await renderRead({
        fields: [temperature, createField("pressure", "float")],
        bound: { [temperature.key]: 0 },
      });
      const box = await screen.findByRole("checkbox", { name: "Read temperature" });
      fireEvent.click(box);
      await waitFor(async () => expect(await retrieveBindings(draft.key)).toEqual([]));
      fireEvent.click(box);
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toEqual([
          { field: temperature.key, channel: 0, dataType: "" },
        ]),
      );
    });

    it("should name the channel of a new field after the cluster, topic, record key, and field", async () => {
      const { dev } = await renderRead({
        fields: [createField("temperature", "float")],
        recordKey: "pump",
      });
      await selectField("temperature");
      const name = await screen.findByDisplayValue(`${dev.name}_oven_pump_temperature`);
      expect((name as HTMLInputElement).disabled).toBe(true);
      const item = screen.getByRole("treeitem", { name: /^temperature/ });
      expect(within(item).getByRole("img", { name: "New channel" })).toBeTruthy();
    });

    it("should offer only the data types that the kind of a field converts to", async () => {
      const count = createField("count", "integer");
      const { draft } = await renderRead({ fields: [count] });
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
    });

    it("should unbind a channel that the kind of its field no longer converts to", async () => {
      const ch = await client.channels.create({
        name: uniqueName("count"),
        dataType: DataType.INT64,
        virtual: true,
      });
      const count = createField("count", "integer");
      const rec = await createRecord({
        fields: [count],
        bound: { [count.key]: ch.key },
      });
      const { draft } = await renderRead({
        records: [rec],
        beforeRender: () => retype(rec, count, "float"),
      });
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toEqual([
          { field: count.key, channel: 0, dataType: "" },
        ]),
      );
    });
  });

  describe("timestamp", () => {
    it("should read the time from the first timestamp field on payload timing", async () => {
      const time = createTimeField();
      const temperature = createField("temperature", "float");
      const { dev, draft } = await renderRead({
        fields: [temperature, time],
        bound: { [temperature.key]: 0 },
      });
      fireEvent.click(await screen.findByRole("button", { name: "Payload value" }));
      await waitFor(async () =>
        expect((await retrieveRecords(draft.key))[0].messages[0]).toEqual({
          message: expect.any(String),
          index: time.key,
          bindings: [
            { field: temperature.key, channel: 0, dataType: "" },
            { field: time.key, channel: 0, dataType: "" },
          ],
        }),
      );
      await screen.findByDisplayValue(`${dev.name}_oven_time`);
      expect(screen.queryByRole("treeitem", { name: /^time/ })).toBeNull();
      fireEvent.click(screen.getByRole("button", { name: "Kafka timestamp" }));
      await waitFor(async () =>
        expect((await retrieveRecords(draft.key))[0].messages[0]).toEqual({
          message: expect.any(String),
          bindings: [{ field: temperature.key, channel: 0, dataType: "" }],
        }),
      );
    });

    it("should keep the index channel when the index moves to another field", async () => {
      const timeCh = await createTimeChannel();
      const time = createTimeField();
      const sent = { ...createTimeField(), name: "sent", pointer: "/sent" };
      const { draft } = await renderRead({
        fields: [time, sent],
        index: time.key,
        bound: { [time.key]: timeCh.key },
      });
      fireEvent.click(await screen.findByRole("treeitem", { name: /JSON.*Any key/ }));
      fireEvent.click(await findDialogTriggerByText("time"));
      const dialogs = await screen.findAllByRole("dialog");
      fireEvent.click(within(dialogs[dialogs.length - 1]).getByText("sent"));
      await waitFor(async () =>
        expect((await retrieveRecords(draft.key))[0].messages[0]).toEqual({
          message: expect.any(String),
          index: sent.key,
          bindings: [{ field: sent.key, channel: timeCh.key, dataType: "" }],
        }),
      );
    });
  });

  describe("messages", () => {
    it("should list each message of a record that reads several, with its fields", async () => {
      const rec = await createRecord({ fields: [createField("temperature", "float")] });
      await addMessage(rec, "status", [createField("mode", "string")], KIND_MATCH);
      await renderRead({ records: [rec] });
      await screen.findByRole("treeitem", { name: "Any key" });
      expect(screen.getByRole("treeitem", { name: "status" })).toBeTruthy();
      expect(screen.getByRole("checkbox", { name: "Read temperature" })).toBeTruthy();
      expect(screen.getByRole("checkbox", { name: "Read mode" })).toBeTruthy();
    });

    it("should add a message that matches at the pointer of the others", async () => {
      const rec = await createRecord({ fields: [createField("temperature", "float")] });
      await addMessage(rec, "status", [createField("mode", "string")], KIND_MATCH);
      const { draft } = await renderRead({ records: [rec] });
      fireEvent.click(await screen.findByRole("treeitem", { name: /JSON.*Any key/ }));
      fireEvent.click(await screen.findByRole("button", { name: "Add message" }));
      await screen.findByDisplayValue("Any key 3");
      expect(screen.getByDisplayValue("/kind")).toBeTruthy();
      await waitFor(async () => {
        const { messages } = await client.icds.retrieve({ key: rec.icd });
        expect(messages).toHaveLength(3);
        expect(messages[2]).toMatchObject({
          name: "Any key 3",
          payload: { format: "json", match: { ...KIND_MATCH, value: "" }, fields: [] },
        });
        const reads = (await retrieveRecords(draft.key))[0].messages;
        expect(reads.map(({ message }) => message)).toEqual([
          rec.messages[0].message,
          rec.messages[1].message,
          messages[2].key,
        ]);
      });
    });

    it("should add a message from the context menu of a record", async () => {
      const rec = await createRecord({ fields: [createField("temperature", "float")] });
      const { draft } = await renderRead({ records: [rec] });
      fireEvent.contextMenu(
        await screen.findByRole("treeitem", { name: /JSON.*Any key/ }),
      );
      fireEvent.click(await screen.findByRole("menuitem", { name: "Add message" }));
      await screen.findByRole("treeitem", { name: "Any key 2" });
      await waitFor(async () =>
        expect((await retrieveRecords(draft.key))[0].messages).toHaveLength(2),
      );
    });

    it("should name the match of a message that takes every value Any value", async () => {
      const rec = await createRecord({ fields: [createField("temperature", "float")] });
      await addMessage(rec, "status", [createField("mode", "string")]);
      await renderRead({ records: [rec] });
      fireEvent.click(await screen.findByRole("treeitem", { name: "status" }));
      expect(await screen.findByText("Any value")).toBeTruthy();
      expect(screen.queryByText("Any document")).toBeNull();
    });

    it("should stop reading a message of a record, keeping it in the ICD", async () => {
      const rec = await createRecord({ fields: [createField("temperature", "float")] });
      const status = await addMessage(
        rec,
        "status",
        [createField("mode", "string")],
        KIND_MATCH,
      );
      const { draft } = await renderRead({ records: [rec] });
      fireEvent.contextMenu(await screen.findByRole("treeitem", { name: "status" }));
      fireEvent.click(await screen.findByText("Remove"));
      await waitFor(async () =>
        expect((await retrieveRecords(draft.key))[0].messages).toEqual([
          rec.messages[0],
        ]),
      );
      const { messages } = await client.icds.retrieve({ key: rec.icd });
      expect(messages.map(({ key }) => key)).toContain(status.key);
    });
  });

  describe("channel name preview", () => {
    it("should preview a numbered name for a new channel whose name is taken", async () => {
      const { dev } = await renderRead({
        fields: [createField("temperature", "float")],
        beforeRender: async ({ name }) =>
          await client.channels.create({
            name: `${name}_oven_temperature`,
            dataType: DataType.FLOAT64,
            virtual: true,
          }),
      });
      await selectField("temperature");
      await screen.findByDisplayValue(`${dev.name}_oven_temperature_2`);
    });

    it("should preview a name for each record that reads one message", async () => {
      const first = await createRecord({
        fields: [createField("temperature", "float")],
      });
      const second = kafka.readRecordZ.parse({
        icd: first.icd,
        recordKey: "",
        messages: first.messages,
      });
      const { dev } = await renderRead({ records: [first, second] });
      const items = await screen.findAllByRole("treeitem", { name: /^temperature/ });
      expect(items).toHaveLength(2);
      const name = `${dev.name}_oven_temperature`;
      fireEvent.click(items[1]);
      await screen.findByDisplayValue(`${name}_2`);
      fireEvent.click(items[0]);
      await screen.findByDisplayValue(name);
      expect(screen.queryByDisplayValue(`${name}_2`)).toBeNull();
    });
  });

  describe("deploying against a live Core", () => {
    it("should reuse the channels of a deploy that failed partway", async () => {
      const rec = await createRecord({ fields: [createField("temperature", "float")] });
      await addMessage(rec, "status", [createField("pressure", "float")], KIND_MATCH);
      const { container, dev, draft, statuses } = await renderRead({ records: [rec] });
      const create = client.channels.create.bind(client.channels);
      const dropped = vi
        .spyOn(client.channels, "create")
        .mockImplementation(async (channels, options) => {
          if (array.toArray(channels).some(({ name }) => name.includes("_status_")))
            throw new Error("Connection lost");
          return await create(channels, options);
        });
      await clickDeploy(container);
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({ description: "Connection lost" }),
        ),
      );
      dropped.mockRestore();
      const made = await retrieveStored(dev.key, rec);
      expect(made.channels["/temperature"]).toBeGreaterThan(0);
      await deploy(container, draft.key);
      const [first] = (await retrieveRecords(draft.key))[0].messages;
      expect(first.bindings[0].channel).toBe(made.channels["/temperature"]);
      expect((await retrieveStored(dev.key, rec)).index).toBe(made.index);
    });

    it("should keep the channel of a field that stops and starts reading", async () => {
      const temperature = createField("temperature", "float");
      const { container, draft } = await renderRead({ fields: [temperature] });
      await deploy(container, draft.key);
      const [{ channel }] = await retrieveBindings(draft.key);
      const box = await screen.findByRole("checkbox", { name: "Read temperature" });
      fireEvent.click(box);
      await waitFor(async () => expect(await retrieveBindings(draft.key)).toEqual([]));
      fireEvent.click(box);
      await waitFor(async () =>
        expect(await retrieveBindings(draft.key)).toMatchObject([
          { field: temperature.key, channel },
        ]),
      );
      await deploy(container, draft.key);
      expect((await retrieveBindings(draft.key))[0].channel).toBe(channel);
    });

    it("should create an index and a data channel for a record stamped by Kafka", async () => {
      const temperature = createField("temperature", "float");
      const { container, dev, draft } = await renderRead({ fields: [temperature] });
      const created = await deploy(container, draft.key);
      expect(created.rack).toBe(dev.rack);
      const [binding] = created.config.topics[0].records[0].messages[0].bindings;
      const dataCh = await client.channels.retrieve(binding.channel);
      expect(dataCh.name).toBe(`${dev.name}_oven_temperature`);
      expect(dataCh.dataType.equals(DataType.FLOAT64)).toBe(true);
      const indexCh = await client.channels.retrieve(dataCh.index);
      expect(indexCh.isIndex).toBe(true);
      expect(indexCh.name).toBe(`${dev.name}_oven_time`);
    });

    it("should name and store the channels of each topic apart", async () => {
      const { container, dev, draft } = await renderRead({
        fields: [createField("temperature", "float")],
        topics: [await createTopic("plant", [createField("temperature", "float")])],
      });
      const created = await deploy(container, draft.key);
      const [oven, plant] = await Promise.all(
        created.config.topics.map(async ({ records: [rec] }) => ({
          rec,
          ch: await client.channels.retrieve(rec.messages[0].bindings[0].channel),
        })),
      );
      expect(oven.ch.name).toBe(`${dev.name}_oven_temperature`);
      expect(plant.ch.name).toBe(`${dev.name}_plant_temperature`);
      const { properties } = await client.devices.retrieve({
        key: dev.key,
        schemas: Kafka.Device.SCHEMAS,
      });
      expect(properties.read[storedKeyOf(oven.rec)].channels["/temperature"]).toBe(
        oven.ch.key,
      );
      expect(
        properties.read[`plant##${plant.rec.messages[0].message}`].channels[
          "/temperature"
        ],
      ).toBe(plant.ch.key);
    });

    it("should bind the index field to the index of the data channels", async () => {
      const time = createTimeField();
      const temperature = createField("temperature", "float");
      const { container, dev, draft } = await renderRead({
        fields: [time, temperature],
        recordKey: "pump",
        index: time.key,
      });
      const created = await deploy(container, draft.key);
      const [read] = created.config.topics[0].records[0].messages;
      const channelOf = (field: string) =>
        read.bindings.find((b) => b.field === field)?.channel ?? 0;
      const indexCh = await client.channels.retrieve(channelOf(time.key));
      expect(indexCh.isIndex).toBe(true);
      expect(indexCh.name).toBe(`${dev.name}_oven_pump_time`);
      const dataCh = await client.channels.retrieve(channelOf(temperature.key));
      expect(dataCh.index).toBe(indexCh.key);
      expect(read.index).toBe(time.key);
    });

    /**
     * Deploys two records: the first binds temperature to a channel, and the device
     * stores that channel for the temperature of the second, which binds none.
     * @returns the stored index, the channel, and the bindings of the second record.
     */
    const deployShared = async (disabled: boolean) => {
      const index = await createTimeChannel();
      const shared = await client.channels.create({
        name: uniqueName("temperature"),
        dataType: DataType.FLOAT64,
        index: index.key,
      });
      const temperature = createField("temperature", "float");
      const owner = await createRecord({
        fields: [temperature],
        recordKey: "pump-1",
        bound: { [temperature.key]: shared.key },
      });
      const other = await createRecord({
        fields: [createField("temperature", "float")],
        recordKey: "pump-2",
      });
      const { container, dev, draft } = await renderRead({
        records: [{ ...owner, disabled }, other],
      });
      await storeChannels(dev.key, other, {
        index: index.key,
        channels: { "/temperature": shared.key },
      });
      const created = await deploy(container, draft.key);
      return {
        index,
        shared,
        bindings: created.config.topics[0].records[1].messages[0].bindings,
      };
    };

    it("should not reuse a stored channel that another record reads", async () => {
      const { index, shared, bindings } = await deployShared(false);
      expect(bindings[0].channel).not.toBe(0);
      expect(bindings[0].channel).not.toBe(shared.key);
      // The Core refuses an index that two messages write.
      expect((await client.channels.retrieve(bindings[0].channel)).index).not.toBe(
        index.key,
      );
    });

    it("should reuse a stored channel that only a disabled record reads", async () => {
      const { shared, bindings } = await deployShared(true);
      expect(bindings[0].channel).toBe(shared.key);
    });

    it("should keep the channels of fields that have them", async () => {
      const index = await createTimeChannel();
      const existing = await client.channels.create({
        name: uniqueName("temperature"),
        dataType: DataType.FLOAT64,
        index: index.key,
      });
      const temperature = createField("temperature", "float");
      const pressure = createField("pressure", "float");
      const { container, draft } = await renderRead({
        fields: [temperature, pressure],
        bound: { [temperature.key]: existing.key, [pressure.key]: 0 },
      });
      const created = await deploy(container, draft.key);
      const [a, b] = created.config.topics[0].records[0].messages[0].bindings;
      expect(a.channel).toBe(existing.key);
      expect((await client.channels.retrieve(b.channel)).index).toBe(index.key);
    });

    it("should create virtual channels with no index for string fields", async () => {
      const { container, draft } = await renderRead({
        fields: [createField("mode", "string")],
      });
      const created = await deploy(container, draft.key);
      const [binding] = created.config.topics[0].records[0].messages[0].bindings;
      const ch = await client.channels.retrieve(binding.channel);
      expect(ch.virtual).toBe(true);
      expect(ch.dataType.equals(DataType.STRING)).toBe(true);
    });

    it("should give each message of a record its own index", async () => {
      const rec = await createRecord({ fields: [createField("temperature", "float")] });
      await addMessage(rec, "flow", [createField("rate", "float")], KIND_MATCH);
      const { container, dev, draft } = await renderRead({ records: [rec] });
      const created = await deploy(container, draft.key);
      const [first, second] = created.config.topics[0].records[0].messages;
      const a = await client.channels.retrieve(first.bindings[0].channel);
      const b = await client.channels.retrieve(second.bindings[0].channel);
      expect(b.name).toBe(`${dev.name}_oven_flow_rate`);
      expect(a.index).not.toBe(b.index);
    });

    it("should store a string field on the index of a number field", async () => {
      const { container, draft } = await renderRead({
        fields: [createField("mode", "string"), createField("temperature", "float")],
      });
      const created = await deploy(container, draft.key);
      const [mode, temperature] =
        created.config.topics[0].records[0].messages[0].bindings;
      const modeCh = await client.channels.retrieve(mode.channel);
      expect(modeCh.virtual).toBe(false);
      expect(modeCh.dataType.equals(DataType.STRING)).toBe(true);
      const temperatureCh = await client.channels.retrieve(temperature.channel);
      expect(modeCh.index).toBe(temperatureCh.index);
    });

    it("should refuse a record that reads no fields", async () => {
      const { container, statuses } = await renderRead({
        fields: [createField("temperature", "float")],
        recordKey: "pump",
        bound: {},
      });
      await clickDeploy(container);
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            message: "Failed to start task",
            description: "The task reads no fields of record pump in topic oven",
          }),
        ),
      );
    });

    it("should refuse a record whose message is gone", async () => {
      const rec = await createRecord({ fields: [createField("temperature", "float")] });
      const { container, statuses } = await renderRead({
        records: [rec],
        beforeRender: () => removeMessage(rec),
      });
      await screen.findByText("Message not found");
      await clickDeploy(container);
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            message: "Failed to start task",
            description:
              "The message format of record Any key in topic oven is missing or not JSON",
          }),
        ),
      );
    });
  });
});
