// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { csv } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type Frame } from "@/framer/frame";
import { type Iterator } from "@/framer/iterator";

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
  const { columns, columnsByIndexKey, emptyGroupStrings } = buildColumnMeta(
    channelPayloads,
    groups,
    headers,
  );
  // Use a cursor-based approach instead of having to call .shift() for O(1) access
  let pendingRecords: RecordEntry[] = [];
  let pendingCursor = 0;
  let stagedRecords: RecordEntry[] = [];

  const extractRecordsFromFrame = (frame: Frame): void => {
    for (const [indexKey] of groups) {
      const indexSeries = frame.get(indexKey);
      if (indexSeries.length === 0) continue;
      const groupColumns = columnsByIndexKey.get(indexKey) ?? [];
      // Pre-fetch all series for this group to avoid repeated lookups
      const seriesData = groupColumns.map((col) => frame.get(col.key));
      for (let i = 0; i < indexSeries.length; i++) {
        const time = indexSeries.at(i, true) as bigint;
        const values = seriesData.map((series) => csv.formatValue(series.at(i, true)));
        stagedRecords.push({ time, values, indexKey });
      }
    }
  };

  const buildCSVRows = (maxRows: number, flush: boolean = false): string[] => {
    if (stagedRecords.length > 0) {
      stagedRecords.sort((a, b) => Number(a.time - b.time));
      if (pendingCursor > 0) {
        pendingRecords = pendingRecords.slice(pendingCursor);
        pendingCursor = 0;
      }
      pendingRecords = mergeSortedRecords(pendingRecords, stagedRecords);
      stagedRecords = [];
    }
    const rows: string[] = [];
    const pendingLen = pendingRecords.length;
    while (pendingCursor < pendingLen && rows.length < maxRows) {
      const minTime = pendingRecords[pendingCursor].time;
      // Don't output the last timestamp unless flushing - more data might arrive
      // Optimization: only check if last record has same time (since array is sorted)
      if (!flush && pendingRecords[pendingLen - 1].time === minTime) break;
      // Collect all records at this timestamp using cursor (O(1) per record)
      // Use Map keyed by indexKey for O(1) lookup instead of find()
      const recordsByGroup = new Map<channel.Key, RecordEntry>();
      while (
        pendingCursor < pendingLen &&
        pendingRecords[pendingCursor].time === minTime
      ) {
        const record = pendingRecords[pendingCursor++];
        recordsByGroup.set(record.indexKey, record);
      }
      // Build row string directly
      const rowParts: string[] = [];
      for (const [indexKey] of groups) {
        const record = recordsByGroup.get(indexKey);
        rowParts.push(
          record?.values.join(",") ?? emptyGroupStrings.get(indexKey) ?? "",
        );
      }
      rows.push(rowParts.join(","));
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
        const bufferedCount =
          pendingRecords.length - pendingCursor + stagedRecords.length;
        if (bufferedCount < 1000) {
          const hasMore = await iterator.next();
          if (hasMore) extractRecordsFromFrame(iterator.value);
        }
        const rows = buildCSVRows(1000);
        if (rows.length > 0)
          controller.enqueue(encoder.encode(`${rows.join(delimiter)}${delimiter}`));
        const remainingPending = pendingRecords.length - pendingCursor;
        if (remainingPending === 0 || stagedRecords.length === 0) {
          const hasMore = await iterator.next();
          if (!hasMore) {
            // Flush remaining records
            const finalRows = buildCSVRows(Infinity, true);
            if (finalRows.length > 0)
              controller.enqueue(
                encoder.encode(`${finalRows.join(delimiter)}${delimiter}`),
              );
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

const groupChannelsByIndex = (
  channels: channel.Payload[],
): Map<channel.Key, channel.Keys> => {
  const groupMap = new Map<channel.Key, channel.Keys>();
  for (const ch of channels) {
    if (ch.index === 0) continue;
    let group = groupMap.get(ch.index);
    if (group == null) {
      group = [ch.index];
      groupMap.set(ch.index, group);
    }
    if (!ch.isIndex && !group.includes(ch.key)) group.push(ch.key);
  }
  return groupMap;
};

interface ColumnMeta {
  key: channel.Key;
  header: string;
}

interface ColumnMetaResult {
  columns: ColumnMeta[];
  columnsByIndexKey: Map<channel.Key, ColumnMeta[]>;
  emptyGroupStrings: Map<channel.Key, string>;
}

const buildColumnMeta = (
  channels: channel.Payload[],
  groups: Map<channel.Key, channel.Keys>,
  headers?: Map<channel.KeyOrName, string>,
): ColumnMetaResult => {
  const channelMap = new Map(channels.map((ch) => [ch.key, ch]));
  const columns: ColumnMeta[] = [];
  const columnsByIndexKey = new Map<channel.Key, ColumnMeta[]>();
  const emptyGroupStrings = new Map<channel.Key, string>();

  for (const [indexKey, channelKeys] of groups) {
    const groupColumns: ColumnMeta[] = [];
    for (const key of channelKeys) {
      const ch = channelMap.get(key);
      if (ch == null) throw new UnexpectedError(`Channel ${key} not found`);
      const meta: ColumnMeta = {
        key,
        header: headers?.get(key) ?? headers?.get(ch.name) ?? ch.name,
      };
      columns.push(meta);
      groupColumns.push(meta);
    }
    columnsByIndexKey.set(indexKey, groupColumns);
    // Pre-compute empty group string for fast row building
    emptyGroupStrings.set(indexKey, ",".repeat(groupColumns.length - 1));
  }

  return { columns, columnsByIndexKey, emptyGroupStrings };
};
interface RecordEntry {
  time: bigint;
  values: string[];
  indexKey: channel.Key;
}

const mergeSortedRecords = (a: RecordEntry[], b: RecordEntry[]): RecordEntry[] => {
  const result: RecordEntry[] = [];
  let i = 0;
  let j = 0;
  while (i < a.length && j < b.length)
    if (a[i].time <= b[j].time) result.push(a[i++]);
    else result.push(b[j++]);
  result.push(...a.slice(i), ...b.slice(j));
  return result;
};
