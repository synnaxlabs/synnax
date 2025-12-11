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
import { TimeRange } from "@synnaxlabs/x";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";

import { Runtime } from "@/runtime";

export interface DownloadArgs {
  timeRanges: TimeRange[];
  keys: channel.Keys;
  keysToNames?: Record<channel.Key, string>;
  name: string;
  afterDownload?: () => void;
  onPercentDownloadedChange?: (percent: number) => void;
}

export const useDownload = (): ((args: DownloadArgs) => void) => {
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  return (args: DownloadArgs) => {
    const { name, onPercentDownloadedChange } = args;
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      try {
        await download({ ...args, client, addStatus });
      } catch (e) {
        onPercentDownloadedChange?.(0);
        throw e;
      }
    }, `Failed to download CSV data for ${name}`);
  };
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
  name,
  afterDownload,
  onPercentDownloadedChange,
  addStatus,
}: DownloadContext): Promise<void> => {
  let savePath: string | null = null;
  if (Runtime.ENGINE === "tauri") {
    savePath = await save({ defaultPath: `${name}.csv` });
    if (savePath == null) return;
  }

  const simplifiedTimeRanges = TimeRange.simplify(timeRanges);
  const mergedTimeRange = new TimeRange({
    start: simplifiedTimeRanges[0].start,
    end: simplifiedTimeRanges[simplifiedTimeRanges.length - 1].end,
  });

  const headers = new Map<channel.Key, string>();
  for (const [key, name] of Object.entries(keysToNames)) headers.set(Number(key), name);

  onPercentDownloadedChange?.(10);
  const stream = await client.read({
    channels: keys,
    timeRange: mergedTimeRange,
    channelNames: new Map(Object.entries(keysToNames)),
    responseType: "csv",
  });
  if (savePath != null) await writeFile(savePath, stream);
  else savePath = await Runtime.downloadStreamFromBrowser(stream, `${name}.csv`);
  addStatus({ variant: "success", message: `Downloaded ${name} to ${savePath}` });
  onPercentDownloadedChange?.(100);
  afterDownload?.();
};
