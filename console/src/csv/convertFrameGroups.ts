// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type Frame, UnexpectedError } from "@synnaxlabs/client";
import { compare, csv, DataType } from "@synnaxlabs/x";

export interface FrameGroup {
  frame: Frame;
  index: channel.Key;
}

export const convertFrameGroups = (
  frameGroups: FrameGroup[],
  newline: csv.RecordDelimiter = "\n",
): string => {
  // validate that keys are not repeated between frames.
  const keySet = new Set<channel.Key>();
  for (const { frame } of frameGroups)
    for (const key of frame.uniqueKeys) {
      if (keySet.has(key))
        throw new Error(`Channel ${key} is repeated between multiple frames`);
      keySet.add(key);
    }

  // validate that all frames have an index key corresponding to a timestamp and the
  // right length for each series.
  for (const { index, frame } of frameGroups) {
    const indexMultiSeries = frame.get(index);
    if (!indexMultiSeries.dataType.equals(DataType.TIMESTAMP))
      throw new Error(`Index channel ${index} is not of type timestamp`);
    const length = indexMultiSeries.length;
    frame.forEachUnique((key, multiSeries) => {
      if (multiSeries.length !== length)
        throw new Error(
          `Multi-series for channel ${key} is not the same length (${multiSeries.length}) as the multi-series for index channel ${index} (${length})`,
        );
    });
  }

  // For each group, iterate through the index series
  const rawBodyEntries: RawBodyEntryInfo[] = [];
  frameGroups.forEach(({ frame, index }) => {
    const records: RecordInfo[] = [];
    const indexSeries = frame.get(index).as("bigint");
    for (let i = 0; i < indexSeries.length; i++) {
      const time = indexSeries.at(i, true);
      const entries: string[] = [];
      frame.uniqueKeys.forEach((key) => {
        const value = frame.get(key).at(i, true);
        entries.push(csv.maybeEscapeField(value.toString()));
      });
      records.push({ time, records: entries.join(",") });
    }
    rawBodyEntries.push({
      indexKey: index,
      records,
      columnCount: frame.uniqueColumns.length,
    });
  });

  const columnCounts: number[] = rawBodyEntries.map(({ columnCount }) => columnCount);
  const bodyEntries: ParsedBodyEntryInfo[] = rawBodyEntries.flatMap((entry, i) => {
    if (entry.records.length === 0) return [];
    return {
      beforeColumnCount: columnCounts.slice(0, i).reduce((acc, curr) => acc + curr, 0),
      columnCount: columnCounts[i],
      afterColumnCount: columnCounts.slice(i + 1).reduce((acc, curr) => acc + curr, 0),
      records: entry.records,
    };
  });

  bodyEntries.sort((a, b) =>
    Number(a.records[0].time.valueOf() - b.records[0].time.valueOf()),
  );

  const rows: string[] = [];
  while (true) {
    const currentEntries: ParsedBodyEntryInfo[] = [];
    const currentEntry = bodyEntries.shift();
    if (currentEntry == null) break;
    currentEntries.push(currentEntry);
    while (true) {
      const nextEntry = bodyEntries[0];
      if (nextEntry == null) break;
      if (nextEntry.records[0].time !== currentEntry.records[0].time) break;
      currentEntries.push(nextEntry);
      bodyEntries.shift();
    }

    currentEntries.sort((a, b) => a.beforeColumnCount - b.beforeColumnCount);
    let row = ",".repeat(currentEntries[0].beforeColumnCount);
    currentEntries.forEach((entry, i) => {
      const record = entry.records.shift();
      if (record == null) throw new UnexpectedError("No records left");
      row += record.records;
      const nextEntry = currentEntries.at(i + 1);
      if (nextEntry == null) {
        row += ",".repeat(entry.afterColumnCount);
        return;
      }
      row += ",".repeat(
        nextEntry.beforeColumnCount - (entry.columnCount - 1 + entry.beforeColumnCount),
      );
    });
    rows.push(row);

    // insert the record into the correct place in the array based off of the timestamps
    // using binary search.
    for (const entry of currentEntries) {
      if (entry.records.length === 0) continue;
      compare.insert(bodyEntries, entry, (a, b) =>
        Number(a.records[0].time - b.records[0].time),
      );
    }
  }
  if (rows.length === 0) return "";
  return rows.join(newline) + newline;
};

interface RecordInfo {
  records: string;
  time: bigint;
}

interface RawBodyEntryInfo {
  indexKey: channel.Key;
  records: RecordInfo[];
  columnCount: number;
}

interface ParsedBodyEntryInfo {
  beforeColumnCount: number;
  columnCount: number;
  afterColumnCount: number;
  records: RecordInfo[];
}
