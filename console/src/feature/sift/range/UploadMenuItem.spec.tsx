// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeStamp } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Sift } from "@/feature/sift";
import { renderMenuItem } from "@/platform/device/testutil";

const client = createTestClient();

describe("Sift.Range.UploadMenuItem", () => {
  it("should open the upload dialog for the range", async () => {
    await renderMenuItem(
      <Sift.Range.UploadMenuItem
        rangeKey={crypto.randomUUID()}
        rangeName="Hotfire 3"
      />,
      { client },
    );
    fireEvent.click(screen.getByText("Upload to Sift"));
    expect(await screen.findByText("Upload Hotfire 3 to Sift")).toBeTruthy();
  });
});

describe("Sift.Range.UploadRegionMenuItem", () => {
  it("should open the upload dialog for the selected time span", async () => {
    const tr = new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(2));
    await renderMenuItem(
      <Sift.Range.UploadRegionMenuItem
        name="Hotfire plot"
        getTimeRange={async () => tr}
      />,
      { client },
    );
    fireEvent.click(screen.getByText("Upload region to Sift"));
    expect(await screen.findByText("Upload Hotfire plot to Sift")).toBeTruthy();
    const start = tr.start.toString("dateTime", "local");
    expect(screen.getByText(start, { exact: false })).toBeTruthy();
  });
});
