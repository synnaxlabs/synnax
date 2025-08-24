// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DisconnectedError, UnexpectedError } from "@synnaxlabs/client";
import { type Channel, Status, Synnax } from "@synnaxlabs/pluto";
import { type TimeRange } from "@synnaxlabs/x";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";

export interface DownloadArgs {
  timeRange: TimeRange;
  lines: Channel.LineProps[];
  name: string;
}

export const useDownloadAsCSV = (): ((args: DownloadArgs) => void) => {
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  return useCallback(
    ({ timeRange, lines, name }: DownloadArgs) =>
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
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
        const res = await client.read({
          timeRange,
          channelNames,
          keys: channelKeys,
          responseType: "csv",
        });
        if (res.body == null) throw new UnexpectedError("HTTP response body is null");
        await writeFile(savePath, res.body);
      }, `Failed to download CSV for ${name}`),
    [client, handleError],
  );
};
