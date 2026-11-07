// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeStamp } from "@synnaxlabs/client";
import { createTestClient, newIndexedPair } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Sift } from "@/feature/sift";
import { createSiftDevice } from "@/feature/sift/testutil";
import { createRunningRack } from "@/platform/device/testutil";
import { renderModalOpener } from "@/platform/modals/testutil";
import { answerNextCommand } from "@/platform/task/testutil";
import { findDialogTriggerByText, uniqueName } from "@/testutil";

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

describe("Sift.Range.useUploadModal", () => {
  it("should require a device, an asset, and channels", async () => {
    const rng = await createRange();
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ rangeKey: rng.key, rangeName: rng.name }],
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
      [{ rangeKey: rng.key, rangeName: rng.name }],
      { client },
    );
    const { answered: listed } = await answerNextCommand(client, scan, {
      type: Sift.Task.LIST_ASSETS_COMMAND_TYPE,
      data: { assets: ["pad", "stand"], last: "stand" },
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
    const cmd = await answered;
    expect(cmd.args).toEqual({
      device: dev.key,
      range: rng.key,
      channels: [data.key],
      asset: "stand",
    });
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("should name a new asset only through New asset", async () => {
    const { rack, scan } = await createScanTask();
    const dev = await createSiftDevice(client, { rack: rack.key });
    const rng = await createRange();
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ rangeKey: rng.key, rangeName: rng.name }],
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
});
