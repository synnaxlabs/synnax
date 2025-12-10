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
  type Frame,
  type Synnax as Client,
  UnexpectedError,
} from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { csv as xcsv, runtime, TimeRange, unique } from "@synnaxlabs/x";
import { save } from "@tauri-apps/plugin-dialog";
import { writeTextFile } from "@tauri-apps/plugin-fs";

import { convertFrameGroups, type FrameGroup } from "@/csv/convertFrameGroups";
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
  const channels = await client.channels.retrieve(keys, { virtual: false });
  onPercentDownloadedChange?.(10);
  const indexes = unique.unique(channels.map(({ index }) => index));
  const indexChannels = await client.channels.retrieve(indexes);
  onPercentDownloadedChange?.(20);
  const channelGroups = new Map<channel.Key, channel.Keys>();
  indexChannels.forEach(({ key }) => {
    channelGroups.set(key, [key]);
  });
  channels.forEach(({ key, index }) => {
    const channelGroup = channelGroups.get(index);
    if (channelGroup == null)
      throw new UnexpectedError(`Index channel ${index} not found`);
    channelGroup.push(key);
  });
  for (const [index, channels] of channelGroups.entries())
    channelGroups.set(index, unique.unique(channels));

  const simplifiedTimeRanges = TimeRange.simplify(timeRanges);

  const intervalCount = simplifiedTimeRanges.length * channelGroups.size;
  let percentDownloaded = 20;
  const delta = intervalCount > 0 ? 70 / intervalCount : 0;
  const readPromiseResults: Record<string, Frame> = {};
  const promises = simplifiedTimeRanges.flatMap((tr) =>
    Array.from(channelGroups.entries()).map(async ([index, keys]) => {
      const frame = await client.read(tr, keys);
      percentDownloaded += delta;
      if (percentDownloaded > 90) percentDownloaded = 90;
      onPercentDownloadedChange?.(percentDownloaded);
      readPromiseResults[getKey(tr, index)] = frame;
    }),
  );
  await Promise.all(promises);

  let headerWritten = false;
  const keysToColumnHeaders = new Map<channel.Key, string>();
  [...channels, ...indexChannels].forEach(({ key, name }) =>
    keysToColumnHeaders.set(key, keysToNames[key] ?? name),
  );
  let csv = "";
  const newlineSeparator = runtime.getOS() === "Windows" ? "\r\n" : "\n";

  for (const tr of simplifiedTimeRanges) {
    const frameGroups: FrameGroup[] = [];
    for (const index of channelGroups.keys()) {
      const frame = readPromiseResults[getKey(tr, index)];
      frameGroups.push({ index, frame });
    }
    if (!headerWritten) {
      const headers: string[] = [];
      for (const { frame } of frameGroups)
        for (const key of frame.uniqueKeys) {
          const header = keysToColumnHeaders.get(key) ?? key.toString();
          headers.push(xcsv.maybeEscapeField(header));
        }
      csv += `${headers.join(",")}${newlineSeparator}`;
      headerWritten = true;
    }
    csv += convertFrameGroups(frameGroups, newlineSeparator);
  }

  if (savePath == null) Runtime.downloadFromBrowser(csv, "text/csv", `${fileName}.csv`);
  else {
    await writeTextFile(savePath, csv);
    addStatus({
      variant: "success",
      message: `Downloaded ${fileName} to ${savePath}`,
    });
  }
  onPercentDownloadedChange?.(100);
  afterDownload?.();
};

const getKey = (tr: TimeRange, index: channel.Key): string =>
  `${tr.toString()}${index}`;
