// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { LinePlot } from "@synnaxlabs/pluto";
import { resolveWindow, TimeRange, TimeStamp, unique } from "@synnaxlabs/x";

import { Framer } from "@/platform/framer";
import { Session } from "@/session";

// DownloadLine carries only the channels the CSV export pulls. CSV columns are
// headed by the channel name, never the line's display label, so none is read.
export interface DownloadLine {
  channels: { x?: channel.Key; y: channel.Key };
}

export interface DownloadAsCSVParams {
  timeRanges: TimeRange[];
  lines: DownloadLine[];
  name: string;
}

export const useDownloadAsCSV = (): ((params: DownloadAsCSVParams) => void) => {
  const openDownloadCSVModal = Framer.useDownloadCSVModal();
  const handleError = Status.useErrorHandler();
  return ({ timeRanges, lines, name }) => {
    const channels = unique.unique(
      lines
        .flatMap((l) => [l.channels.y, l.channels.x])
        .filter((v): v is channel.Key => v != null && v !== 0),
    );
    const timeRange = TimeRange.merge(...timeRanges);
    handleError(
      async () =>
        await openDownloadCSVModal({
          timeRange: timeRange.numeric,
          name,
          channels,
          icon: <Icon.LinePlot />,
        }),
      `Failed to download CSV data for ${name}`,
    );
  };
};

export const useDownloadPlotAsCSV = (key: string): (() => void) => {
  const downloadAsCSV = useDownloadAsCSV();
  const derived = LinePlot.useLines({ key });
  const ranges = LinePlot.useRanges({ key });
  const name = LinePlot.useName({ key });
  const windows = Session.Range.useResolveWindows(ranges);
  return () => {
    const now = TimeStamp.now();
    const lines: DownloadLine[] = derived.map((d) => ({
      channels: { x: d.xChannel, y: d.yChannel },
    }));
    const timeRanges = windows.map((w) => resolveWindow(w, now));
    downloadAsCSV({ timeRanges, lines, name });
  };
};
