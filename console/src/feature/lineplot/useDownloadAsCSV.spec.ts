// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeStamp } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useDownloadAsCSV } from "@/feature/lineplot/useDownloadAsCSV";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";

describe("useDownloadAsCSV", () => {
  it("should open the CSV dialog on each time range", async () => {
    await renderModalOpener(useDownloadAsCSV, [
      {
        timeRanges: [
          new TimeRange(
            new TimeStamp(new Date(2020, 6, 9, 12)),
            new TimeStamp(new Date(2020, 6, 9, 13)),
          ),
          new TimeRange(
            new TimeStamp(new Date(2020, 2, 5, 12)),
            new TimeStamp(new Date(2020, 2, 5, 13)),
          ),
        ],
        lines: [{ channels: { x: 1, y: 2 } }],
        name: "My Plot",
      },
    ]);
    await waitFor(() =>
      expect(document.body.textContent).toMatch(
        new RegExp(
          String.raw`Mar 5, 2020\s*12:00.*Mar 5, 2020\s*13:00.*` +
            String.raw`Jul 9, 2020\s*12:00.*Jul 9, 2020\s*13:00`,
        ),
      ),
    );
    expect(findButton("Download").className).not.toContain("lyra--disabled");
  });
});
