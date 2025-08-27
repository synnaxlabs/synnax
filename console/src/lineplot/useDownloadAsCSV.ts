// Copyright 2025 Synnax Labs, Inc.
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
  type framer,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { type Channel, Status, Synnax } from "@synnaxlabs/pluto";
import { runtime, TimeRange, TimeStamp, unique } from "@synnaxlabs/x";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { buildLines } from "@/lineplot/buildLines";
import { select, selectRanges } from "@/lineplot/selectors";
import { type RootState } from "@/store";

interface DownloadArgs extends DownloadAsCSVArgs {
  client: Client;
}

const frameToCSV = (columns: Map<channel.Key, string>, frame: framer.Frame): string => {
  if (frame.series.length === 0) throw new Error("No data selected");
  const count = frame.series[0].length;
  const rows: string[] = [];
  const header: string[] = [];
  for (let i = 1; i < count; i++) {
    const row: string[] = [];
    columns.forEach((name, key) => {
      const d = frame.get(key).at(i, true);
      if (i === 1) header.push(name);
      row.push(d.toString());
    });
    if (i === 1) rows.push(header.join(","));
    rows.push(row.join(","));
  }
  const os = runtime.getOS();
  return rows.join(os === "Windows" ? "\r\n" : "\n");
};

const download = async ({
  lines,
  client,
  timeRanges,
  name,
}: DownloadArgs): Promise<void> => {
  let keys: channel.Keys = unique.unique(
    lines
      .flatMap((l) => [l.channels.y, l.channels.x])
      .filter((v): v is channel.Key => v != null && v != 0 && typeof v === "number"),
  );
  const channels = await client.channels.retrieve(keys);
  const indexes = unique.unique(channels.map((c) => c.index));
  keys = unique.unique([...keys, ...indexes]);
  const indexChannels = await client.channels.retrieve(indexes);
  const columns = new Map<channel.Key, string>();
  channels.forEach((c) =>
    columns.set(c.key, lines.find((l) => l.channels.y === c.key)?.label ?? c.name),
  );
  indexChannels.forEach((c) =>
    columns.set(c.key, lines.find((l) => l.channels.x === c.key)?.label ?? c.name),
  );
  if (timeRanges.length === 0) throw new Error("No time ranges provided");
  const simplifiedTimeRanges = TimeRange.simplify(timeRanges);
  const frames = await Promise.all(
    simplifiedTimeRanges.map((tr) => client.read(tr, keys)),
  );
  const frame = frames.reduce((acc, curr) => {
    acc.push(curr);
    return acc;
  });
  const csv = frameToCSV(columns, frame);
  const savePath = await save({ defaultPath: `${name}.csv` });
  if (savePath == null) return;
  const data = new TextEncoder().encode(csv);
  await writeFile(savePath, data);
};

export interface DownloadAsCSVArgs {
  timeRanges: TimeRange[];
  lines: Channel.LineProps[];
  name: string;
}

export const useDownloadAsCSV = (): ((args: DownloadAsCSVArgs) => void) => {
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  return ({ timeRanges: timeRange, lines, name }: DownloadAsCSVArgs) =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      await download({ timeRanges: timeRange, lines, client, name });
    }, "Failed to download CSV");
};

export const useDownloadPlotAsCSV = (key: string): (() => void) => {
  const downloadAsCSV = useDownloadAsCSV();
  const store = useStore<RootState>();
  return () => {
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
  };
};
