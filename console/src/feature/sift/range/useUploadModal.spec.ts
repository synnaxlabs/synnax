// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  access,
  type channel,
  DataType,
  device,
  type task,
  TimeRange,
  TimeStamp,
  user,
} from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
  newIndexedPair,
} from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Sift } from "@/feature/sift";
import { createSiftDevice } from "@/feature/sift/testutil";
import { createRunningRack } from "@/platform/device/testutil";
import { renderModalOpener } from "@/platform/modals/testutil";
import { answerNextCommand, type CommandAnswer } from "@/platform/task/testutil";
import { CaptureStatuses, findDialogTriggerByText, uniqueName } from "@/testutil";

const client = createTestClient();

const createRange = async () =>
  await client.ranges.create({
    name: uniqueName("hotfire"),
    timeRange: new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(2)),
  });

// The device list pages, so a search finds the device however many the Core holds.
const selectDevice = async (name: string) => {
  fireEvent.click(await findDialogTriggerByText("Select device"));
  fireEvent.change(await screen.findByPlaceholderText("Search devices..."), {
    target: { value: name },
  });
  fireEvent.click(await screen.findByText(name));
};

const clickUpload = () =>
  fireEvent.click(screen.getByRole("button", { name: /Upload/ }));

const createScanTask = async () => {
  const rack = await createRunningRack(client, "sift");
  const scan = await rack.createTask({
    name: uniqueName("sift_scan"),
    type: Sift.Task.SCAN_TYPE,
    config: {},
  });
  return { rack, scan };
};

// Picks the device, the asset Sift lists last, and the channel, then uploads.
const fillAndUpload = async (
  scan: task.Task,
  dev: device.Device,
  ch: channel.Channel,
  answer: CommandAnswer = {},
) => {
  const { answered: listed } = await answerNextCommand(client, scan, {
    type: Sift.Task.LIST_ASSETS_COMMAND_TYPE,
    data: { assets: ["pad", "stand"], last: "stand" },
  });
  await selectDevice(dev.name);
  await listed;
  expect(await screen.findByText("stand")).toBeTruthy();
  fireEvent.click(screen.getByText("Select channels to upload"));
  fireEvent.change(await screen.findByPlaceholderText("Search channels..."), {
    target: { value: ch.name },
  });
  fireEvent.click(await screen.findByText(ch.name));
  const { answered } = await answerNextCommand(client, scan, {
    type: Sift.Task.UPLOAD_COMMAND_TYPE,
    ...answer,
  });
  clickUpload();
  return await answered;
};

describe("Sift.Range.useUploadModal", () => {
  // The dialog picks the only Sift device on its own, so two keep the pick empty.
  beforeAll(async () => {
    await createSiftDevice(client);
    await createSiftDevice(client);
  });

  it("should pick the device when it is the only one the user can see", async () => {
    const dev = await createSiftDevice(client);
    const rng = await createRange();
    const restricted = await createTestClientWithPolicy(client, {
      name: uniqueName("policy"),
      objects: [
        device.resourceID(dev.key),
        user.TYPE_RESOURCE_ID,
        access.role.TYPE_RESOURCE_ID,
        access.policy.TYPE_RESOURCE_ID,
      ],
      actions: ["retrieve"],
    });
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ name: rng.name, source: rng.key }],
      { client: restricted },
    );
    expect(await findDialogTriggerByText(dev.name)).toBeTruthy();
  });

  it("should require a device, an asset, and channels", async () => {
    const rng = await createRange();
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ name: rng.name, source: rng.key }],
      { client },
    );
    expect(await screen.findByText(`Upload ${rng.name} to Sift`)).toBeTruthy();
    clickUpload();
    expect(await screen.findByText("Device is required")).toBeTruthy();
    expect(screen.getByText("Asset is required")).toBeTruthy();
    expect(screen.getByText("Select at least one channel")).toBeTruthy();
  });

  it("should queue an upload on the scan task of the device's rack", async () => {
    const { rack, scan } = await createScanTask();
    const dev = await createSiftDevice(client, { rack: rack.key });
    const [, data] = await newIndexedPair(client);
    const rng = await createRange();
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ name: rng.name, source: rng.key }],
      { client },
    );
    const cmd = await fillAndUpload(scan, dev, data);
    expect(cmd.args).toEqual({
      device: dev.key,
      range: rng.key,
      channels: [data.key],
      asset: "stand",
    });
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("should start again at the last asset of a new device", async () => {
    const { rack, scan } = await createScanTask();
    const first = await createSiftDevice(client, { rack: rack.key });
    const second = await createSiftDevice(client, { rack: rack.key });
    const rng = await createRange();
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ name: rng.name, source: rng.key }],
      { client },
    );
    const { answered: listedFirst } = await answerNextCommand(client, scan, {
      type: Sift.Task.LIST_ASSETS_COMMAND_TYPE,
      data: { assets: ["stand"], last: "stand" },
    });
    await selectDevice(first.name);
    await listedFirst;
    expect(await screen.findByText("stand")).toBeTruthy();
    const { answered: listedSecond } = await answerNextCommand(client, scan, {
      type: Sift.Task.LIST_ASSETS_COMMAND_TYPE,
      data: { assets: ["pad"], last: "" },
    });
    fireEvent.click(await findDialogTriggerByText(first.name));
    fireEvent.change(await screen.findByPlaceholderText("Search devices..."), {
      target: { value: second.name },
    });
    fireEvent.click(await screen.findByText(second.name));
    await listedSecond;
    await waitFor(() => expect(screen.queryByText("stand")).toBeNull());
  });

  it("should leave the asset empty when the assets of a new device fail to list", async () => {
    const { rack, scan } = await createScanTask();
    const first = await createSiftDevice(client, { rack: rack.key });
    const second = await createSiftDevice(client, { rack: rack.key });
    const rng = await createRange();
    let statuses: Status.NotificationSpec[] = [];
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ name: rng.name, source: rng.key }],
      {
        client,
        extra: createElement(CaptureStatuses, { onStatuses: (s) => (statuses = s) }),
      },
    );
    const { answered: listedFirst } = await answerNextCommand(client, scan, {
      type: Sift.Task.LIST_ASSETS_COMMAND_TYPE,
      data: { assets: ["stand"], last: "stand" },
    });
    await selectDevice(first.name);
    await listedFirst;
    expect(await screen.findByText("stand")).toBeTruthy();
    const { answered: listedSecond } = await answerNextCommand(client, scan, {
      type: Sift.Task.LIST_ASSETS_COMMAND_TYPE,
      variant: "error",
      message: "Sift is unreachable",
    });
    fireEvent.click(await findDialogTriggerByText(first.name));
    fireEvent.change(await screen.findByPlaceholderText("Search devices..."), {
      target: { value: second.name },
    });
    fireEvent.click(await screen.findByText(second.name));
    await listedSecond;
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain("Failed to list Sift assets"),
    );
    expect(screen.queryByText("stand")).toBeNull();
  });

  it("should name a new asset only through New asset", async () => {
    const { rack, scan } = await createScanTask();
    const dev = await createSiftDevice(client, { rack: rack.key });
    const rng = await createRange();
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ name: rng.name, source: rng.key }],
      { client },
    );
    const { answered } = await answerNextCommand(client, scan, {
      type: Sift.Task.LIST_ASSETS_COMMAND_TYPE,
      data: { assets: [], last: "" },
    });
    await selectDevice(dev.name);
    await answered;
    fireEvent.click(screen.getByRole("button", { name: /New asset/ }));
    fireEvent.change(await screen.findByPlaceholderText("Asset name"), {
      target: { value: "Rig 2" },
    });
    fireEvent.click(screen.getByRole("button", { name: /Save/ }));
    expect(await screen.findByText("Rig 2")).toBeTruthy();
  });

  it("should queue an upload of a plain time span", async () => {
    const { rack, scan } = await createScanTask();
    const dev = await createSiftDevice(client, { rack: rack.key });
    const [, data] = await newIndexedPair(client);
    const tr = new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(2));
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ name: "Hotfire plot", source: tr.numeric }],
      { client },
    );
    expect(await screen.findByText("Upload Hotfire plot to Sift")).toBeTruthy();
    const start = tr.start.toString("dateTime", "local");
    expect(screen.getByText(start, { exact: false })).toBeTruthy();
    const cmd = await fillAndUpload(scan, dev, data);
    expect(cmd.args).toEqual({
      device: dev.key,
      timeRange: {
        start: tr.start.valueOf().toString(),
        end: tr.end.valueOf().toString(),
      },
      channels: [data.key],
      asset: "stand",
    });
  });

  it("should not list calculated channels, which the Core does not store", async () => {
    const [, data] = await newIndexedPair(client);
    const calc = await client.channels.create({
      name: uniqueName("sift_calc"),
      dataType: DataType.FLOAT64,
      virtual: true,
      expression: `return ${data.name} * 2`,
    });
    const rng = await createRange();
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ name: rng.name, source: rng.key }],
      { client },
    );
    fireEvent.click(await screen.findByText("Select channels to upload"));
    const search = await screen.findByPlaceholderText("Search channels...");
    fireEvent.change(search, { target: { value: data.name } });
    expect(await screen.findByText(data.name)).toBeTruthy();
    fireEvent.change(search, { target: { value: calc.name } });
    await waitFor(() => expect(screen.queryByText(data.name)).toBeNull());
    expect(screen.queryByText(calc.name)).toBeNull();
  });

  it("should disable Upload while the scan task queues the upload", async () => {
    const { rack, scan } = await createScanTask();
    const dev = await createSiftDevice(client, { rack: rack.key });
    const [, data] = await newIndexedPair(client);
    const rng = await createRange();
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ name: rng.name, source: rng.key }],
      { client },
    );
    const { answered: listed } = await answerNextCommand(client, scan, {
      type: Sift.Task.LIST_ASSETS_COMMAND_TYPE,
      data: { assets: ["stand"], last: "stand" },
    });
    await selectDevice(dev.name);
    await listed;
    expect(await screen.findByText("stand")).toBeTruthy();
    fireEvent.click(screen.getByText("Select channels to upload"));
    fireEvent.change(await screen.findByPlaceholderText("Search channels..."), {
      target: { value: data.name },
    });
    fireEvent.click(await screen.findByText(data.name));
    const { answered } = await answerNextCommand(client, scan, {
      type: Sift.Task.UPLOAD_COMMAND_TYPE,
    });
    clickUpload();
    expect(
      screen.getByRole("button", { name: /Upload/ }).getAttribute("aria-disabled"),
    ).toBe("true");
    await answered;
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("should stay open and show why the scan task rejected an upload", async () => {
    const { rack, scan } = await createScanTask();
    const dev = await createSiftDevice(client, { rack: rack.key });
    const [, data] = await newIndexedPair(client);
    const rng = await createRange();
    let statuses: Status.NotificationSpec[] = [];
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ name: rng.name, source: rng.key }],
      {
        client,
        extra: createElement(CaptureStatuses, { onStatuses: (s) => (statuses = s) }),
      },
    );
    await fillAndUpload(scan, dev, data, {
      variant: "error",
      message: "Failed to upload to Sift",
      description: "range has no data",
    });
    expect(await screen.findByText("range has no data")).toBeTruthy();
    expect(screen.getByText(`Upload ${rng.name} to Sift`)).toBeTruthy();
    expect(statuses).toEqual([]);
    // A retry that fails for another cause clears the earlier reason.
    await client.tasks.delete(scan.key);
    clickUpload();
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain("Failed to upload to Sift"),
    );
    expect(screen.queryByText("range has no data")).toBeNull();
    expect(screen.getByText(`Upload ${rng.name} to Sift`)).toBeTruthy();
  });
});
