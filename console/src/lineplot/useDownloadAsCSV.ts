// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type Channel } from "@synnaxlabs/pluto";
import { TimeRange, TimeStamp, unique } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useStore } from "react-redux";

import { CSV } from "@/csv";
import { Layout } from "@/layout";
import { buildLines } from "@/lineplot/buildLines";
import { select, selectRanges } from "@/lineplot/selectors";
import { type RootState } from "@/store";

export interface DownloadAsCSVArgs {
  timeRanges: TimeRange[];
  lines: Channel.LineProps[];
  name: string;
}

export const useDownloadAsCSV = (): ((args: DownloadAsCSVArgs) => void) => {
  const openDownloadCSVModal = CSV.useDownloadModal();
  return useCallback(
    ({ timeRanges, lines, name }) => {
      const channels = unique.unique(
        lines
          .flatMap((l) => [l.channels.y, l.channels.x])
          .filter(
            (v): v is channel.Key => v != null && v != 0 && typeof v === "number",
          ),
      );
      const channelNames = lines.reduce<Record<channel.Key, string>>((acc, l) => {
        if (l.label == null) return acc;
        if (typeof l.channels.y === "number") acc[l.channels.y] = l.label;
        if (typeof l.channels.x === "number") acc[l.channels.x] = l.label;
        return acc;
      }, {});
      const timeRange = TimeRange.merge(...timeRanges);
      void openDownloadCSVModal({
        timeRange: timeRange.numeric,
        name,
        channels,
        channelNames,
      });
    },
    [openDownloadCSVModal],
  );
};

export const useDownloadPlotAsCSV = (key: string): (() => void) => {
  const downloadAsCSV = useDownloadAsCSV();
  const store = useStore<RootState>();
  return useCallback(() => {
    const now = TimeStamp.now();
    const storeState = store.getState();
    const { name } = Layout.selectRequired(storeState, key);
    const state = select(storeState, key);
    const ranges = selectRanges(storeState, key);
    const lines = buildLines(state, ranges);
    const timeRanges = Object.values(ranges).flatMap((ranges) =>
      ranges.map((r) => {
        if (r.variant === "static") return new TimeRange(r.timeRange);
        return new TimeRange({ start: now.sub(r.span), end: now });
      }),
    );
    downloadAsCSV({ timeRanges, lines, name });
  }, [downloadAsCSV, store]);
};
