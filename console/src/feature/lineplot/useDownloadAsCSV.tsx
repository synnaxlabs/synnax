// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  DisconnectedError,
  type Synnax as Client,
  UnsupportedError,
} from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { LinePlot, Synnax } from "@synnaxlabs/pluto";
import { resolveWindow, TimeRange, TimeStamp, unique } from "@synnaxlabs/x";

import { FLAGS } from "@/flags";
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

const channelsOf = (lines: DownloadLine[]): channel.Key[] =>
  unique.unique(
    lines
      .flatMap((l) => [l.channels.y, l.channels.x])
      .filter((v): v is channel.Key => v != null && v !== 0),
  );

// Returns the end of the newest stored sample of the channel, or null when it stores
// none.
const readNewest = async (
  client: Client,
  key: channel.Key,
): Promise<TimeStamp | null> => {
  try {
    const { series } = await client.readLatest([key], 1);
    return series.findLast((s) => !s.timeRange.isZero)?.timeRange.end ?? null;
  } catch (e) {
    // The Core stores no samples for a calculation over virtual channels.
    if (UnsupportedError.matches(e)) return null;
    throw e as Error;
  }
};

// Returns where a live window of the plot ends: the end of the newest stored sample of
// its channels, or now when they store none. Each channel reads alone, since the Core
// rejects a read that holds a calculation over virtual channels.
const readEdge = async (
  client: Client,
  channels: channel.Key[],
): Promise<TimeStamp> => {
  const newest = await Promise.all(channels.map((k) => readNewest(client, k)));
  const ends = newest.filter((t): t is TimeStamp => t != null);
  return ends.length === 0
    ? TimeStamp.now()
    : ends.reduce((a, b) => TimeStamp.max(a, b));
};

export const useDownloadAsCSV = (): ((params: DownloadAsCSVParams) => void) => {
  const openDownloadCSVModal = Framer.useDownloadCSVModal();
  return ({ timeRanges, lines, name }) =>
    openDownloadCSVModal({
      timeRanges,
      name,
      channels: channelsOf(lines),
      icon: <Icon.LinePlot />,
    });
};

export const useDownloadPlotAsCSV = (key: string): (() => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const downloadAsCSV = useDownloadAsCSV();
  const derived = LinePlot.useLines({ key });
  const ranges = LinePlot.useRanges({ key });
  const name = LinePlot.useName({ key });
  const windows = Session.Range.useResolveWindows(ranges);
  return () =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const lines: DownloadLine[] = derived.map((d) => ({
        channels: { x: d.xChannel, y: d.yChannel },
      }));
      const live = windows.some((w) => w.variant === "dynamic");
      const end = live ? await readEdge(client, channelsOf(lines)) : TimeStamp.now();
      const resolved = windows.map((w) => resolveWindow(w, end));
      const timeRanges = FLAGS.breaks
        ? LinePlot.shownSpans(resolved)
        : [TimeRange.merge(...resolved)];
      downloadAsCSV({ timeRanges, lines, name });
    }, "Failed to download plot as CSV");
};
