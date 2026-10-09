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
  it("should add a channel and reveal the enum mapping for a string JSON type", async () => {
    const dev = await createKafkaDevice(client);
    const draft = await createDraft(client, {
      ...Kafka.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    const { container } = await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    fireEvent.click(getIconButton(container, "add"));
    await screen.findByText("JSON type");
    expect(screen.queryByText("Enum mapping")).toBeNull();
    await selectFromDropdown("Number", "String");
    await screen.findByText("Enum mapping");
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
      channels: [{ ...kafka.writeChannelZ.parse({}), key: "c1", channel: ch.key }],
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
      channels: [{ ...kafka.writeChannelZ.parse({}), key: "c1", channel: ch.key }],
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
});
