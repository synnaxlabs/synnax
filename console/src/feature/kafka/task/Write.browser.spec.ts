// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  type device,
  icd,
  kafka,
  type Synnax,
  task,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Kafka } from "@/feature/kafka";
import {
  browseOf,
  createInferredField,
  createKafkaDevice,
  createScanTask,
  listTopic,
  sampleOfTopic,
} from "@/feature/kafka/testutil";
import { ICD } from "@/platform/icd";
import { pickMessageFormat, setField } from "@/platform/icd/testutil";
import {
  answerCommand,
  answerCommandsUntil,
  answerNextCommand,
  awaitCommand,
  clickDeploy,
  type CommandAnswer,
  countFailedSaves,
  deployAndAwaitTask,
  preloadForms,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";
import { findDialogTriggerByText } from "@/testutil";

const client = createTestClient();

preloadForms(Kafka.Task.FORMS);

// Drafts carry no key; the created task mints its own.
const ZERO_DRAFT: task.New<Kafka.Task.WriteSchemas> = {
  name: Kafka.Task.DEFAULT_WRITE_NAME,
  type: Kafka.Task.WRITE_TYPE,
  config: Kafka.Task.WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (client: Synnax, config: kafka.WriteConfig) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, Kafka.Task.WRITE_SCHEMAS);

const retrieveTargets = async (key: task.Key) =>
  (await client.tasks.retrieve({ key, schemas: Kafka.Task.WRITE_SCHEMAS })).config
    .targets;

const createField = (name: string, type: icd.JSONField["type"]): icd.JSONField =>
  icd.jsonFieldZ.parse({ name, pointer: `/${name}`, type });

/**
 * Creates an ICD that holds one JSON message of the fields, and a target that sends it,
 * with no channels.
 */
const createTarget = async (
  fields: icd.JSONField[],
  overrides: Partial<Kafka.Task.WriteTarget> = {},
) => {
  const message = icd.messageZ.parse({
    name: uniqueName("command"),
    payload: { format: "json", fields },
  });
  const ic = await client.icds.create({
    name: uniqueName("pump"),
    messages: [message],
  });
  const target = kafka.writeTargetZ.parse({
    key: "t1",
    topic: "plant.pumps",
    icd: ic.key,
    message: { message: message.key, bindings: ICD.Write.suit([], fields) },
    ...overrides,
  });
  return { ic, message, target };
};

interface RenderWriteOptions extends RenderTaskFormTabOptions {
  /** The cluster of the task. Absent creates one. */
  dev?: device.Device;
}

// The form body renders against a configured cluster only.
const renderWrite = async (
  targets: Kafka.Task.WriteTarget[] = [],
  { dev: given, ...options }: RenderWriteOptions = {},
) => {
  const dev = given ?? (await createKafkaDevice(client));
  const draft = await createDraft(client, {
    ...Kafka.Task.WRITE_SCHEMAS.config.parse({}),
    device: dev.key,
    targets,
  });
  const result = await renderTaskFormTab(Kafka.Task.FORMS[Kafka.Task.WRITE_TYPE], {
    client,
    taskKey: draft.key,
    ...options,
  });
  return { ...result, dev, draft };
};

const deployWrite = async (container: HTMLElement, key: task.Key) =>
  (await deployAndAwaitTask(client, container, key, Kafka.Task.WRITE_SCHEMAS)).config
    .targets;

/** @returns the channel of each channel binding of the target. */
const boundChannels = ({ message }: Kafka.Task.WriteTarget): channel.Key[] =>
  message.bindings.flatMap((b) => (b.type === "channel" ? [b.channel] : []));

const retrieveWriteRecord = async (key: device.Key) =>
  (await client.devices.retrieve({ key, schemas: Kafka.Device.SCHEMAS })).properties
    .write;

const TOPIC = "plant.pumps";
const MISSING: kafka.SampledTopic = {
  topic: TOPIC,
  records: [],
  shapes: [],
  error: "The topic does not exist.",
  missing: true,
};

/** @returns a shape of count records that hold the fields, sent by the keys. */
const createShape = (sources: string[], count: number, ...pointers: string[]) => ({
  sources,
  fields: pointers.map((p) => createInferredField(p, "float64", "number")),
  index: -1,
  count,
  present: -1,
});

/**
 * Answers the commands of the Add targets dialog of the task on dev until one of the
 * type last.
 */
const answerScan = async (
  dev: device.Device,
  last: "browse" | "sample",
  answers: Record<string, Omit<CommandAnswer, "type">>,
) =>
  await answerCommandsUntil(client, [await createScanTask(client, dev)], last, answers);

/**
 * Opens the Add targets dialog, which renders once the form leaves its loading preview.
 * @returns queries scoped to the dialog.
 */
const clickAddTargets = async () => {
  fireEvent.click(await screen.findByRole("button", { name: "Add targets" }));
  return within(await screen.findByRole("dialog"));
};

/**
 * Adds the target of a topic that the browse lists, from a sample of the shapes.
 * @returns the targets of the task once it holds count of them.
 */
const addSampledTarget = async (
  dev: device.Device,
  draft: task.Key,
  shapes: object[],
  count = 1,
) => {
  const { answered } = await answerScan(dev, "sample", {
    browse: { data: browseOf(listTopic(TOPIC)) },
    sample: { data: sampleOfTopic(TOPIC, ...shapes) },
  });
  const dialog = await clickAddTargets();
  await answered;
  await dialog.findByText(/ · \d+ records?$/);
  fireEvent.click(dialog.getByRole("checkbox", { name: `Pick ${TOPIC}` }));
  fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
  return await waitFor(async () => {
    const targets = await retrieveTargets(draft);
    expect(targets).toHaveLength(count);
    return targets;
  });
};

describe("Kafka Write form", () => {
  // This first test pays the one-time load of the lazy form, which takes over 10 s on
  // a loaded CI runner.
  it("should show the empty state and add + select a target", async () => {
    const { dev, draft } = await renderWrite();
    await screen.findByText("Add a target to send records to a topic");
    await addSampledTarget(dev, draft.key, [createShape(["pump-1"], 2, "/speed")]);
    await screen.findByText("Produce");
    expect(screen.getByDisplayValue(TOPIC)).toBeTruthy();
  }, 30_000);

  it("should send the shape of the most records, in a message named after the topic", async () => {
    let statuses: Status.NotificationSpec[] = [];
    const { dev, draft } = await renderWrite([], {
      onStatuses: (s) => (statuses = s),
    });
    const [target] = await addSampledTarget(dev, draft.key, [
      createShape(["pump-1"], 2, "/speed"),
      createShape(["pump-2"], 5, "/speed", "/mode"),
    ]);
    expect(target).toMatchObject({ topic: TOPIC, recordKey: "pump-2" });
    const [ic] = await client.icds.retrieve([target.icd]);
    expect(ic.name).toBe(Kafka.Task.DEFAULT_WRITE_NAME);
    expect(ic.messages).toMatchObject([
      {
        key: target.message.message,
        name: TOPIC,
        payload: { fields: [{ pointer: "/speed" }, { pointer: "/mode" }] },
      },
    ]);
    expect(target.message.bindings).toMatchObject([
      { type: "channel", channel: 0 },
      { type: "channel", channel: 0 },
    ]);
    expect(countFailedSaves(statuses)).toBe(0);
  });

  it("should give a target no record key when several keys sent its shape", async () => {
    const { dev, draft } = await renderWrite();
    const [target] = await addSampledTarget(dev, draft.key, [
      createShape(["pump-1", "pump-2"], 2, "/speed"),
    ]);
    expect(target.recordKey).toBe("");
  });

  it("should add the message of a new target to the ICD of the most targets", async () => {
    const { ic, target } = await createTarget([createField("speed", "float")]);
    const { dev, draft } = await renderWrite([target]);
    await screen.findByRole("treeitem", { name: /Topic.*plant\.pumps/ });
    const [, added] = await addSampledTarget(
      dev,
      draft.key,
      [createShape(["pump-1"], 2, "/speed")],
      2,
    );
    expect(added.icd).toBe(ic.key);
    await waitFor(async () => {
      const [saved] = await client.icds.retrieve([ic.key]);
      expect(saved.messages.at(-1)).toMatchObject({
        key: added.message.message,
        name: TOPIC,
      });
    });
  });

  it("should show the fields of a topic as a preview, picked by the topic", async () => {
    const { dev, draft } = await renderWrite();
    const { answered } = await answerScan(dev, "sample", {
      browse: { data: browseOf(listTopic(TOPIC)) },
      sample: { data: sampleOfTopic(TOPIC, createShape(["pump-1"], 2, "/speed")) },
    });
    const dialog = await clickAddTargets();
    await answered;
    await dialog.findByText("/speed");
    expect(dialog.queryByRole("checkbox", { name: "Record /speed" })).toBeNull();
    expect(dialog.queryByRole("button", { name: "Select all" })).toBeNull();
    fireEvent.click(dialog.getByRole("checkbox", { name: `Pick ${TOPIC}` }));
    fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
    await waitFor(async () =>
      expect(await retrieveTargets(draft.key)).toMatchObject([{ topic: TOPIC }]),
    );
  });

  it("should add a target with an empty message for a topic that holds no records", async () => {
    const { dev, draft } = await renderWrite();
    const { answered } = await answerScan(dev, "browse", {
      browse: { data: browseOf(listTopic(TOPIC, { records: 0 })) },
    });
    const dialog = await clickAddTargets();
    await answered;
    fireEvent.click(await dialog.findByRole("checkbox", { name: `Pick ${TOPIC}` }));
    fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
    const target = await waitFor(async () => {
      const [t] = await retrieveTargets(draft.key);
      expect(t).toBeDefined();
      return t;
    });
    expect(target.message.bindings).toEqual([]);
    const [ic] = await client.icds.retrieve([target.icd]);
    expect(ic.messages).toMatchObject([
      { name: TOPIC, payload: { format: "json", fields: [] } },
    ]);
  });

  it("should take the shape of a topic whose read failed once Retry reads it", async () => {
    const { dev, draft } = await renderWrite();
    const scan = await createScanTask(client, dev);
    const failed = {
      variant: "error",
      message: "Failed to reach the cluster",
    } as const;
    const first = await answerCommandsUntil(client, [scan], "sample", {
      browse: { data: browseOf(listTopic(TOPIC)) },
      sample: failed,
    });
    const dialog = await clickAddTargets();
    await first.answered;
    await dialog.findByText("Failed to read records");
    const second = await answerNextCommand(client, scan, { type: "sample", ...failed });
    fireEvent.click(dialog.getByRole("checkbox", { name: `Pick ${TOPIC}` }));
    await second.answered;
    await dialog.findByRole("button", { name: "Add 1 target" });
    const third = await answerNextCommand(client, scan, {
      type: "sample",
      data: sampleOfTopic(TOPIC, createShape(["pump-1"], 2, "/speed")),
    });
    fireEvent.click(await dialog.findByRole("button", { name: "Retry" }));
    await third.answered;
    await dialog.findByText("/speed");
    fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
    const target = await waitFor(async () => {
      const [t] = await retrieveTargets(draft.key);
      expect(t).toBeDefined();
      return t;
    });
    const [ic] = await client.icds.retrieve([target.icd]);
    expect(ic.messages).toMatchObject([
      { name: TOPIC, payload: { fields: [{ pointer: "/speed" }] } },
    ]);
  });

  it("should add a target for a typed topic while the cluster cannot be reached", async () => {
    const { dev, draft } = await renderWrite();
    const failed = {
      variant: "error",
      message: "Failed to reach the cluster",
    } as const;
    const { answered } = await answerScan(dev, "sample", {
      browse: failed,
      sample: failed,
    });
    const dialog = await clickAddTargets();
    await dialog.findByText("Failed to list topics");
    fireEvent.change(dialog.getByPlaceholderText("Search topics"), {
      target: { value: TOPIC },
    });
    fireEvent.click(await dialog.findByRole("button", { name: `Write “${TOPIC}”` }));
    await answered;
    fireEvent.click(await dialog.findByRole("button", { name: "Add 1 target" }));
    const target = await waitFor(async () => {
      const [t] = await retrieveTargets(draft.key);
      expect(t).toBeDefined();
      return t;
    });
    expect(target.topic).toBe(TOPIC);
    const [ic] = await client.icds.retrieve([target.icd]);
    expect(ic.messages).toMatchObject([
      { name: TOPIC, payload: { format: "json", fields: [] } },
    ]);
  });

  it("should add a target for a topic that the cluster does not let Synnax read", async () => {
    const { dev, draft } = await renderWrite();
    const { answered } = await answerScan(dev, "browse", {
      browse: { data: browseOf(listTopic(TOPIC, { readable: false })) },
    });
    const dialog = await clickAddTargets();
    await answered;
    await dialog.findByText(
      "The cluster does not let Synnax read this topic. Its target gets a message with no fields.",
    );
    fireEvent.click(dialog.getByRole("checkbox", { name: `Pick ${TOPIC}` }));
    fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
    await waitFor(async () =>
      expect(await retrieveTargets(draft.key)).toMatchObject([
        { topic: TOPIC, message: { bindings: [] } },
      ]),
    );
  });

  it("should add a target for a topic typed by name", async () => {
    const { dev, draft } = await renderWrite();
    const { answered } = await answerScan(dev, "sample", {
      browse: { data: browseOf() },
      sample: { data: sampleOfTopic(TOPIC) },
    });
    const dialog = await clickAddTargets();
    fireEvent.change(await dialog.findByPlaceholderText("Search topics"), {
      target: { value: TOPIC },
    });
    fireEvent.click(await dialog.findByRole("button", { name: `Write “${TOPIC}”` }));
    await answered;
    fireEvent.click(await dialog.findByRole("button", { name: "Add 1 target" }));
    await waitFor(async () =>
      expect(await retrieveTargets(draft.key)).toMatchObject([
        { topic: TOPIC, recordKey: "" },
      ]),
    );
  });

  it("should show that a typed topic the cluster does not list does not exist yet", async () => {
    const { dev } = await renderWrite();
    const { answered } = await answerScan(dev, "sample", {
      browse: { data: browseOf() },
      sample: { data: { topics: [MISSING], truncated: false } },
    });
    const dialog = await clickAddTargets();
    fireEvent.change(await dialog.findByPlaceholderText("Search topics"), {
      target: { value: TOPIC },
    });
    fireEvent.click(await dialog.findByRole("button", { name: `Write “${TOPIC}”` }));
    await answered;
    await dialog.findByText("Topic does not exist yet");
    dialog.getByText(
      "Its target gets a message with no fields. Add fields in the target pane.",
    );
    expect(dialog.queryByText("Read no records")).toBeNull();
  });

  it("should show why a typed topic that the cluster does not list read no records", async () => {
    const { dev } = await renderWrite();
    const error = "The cluster does not let the client read the topic.";
    const { answered } = await answerScan(dev, "sample", {
      browse: { data: browseOf() },
      sample: {
        data: { topics: [{ ...MISSING, error, missing: false }], truncated: false },
      },
    });
    const dialog = await clickAddTargets();
    fireEvent.change(await dialog.findByPlaceholderText("Search topics"), {
      target: { value: TOPIC },
    });
    fireEvent.click(await dialog.findByRole("button", { name: `Write “${TOPIC}”` }));
    await answered;
    await dialog.findByText("Read no records");
    dialog.getByText(error);
    expect(dialog.queryByText("Topic does not exist yet")).toBeNull();
  });

  it("should show the records of a typed topic past the end of a full list", async () => {
    const { dev } = await renderWrite();
    const { answered } = await answerScan(dev, "sample", {
      browse: { data: { ...browseOf(), truncated: true } },
      sample: { data: sampleOfTopic(TOPIC, createShape(["pump-1"], 2, "/speed")) },
    });
    const dialog = await clickAddTargets();
    fireEvent.change(await dialog.findByPlaceholderText("Search topics"), {
      target: { value: TOPIC },
    });
    fireEvent.click(await dialog.findByRole("button", { name: `Write “${TOPIC}”` }));
    await answered;
    await dialog.findByText("/speed");
    expect(dialog.queryByText("Topic does not exist yet")).toBeNull();
  });

  it("should keep the records of a typed topic read before the topics list", async () => {
    const { dev } = await renderWrite();
    const scan = await createScanTask(client, dev);
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    try {
      const dialog = await clickAddTargets();
      const browse = await awaitCommand(streamer, scan.key, "browse");
      const { answered } = await answerNextCommand(client, scan, {
        type: "sample",
        data: sampleOfTopic(TOPIC, createShape(["pump-1"], 2, "/speed")),
      });
      fireEvent.change(await dialog.findByPlaceholderText("Search topics"), {
        target: { value: TOPIC },
      });
      fireEvent.click(await dialog.findByRole("button", { name: `Write “${TOPIC}”` }));
      await answered;
      await dialog.findByText("/speed");
      await answerCommand(client, scan, browse, { data: browseOf() });
      await dialog.findByText("1 topic");
      dialog.getByText("/speed");
    } finally {
      streamer.close();
    }
  });

  it("should count a topic of the task that the cluster does not list", async () => {
    const held = "plant.commands";
    const { target } = await createTarget([], { topic: held });
    const { dev } = await renderWrite([target]);
    const { answered } = await answerScan(dev, "sample", {
      browse: { data: browseOf(listTopic(TOPIC)) },
      sample: { data: { topics: [{ ...MISSING, topic: held }], truncated: false } },
    });
    const dialog = await clickAddTargets();
    await answered;
    await dialog.findByText("Topic does not exist yet");
    dialog.getByText("2 topics");
  });

  it("should show one Message section for a target whose message has no fields", async () => {
    const { target } = await createTarget([]);
    await renderWrite([target]);
    fireEvent.click(await screen.findByRole("treeitem", { name: /JSON.*No key/ }));
    await screen.findByText("Message has no fields");
    expect(screen.getAllByText("Message", { exact: true })).toHaveLength(1);
  });

  it("should send a picked message format from a new target", async () => {
    const { ic, message } = await createTarget([createField("speed", "float")]);
    const { dev, draft } = await renderWrite();
    const { answered } = await answerScan(dev, "sample", {
      browse: { data: browseOf(listTopic(TOPIC)) },
      sample: { data: sampleOfTopic(TOPIC, createShape(["pump-1"], 2, "/mode")) },
    });
    const dialog = await clickAddTargets();
    await answered;
    await dialog.findByText("/mode");
    fireEvent.click(dialog.getByRole("checkbox", { name: `Pick ${TOPIC}` }));
    await pickMessageFormat("Select message format", message.name, ic.name);
    await dialog.findByText(`Each picked topic sends “${message.name}”.`);
    fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
    await waitFor(async () =>
      expect(await retrieveTargets(draft.key)).toMatchObject([
        { topic: TOPIC, icd: ic.key, message: { message: message.key } },
      ]),
    );
    const [saved] = await client.icds.retrieve([ic.key]);
    expect(saved.messages).toHaveLength(1);
  });

  it("should show the shapes of the records again after the message format is cleared", async () => {
    const { ic, message } = await createTarget([createField("speed", "float")]);
    const { dev } = await renderWrite();
    const { answered } = await answerScan(dev, "sample", {
      browse: { data: browseOf(listTopic(TOPIC)) },
      sample: { data: sampleOfTopic(TOPIC, createShape(["pump-1"], 2, "/mode")) },
    });
    const dialog = await clickAddTargets();
    await answered;
    await dialog.findByText("/mode");
    await pickMessageFormat("Select message format", message.name, ic.name);
    const note = `Each picked topic sends “${message.name}”.`;
    await dialog.findByText(note);
    fireEvent.click(await findDialogTriggerByText(message.name));
    const dialogs = await screen.findAllByRole("dialog");
    const select = within(dialogs[dialogs.length - 1]);
    fireEvent.click(await select.findByText("No message format"));
    await dialog.findByText("/mode");
    expect(dialog.queryByText(note)).toBeNull();
  });

  it("should show the topic and record key of a target", async () => {
    const { target } = await createTarget([createField("speed", "float")], {
      recordKey: "pump-1",
    });
    await renderWrite([target]);
    fireEvent.click(await screen.findByRole("treeitem", { name: /JSON.*pump-1/ }));
    await screen.findByText("Produce");
    expect(screen.getByDisplayValue("plant.pumps")).toBeTruthy();
    expect(screen.getByDisplayValue("pump-1")).toBeTruthy();
  });

  it("should show each topic above its record keys, and each record key above its fields", async () => {
    const fields = [createField("speed", "float")];
    const { target: keyed } = await createTarget(fields, { recordKey: "pump-1" });
    const unkeyed = { ...keyed, key: "t2", recordKey: "" };
    await renderWrite([keyed, unkeyed]);
    const level = async (name: RegExp) =>
      (await screen.findByRole("treeitem", { name })).getAttribute("aria-level");
    expect(await level(/Topic.*plant\.pumps/)).toBe("1");
    expect(await level(/JSON.*pump-1/)).toBe("2");
    expect(await level(/JSON.*No key/)).toBe("2");
    expect(screen.getAllByRole("treeitem", { name: /^speed/ })).toHaveLength(2);
    expect(screen.getAllByRole("treeitem", { name: /Topic/ })).toHaveLength(1);
  });

  it("should keep a new target in the tree while its topic is typed", async () => {
    const { dev, draft } = await renderWrite();
    await addSampledTarget(dev, draft.key, [createShape(["pump-1"], 2, "/speed")]);
    fireEvent.change(await screen.findByDisplayValue(TOPIC), {
      target: { value: "plant" },
    });
    await screen.findByRole("treeitem", { name: /Topic.*plant$/ });
    expect(screen.getByRole("treeitem", { name: /JSON.*pump-1/ }).ariaSelected).toBe(
      "true",
    );
    expect(screen.getByDisplayValue("plant")).toBeTruthy();
  });

  it("should show the numbered name that deploying gives a command channel whose name another target takes", async () => {
    const { target } = await createTarget([createField("speed", "float")], {
      recordKey: "pump-1",
    });
    const { dev } = await renderWrite([
      target,
      { ...target, key: "t2", recordKey: "pump_1" },
    ]);
    const name = `${channel.escapeInvalidName(dev.name)}_plant_pumps_pump_1_speed_cmd`;
    const [first, second] = await screen.findAllByRole("treeitem", { name: /^speed/ });
    fireEvent.click(second);
    expect(await screen.findByDisplayValue(`${name}_2`)).toBeTruthy();
    fireEvent.click(first);
    expect(await screen.findByDisplayValue(name)).toBeTruthy();
  });

  it("should show the stored command channel that deploying keeps for a field with no channel, not a new one", async () => {
    const setpoint = createField("setpoint", "float");
    const { target } = await createTarget([setpoint]);
    const stored = await client.channels.create({
      name: uniqueName("kafka_cmd"),
      dataType: "float64",
      virtual: true,
    });
    const key = `${target.topic}#${target.recordKey}#${setpoint.key}`;
    const dev = await createKafkaDevice(client, {
      properties: { write: { [key]: stored.key } },
    });
    await renderWrite([target], { dev });
    fireEvent.click(await screen.findByRole("treeitem", { name: /^setpoint/ }));
    expect(await screen.findByDisplayValue(stored.name)).toBeTruthy();
    expect(screen.queryByRole("img", { name: "New channel" })).toBeNull();
  });

  it.each([
    ["topic", "plant.pumps", "plant.valves", (to: string) => `${to}#pump-1`],
    ["record key", "pump-1", "pump-2", (to: string) => `plant.pumps#${to}`],
  ])(
    "should give a deployed target new command channels when its %s changes, and leave the old ones to the old address",
    async (_, from, to, addressOf) => {
      const speed = createField("speed", "float");
      const { target } = await createTarget([speed], { recordKey: "pump-1" });
      const first = await renderWrite([target]);
      const [stored] = boundChannels(
        (await deployWrite(first.container, first.draft.key))[0],
      );
      expect(stored).not.toBe(0);
      first.unmount();
      const { container } = await renderTaskFormTab(
        Kafka.Task.FORMS[Kafka.Task.WRITE_TYPE],
        { client, taskKey: first.draft.key },
      );
      fireEvent.click(await screen.findByRole("treeitem", { name: /JSON.*pump-1/ }));
      fireEvent.change(await screen.findByDisplayValue(from), {
        target: { value: to },
      });
      const [next] = boundChannels((await deployWrite(container, first.draft.key))[0]);
      expect(next).not.toBe(0);
      expect(next).not.toBe(stored);
      expect(await retrieveWriteRecord(first.dev.key)).toEqual({
        [`plant.pumps#pump-1#${speed.key}`]: stored,
        [`${addressOf(to)}#${speed.key}`]: next,
      });
    },
  );

  it("should show the planned command channel name of a field after a rename", async () => {
    const { target } = await createTarget([createField("speed", "float")], {
      recordKey: "pump-1",
    });
    const { dev } = await renderWrite([
      target,
      { ...target, key: "t2", recordKey: "pump_1" },
    ]);
    const prefix = `${channel.escapeInvalidName(dev.name)}_plant_pumps_pump_1`;
    const [, second] = await screen.findAllByRole("treeitem", { name: /^speed/ });
    fireEvent.click(second);
    await screen.findByDisplayValue(`${prefix}_speed_cmd_2`);
    fireEvent.change(screen.getByDisplayValue("speed"), { target: { value: "rate" } });
    expect(await screen.findByDisplayValue(`${prefix}_rate_cmd_2`)).toBeTruthy();
  });

  it("should name the topic in the details header when a topic is selected", async () => {
    const { target } = await createTarget([createField("speed", "float")], {
      recordKey: "pump-1",
    });
    await renderWrite([target]);
    fireEvent.click(
      await screen.findByRole("treeitem", { name: /Topic.*plant\.pumps/ }),
    );
    expect(await screen.findByText("Select a record key of the topic")).toBeTruthy();
    expect(screen.queryByText("Details")).toBeNull();
    expect(screen.getAllByText("plant.pumps")).toHaveLength(2);
  });

  it("should name a new command channel after the device, topic, record key, and field", async () => {
    const { target } = await createTarget([createField("speed", "float")], {
      recordKey: "pump-1",
    });
    const { dev } = await renderWrite([target]);
    fireEvent.click(await screen.findByRole("treeitem", { name: /^speed/ }));
    const name = `${channel.escapeInvalidName(dev.name)}_plant_pumps_pump_1_speed_cmd`;
    expect(await screen.findByDisplayValue(name)).toBeTruthy();
  });

  it("should duplicate a target with its message and bindings, but no command channels", async () => {
    const { target } = await createTarget([createField("speed", "float")]);
    const cmd = await client.channels.create({
      name: uniqueName("kafka_cmd"),
      dataType: "float64",
      virtual: true,
    });
    const [{ field }] = target.message.bindings;
    const bound: Kafka.Task.WriteTarget = {
      ...target,
      message: {
        ...target.message,
        bindings: [{ type: "channel", field, channel: cmd.key }],
      },
    };
    const { draft } = await renderWrite([bound]);
    fireEvent.contextMenu(
      await screen.findByRole("treeitem", { name: /JSON.*No key/ }),
    );
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(async () => {
      const targets = await retrieveTargets(draft.key);
      expect(targets).toMatchObject([
        { key: "t1", message: bound.message },
        { topic: target.topic, icd: target.icd, message: target.message },
      ]);
      expect(targets[1].key).not.toBe("t1");
    });
  });

  it("should refuse to deploy an enabled target with no topic", async () => {
    const { target } = await createTarget([createField("speed", "float")], {
      topic: "",
    });
    const { container } = await renderWrite([target]);
    await screen.findByRole("treeitem", { name: /Topic.*New topic/ });
    await clickDeploy(container);
    await screen.findByText("Topic is required");
  });
  describe("adding fields", () => {
    const retrieveFields = async (ic: icd.ICD, message: icd.Message) => {
      const [retrieved] = await client.icds.retrieve([ic.key]);
      return ICD.jsonFields(retrieved.messages.find(({ key }) => key === message.key));
    };

    const addField = async () => {
      fireEvent.click(await screen.findByRole("treeitem", { name: /JSON.*No key/ }));
      fireEvent.click(await screen.findByRole("button", { name: "Add field" }));
    };

    it("should add a field to the message in its ICD, with the focus on its name, then put it on a channel", async () => {
      const speed = createField("speed", "float");
      const { target, ic, message } = await createTarget([speed]);
      const { draft } = await renderWrite([target]);
      await addField();
      const name = await screen.findByDisplayValue("Field 2");
      await waitFor(() => expect(document.activeElement).toBe(name));
      screen.getByDisplayValue("/field_2");
      fireEvent.change(screen.getByPlaceholderText("/temperature"), {
        target: { value: "/limit" },
      });
      let added: icd.JSONField | undefined;
      await waitFor(async () => {
        added = (await retrieveFields(ic, message)).find((f) => f.pointer === "/limit");
        expect(added).toBeDefined();
      });
      await waitFor(async () =>
        expect((await retrieveTargets(draft.key))[0]).toMatchObject({
          message: {
            bindings: [
              { type: "channel", field: speed.key },
              { type: "channel", field: added?.key, channel: 0 },
            ],
          },
        }),
      );
    });

    it("should move the pointer of an added field when the field is renamed", async () => {
      const { target } = await createTarget([createField("speed", "float")]);
      await renderWrite([target]);
      await addField();
      fireEvent.change(await screen.findByDisplayValue("Field 2"), {
        target: { value: "Pump Pressure" },
      });
      await screen.findByDisplayValue("/pump_pressure");
    });

    it("should move the pointer of an added field renamed after another item was selected", async () => {
      const { target } = await createTarget([createField("speed", "float")]);
      await renderWrite([target]);
      await addField();
      await screen.findByDisplayValue("Field 2");
      fireEvent.click(screen.getByRole("treeitem", { name: /^speed/ }));
      fireEvent.click(await screen.findByRole("treeitem", { name: /^Field 2/ }));
      fireEvent.change(await screen.findByDisplayValue("Field 2"), {
        target: { value: "pressure" },
      });
      await screen.findByDisplayValue("/pressure");
    });

    it("should keep the pointer of an added field once the user edits it", async () => {
      const { target } = await createTarget([createField("speed", "float")]);
      await renderWrite([target]);
      await addField();
      fireEvent.change(await screen.findByDisplayValue("/field_2"), {
        target: { value: "/limit" },
      });
      fireEvent.change(screen.getByDisplayValue("Field 2"), {
        target: { value: "pressure" },
      });
      await screen.findByDisplayValue("pressure");
      screen.getByDisplayValue("/limit");
    });

    it("should number the pointer of an added field renamed onto the pointer of another field", async () => {
      const { target } = await createTarget([createField("speed", "float")]);
      await renderWrite([target]);
      await addField();
      fireEvent.change(await screen.findByDisplayValue("Field 2"), {
        target: { value: "Speed" },
      });
      await screen.findByDisplayValue("/speed_2");
    });

    it("should move the numbered pointer of an added field when it is renamed again", async () => {
      const { target } = await createTarget([createField("speed", "float")]);
      await renderWrite([target]);
      await addField();
      fireEvent.change(await screen.findByDisplayValue("Field 2"), {
        target: { value: "Speed" },
      });
      fireEvent.change(await screen.findByDisplayValue("Speed"), {
        target: { value: "Speedometer" },
      });
      await screen.findByDisplayValue("/speedometer");
    });

    it("should keep the pointer of an added field once the ICD editor edits it", async () => {
      const { target, ic, message } = await createTarget([
        createField("speed", "float"),
      ]);
      await renderWrite([target]);
      await addField();
      const added = await waitFor(async () => {
        const found = (await retrieveFields(ic, message)).find(
          (f) => f.name === "Field 2",
        );
        if (found == null) throw new Error("the added field is not in the ICD");
        return found;
      });
      await setField(
        client,
        { icd: ic.key, message: message.key },
        { ...added, pointer: "/limit" },
      );
      await screen.findByDisplayValue("/limit");
      fireEvent.change(screen.getByDisplayValue("Field 2"), {
        target: { value: "pressure" },
      });
      await screen.findByDisplayValue("pressure");
      screen.getByDisplayValue("/limit");
    });

    it("should keep the pointer of a field the message already had when it is renamed", async () => {
      const { target } = await createTarget([createField("speed", "float")]);
      await renderWrite([target]);
      fireEvent.click(await screen.findByRole("treeitem", { name: /^speed/ }));
      fireEvent.change(await screen.findByDisplayValue("speed"), {
        target: { value: "pressure" },
      });
      await screen.findByDisplayValue("pressure");
      screen.getByDisplayValue("/speed");
    });

    it("should add a first field to a message with no fields", async () => {
      const { target } = await createTarget([]);
      await renderWrite([target]);
      fireEvent.click(await screen.findByRole("treeitem", { name: /JSON.*No key/ }));
      await screen.findByText("Message has no fields");
      fireEvent.click(screen.getByRole("button", { name: "Add field" }));
      await screen.findByRole("treeitem", { name: /^Field 1/ });
      screen.getByDisplayValue("Field 1");
    });
  });

  describe("deploying against a live Core", () => {
    it("should create a command channel for each channel binding and keep it in the cluster", async () => {
      const speed = createField("speed", "float");
      const open = createField("open", "boolean");
      const { target } = await createTarget([speed, open], { recordKey: "Pump_1" });
      const { container, dev, draft } = await renderWrite([target]);
      const [deployed] = await deployWrite(container, draft.key);
      const keys = boundChannels(deployed);
      const channels = await client.channels.retrieve(keys);
      const prefix = `${channel.escapeInvalidName(dev.name)}_plant_pumps_Pump_1`;
      expect(channels.map(({ name }) => name)).toEqual([
        `${prefix}_speed_cmd`,
        `${prefix}_open_cmd`,
      ]);
      expect(channels.map((ch) => ch.dataType.toString())).toEqual([
        "float64",
        "uint8",
      ]);
      expect(await retrieveWriteRecord(dev.key)).toEqual({
        [`plant.pumps#Pump_1#${speed.key}`]: keys[0],
        [`plant.pumps#Pump_1#${open.key}`]: keys[1],
      });
    });

    it("should give two record keys of one topic their own command channels", async () => {
      const speed = createField("speed", "float");
      const { target } = await createTarget([speed], { recordKey: "pump1" });
      const second = { ...target, key: "t2", recordKey: "pump2" };
      const { container, dev, draft } = await renderWrite([target, second]);
      const [first, other] = (await deployWrite(container, draft.key)).map(
        boundChannels,
      );
      expect(first[0]).not.toEqual(other[0]);
      expect(await retrieveWriteRecord(dev.key)).toEqual({
        [`plant.pumps#pump1#${speed.key}`]: first[0],
        [`plant.pumps#pump2#${speed.key}`]: other[0],
      });
    });

    it("should number the command channel of a record key that escapes to the name of another", async () => {
      const speed = createField("speed", "float");
      const { target } = await createTarget([speed], { recordKey: "pump-1" });
      const second = { ...target, key: "t2", recordKey: "pump_1" };
      const { container, dev, draft } = await renderWrite([target, second]);
      const deployed = await deployWrite(container, draft.key);
      const channels = await client.channels.retrieve(deployed.flatMap(boundChannels));
      const name = `${channel.escapeInvalidName(dev.name)}_plant_pumps_pump_1_speed_cmd`;
      expect(channels.map((ch) => ch.name)).toEqual([name, `${name}_2`]);
    });

    it("should bind a field to the command channel that the cluster keeps for it", async () => {
      const speed = createField("speed", "float");
      const { target } = await createTarget([speed]);
      const index = await client.channels.create({
        name: uniqueName("kafka_cmd_time"),
        dataType: "timestamp",
        isIndex: true,
      });
      const kept = await client.channels.create({
        name: uniqueName("kafka_cmd"),
        dataType: "float64",
        index: index.key,
      });
      const dev = await createKafkaDevice(client, {
        properties: { write: { [`plant.pumps##${speed.key}`]: kept.key } },
      });
      const { container, draft } = await renderWrite([target], { dev });
      const [deployed] = await deployWrite(container, draft.key);
      expect(boundChannels(deployed)).toEqual([kept.key]);
    });
  });
});
