// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ranger,
  type sift,
  type status,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Sift } from "@/feature/sift";
import { createSiftDevice } from "@/feature/sift/testutil";
import { Modals } from "@/platform/modals";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const START = TimeStamp.seconds(10);

// The device sits on a new rack with no uploader, so a pending upload stays pending.
const createUpload = async (
  rng: ranger.Range,
  variant: status.Variant,
  reached: TimeSpan[] = [TimeSpan.seconds(5)],
) => {
  const dev = await createSiftDevice(client);
  return await client.statuses.set<typeof sift.uploadDetailsZ>({
    key: `sift_upload:${uniqueName("upload")}`,
    name: rng.name,
    variant,
    message: "Uploading to Sift",
    time: TimeStamp.now(),
    details: {
      device: dev.key,
      range: rng.key,
      timeRange: new TimeRange(START, START.add(TimeSpan.seconds(10))),
      channels: [],
      asset: "stand",
      positions: reached.map((p, i) => ({ index: i + 1, time: START.add(p) })),
      samples: 5,
    },
  });
};

const renderUploads = async (rangeKey: ranger.Key) => {
  const { wrapper } = await createConsoleWrapper({ client });
  render(
    <>
      <Sift.Range.Uploads rangeKey={rangeKey} />
      <Modals.Stack />
    </>,
    { wrapper },
  );
};

const createRange = async () =>
  await client.ranges.create({
    name: uniqueName("hotfire"),
    timeRange: new TimeRange(START, START.add(TimeSpan.seconds(10))),
  });

describe("Sift.Range.Uploads", () => {
  it("should show each upload of the range with its progress", async () => {
    const rng = await createRange();
    await createUpload(rng, "loading");
    await renderUploads(rng.key);
    expect(await screen.findByText("stand")).toBeTruthy();
    expect(screen.getByText("Uploading to Sift")).toBeTruthy();
    expect(screen.getByText("50%")).toBeTruthy();
  });

  it("should show no progress before the first checkpoint", async () => {
    const rng = await createRange();
    await createUpload(rng, "loading", []);
    await renderUploads(rng.key);
    expect(await screen.findByText("0%")).toBeTruthy();
  });

  it("should show the progress of the index that has reached the least far", async () => {
    const rng = await createRange();
    await createUpload(rng, "loading", [TimeSpan.seconds(8), TimeSpan.seconds(4)]);
    await renderUploads(rng.key);
    expect(await screen.findByText("40%")).toBeTruthy();
  });

  it("should stop the progress at 100%", async () => {
    const rng = await createRange();
    await createUpload(rng, "loading", [TimeSpan.seconds(20)]);
    await renderUploads(rng.key);
    expect(await screen.findByText("100%")).toBeTruthy();
  });

  it("should show the error when the uploads fail to list", async () => {
    await createSiftDevice(client);
    await renderUploads("not-a-range");
    expect(await screen.findByText("Failed to retrieve Sift uploads")).toBeTruthy();
  });

  it("should open the upload modal for the range from the header", async () => {
    const rng = await createRange();
    await createUpload(rng, "success");
    await renderUploads(rng.key);
    fireEvent.click(await screen.findByRole("button", { name: "Upload to Sift" }));
    expect(await screen.findByText(`Upload ${rng.name} to Sift`)).toBeTruthy();
  });

  it("should cancel an upload by deleting its status", async () => {
    const rng = await createRange();
    const upload = await createUpload(rng, "loading");
    await renderUploads(rng.key);
    fireEvent.click(await screen.findByRole("button", { name: "Cancel upload" }));
    await waitFor(async () =>
      expect(await client.statuses.retrieve({ key: [upload.key] })).toHaveLength(0),
    );
  });

  it("should show an upload that starts while the section is open", async () => {
    const rng = await createRange();
    await createUpload(rng, "success");
    await renderUploads(rng.key);
    expect(await screen.findAllByText("stand")).toHaveLength(1);
    await createUpload(rng, "loading");
    await waitFor(() => expect(screen.getAllByText("stand")).toHaveLength(2));
  });

  it("should drop an upload that is deleted elsewhere", async () => {
    const rng = await createRange();
    const upload = await createUpload(rng, "success");
    await renderUploads(rng.key);
    expect(await screen.findByText("stand")).toBeTruthy();
    await client.statuses.delete(upload.key);
    await waitFor(() => expect(screen.queryByText("stand")).toBeNull());
  });

  it("should not show the uploads of another range", async () => {
    const rng = await createRange();
    await createUpload(rng, "success");
    const other = await createRange();
    await createUpload(other, "loading");
    await renderUploads(rng.key);
    expect(await screen.findAllByText("stand")).toHaveLength(1);
    expect(screen.queryByText("50%")).toBeNull();
  });

  it("should not show the progress of a finished upload", async () => {
    const rng = await createRange();
    await createUpload(rng, "success");
    await renderUploads(rng.key);
    expect(await screen.findByRole("button", { name: "Delete upload" })).toBeTruthy();
    expect(screen.queryByText("50%")).toBeNull();
  });
});
