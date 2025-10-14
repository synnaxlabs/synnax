// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type Frame, UnexpectedError } from "@synnaxlabs/client";
import { DataType, type MultiSeries, type TimeStamp } from "@synnaxlabs/x/telem";

export type CSVGroup = {
  index: channel.KeyOrName;
  frame: Frame;
};

export const convertFrameGroups = (
  // use maps instead of objects to avoid the key being converted to a string
  groups: CSVGroup[],
  newline: "\r\n" | "\n" = "\n",
): string => {
  // step1: validate that keys are not repeated between frames.
  const keyOrNamesSet = new Set<channel.KeyOrName>();
  for (const { frame } of groups)
    for (const keyOrName of frame.uniqueColumns) {
      if (keyOrNamesSet.has(keyOrName))
        throw new Error(`Channel ${keyOrName} is repeated between multiple frames`);
      keyOrNamesSet.add(keyOrName);
    }

  // step2: validate that all frames have an index key corresponding to a timestamp and
  // the right length for each series.
  for (const { index, frame } of groups) {
    const indexSeries = frame.get(index);
    if (indexSeries.length === 0)
      throw new Error(`No data found for index channel ${index}`);
    if (!indexSeries.dataType.equals(DataType.TIMESTAMP))
      throw new Error(`Index channel ${index} is not of type timestamp`);
    if (!frame.isVertical)
      throw new Error(`Frame with index channel ${index} is not vertical`);
    const length = indexSeries.length;
    frame.forEach((key, series) => {
      if (series.length !== length)
        throw new Error(
          `Series for channel ${key} is not the same length (${series.length}) as the series for index channel ${index} (${length})`,
        );
    });
  }
  // step3: For each group, iterate through the index series
  const bodyEntries: BodyEntryInfo[] = [];
  groups.forEach(({ index, frame }) => {
    const records: SeveralValueEntryInfo[] = [];
    const indexSeries = frame.get(index) as MultiSeries<TimeStamp>;
    for (let i = 0; i < indexSeries.length; i++) {
      const time = indexSeries.at(i, true);
      const entries: string[] = [];
      frame.uniqueColumns.forEach((col) => {
        const value = frame.get(col).at(i, true);
        entries.push(sanitizeValue(value.toString()));
      });
      records.push({ time, records: entries.join(",") });
    }
    bodyEntries.push({
      indexKey: index,
      records,
      columnCount: frame.uniqueColumns.length,
    });
  });

  const columnCounts: number[] = bodyEntries.map((entry) => entry.columnCount);
  const parsedBodyEntries: ParsedBodyEntryInfo[] = bodyEntries.map((entry, i) => ({
    beforeCommaCount: columnCounts.slice(0, i).reduce((acc, curr) => acc + curr, 0),
    afterCommaCount: columnCounts.slice(i + 1).reduce((acc, curr) => acc + curr, 0),
    records: entry.records,
  }));

  const rows: string[] = [];
  while (true) {
    const currentEntry = parsedBodyEntries.shift();
    if (currentEntry == null) break;
    const { beforeCommaCount, afterCommaCount, records } = currentEntry;
    const currentRecord = records.shift();
    if (currentRecord == null) throw new UnexpectedError("No records left");
    const string = currentRecord.records;
    const row = ",".repeat(beforeCommaCount) + string + ",".repeat(afterCommaCount);
    rows.push(row);
    if (records.length === 0) continue;
    // insert the record into the correct place in the array based off of the
    // timestamps using binary search.
    const nextTime = records[0].time;
    let left = 0;
    let right = parsedBodyEntries.length;
    while (left < right) {
      const mid = Math.floor((left + right) / 2);
      if (parsedBodyEntries[mid].records[0].time.valueOf() > nextTime.valueOf())
        right = mid;
      else left = mid + 1;
    }
    if (left === parsedBodyEntries.length) parsedBodyEntries.push(currentEntry);
    else parsedBodyEntries.splice(left, 0, currentEntry);
  }
  return rows.join(newline) + newline;
};

export const sanitizeValue = (value: string): string => {
  if (!/[",\n]/.test(value)) return value;
  // If value contains a comma, quote, or newline, wrap in double quotes and escape
  // existing double quotes.
  const escaped = value.replace(/"/g, '""');
  return `"${escaped}"`;
};

type SeveralValueEntryInfo = {
  time: TimeStamp;
  records: string;
};

type BodyEntryInfo = {
  indexKey: channel.KeyOrName;
  records: SeveralValueEntryInfo[];
  columnCount: number;
};

type ParsedBodyEntryInfo = {
  beforeCommaCount: number;
  afterCommaCount: number;
  records: SeveralValueEntryInfo[];
};
