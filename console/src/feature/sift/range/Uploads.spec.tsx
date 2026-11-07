// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ranger,
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
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const START = TimeStamp.seconds(10);

// The device sits on a new rack with no uploader, so a pending upload stays pending.
const createUpload = async (rng: ranger.Range, variant: status.Variant) => {
  const dev = await createSiftDevice(client);
  return await client.statuses.set(
    {
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
        positions: [{ index: 1, time: START.add(TimeSpan.seconds(5)) }],
        samples: 5,
      },
    },
    { parent: ranger.ontologyID(rng.key) },
  );
};

const renderUploads = async (rng: ranger.Range) => {
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Sift.Range.Uploads rangeKey={rng.key} />, { wrapper });
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
    await renderUploads(rng);
    expect(await screen.findByText("stand")).toBeTruthy();
    expect(screen.getByText("Uploading to Sift")).toBeTruthy();
    expect(screen.getByText("50%")).toBeTruthy();
  });

  it("should cancel an upload by deleting its status", async () => {
    const rng = await createRange();
    const upload = await createUpload(rng, "loading");
    await renderUploads(rng);
    fireEvent.click(await screen.findByRole("button", { name: "Cancel upload" }));
    await waitFor(async () =>
      expect(await client.statuses.retrieve({ key: [upload.key] })).toHaveLength(0),
    );
  });

  it("should not show the progress of a finished upload", async () => {
    const rng = await createRange();
    await createUpload(rng, "success");
    await renderUploads(rng);
    expect(await screen.findByRole("button", { name: "Remove upload" })).toBeTruthy();
    expect(screen.queryByText("50%")).toBeNull();
  });
});
