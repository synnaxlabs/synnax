// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, mqtt } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { DataType, type payload, uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker, openAddData, openScanner } from "@/feature/mqtt/testutil";
import {
  clickDeploy,
  clickFailedDeploy,
  deployAndAwaitTask,
  renderTaskFormTab,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

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

interface RenderParams {
  /** The fields of the topic's message format. Absent renders a task with no topics. */
  fields?: icd.JSONField[];
  index?: string;
  /** The channel each field is bound to before the deploy. */
  bound?: Record<string, number>;
  properties?: Partial<MQTT.Device.Properties>;
}

const createEntry = async (
  fields: icd.JSONField[],
  index: string | undefined,
  bound: Record<string, number>,
): Promise<MQTT.Task.ICDReadEntry> => {
  const message: icd.Message = {
    key: uuid.create(),
    name: TOPIC,
    payload: { format: "json", fields },
  };
  const format = await client.icds.create({
    name: uniqueName("format"),
    messages: [message],
  });
  const bindings = Object.entries(bound).map(([field, channel]) => ({
    field,
    channel,
  }));
  return mqtt.icdReadEntryZ.parse({
    icd: format.key,
    topic: TOPIC,
    messages: [{ message: message.key, index, bindings }],
  });
};

// The form body renders against a configured broker only.
const renderICDRead = async ({
  fields,
  index,
  bound = {},
  properties,
}: RenderParams = {}) => {
  const dev = await createBroker(client, { properties });
  const config = MQTT.Task.ICD_READ_SCHEMAS.config.parse({
    device: dev.key,
    entries: fields == null ? [] : [await createEntry(fields, index, bound)],
  });
  const draft = await client.tasks.create(
    { name: "MQTT ICD read task", type: MQTT.Task.ICD_READ_TYPE, config },
    MQTT.Task.ICD_READ_SCHEMAS,
  );
  const statuses: Status.NotificationSpec[] = [];
  const result = await renderTaskFormTab(MQTT.Task.ICDRead, {
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
  (await client.tasks.retrieve({ key, schemas: MQTT.Task.ICD_READ_SCHEMAS })).config
    .entries;

const deploy = async (container: HTMLElement, key: string) =>
  await deployAndAwaitTask(client, container, key, MQTT.Task.ICD_READ_SCHEMAS);

const retrieveTopicProps = async (key: string) =>
  (await client.devices.retrieve({ key, schemas: MQTT.Device.SCHEMAS })).properties
    .read[TOPIC];

describe("MQTT ICD read form", () => {
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
    const { dev, draft } = await renderICDRead();
    await screen.findByText("Add data to read a topic");
    const scanner = await openScanner(client, dev.rack);
    let dialog = await openAddData();
    await scanner.answer({ data: { topics: TOPICS, truncated: false } });
    fireEvent.click(await dialog.findByRole("checkbox", { name: "Pick oven" }));
    fireEvent.click(dialog.getByRole("button", { name: "Add 1 field" }));
    const [entry] = await waitFor(async () => {
      const entries = await retrieveEntries(draft.key);
      expect(entries).toHaveLength(1);
      return entries;
    });
    dialog = await openAddData();
    await scanner.answer({ data: { topics: TOPICS, truncated: false } });
    scanner.close();
    fireEvent.click(await dialog.findByRole("checkbox", { name: "Record label" }));
    fireEvent.click(dialog.getByRole("button", { name: "Add 1 field" }));
    await waitFor(async () => {
      const format = await client.icds.retrieve(entry.icd);
      expect(format.name).toBe("MQTT ICD read task");
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
    });
    expect(await retrieveEntries(draft.key)).toHaveLength(1);
  });

  it("should disable and enable a topic through the context menu", async () => {
    const { draft } = await renderICDRead({
      fields: [createField("temperature", "float")],
    });
    const item = () => screen.findByRole("option", { name: /plant\/oven/ });
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
    const { draft } = await renderICDRead({
      fields: [createField("temperature", "float"), time],
    });
    fireEvent.click(await screen.findByRole("button", { name: "Payload value" }));
    await waitFor(async () =>
      expect((await retrieveEntries(draft.key))[0].messages[0].index).toBe(time.key),
    );
  });

  it("should rename the message of a topic when the topic changes", async () => {
    const { draft } = await renderICDRead({
      fields: [createField("temperature", "float")],
    });
    const input = await screen.findByDisplayValue(TOPIC);
    fireEvent.change(input, { target: { value: "plant/kiln" } });
    fireEvent.blur(input);
    const [entry] = await retrieveEntries(draft.key);
    await waitFor(async () =>
      expect((await client.icds.retrieve(entry.icd)).messages[0].name).toBe(
        "plant/kiln",
      ),
    );
  });

  describe("deploying against a live Core", () => {
    it("should not deploy while the message format holds an invalid edit", async () => {
      const { container, dev } = await renderICDRead({
        fields: [createField("temperature", "float")],
      });
      const name = await screen.findByLabelText("Name 1");
      fireEvent.change(name, { target: { value: "" } });
      fireEvent.blur(name);
      await clickFailedDeploy(container);
      expect(await retrieveTopicProps(dev.key)).toBeUndefined();
    });

    it("should create and bind an index channel and a data channel", async () => {
      const time = createTimeField();
      const temperature = createField("temperature", "float");
      const { container, dev, draft } = await renderICDRead({
        fields: [time, temperature],
        index: time.key,
      });
      const created = await deploy(container, draft.key);
      const { index, channels } = await retrieveTopicProps(dev.key);
      const indexCh = await client.channels.retrieve(index);
      expect(indexCh.isIndex).toBe(true);
      expect(indexCh.name).toBe(`${dev.name}_plant_oven_time`);
      const dataCh = await client.channels.retrieve(channels["/temperature"]);
      expect(dataCh.index).toBe(index);
      expect(dataCh.dataType.equals(DataType.FLOAT64)).toBe(true);
      expect(dataCh.name).toBe(`${dev.name}_plant_oven_temperature`);
      expect(created.config.entries[0].messages).toEqual([
        {
          message: expect.any(String),
          index: time.key,
          bindings: [
            { field: time.key, channel: index },
            { field: temperature.key, channel: dataCh.key },
          ],
        },
      ]);
    });

    it("should create virtual channels with no index for string fields", async () => {
      const mode = createField("mode", "string");
      const { container, dev, draft } = await renderICDRead({ fields: [mode] });
      await deploy(container, draft.key);
      const { index, channels } = await retrieveTopicProps(dev.key);
      expect(index).toBe(0);
      const ch = await client.channels.retrieve(channels["/mode"]);
      expect(ch.virtual).toBe(true);
      expect(ch.dataType.equals(DataType.STRING)).toBe(true);
    });

    it("should refuse a format that mixes string and number fields", async () => {
      const { container, dev, statuses } = await renderICDRead({
        fields: [createField("mode", "string"), createField("temperature", "float")],
      });
      await clickDeploy(container);
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            message: "Failed to start task",
            description: expect.stringContaining(`In ${TOPIC}, mode: a string`),
          }),
        ),
      );
      expect(await retrieveTopicProps(dev.key)).toBeUndefined();
    });

    it("should refuse a format with no fields", async () => {
      const { container, statuses } = await renderICDRead({ fields: [] });
      await clickDeploy(container);
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            message: "Failed to start task",
            description: `The message format of ${TOPIC} has no fields`,
          }),
        ),
      );
    });

    it("should replace only the channel of a field whose kind changed", async () => {
      const time = await client.channels.create({
        name: uniqueName("time"),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const [temperatureCh, countCh] = await client.channels.create([
        {
          name: uniqueName("temperature"),
          dataType: DataType.FLOAT64,
          index: time.key,
        },
        { name: uniqueName("count"), dataType: DataType.FLOAT64, index: time.key },
      ]);
      const timeField = createTimeField();
      const temperature = createField("temperature", "float");
      const count = createField("count", "integer");
      const { container, dev, draft } = await renderICDRead({
        fields: [timeField, temperature, count],
        index: timeField.key,
        bound: {
          [timeField.key]: time.key,
          [temperature.key]: temperatureCh.key,
          [count.key]: countCh.key,
        },
        properties: {
          read: {
            [TOPIC]: {
              index: time.key,
              channels: { "/temperature": temperatureCh.key, "/count": countCh.key },
            },
          },
        },
      });
      const created = await deploy(container, draft.key);
      const { channels } = await retrieveTopicProps(dev.key);
      expect(channels["/temperature"]).toBe(temperatureCh.key);
      expect(channels["/count"]).not.toBe(countCh.key);
      const replaced = await client.channels.retrieve(channels["/count"]);
      expect(replaced.dataType.equals(DataType.INT64)).toBe(true);
      expect(replaced.index).toBe(time.key);
      expect(created.config.entries[0].messages[0].bindings).toEqual([
        { field: timeField.key, channel: time.key },
        { field: temperature.key, channel: temperatureCh.key },
        { field: count.key, channel: replaced.key },
      ]);
    });
  });
});
