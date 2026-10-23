// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { kafka, type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Kafka } from "@/feature/kafka";
import { createKafkaDevice } from "@/feature/kafka/testutil";
import {
  awaitEditableForm,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  selectFromDropdown,
} from "@/platform/task/testutil";

const client = createTestClient();

const renderRead = async (options: RenderTaskFormTabOptions) => {
  const rendered = await renderTaskFormTab(Kafka.Task.Read, options);
  await awaitEditableForm();
  return rendered;
};

const ZERO_DRAFT: task.New<Kafka.Task.ReadSchemas> = {
  name: "Kafka read task",
  type: Kafka.Task.READ_TYPE,
  config: Kafka.Task.READ_SCHEMAS.config.parse({}),
};

const createDraft = async (
  client: Synnax,
  config: task.Payload<Kafka.Task.ReadSchemas>["config"],
) => await client.tasks.create({ ...ZERO_DRAFT, config }, Kafka.Task.READ_SCHEMAS);

const createField = (
  key: string,
  pointer: string,
  overrides: Partial<Kafka.Task.ReadField> = {},
): Kafka.Task.ReadField => ({
  ...kafka.readFieldZ.parse({}),
  key,
  pointer,
  ...overrides,
});

const createRecord = (
  key: string,
  fields: Kafka.Task.ReadField[],
  overrides: Partial<Kafka.Task.ReadRecord> = {},
): Kafka.Task.ReadRecord => ({
  ...kafka.readRecordZ.parse({}),
  key,
  fields,
  ...overrides,
});

const commitPointer = (value: string): void => {
  const input = screen.getByPlaceholderText("/temperature");
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

/** Renders a draft with one record that reads any key, and adds a field to it. */
const renderWithField = async () => {
  const dev = await createKafkaDevice(client);
  const draft = await createDraft(client, {
    ...Kafka.Task.READ_SCHEMAS.config.parse({}),
    device: dev.key,
    records: [createRecord("r", [])],
  });
  await renderRead({ client, taskKey: draft.key });
  await screen.findByText(dev.name);
  fireEvent.click((await screen.findAllByRole("button", { name: "Add field" }))[0]);
  await screen.findByPlaceholderText("/temperature");
};

describe("Read", () => {
  it("should add a record that reads any key", async () => {
    const dev = await createKafkaDevice(client);
    const draft = await createDraft(client, {
      ...Kafka.Task.READ_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    await renderRead({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    fireEvent.click(await screen.findByRole("button", { name: "Add record" }));
    await screen.findByPlaceholderText("Any key");
    expect(screen.getAllByText("Any key").length).toBeGreaterThan(0);
  });

  it("should add a field, edit its pointer, and show the enum mapping editor", async () => {
    await renderWithField();
    await screen.findByText("Enum mapping");
    commitPointer("/payload/temp");
    await waitFor(() => expect(screen.getAllByText("/payload/temp")).toHaveLength(2));
  });

  it("should hide the enum mapping and show the time format for a timestamp field", async () => {
    await renderWithField();
    await screen.findByText("Enum mapping");
    await selectFromDropdown("float64", "Timestamp");
    await screen.findByText("Time format");
    await waitFor(() => expect(screen.queryByText("Enum mapping")).toBeNull());
  });

  it("should offer the string type and hide the enum mapping for it", async () => {
    await renderWithField();
    await screen.findByText("Enum mapping");
    await selectFromDropdown("float64", "String");
    await waitFor(() => expect(screen.queryByText("Enum mapping")).toBeNull());
  });

  it("should stamp a record by a value it carries, hidden from the tree", async () => {
    const dev = await createKafkaDevice(client);
    const draft = await createDraft(client, {
      ...Kafka.Task.READ_SCHEMAS.config.parse({}),
      device: dev.key,
      records: [createRecord("r", [])],
    });
    await renderRead({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    fireEvent.click(await screen.findByText("Record value"));
    const pointer = await screen.findByPlaceholderText("/timestamp");
    expect((pointer as HTMLInputElement).value).toBe("/timestamp");
    await screen.findByText("Time format");
    // The index field is edited here, never listed as a field of the record.
    expect(screen.queryByText("New field")).toBeNull();
    expect(screen.queryByText("/timestamp")).toBeNull();
    fireEvent.click(screen.getByText("Arrival time"));
    await waitFor(() => expect(screen.queryByPlaceholderText("/timestamp")).toBeNull());
  });

  it("should create an index per record and a channel per field on deploy", async () => {
    const dev = await createKafkaDevice(client);
    const draft = await createDraft(client, {
      ...Kafka.Task.READ_SCHEMAS.config.parse({}),
      device: dev.key,
      topic: "readings",
      records: [
        createRecord(
          "any",
          [
            createField("ts", "/ts", { dataType: "timestamp", timeFormat: "unix_ms" }),
            createField("temp", "/temp", { name: "kafka_temp" }),
          ],
          { index: "ts" },
        ),
        createRecord("motor", [createField("rpm", "/rpm")], { recordKey: "motor" }),
      ],
    });
    const { container } = await renderRead({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Kafka.Task.READ_SCHEMAS,
    );
    expect(created.rack).toBe(dev.rack);
    const [any, motor] = created.config.records;
    const [ts, temp] = any.fields;
    const [rpm] = motor.fields;

    const index = await client.channels.retrieve(ts.channel);
    expect(index.isIndex).toBe(true);
    expect(index.name).toBe(`${dev.name}_time`);
    const tempCh = await client.channels.retrieve(temp.channel);
    expect(tempCh.name).toBe("kafka_temp");
    expect(tempCh.index).toBe(index.key);
    expect(tempCh.dataType.toString()).toBe("float64");

    const rpmCh = await client.channels.retrieve(rpm.channel);
    expect(rpmCh.name).toBe(`${dev.name}_motor_rpm`);
    const motorIndex = await client.channels.retrieve(rpmCh.index);
    expect(motorIndex.isIndex).toBe(true);
    expect(motorIndex.name).toBe(`${dev.name}_motor_time`);
  });

  it("should keep the channels of fields that already have them on deploy", async () => {
    const dev = await createKafkaDevice(client);
    const index = await client.channels.create({
      name: `${dev.name}_existing_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    const existing = await client.channels.create({
      name: `${dev.name}_existing`,
      dataType: "float64",
      index: index.key,
    });
    const draft = await createDraft(client, {
      ...Kafka.Task.READ_SCHEMAS.config.parse({}),
      device: dev.key,
      topic: "readings",
      records: [
        createRecord("r", [
          createField("a", "/a", { channel: existing.key }),
          createField("b", "/b"),
        ]),
      ],
    });
    const { container } = await renderRead({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Kafka.Task.READ_SCHEMAS,
    );
    const [a, b] = created.config.records[0].fields;
    expect(a.channel).toBe(existing.key);
    const bCh = await client.channels.retrieve(b.channel);
    expect(bCh.index).toBe(index.key);
  });
});
