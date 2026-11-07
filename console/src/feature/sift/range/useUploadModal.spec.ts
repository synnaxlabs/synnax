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
import { answerNextCommand, selectFromDropdown } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const createRange = async () =>
  await client.ranges.create({
    name: uniqueName("hotfire"),
    timeRange: new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(2)),
  });

const clickUpload = () =>
  fireEvent.click(screen.getByRole("button", { name: /Upload/ }));

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
    const rack = await createRunningRack(client, "sift");
    const scan = await rack.createTask({
      name: uniqueName("sift_scan"),
      type: Sift.Task.SCAN_TYPE,
      config: {},
    });
    const dev = await createSiftDevice(client, { rack: rack.key });
    const [, data] = await newIndexedPair(client);
    const rng = await createRange();
    await renderModalOpener(
      Sift.Range.useUploadModal,
      [{ rangeKey: rng.key, rangeName: rng.name }],
      { client },
    );
    await selectFromDropdown("Select device", dev.name);
    fireEvent.change(await screen.findByPlaceholderText("Test stand 1"), {
      target: { value: "Test stand 1" },
    });
    fireEvent.click(screen.getByText("Select channels to upload"));
    fireEvent.change(await screen.findByPlaceholderText("Search channels..."), {
      target: { value: data.name },
    });
    fireEvent.click(await screen.findByText(data.name));
    const { answered } = await answerNextCommand(client, scan);
    clickUpload();
    const cmd = await answered;
    expect(cmd.type).toBe(Sift.Task.UPLOAD_COMMAND_TYPE);
    expect(cmd.args).toEqual({
      device: dev.key,
      range: rng.key,
      channels: [data.key],
      asset: "Test stand 1",
    });
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });
});
