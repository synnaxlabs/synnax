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
  type task,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Kafka } from "@/feature/kafka";
import { createKafkaDevice } from "@/feature/kafka/testutil";
import { ICD } from "@/platform/icd";
import {
  clickDeploy,
  countFailedSaves,
  deployAndAwaitTask,
  preloadForms,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";

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

// The add button renders once the form leaves its loading preview.
const addTarget = async () =>
  fireEvent.click(await screen.findByRole("button", { name: "Add target" }));

describe("Kafka Write form", () => {
  // This first test pays the one-time load of the lazy form, which takes over 10 s on
  // a loaded CI runner.
  it("should show the empty state and add + select a target", async () => {
    await renderWrite();
    await screen.findByText("Add a target to send records to a topic");
    await addTarget();
    await screen.findByText("Produce");
    expect(screen.getByText("Message has no fields")).toBeTruthy();
  }, 30_000);

  it("should save a new target with an empty message in an ICD named after the task", async () => {
    let statuses: Status.NotificationSpec[] = [];
    const { draft } = await renderWrite([], { onStatuses: (s) => (statuses = s) });
    await addTarget();
    const target = await waitFor(async () => {
      const [t] = await retrieveTargets(draft.key);
      expect(t).toBeDefined();
      return t;
    });
    expect(target.message.bindings).toEqual([]);
    const [ic] = await client.icds.retrieve([target.icd]);
    expect(ic.name).toBe(Kafka.Task.DEFAULT_WRITE_NAME);
    expect(ic.messages).toMatchObject([
      {
        key: target.message.message,
        name: "Untitled",
        payload: { format: "json", fields: [] },
      },
    ]);
    expect(countFailedSaves(statuses)).toBe(0);
  });

  it("should add the message of a new target to the ICD of the most targets", async () => {
    const { ic, target } = await createTarget([createField("speed", "float")]);
    const { draft } = await renderWrite([target]);
    await screen.findByRole("treeitem", { name: /Topic.*plant\.pumps/ });
    await addTarget();
    const added = await waitFor(async () => {
      const targets = await retrieveTargets(draft.key);
      expect(targets).toHaveLength(2);
      return targets[1];
    });
    expect(added.icd).toBe(ic.key);
    await waitFor(async () => {
      const [saved] = await client.icds.retrieve([ic.key]);
      expect(saved.messages.at(-1)).toMatchObject({
        key: added.message.message,
        name: "Untitled",
      });
    });
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
