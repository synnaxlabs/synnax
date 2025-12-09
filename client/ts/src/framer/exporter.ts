// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { csv, unique } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { type Frame } from "@/framer/frame";
import { AUTO_SPAN, type Iterator } from "@/framer/iterator";

export interface CreateCSVExportStreamParams {
  iterator: Iterator;
  channelPayloads: channel.Payload[];
  headers?: Map<channel.KeyOrName, string>;
  delimiter?: csv.RecordDelimiter;
}

export const createCSVExportStream = ({
  iterator,
  channelPayloads,
  headers,
  delimiter = "\r\n",
}: CreateCSVExportStreamParams): ReadableStream<Uint8Array> => {
  const encoder = new TextEncoder();
  let headerWritten = false;
  let seekDone = false;
  const groups = groupChannelsByIndex(channelPayloads);
  const columns = buildColumnMeta(channelPayloads, groups, headers);
  const pendingRecords: RecordEntry[] = [];

  const extractRecordsFromFrame = (frame: Frame): void => {
    groups.forEach((group, groupIdx) => {
      const indexSeries = frame.get(group.indexKey);
      if (indexSeries.length === 0) return;
      const groupColumns = columns.filter((c) => c.groupIdx === groupIdx);
      for (let i = 0; i < indexSeries.length; i++) {
        const time = indexSeries.at(i, true) as bigint;
        const values: string[] = groupColumns.map((col) => {
          const series = frame.get(col.key);
          const value = series.at(i, true);
          return csv.formatValue(value);
        });
        pendingRecords.push({ time, values, groupIdx });
      }
    });
    pendingRecords.sort((a, b) => Number(a.time - b.time));
  };

  const buildCSVRows = (maxRows: number): string[] => {
    const rows: string[] = [];
    while (pendingRecords.length > 0 && rows.length < maxRows) {
      const minTime = pendingRecords[0].time;
      const recordsAtTime: RecordEntry[] = [];
      while (pendingRecords.length > 0 && pendingRecords[0].time === minTime)
        recordsAtTime.push(pendingRecords.shift()!);
      const row: string[] = [];
      for (let groupIdx = 0; groupIdx < groups.length; groupIdx++) {
        const groupColumns = columns.filter((c) => c.groupIdx === groupIdx);
        const record = recordsAtTime.find((r) => r.groupIdx === groupIdx);
        if (record != null) row.push(...record.values);
        else row.push(...groupColumns.map(() => ""));
      }
      rows.push(row.join(","));
    }
    return rows;
  };

  return new ReadableStream<Uint8Array>({
    async pull(controller): Promise<void> {
      try {
        if (!seekDone) {
          await iterator.seekFirst();
          seekDone = true;
        }
        if (!headerWritten) {
          const headerRow = columns.map((c) => csv.formatValue(c.header)).join(",");
          controller.enqueue(encoder.encode(`${headerRow}${delimiter}`));
          headerWritten = true;
        }
        if (pendingRecords.length < 1000) {
          const hasMore = await iterator.next(AUTO_SPAN);
          if (hasMore) extractRecordsFromFrame(iterator.value);
        }
        const rows = buildCSVRows(1000);
        if (rows.length > 0)
          controller.enqueue(encoder.encode(`${rows.join(delimiter)}${delimiter}`));
        if (pendingRecords.length === 0) {
          const hasMore = await iterator.next(AUTO_SPAN);
          if (!hasMore) {
            await iterator.close();
            controller.close();
            return;
          }
          extractRecordsFromFrame(iterator.value);
        }
      } catch (error) {
        await iterator.close();
        controller.error(error);
      }
    },

    async cancel(): Promise<void> {
      await iterator.close();
    },
  });
};

interface ChannelGroup {
  indexKey: channel.Key;
  channelKeys: channel.Keys;
}

const groupChannelsByIndex = (channels: channel.Payload[]): ChannelGroup[] => {
  const groupMap = new Map<channel.Key, channel.Keys>();
  const indexKeys = unique.unique(
    channels.map((ch) => ch.index).filter((k) => k !== 0),
  );
  indexKeys.forEach((indexKey) => {
    groupMap.set(indexKey, [indexKey]);
  });
  channels.forEach((ch) => {
    if (ch.isIndex) return;
    const group = groupMap.get(ch.index);
    if (group != null && !group.includes(ch.key)) group.push(ch.key);
  });
  return Array.from(groupMap.entries()).map(([indexKey, channelKeys]) => ({
    indexKey,
    channelKeys,
  }));
};

interface ColumnMeta {
  key: channel.Key;
  header: string;
  groupIdx: number;
}

const buildColumnMeta = (
  channels: channel.Payload[],
  groups: ChannelGroup[],
  headers?: Map<channel.KeyOrName, string>,
): ColumnMeta[] => {
  const channelMap = new Map(channels.map((ch) => [ch.key, ch]));
  const result: ColumnMeta[] = [];
  groups.forEach((group, groupIdx) => {
    group.channelKeys.forEach((key) => {
      const ch = channelMap.get(key);
      if (ch == null) return;
      result.push({
        key,
        header: headers?.get(key) ?? headers?.get(ch.name) ?? ch.name,
        groupIdx,
      });
    });
  });
  return result;
};

interface RecordEntry {
  time: bigint;
  values: string[];
  groupIdx: number;
}
