// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, DataType } from "@synnaxlabs/client";
import { id, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CSV } from "@/platform/csv";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import {
  captureBrowserDownloads,
  type CapturedDownloads,
  removeFilePickers,
} from "@/testutil";

const TIME_RANGE = new TimeRange(TimeStamp.seconds(0), TimeStamp.seconds(10));

const openDownloadModal = async (
  params: CSV.DownloadModalParams,
  options: Parameters<typeof renderModalOpener>[2] = {},
) => await renderModalOpener(() => CSV.useDownloadModal(), [params], options);

// jsdom's Blob has no text(); FileReader is the portable way to read it back.
const readBlob = (blob: Blob): Promise<string> =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = () => reject(reader.error ?? new Error("failed to read blob"));
    reader.readAsText(blob);
  });

describe("useDownloadModal", () => {
  describe("form step (no client)", () => {
    it("disables the Download button when no channels are selected", async () => {
      await openDownloadModal({ timeRange: TIME_RANGE, channels: [], name: "My Plot" });
      await waitFor(() =>
        expect(findButton("Download").className).toContain("pluto--disabled"),
      );
    });

    it("enables the Download button when channels are preselected", async () => {
      await openDownloadModal({
        timeRange: TIME_RANGE,
        channels: [1, 2],
        name: "My Plot",
      });
      await waitFor(() =>
        expect(findButton("Download").className).not.toContain("pluto--disabled"),
      );
    });

    it("keeps the modal open when the download fails without a client", async () => {
      await openDownloadModal({
        timeRange: TIME_RANGE,
        channels: [1, 2],
        name: "My Plot",
      });
      const button = await waitFor(() => {
        const btn = findButton("Download");
        expect(btn.className).not.toContain("pluto--disabled");
        return btn;
      });
      await act(async () => {
        fireEvent.click(button);
      });
      expect(findButton("Download")).toBeTruthy();
    });
  });

  describe("with test client", () => {
    let downloads: CapturedDownloads;
    beforeEach(() => {
      removeFilePickers();
      downloads = captureBrowserDownloads();
    });
    afterEach(() => {
      vi.restoreAllMocks();
    });

    it("reads the selected channels, downloads their data as a CSV, and closes", async () => {
      const client = createTestClient();
      const suffix = id.create().replace(/-/g, "_");
      const indexCh = await client.channels.create({
        name: `csv_dl_time_${suffix}`,
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const dataCh = await client.channels.create({
        name: `csv_dl_data_${suffix}`,
        dataType: DataType.FLOAT64,
        index: indexCh.key,
      });
      const timestamps = Array.from({ length: 5 }, (_, i) => TimeStamp.seconds(i + 1));
      await client.write(TimeStamp.seconds(1), {
        [indexCh.key]: timestamps,
        [dataCh.key]: [1, 2, 3, 4, 5],
      });

      await openDownloadModal(
        {
          timeRange: new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(6)),
          channels: [dataCh.key],
          name: "export",
        },
        { client },
      );
      const downloadButton = await waitFor(() => {
        const btn = findButton("Download");
        expect(btn.className).not.toContain("pluto--disabled");
        return btn;
      });
      await act(async () => {
        fireEvent.click(downloadButton);
      });

      await waitFor(() => expect(downloads.anchors).toHaveLength(1));
      expect(downloads.anchors[0].download).toEqual("export.csv");

      const text = await readBlob(downloads.blobs[0]);
      const rows = text.trim().split("\n");
      expect(rows).toHaveLength(6);
      expect(rows[0]).toContain(dataCh.name);
      expect(rows.slice(1).map((r) => Number(r.split(",").at(-1)))).toEqual([
        1, 2, 3, 4, 5,
      ]);

      await waitFor(() =>
        expect(screen.queryByText("Download data for export to a CSV")).toBeNull(),
      );
    });
  });
});
