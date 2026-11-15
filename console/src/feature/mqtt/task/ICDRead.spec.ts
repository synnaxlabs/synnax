// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { DataType, type payload, uuid } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker } from "@/feature/mqtt/testutil";
import {
  clickDeploy,
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
  fields: icd.JSONField[];
  index?: string;
  /** The channel each field is bound to before the deploy. */
  bound?: Record<string, number>;
  properties?: Partial<MQTT.Device.Properties>;
}

// The form body renders against a configured broker only.
const renderICDRead = async ({
  fields,
  index,
  bound = {},
  properties,
}: RenderParams) => {
  const dev = await createBroker(client, { properties });
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
  const config = MQTT.Task.ICD_READ_SCHEMAS.config.parse({
    device: dev.key,
    entries: [
      {
        icd: format.key,
        topic: TOPIC,
        messages: [{ message: message.key, index, bindings }],
      },
    ],
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

const deploy = async (container: HTMLElement, key: string) =>
  await deployAndAwaitTask(client, container, key, MQTT.Task.ICD_READ_SCHEMAS);

const retrieveTopicProps = async (key: string) =>
  (await client.devices.retrieve({ key, schemas: MQTT.Device.SCHEMAS })).properties
    .read[TOPIC];

describe("MQTT ICD read form", () => {
  describe("deploying against a live Core", () => {
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
