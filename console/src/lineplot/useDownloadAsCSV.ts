// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type framer, type Synnax } from "@synnaxlabs/client";
import { type Channel, Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { type TimeRange, unique } from "@synnaxlabs/x";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";

import { NULL_CLIENT_ERROR } from "@/errors";

interface DownloadProps {
  timeRange: TimeRange;
  client: Synnax;
  lines: Channel.LineProps[];
  name: string;
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
  return rows.join("\n");
};

const download = async ({
  lines,
  client,
  timeRange,
  name = "synnax-data",
}: DownloadProps): Promise<void> => {
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
  const frame = await client.read(timeRange, keys);
  const csv = frameToCSV(columns, frame);
  const savePath = await save({ defaultPath: `${name}.csv` });
  if (savePath == null) return;
  const data = new TextEncoder().encode(csv);
  await writeFile(savePath, data);
};

interface DownloadArgs extends Omit<DownloadProps, "client"> {}

export const useDownloadAsCSV = (): ((args: DownloadArgs) => void) => {
  const handleError = Status.useErrorHandler();
  const client = PSynnax.use();
  return ({ timeRange, lines, name }: DownloadArgs) =>
    handleError(async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      await download({ timeRange, lines, client, name });
    }, "Failed to download CSV");
};
