// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type rtsp, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { DataType, TimeSpan } from "@synnaxlabs/x";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { RTSP } from "@/feature/rtsp";
import {
  awaitEditableForm,
  clickDeploy,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const URL_PLACEHOLDER = "rtsp://user:password@192.168.1.10:554/stream";

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<RTSP.Task.ReadSchemas> = {
  name: "RTSP read task",
  type: RTSP.Task.READ_TYPE,
  config: RTSP.Task.READ_SCHEMAS.config.parse({}),
};

// The form renders read-only until the update grant lands.
const renderRead = async (options: RenderTaskFormTabOptions = {}) => {
  const rendered = await renderTaskFormTab(RTSP.Task.Read, {
    client,
    task: ZERO_DRAFT,
    ...options,
  });
  await awaitEditableForm();
  return rendered;
};

const createDraft = async (config: Partial<rtsp.ReadConfig>, rack = 0) =>
  await client.tasks.create(
    {
      ...ZERO_DRAFT,
      name: uniqueName("camera"),
      config: { ...ZERO_DRAFT.config, ...config },
      rack,
    },
    RTSP.Task.READ_SCHEMAS,
  );

describe("RTSP read form", () => {
  it("should show the URL, transport, rack, channel, and timing fields", async () => {
    await renderRead();
    expect(await screen.findByPlaceholderText(URL_PLACEHOLDER)).toBeTruthy();
    expect(screen.getByText("Transport")).toBeTruthy();
    expect(screen.getByText("Connect from")).toBeTruthy();
    expect(screen.getByText("Video channel")).toBeTruthy();
    expect(screen.getByText("Time source")).toBeTruthy();
    expect(screen.getByText("Offset")).toBeTruthy();
  });

  it("should load the time source and offset of the task row", async () => {
    const draft = await createDraft({
      timeSource: "capture",
      offset: TimeSpan.milliseconds(-250),
    });
    await renderRead({ taskKey: draft.key });
    expect(await screen.findByDisplayValue("-250")).toBeTruthy();
    expect(screen.getByText("Capture")).toBeTruthy();
  });

  it("should load the URL of the task row", async () => {
    const draft = await createDraft({ url: "rtsp://camera.local/stream" });
    await renderRead({ taskKey: draft.key });
    expect(await screen.findByDisplayValue("rtsp://camera.local/stream")).toBeTruthy();
  });

  it("should block a deploy of a URL that is not RTSP", async () => {
    const draft = await createDraft({ url: "http://camera.local/stream" });
    const { container } = await renderRead({ taskKey: draft.key });
    await clickDeploy(container);
    expect(
      await screen.findByText("URL must start with rtsp:// or rtsps://"),
    ).toBeTruthy();
  });

  describe("deploying against a live Core", () => {
    it("should create a video channel and its index named after the task", async () => {
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const draft = await createDraft({ url: "rtsp://camera.local/stream" }, rack.key);
      const { container } = await renderRead({ taskKey: draft.key });
      const deployed = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        RTSP.Task.READ_SCHEMAS,
      );
      const ch = await client.channels.retrieve(deployed.config.channel);
      expect(ch.name).toBe(channel.escapeInvalidName(draft.name));
      expect(ch.dataType.equals(DataType.VIDEO)).toBe(true);
      expect(ch.virtual).toBe(false);
      expect((await client.channels.retrieve(ch.index)).name).toBe(
        `${channel.escapeInvalidName(draft.name)}_time`,
      );
      expect(deployed.config.url).toBe("rtsp://camera.local/stream");
    });

    it("should create a virtual video channel for a task that saves no data", async () => {
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const draft = await createDraft(
        { dataSavingDisabled: true, url: "rtsp://camera.local/stream" },
        rack.key,
      );
      const { container } = await renderRead({ taskKey: draft.key });
      const deployed = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        RTSP.Task.READ_SCHEMAS,
      );
      const ch = await client.channels.retrieve(deployed.config.channel);
      expect(ch.name).toBe(channel.escapeInvalidName(draft.name));
      expect(ch.dataType.equals(DataType.VIDEO)).toBe(true);
      expect(ch.virtual).toBe(true);
      expect(deployed.config.url).toBe("rtsp://camera.local/stream");
    });

    it("should keep the video channel of the task", async () => {
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const ch = await client.channels.create({
        name: uniqueName("video"),
        dataType: DataType.VIDEO,
        virtual: true,
      });
      const draft = await createDraft(
        { url: "rtsp://camera.local/stream", channel: ch.key },
        rack.key,
      );
      const { container } = await renderRead({ taskKey: draft.key });
      const deployed = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        RTSP.Task.READ_SCHEMAS,
      );
      expect(deployed.config.channel).toBe(ch.key);
    });
  });
});
