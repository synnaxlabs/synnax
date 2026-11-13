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
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Kafka } from "@/feature/kafka";
import { createKafkaDevice } from "@/feature/kafka/testutil";
import {
  awaitEditableForm,
  deployAndAwaitTask,
  findChannelListItem,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  selectFromDropdown,
} from "@/platform/task/testutil";
import { getIconButton, uniqueName } from "@/testutil";

const client = createTestClient();

const renderWrite = async (options: RenderTaskFormTabOptions) => {
  const rendered = await renderTaskFormTab(Kafka.Task.Write, options);
  await awaitEditableForm();
  return rendered;
};

const ZERO_DRAFT: task.New<Kafka.Task.WriteSchemas> = {
  name: "Kafka write task",
  type: Kafka.Task.WRITE_TYPE,
  config: Kafka.Task.WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (
  client: Synnax,
  config: task.Payload<Kafka.Task.WriteSchemas>["config"],
) => await client.tasks.create({ ...ZERO_DRAFT, config }, Kafka.Task.WRITE_SCHEMAS);

describe("Write", () => {
  it("should add a channel that writes numbers with an enum mapping", async () => {
    const dev = await createKafkaDevice(client);
    const draft = await createDraft(client, {
      ...Kafka.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    const { container } = await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    fireEvent.click(getIconButton(container, "add"));
    await screen.findByText("Enum mapping");
  });

  it("should offer the kinds a float channel can be written as", async () => {
    const dev = await createKafkaDevice(client);
    const ch = await client.channels.create({
      name: uniqueName("kafka_out"),
      dataType: "float64",
      virtual: true,
    });
    const draft = await createDraft(client, {
      ...Kafka.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
      topic: "samples",
      channels: [
        { ...kafka.writeChannelZ.parse({ type: "float" }), key: "c1", channel: ch.key },
      ],
    });
    const { container } = await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    fireEvent.click((await screen.findAllByText(ch.name))[0]);
    await screen.findByText("Write as");
    await selectFromDropdown("Number", "Boolean");
    fireEvent.click(await screen.findByRole("button", { name: "Add mapping" }));
    fireEvent.change(await screen.findByPlaceholderText("String (e.g. ON)"), {
      target: { value: "ON" },
    });
    fireEvent.click(screen.getByRole("checkbox", { name: "Value 1" }));
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Kafka.Task.WRITE_SCHEMAS,
    );
    expect(created.config.channels[0]).toMatchObject({
      type: "boolean",
      enumValues: [{ label: "ON", value: true }],
    });
  });

  it("should show the channel name in the list once a channel is selected", async () => {
    const dev = await createKafkaDevice(client);
    const ch = await client.channels.create({
      name: uniqueName("kafka_out"),
      dataType: "float64",
      virtual: true,
    });
    const draft = await createDraft(client, {
      ...Kafka.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
      channels: [
        { ...kafka.writeChannelZ.parse({ type: "float" }), key: "c1", channel: ch.key },
      ],
    });
    await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    await waitFor(() => expect(screen.getAllByText(ch.name).length).toBeGreaterThan(0));
  });

  it("should omit cleared pointers from the record on deploy", async () => {
    const dev = await createKafkaDevice(client);
    const ch = await client.channels.create({
      name: uniqueName("kafka_out"),
      dataType: "float64",
      virtual: true,
    });
    const draft = await createDraft(client, {
      ...Kafka.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
      topic: "samples",
      record: { ...kafka.recordZ.parse({}), channelPointer: "", timestampPointer: "" },
      channels: [
        { ...kafka.writeChannelZ.parse({ type: "float" }), key: "c1", channel: ch.key },
      ],
    });
    const { container } = await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Kafka.Task.WRITE_SCHEMAS,
    );
    expect(created.rack).toBe(dev.rack);
    expect(created.config.record.channelPointer).toBeUndefined();
    expect(created.config.record.timestampPointer).toBeUndefined();
    expect(created.config.record.valuePointer).toBe("/value");
    expect(created.config.channels[0].channel).toBe(ch.key);
  });

  it("should deploy an empty value pointer that writes raw bytes", async () => {
    const dev = await createKafkaDevice(client);
    const ch = await client.channels.create({
      name: uniqueName("kafka_out"),
      dataType: "bytes",
      virtual: true,
    });
    const draft = await createDraft(client, {
      ...Kafka.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
      topic: "samples",
      record: {
        ...kafka.recordZ.parse({}),
        valuePointer: "",
        channelPointer: "",
        timestampPointer: "",
      },
      channels: [
        { ...kafka.writeChannelZ.parse({ type: "bytes" }), key: "c1", channel: ch.key },
      ],
    });
    const { container } = await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Kafka.Task.WRITE_SCHEMAS,
    );
    expect(created.config.record.valuePointer).toBe("");
    expect(created.config.record.channelPointer).toBeUndefined();
  });

  it("should show no channel, never a new one, before a channel is picked", async () => {
    const dev = await createKafkaDevice(client);
    const draft = await createDraft(client, {
      ...Kafka.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
      channels: [{ ...kafka.writeChannelZ.parse({ type: "float" }), key: "c1" }],
    });
    await renderWrite({ client, taskKey: draft.key });
    const item = await findChannelListItem("No channel");
    expect(within(item).queryByRole("img", { name: "New channel" })).toBeNull();
  });

  // A Core clears the channel of a row whose channel it no longer holds.
  it("should deploy when a disabled row has no channel", async () => {
    const dev = await createKafkaDevice(client);
    const ch = await client.channels.create({
      name: uniqueName("kafka_out"),
      dataType: "float64",
      virtual: true,
    });
    const draft = await createDraft(client, {
      ...Kafka.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
      topic: "samples",
      channels: [
        { ...kafka.writeChannelZ.parse({ type: "float" }), key: "c1", channel: ch.key },
        {
          ...kafka.writeChannelZ.parse({ type: "float" }),
          key: "c2",
          channel: 0,
          disabled: true,
        },
      ],
    });
    const { container } = await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Kafka.Task.WRITE_SCHEMAS,
    );
    expect(created.config.channels.map((c) => c.channel)).toEqual([ch.key, 0]);
  });

  // A disabled row skips the Core's kind check, so its stored kind can disagree with
  // the channel until the row is enabled.
  it("should write a string channel as a string", async () => {
    const dev = await createKafkaDevice(client);
    const ch = await client.channels.create({
      name: uniqueName("kafka_out"),
      dataType: "string",
      virtual: true,
    });
    const draft = await createDraft(client, {
      ...Kafka.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
      topic: "samples",
      channels: [
        kafka.writeChannelZ.parse({
          type: "float",
          key: "c1",
          channel: ch.key,
          disabled: true,
        }),
      ],
    });
    const { container } = await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    fireEvent.click((await screen.findAllByText(ch.name))[0]);
    // The section title and the field label, once the details render.
    await waitFor(() => expect(screen.getAllByText("Channel")).toHaveLength(2));
    expect(screen.queryByText("Write as")).toBeNull();
    fireEvent.click(screen.getByRole("checkbox", { name: "Enabled" }));
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Kafka.Task.WRITE_SCHEMAS,
    );
    expect(created.config.channels[0]).toMatchObject({
      type: "string",
      disabled: false,
    });
  });

  it("should write a timestamp channel in a time format", async () => {
    const dev = await createKafkaDevice(client);
    const ch = await client.channels.create({
      name: uniqueName("kafka_time"),
      dataType: "timestamp",
      isIndex: true,
    });
    const draft = await createDraft(client, {
      ...Kafka.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
      topic: "samples",
      channels: [
        kafka.writeChannelZ.parse({
          type: "float",
          key: "c1",
          channel: ch.key,
          disabled: true,
          enumValues: [{ label: "ON", value: 1 }],
        }),
      ],
    });
    const { container } = await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    fireEvent.click((await screen.findAllByText(ch.name))[0]);
    // The record's time format and the channel's, once the details render.
    await waitFor(() => expect(screen.getAllByText("Time format")).toHaveLength(2));
    expect(screen.queryByText("Enum mapping")).toBeNull();
    fireEvent.click(screen.getByRole("checkbox", { name: "Enabled" }));
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Kafka.Task.WRITE_SCHEMAS,
    );
    expect(created.config.channels[0]).toMatchObject({
      type: "timestamp",
      timeFormat: "iso8601",
      disabled: false,
    });
  });
});
