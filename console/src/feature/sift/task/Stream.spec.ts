// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, type task } from "@synnaxlabs/client";
import { createTestClient, newIndexedPair } from "@synnaxlabs/client/testutil";
import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Sift } from "@/feature/sift";
import { createSiftDevice } from "@/feature/sift/testutil";
import {
  awaitEditableForm,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const ZERO_DRAFT: task.New<Sift.Task.StreamSchemas> = {
  name: "Sift stream task",
  type: Sift.Task.STREAM_TYPE,
  config: Sift.Task.STREAM_SCHEMAS.config.parse({}),
};

const renderStream = async (options: RenderTaskFormTabOptions = {}) => {
  const rendered = await renderTaskFormTab(Sift.Task.Stream, {
    client,
    task: ZERO_DRAFT,
    ...options,
  });
  await awaitEditableForm();
  return rendered;
};

describe("Sift stream form", () => {
  it("should show the asset, channel, and label fields", async () => {
    await renderStream();
    expect(await screen.findByPlaceholderText("Test stand 1")).toBeTruthy();
    expect(screen.getByText("Select channels to send")).toBeTruthy();
    expect(
      screen.getByText("A range with one of these labels becomes a Sift run."),
    ).toBeTruthy();
  });

  it("should start the task on the rack of its device", async () => {
    const rack = await client.racks.create({ name: "sift" });
    const dev = await createSiftDevice(client, { rack: rack.key });
    const [, data] = await newIndexedPair(client);
    const t = await client.tasks.create(
      {
        ...ZERO_DRAFT,
        config: {
          ...ZERO_DRAFT.config,
          device: dev.key,
          asset: "Test stand 1",
          channels: [data.key],
        },
      },
      Sift.Task.STREAM_SCHEMAS,
    );
    const { container } = await renderStream({ taskKey: t.key });
    const deployed = await deployAndAwaitTask(
      client,
      container,
      t.key,
      Sift.Task.STREAM_SCHEMAS,
    );
    expect(deployed.rack).toBe(rack.key);
    expect(deployed.config).toMatchObject({
      device: dev.key,
      asset: "Test stand 1",
      channels: [data.key],
    });
  });

  it("should save the labels that the user picks", async () => {
    const dev = await createSiftDevice(client);
    const [, data] = await newIndexedPair(client);
    const hotfire = await client.labels.create({
      name: uniqueName("hotfire"),
      color: "#FF0000",
    });
    const t = await client.tasks.create(
      {
        ...ZERO_DRAFT,
        config: {
          ...ZERO_DRAFT.config,
          device: dev.key,
          asset: "Test stand 1",
          channels: [data.key],
        },
      },
      Sift.Task.STREAM_SCHEMAS,
    );
    const { container } = await renderStream({ taskKey: t.key });
    fireEvent.click(await screen.findByText("Select labels"));
    fireEvent.change(await screen.findByPlaceholderText("Search labels..."), {
      target: { value: hotfire.name },
    });
    fireEvent.click(await screen.findByText(hotfire.name));
    const deployed = await deployAndAwaitTask(
      client,
      container,
      t.key,
      Sift.Task.STREAM_SCHEMAS,
    );
    expect(deployed.config.labels).toEqual([hotfire.key]);
  });

  it("should not list calculated channels, which the Core does not store", async () => {
    const prefix = uniqueName("sift_stored");
    const index = await client.channels.create({
      name: `${prefix}_time`,
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const data = await client.channels.create({
      name: `${prefix}_data`,
      dataType: DataType.FLOAT64,
      index: index.key,
    });
    await client.channels.create({
      name: `${prefix}_calc`,
      dataType: DataType.FLOAT64,
      virtual: true,
      expression: `return ${data.name} * 2`,
    });
    await renderStream();
    fireEvent.click(screen.getByText("Select channels to send"));
    fireEvent.change(await screen.findByPlaceholderText("Search channels..."), {
      target: { value: prefix },
    });
    expect(await screen.findByText(data.name)).toBeTruthy();
    expect(screen.queryByText(`${prefix}_calc`)).toBeNull();
  });

  it("should show the progress of the stream once the Core writes it", async () => {
    const t = await client.tasks.create(ZERO_DRAFT, Sift.Task.STREAM_SCHEMAS);
    await renderStream({ taskKey: t.key });
    const description = "Sent 12 samples to asset Test stand 1 since the task started.";
    await client.statuses.set({
      key: `${Sift.Task.PROGRESS_KEY_PREFIX}${t.key}`,
      name: t.name,
      variant: "info",
      message: "Sent to Sift",
      description,
      time: TimeStamp.now(),
    });
    expect(await screen.findByText(description)).toBeTruthy();
    expect(screen.getByText("Sent to Sift")).toBeTruthy();
  });
});
