// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type Channel, Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { type TimeRange } from "@synnaxlabs/x";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";

import { NULL_CLIENT_ERROR } from "@/errors";

export interface DownloadArgs {
  timeRange: TimeRange;
  lines: Channel.LineProps[];
  name: string;
}

export const useDownloadAsCSV = (): ((args: DownloadArgs) => void) => {
  const handleError = Status.useErrorHandler();
  const client = PSynnax.use();
  return useCallback(
    ({ timeRange, lines, name }: DownloadArgs) =>
      handleError(async () => {
        if (client == null) throw NULL_CLIENT_ERROR;
        const channelNames: Record<channel.Key, string> = {};
        const channelKeys: channel.Key[] = [];
        lines.forEach(({ channels, label, variant }) => {
          const keys = [channels.y, channels.x];
          const hasLabel = variant !== "dynamic" && label != null;
          keys.forEach((key) => {
            if (key != null && key !== 0 && typeof key === "number") {
              channelKeys.push(key);
              if (hasLabel) channelNames[key] = label;
            }
          });
        });
        const savePath = await save({
          title: "title",
          defaultPath: `${name}.csv`,
          filters: [{ name: "CSV", extensions: ["csv"] }],
        });
        if (savePath == null) return;
        const res = await client.export.csv({
          timeRange,
          channelNames,
          keys: channelKeys,
        });
        // Ideally, we would pass `res.body` directly to `writeFile`, that way we avoid
        // loading the data for the CSV into memory. However, there are currently bugs
        // in Tauri's implementation of `writeFile` that prevent this from working for
        // passing in `ReadableStream<Uint8Array>`s:
        // https://github.com/tauri-apps/plugins-workspace/issues/2835
        //
        // Loading into memory is a workaround for now:
        // https://linear.app/synnax/issue/SY-2656/dont-load-csv-data-into-memory
        const data = await res.bytes();
        await writeFile(savePath, data);
      }, `Failed to download CSV for ${name}`),
    [client, handleError],
  );
};
