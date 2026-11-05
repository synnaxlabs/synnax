// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type task, type whip } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { DataType } from "@synnaxlabs/x";
import { act, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { WHIP } from "@/feature/whip";
import {
  awaitEditableForm,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";
import { Session } from "@/session";
import { uniqueName } from "@/testutil";

const client = createTestClient();

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<WHIP.Task.ReadSchemas> = {
  name: "WHIP read task",
  type: WHIP.Task.READ_TYPE,
  config: WHIP.Task.READ_SCHEMAS.config.parse({}),
};

// The form renders read-only until the update grant lands.
const renderRead = async (options: RenderTaskFormTabOptions = {}) => {
  const rendered = await renderTaskFormTab(WHIP.Task.Read, {
    client,
    task: ZERO_DRAFT,
    ...options,
  });
  await awaitEditableForm();
  return rendered;
};

const createDraft = async (rack = 0, config: Partial<whip.ReadConfig> = {}) =>
  await client.tasks.create(
    {
      ...ZERO_DRAFT,
      name: uniqueName("obs"),
      config: { ...ZERO_DRAFT.config, ...config },
      rack,
    },
    WHIP.Task.READ_SCHEMAS,
  );

describe("WHIP read form", () => {
  it("should show the URL that an encoder publishes the task to", async () => {
    const draft = await createDraft();
    const { store } = await renderRead({ taskKey: draft.key });
    act(() => {
      store.dispatch(Session.Core.select("LOCAL"));
    });
    expect(
      await screen.findByText(`http://localhost:9090/api/v1/whip/${draft.key}`),
    ).toBeTruthy();
    expect(screen.getByText("Receive on")).toBeTruthy();
    expect(screen.getByText("Video channel")).toBeTruthy();
    expect(screen.getByText("Time source")).toBeTruthy();
    expect(screen.getByText("Offset")).toBeTruthy();
  });

  describe("deploying against a live Core", () => {
    it("should create a video channel and its index named after the task", async () => {
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const draft = await createDraft(rack.key);
      const { container } = await renderRead({ taskKey: draft.key });
      const deployed = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        WHIP.Task.READ_SCHEMAS,
      );
      const ch = await client.channels.retrieve(deployed.config.channel);
      expect(ch.name).toBe(channel.escapeInvalidName(draft.name));
      expect(ch.dataType.equals(DataType.VIDEO)).toBe(true);
      expect(ch.virtual).toBe(false);
      expect((await client.channels.retrieve(ch.index)).name).toBe(
        `${channel.escapeInvalidName(draft.name)}_time`,
      );
    });

    it("should create a virtual video channel for a task that saves no data", async () => {
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const draft = await createDraft(rack.key, { dataSavingDisabled: true });
      const { container } = await renderRead({ taskKey: draft.key });
      const deployed = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        WHIP.Task.READ_SCHEMAS,
      );
      const ch = await client.channels.retrieve(deployed.config.channel);
      expect(ch.dataType.equals(DataType.VIDEO)).toBe(true);
      expect(ch.virtual).toBe(true);
    });
  });
});
