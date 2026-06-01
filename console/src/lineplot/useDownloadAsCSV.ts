// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type Channel, LinePlot as PLinePlot, Status } from "@synnaxlabs/pluto";
import { color, TimeRange, TimeStamp, unique } from "@synnaxlabs/x";
import { useCallback } from "react";

import { CSV } from "@/csv";
import { Layout } from "@/layout";
import { useDerivedLines } from "@/lineplot/useDerivedLines";
import { Range } from "@/range";

export interface DownloadAsCSVArgs {
  timeRanges: TimeRange[];
  lines: Channel.LineProps[];
  name: string;
}

export const useDownloadAsCSV = (): ((args: DownloadAsCSVArgs) => void) => {
  const openDownloadCSVModal = CSV.useDownloadModal();
  const handleError = Status.useErrorHandler();
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
      handleError(
        async () =>
          openDownloadCSVModal(
            {
              timeRange: timeRange.numeric,
              name,
              channels,
              channelNames,
            },
            { icon: "LinePlot" },
          ),
        `Failed to download CSV data for ${name}`,
      );
    },
    [openDownloadCSVModal],
  );
};

export const useDownloadPlotAsCSV = (key: string): (() => void) => {
  const downloadAsCSV = useDownloadAsCSV();
  const derived = useDerivedLines(key);
  const ranges = PLinePlot.useSelectRanges({ key });
  const { name } = Layout.useSelectRequired(key);
  const rangeKeys = unique.unique([...ranges.x1, ...ranges.x2]);
  const resolved = Range.useSelectMultiple(rangeKeys);
  return useCallback(() => {
    const now = TimeStamp.now();
    const lines: Channel.LineProps[] = derived.map((d) => ({
      key: d.key,
      axes: { x: d.xAxis, y: d.yAxis },
      channels: { x: d.xChannel, y: d.yChannel },
      color: d.color ?? color.ZERO,
      strokeWidth: d.strokeWidth,
      downsample: d.downsample,
      downsampleMode: d.downsampleMode,
      label: d.label,
      variant: "static",
      timeRange: TimeRange.ZERO,
    }));
    const timeRanges = resolved.map((r) => {
      if (r.variant === "static") return new TimeRange(r.timeRange);
      return new TimeRange({ start: now.sub(r.span), end: now });
    });
    downloadAsCSV({ timeRanges, lines, name });
  }, [downloadAsCSV, derived, resolved, name]);
};
