// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Framer } from "@/platform/framer";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import {
  captureBrowserDownloads,
  type CapturedDownloads,
  readBlob,
  removeSaveFilePicker,
} from "@/testutil";

const TIME_RANGE = new TimeRange(TimeStamp.seconds(0), TimeStamp.seconds(10));
const MARCH_RANGE = new TimeRange(
  new TimeStamp(new Date(2020, 2, 5, 12)),
  new TimeStamp(new Date(2020, 2, 5, 13)),
);
const JULY_RANGE = new TimeRange(
  new TimeStamp(new Date(2020, 6, 9, 12)),
  new TimeStamp(new Date(2020, 6, 9, 13)),
);

const openDownloadModal = async (
  params: Framer.DownloadCSVModalParams,
  options: Parameters<typeof renderModalOpener>[2] = {},
) => await renderModalOpener(() => Framer.useDownloadCSVModal(), [params], options);

describe("useDownloadCSVModal", () => {
  describe("form step (no client)", () => {
    it("disables the Download button when no channels are selected", async () => {
      await openDownloadModal({
        timeRanges: [TIME_RANGE],
        channels: [],
        name: "My Plot",
      });
      await waitFor(() =>
        expect(findButton("Download").className).toContain("lyra--disabled"),
      );
    });

    it("enables the Download button when channels are preselected", async () => {
      await openDownloadModal({
        timeRanges: [TIME_RANGE],
        channels: [1, 2],
        name: "My Plot",
      });
      await waitFor(() =>
        expect(findButton("Download").className).not.toContain("lyra--disabled"),
      );
    });

    it("shows a range that still runs as ending now", async () => {
      await openDownloadModal({
        timeRanges: [new TimeRange(TimeStamp.seconds(0), TimeStamp.MAX).numeric],
        channels: [],
        name: "My Plot",
      });
      await waitFor(() => expect(document.body.textContent).toMatch(/To\s*Today/));
    });

    it("shows a range that has not started as starting now", async () => {
      await openDownloadModal({
        timeRanges: [new TimeRange(TimeStamp.MAX, TimeStamp.MAX).numeric],
        channels: [],
        name: "My Plot",
      });
      await waitFor(() =>
        expect(document.body.textContent).toMatch(/From\s*Today.*To\s*Today/),
      );
    });

    it("lets the user edit one time range", async () => {
      await openDownloadModal({
        timeRanges: [MARCH_RANGE],
        channels: [],
        name: "My Plot",
      });
      await waitFor(() => expect(screen.getAllByText("Mar 5, 2020")).toHaveLength(2));
      for (const day of screen.getAllByText("Mar 5, 2020"))
        expect(day.closest("button")?.className).not.toContain("lyra-btn--preview");
    });

    it("lists several time ranges in time order with no edit", async () => {
      await openDownloadModal({
        timeRanges: [JULY_RANGE, MARCH_RANGE],
        channels: [],
        name: "My Plot",
      });
      await waitFor(() =>
        expect(document.body.textContent).toMatch(
          /From\s*Mar 5, 2020\s*12:00.*To\s*Mar 5, 2020\s*13:00.*Jul 9, 2020\s*12:00.*Jul 9, 2020\s*13:00/,
        ),
      );
      expect(screen.getAllByText("From")).toHaveLength(1);
      for (const day of [
        ...screen.getAllByText("Mar 5, 2020"),
        ...screen.getAllByText("Jul 9, 2020"),
      ])
        expect(day.closest("button")?.className).toContain("lyra-btn--preview");
    });

    it("keeps the modal open when the download fails without a client", async () => {
      await openDownloadModal({
        timeRanges: [TIME_RANGE],
        channels: [1, 2],
        name: "My Plot",
      });
      const button = await waitFor(() => {
        const btn = findButton("Download");
        expect(btn.className).not.toContain("lyra--disabled");
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
      removeSaveFilePicker();
      downloads = captureBrowserDownloads();
    });
    afterEach(() => {
      vi.restoreAllMocks();
    });

    it("reads the selected channels, downloads their data as CSV, and closes", async () => {
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
          timeRanges: [new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(6))],
          channels: [dataCh.key],
          name: "export",
        },
        { client },
      );
      const downloadButton = await waitFor(() => {
        const btn = findButton("Download");
        expect(btn.className).not.toContain("lyra--disabled");
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
        expect(screen.queryByText("Download data for export as CSV")).toBeNull(),
      );
    });

    it("reads only the samples inside each of several time ranges", async () => {
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
      const timestamps = Array.from({ length: 8 }, (_, i) => TimeStamp.seconds(i + 1));
      await client.write(TimeStamp.seconds(1), {
        [indexCh.key]: timestamps,
        [dataCh.key]: [1, 2, 3, 4, 5, 6, 7, 8],
      });
      await openDownloadModal(
        {
          timeRanges: [
            new TimeRange(TimeStamp.seconds(6), TimeStamp.seconds(8)),
            new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(3)),
          ],
          channels: [dataCh.key],
          name: "ranges",
        },
        { client },
      );
      const downloadButton = await waitFor(() => {
        const btn = findButton("Download");
        expect(btn.className).not.toContain("lyra--disabled");
        return btn;
      });
      await act(async () => {
        fireEvent.click(downloadButton);
      });
      await waitFor(() => expect(downloads.blobs).toHaveLength(1));
      const rows = (await readBlob(downloads.blobs[0])).trim().split("\n");
      expect(rows[0]).toEqual(`${indexCh.name},${dataCh.name}`);
      expect(rows.slice(1).map((r) => Number(r.split(",").at(-1)))).toEqual([
        1, 2, 6, 7,
      ]);
    });

    it("reads every sample of a range that still runs up to the download", async () => {
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
      // A float of this start rounds up past it.
      const start = new TimeStamp((TimeStamp.now().valueOf() / 256n) * 256n + 200n);
      await client.write(start, { [indexCh.key]: [start], [dataCh.key]: [1] });
      await openDownloadModal(
        {
          timeRanges: [new TimeRange(start, TimeStamp.MAX)],
          channels: [dataCh.key],
          name: "running",
        },
        { client },
      );
      const downloadButton = await waitFor(() => {
        const btn = findButton("Download");
        expect(btn.className).not.toContain("lyra--disabled");
        return btn;
      });
      const late = TimeStamp.now().add(TimeSpan.SECOND);
      await client.write(late, { [indexCh.key]: [late], [dataCh.key]: [2] });
      await act(async () => {
        fireEvent.click(downloadButton);
      });
      await waitFor(() => expect(downloads.blobs).toHaveLength(1));
      const rows = (await readBlob(downloads.blobs[0])).trim().split("\n");
      expect(rows.slice(1).map((r) => Number(r.split(",").at(-1)))).toEqual([1, 2]);
    });
  });
});
