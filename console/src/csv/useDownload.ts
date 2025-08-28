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
import { Status, Synnax } from "@synnaxlabs/pluto";
import { runtime, TimeRange, unique } from "@synnaxlabs/x";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";

export interface DownloadArgs {
  timeRanges: TimeRange[];
  keys: channel.Keys;
  keysToNames?: Record<channel.Key, string>;
  fileName: string;
  afterDownload?: () => void;
  onPercentDownloadedChange?: (percent: number) => void;
}

export const useDownload = (): ((args: DownloadArgs) => void) => {
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  return (args: DownloadArgs) =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      try {
        await download({ ...args, client, addStatus });
      } catch (e) {
        args.onPercentDownloadedChange?.(0);
        throw e;
      }
    }, "Failed to download CSV");
};

interface DownloadContext extends DownloadArgs {
  client: Client;
  addStatus: Status.Adder;
}

const download = async ({
  timeRanges,
  keys,
  keysToNames = {},
  client,
  fileName,
  afterDownload,
  onPercentDownloadedChange,
  addStatus,
}: DownloadContext): Promise<void> => {
  const savePath = await save({ defaultPath: `${fileName}.csv` });
  if (savePath == null) return;
  const channels = await client.channels.retrieve(keys);
  onPercentDownloadedChange?.(10);
  const indexes = unique.unique(channels.map(({ index }) => index));
  const indexChannels = await client.channels.retrieve(indexes);
  onPercentDownloadedChange?.(20);
  const columns = new Map<channel.Key, string>();
  indexChannels.forEach(({ key, name }) => columns.set(key, keysToNames[key] ?? name));
  channels.forEach(({ key, name }) => columns.set(key, keysToNames[key] ?? name));
  const simplifiedTimeRanges = TimeRange.simplify(timeRanges);
  const allKeys = unique.unique([...keys, ...indexes]);
  let percentDownloaded = 20;
  const totalFrames = simplifiedTimeRanges.length;
  const delta = totalFrames > 0 ? 60 / totalFrames : 0;
  const frames = await Promise.all(
    simplifiedTimeRanges.map(async (tr) => {
      const frame = await client.read(tr, allKeys);
      percentDownloaded += delta;
      if (percentDownloaded >= 80) percentDownloaded = 80;
      onPercentDownloadedChange?.(percentDownloaded);
      return frame;
    }),
  );
  const frame = frames.reduce((acc, curr) => {
    acc.push(curr);
    return acc;
  });
  const csv = frameToCSV(columns, frame);
  const data = new TextEncoder().encode(csv);
  await writeFile(savePath, data);
  addStatus({
    variant: "success",
    message: `Downloaded ${fileName} to ${savePath}`,
  });
  onPercentDownloadedChange?.(100);
  afterDownload?.();
};

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
