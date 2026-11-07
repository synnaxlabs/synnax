// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { createTestClient, newIndexedPair } from "@synnaxlabs/client/testutil";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Sift } from "@/feature/sift";
import { createSiftDevice } from "@/feature/sift/testutil";
import {
  awaitEditableForm,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";

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
    expect(await screen.findByRole("button", { name: /New asset/ })).toBeTruthy();
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
});
