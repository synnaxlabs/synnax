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
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { runtime, TimeRange, unique } from "@synnaxlabs/x";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";

import { convertFrameGroups, type CSVGroup, sanitizeValue } from "@/csv/util";
import { Runtime } from "@/runtime";

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
  let savePath: string | null = null;
  if (Runtime.ENGINE === "tauri") {
    savePath = await save({ defaultPath: `${fileName}.csv` });
    if (savePath == null) return;
  }
  const channels = await client.channels.retrieve(keys);
  onPercentDownloadedChange?.(10);
  const indexes = unique.unique(channels.map(({ index }) => index));
  const indexChannels = await client.channels.retrieve(indexes);
  onPercentDownloadedChange?.(20);
  const channelGroups = new Map<channel.Key, Set<channel.Key>>();
  indexChannels.forEach(({ key }) => {
    channelGroups.set(key, new Set([key]));
  });
  channels.forEach(({ key, index }) => {
    const channelGroup = channelGroups.get(index);
    if (channelGroup == null) throw new Error(`Index channel ${index} not found`);
    channelGroup.add(key);
  });
  const keysToColumnHeaders = new Map<channel.Key, string>();
  indexChannels.forEach(({ key, name }) =>
    keysToColumnHeaders.set(key, keysToNames[key] ?? name),
  );
  channels.forEach(({ key, name }) =>
    keysToColumnHeaders.set(key, keysToNames[key] ?? name),
  );

  const simplifiedTimeRanges = TimeRange.simplify(timeRanges);
  // turn the channel groups into arrays
  const channelGroupsAsArrays: Map<channel.Key, channel.Keys> = new Map(
    Array.from(channelGroups.entries()).map(([key, group]) => [key, Array.from(group)]),
  );

  let headerWritten = false;
  let csv = "";
  const newlineSeparator = runtime.getOS() === "Windows" ? "\r\n" : "\n";

  const intervalCount = simplifiedTimeRanges.length * channelGroupsAsArrays.size;
  let percentDownloaded = 20;
  const delta = intervalCount > 0 ? 70 / intervalCount : 0;

  for (const tr of simplifiedTimeRanges) {
    const csvGroups: CSVGroup[] = [];
    for (const [index, keys] of channelGroupsAsArrays) {
      const frame = await client.read(tr, keys);
      csvGroups.push({ index, frame });
      percentDownloaded += delta;
      if (percentDownloaded >= 90) percentDownloaded = 90;
      onPercentDownloadedChange?.(percentDownloaded);
    }
    if (!headerWritten) {
      const headers: string[] = [];
      for (const { frame } of csvGroups)
        for (const key of frame.uniqueKeys) {
          const header = keysToColumnHeaders.get(key) ?? key.toString();
          headers.push(sanitizeValue(header));
        }
      csv += `${headers.join(",")}${newlineSeparator}`;
      headerWritten = true;
    }
    csv += convertFrameGroups(csvGroups, newlineSeparator);
  }
  const data = new TextEncoder().encode(csv);
  if (savePath == null) Runtime.downloadFromBrowser(csv, "text/csv", `${fileName}.csv`);
  else {
    await writeFile(savePath, data);
    addStatus({
      variant: "success",
      message: `Downloaded ${fileName} to ${savePath}`,
    });
  }
  onPercentDownloadedChange?.(100);
  afterDownload?.();
};
