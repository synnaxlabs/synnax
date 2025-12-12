// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, runtime, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { createTestClient } from "@/testutil/client";
import { secondsLinspace } from "@/testutil/telem";

const client = createTestClient();

const delimiter = runtime.getOS() === "Windows" ? "\r\n" : "\n";

/** Helper to collect stream into a string */
const streamToString = async (stream: ReadableStream<Uint8Array>): Promise<string> => {
  const reader = stream.getReader();
  const chunks: Uint8Array[] = [];
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
  }
  const decoder = new TextDecoder();
  return chunks.map((c) => decoder.decode(c)).join("");
};

const parseCSV = (csv: string): string[][] => {
  const lines = csv.trim().split(delimiter);
  return lines.map((line) => line.split(","));
};

const streamToRecords = async (
  stream: ReadableStream<Uint8Array>,
): Promise<string[][]> => {
  const csv = await streamToString(stream);
  return parseCSV(csv);
};

describe("Reader", () => {
  describe("CSV", () => {
    it("should export channels with the same index", async () => {
      const index = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data1 = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: index.key,
      });
      const data2 = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: index.key,
      });
      const start = TimeStamp.seconds(1);
      const writer = await client.openWriter({
        start,
        channels: [index.key, data1.key, data2.key],
      });
      await writer.write({
        [index.key]: [TimeStamp.seconds(1), TimeStamp.seconds(2), TimeStamp.seconds(3)],
        [data1.key]: [10, 20, 30],
        [data2.key]: [100, 200, 300],
      });
      await writer.commit();
      await writer.close();
      const stream = await client.read({
        channels: [index.key, data1.key, data2.key],
        timeRange: { start: TimeStamp.seconds(0), end: TimeStamp.seconds(10) },
        channelNames: new Map([
          [index.key, "Time"],
          [data1.key, "Sensor1"],
          [data2.key, "Sensor2"],
        ]),
        responseType: "csv",
      });
      const records = await streamToRecords(stream);
      expect(records).toEqual([
        ["Time", "Sensor1", "Sensor2"],
        ["1000000000", "10", "100"],
        ["2000000000", "20", "200"],
        ["3000000000", "30", "300"],
      ]);
    });
    it("should export multiple channels with different indexes", async () => {
      const index1 = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data1 = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: index1.key,
      });
      const index2 = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data2 = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: index2.key,
      });
      // Write to first group - timestamps 1, 3, 5
      const writer1 = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels: [index1.key, data1.key],
      });
      await writer1.write({
        [index1.key]: [
          TimeStamp.seconds(1),
          TimeStamp.seconds(3),
          TimeStamp.seconds(5),
        ],
        [data1.key]: [100, 300, 500],
      });
      await writer1.commit();
      await writer1.close();

      // Write to second group - timestamps 2, 4, 6
      const writer2 = await client.openWriter({
        start: TimeStamp.seconds(2),
        channels: [index2.key, data2.key],
      });
      await writer2.write({
        [index2.key]: [
          TimeStamp.seconds(2),
          TimeStamp.seconds(4),
          TimeStamp.seconds(6),
        ],
        [data2.key]: [200, 400, 600],
      });
      await writer2.commit();
      await writer2.close();
      const stream = await client.read({
        channels: [data1.key, data2.key], // Just data channels - indexes auto-included
        timeRange: { start: TimeStamp.seconds(0), end: TimeStamp.seconds(10) },
        channelNames: new Map([
          [index1.key, "Time1"],
          [data1.key, "Data1"],
          [index2.key, "Time2"],
          [data2.key, "Data2"],
        ]),
        responseType: "csv",
      });
      const records = await streamToRecords(stream);
      expect(records).toEqual([
        ["Time1", "Data1", "Time2", "Data2"],
        ["1000000000", "100", "", ""],
        ["", "", "2000000000", "200"],
        ["3000000000", "300", "", ""],
        ["", "", "4000000000", "400"],
        ["5000000000", "500", "", ""],
        ["", "", "6000000000", "600"],
      ]);
    });
    it("should allow downsampling", async () => {
      const index = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: index.key,
      });
      const writer = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels: [index.key, data.key],
      });
      await writer.write({
        [index.key]: [TimeStamp.seconds(1), TimeStamp.seconds(2), TimeStamp.seconds(3)],
        [data.key]: [10, 20, 30],
      });
      await writer.commit();
      await writer.close();
      const stream = await client.read({
        channels: [data.key],
        timeRange: { start: TimeStamp.seconds(0), end: TimeStamp.seconds(10) },
        responseType: "csv",
        iteratorConfig: { downsampleFactor: 2 },
      });
      const records = await streamToRecords(stream);
      expect(records).toEqual([
        [index.name, data.name],
        ["1000000000", "10"],
        ["3000000000", "30"],
      ]);
    });
    it("should handle channels at different uneven rates with correct row ordering", async () => {
      const indexFast = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const dataFast = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: indexFast.key,
      });
      const indexSlow = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const dataSlow = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: indexSlow.key,
      });
      const baseTime = TimeStamp.nanoseconds(0);
      // Write fast data: 0ns, 1ns, 2ns, 3ns, 4ns, 5ns
      const writerFast = await client.openWriter({
        start: baseTime,
        channels: [indexFast.key, dataFast.key],
      });
      await writerFast.write({
        [indexFast.key]: [
          TimeStamp.nanoseconds(0),
          TimeStamp.nanoseconds(1),
          TimeStamp.nanoseconds(2),
          TimeStamp.nanoseconds(3),
          TimeStamp.nanoseconds(4),
          TimeStamp.nanoseconds(5),
        ],
        [dataFast.key]: [1.0, 1.1, 1.2, 1.3, 1.4, 1.5],
      });
      await writerFast.commit();
      await writerFast.close();

      // Write slow data: 0ns, 5ns
      const writerSlow = await client.openWriter({
        start: baseTime,
        channels: [indexSlow.key, dataSlow.key],
      });
      await writerSlow.write({
        [indexSlow.key]: [TimeStamp.nanoseconds(0), TimeStamp.nanoseconds(5)],
        [dataSlow.key]: [2.0, 2.5],
      });
      await writerSlow.commit();
      await writerSlow.close();

      const stream = await client.read({
        channels: [dataFast.key, dataSlow.key],
        timeRange: {
          start: baseTime,
          end: TimeStamp.nanoseconds(6),
        },
        responseType: "csv",
      });
      const records = await streamToRecords(stream);
      expect(records).toEqual([
        [indexFast.name, dataFast.name, indexSlow.name, dataSlow.name],
        ["0", "1", "0", "2"],
        ["1", "1.1", "", ""],
        ["2", "1.2", "", ""],
        ["3", "1.3", "", ""],
        ["4", "1.4", "", ""],
        ["5", "1.5", "5", "2.5"],
      ]);
    });
    it("should handle large amounts of channels", async () => {
      const numGroups = 5;
      const channelsPerGroup = 3;
      const dataKeys: channel.Keys = [];
      const expectedColumns = numGroups * (1 + channelsPerGroup);

      // Store timestamps written per group for building expected rows later
      interface GroupWrite {
        groupIdx: number;
        timestamps: bigint[];
        values: number[][]; // values[sampleIdx][channelIdx]
      }
      const groupWrites: GroupWrite[] = [];

      for (let g = 0; g < numGroups; g++) {
        const index = await client.channels.create({
          name: id.create(),
          dataType: DataType.TIMESTAMP,
          isIndex: true,
        });
        const groupChannels: channel.Keys = [index.key];
        for (let c = 0; c < channelsPerGroup; c++) {
          const data = await client.channels.create({
            name: id.create(),
            dataType: DataType.FLOAT64,
            index: index.key,
          });
          dataKeys.push(data.key);
          groupChannels.push(data.key);
        }
        const writer = await client.openWriter({
          start: TimeStamp.seconds(g + 1),
          channels: groupChannels,
        });
        // Write two timestamps for this group
        const ts1 = TimeStamp.seconds(g + 1);
        const ts2 = TimeStamp.seconds(g + 2);
        const writeData: Record<number, unknown[]> = {
          [index.key]: [ts1, ts2],
        };
        // Write sample values for all channels
        for (let c = 0; c < channelsPerGroup; c++)
          writeData[groupChannels[c + 1]] = [g * 10 + c, g * 10 + c + 1];

        await writer.write(writeData);
        await writer.commit();
        await writer.close();

        // Store the write info
        groupWrites.push({
          groupIdx: g,
          timestamps: [ts1.valueOf(), ts2.valueOf()],
          values: [
            Array.from({ length: channelsPerGroup }, (_, c) => g * 10 + c),
            Array.from({ length: channelsPerGroup }, (_, c) => g * 10 + c + 1),
          ],
        });
      }

      // Build expected rows AFTER all groups created (now we know total columns)
      const rowsByTime = new Map<string, string[]>();
      for (const gw of groupWrites)
        for (let i = 0; i < gw.timestamps.length; i++) {
          const timeStr = gw.timestamps[i].toString();
          if (!rowsByTime.has(timeStr))
            rowsByTime.set(timeStr, Array(expectedColumns).fill(""));

          const row = rowsByTime.get(timeStr)!;
          const colOffset = gw.groupIdx * (1 + channelsPerGroup);
          row[colOffset] = timeStr; // index timestamp
          for (let c = 0; c < channelsPerGroup; c++)
            row[colOffset + 1 + c] = gw.values[i][c].toString();
        }

      // Compose expected rows in time order (ascending)
      const sortedTimes = Array.from(rowsByTime.keys())
        .map((k) => BigInt(k))
        .sort((a, b) => (a < b ? -1 : a > b ? 1 : 0))
        .map((k) => k.toString());

      const expectedRows: string[][] = [];
      for (const timeStr of sortedTimes) expectedRows.push(rowsByTime.get(timeStr)!);

      const stream = await client.read({
        channels: dataKeys,
        timeRange: { start: TimeStamp.seconds(0), end: TimeStamp.seconds(20) },
        responseType: "csv",
      });
      const rows = await streamToRecords(stream);
      // There should be a header and at least the expected number of rows
      expect(rows.length).toBeGreaterThan(1);
      expect(rows.slice(1)).toEqual(expectedRows);
      // Each row should have columns for all groups (index + data channels each)
      rows.forEach((row) => {
        expect(row).toHaveLength(expectedColumns);
      });
    });

    it("should handle empty data gracefully", async () => {
      const index = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: index.key,
      });
      const stream = await client.read({
        channels: [data.key],
        timeRange: { start: TimeStamp.seconds(0), end: TimeStamp.seconds(10) },
        responseType: "csv",
      });
      const rows = await streamToRecords(stream);
      expect(rows).toEqual([[index.name, data.name]]);
    });

    it("should use channel names as default headers", async () => {
      const uniqueSuffix = id.create();
      const indexName = `my_timestamp_${uniqueSuffix}`;
      const dataName = `my_sensor_data_${uniqueSuffix}`;
      const index = await client.channels.create({
        name: indexName,
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data = await client.channels.create({
        name: dataName,
        dataType: DataType.FLOAT64,
        index: index.key,
      });
      const writer = await client.openWriter({
        start: TimeStamp.nanoseconds(1),
        channels: [index.key, data.key],
      });
      await writer.write({
        [index.key]: [TimeStamp.nanoseconds(1)],
        [data.key]: [42],
      });
      await writer.commit();
      await writer.close();
      const stream = await client.read({
        channels: [data.key],
        timeRange: { start: TimeStamp.seconds(0), end: TimeStamp.seconds(100000) },
        responseType: "csv",
      });
      const records = await streamToRecords(stream);
      expect(records).toEqual([
        [index.name, data.name],
        ["1", "42"],
      ]);
    });
    it("should handle channels across domains with gaps in them", async () => {
      const index = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: index.key,
      });
      let writer = await client.openWriter({
        start: TimeStamp.nanoseconds(101),
        channels: [index.key, data.key],
      });
      await writer.write({
        [index.key]: [
          TimeStamp.nanoseconds(101),
          TimeStamp.nanoseconds(102),
          TimeStamp.nanoseconds(103),
        ],
        [data.key]: [10, 11, 12],
      });
      await writer.commit();
      await writer.close();
      writer = await client.openWriter({
        start: TimeStamp.nanoseconds(1),
        channels: [index.key, data.key],
      });
      await writer.write({
        [index.key]: [
          TimeStamp.nanoseconds(1),
          TimeStamp.nanoseconds(2),
          TimeStamp.nanoseconds(3),
        ],
        [data.key]: [1, 2, 3],
      });
      await writer.commit();
      await writer.close();
      const stream = await client.read({
        channels: [data.key],
        timeRange: { start: TimeStamp.nanoseconds(3), end: TimeStamp.nanoseconds(103) },
        responseType: "csv",
      });
      const rows = await streamToRecords(stream);
      expect(rows).toEqual([
        [index.name, data.name],
        ["3", "3"],
        ["101", "10"],
        ["102", "11"],
      ]);
    });
    it("should handle non-overlapping data across domains", async () => {
      // first index will get written from times 10-15, second index from times 13-18
      const index1 = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const index2 = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data1 = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: index1.key,
      });
      const data2 = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: index2.key,
      });
      const writer1 = await client.openWriter({
        start: TimeStamp.nanoseconds(10),
        channels: [index1.key, data1.key],
      });
      await writer1.write({
        [index1.key]: [
          TimeStamp.nanoseconds(10),
          TimeStamp.nanoseconds(11),
          TimeStamp.nanoseconds(12),
          TimeStamp.nanoseconds(13),
          TimeStamp.nanoseconds(14),
          TimeStamp.nanoseconds(15),
        ],
        [data1.key]: [1, 2, 3, 4, 5, 6],
      });
      await writer1.commit();
      await writer1.close();
      const writer2 = await client.openWriter({
        start: TimeStamp.nanoseconds(15),
        channels: [index2.key, data2.key],
      });
      await writer2.write({
        [index2.key]: [
          TimeStamp.nanoseconds(13),
          TimeStamp.nanoseconds(14),
          TimeStamp.nanoseconds(15),
          TimeStamp.nanoseconds(16),
          TimeStamp.nanoseconds(17),
          TimeStamp.nanoseconds(18),
        ],
        [data2.key]: [11, 12, 13, 14, 15, 16],
      });
      await writer2.commit();
      await writer2.close();
      const stream = await client.read({
        channels: [data1.key, data2.key],
        timeRange: { start: TimeStamp.nanoseconds(0), end: TimeStamp.nanoseconds(19) },
        responseType: "csv",
      });
      const rows = await streamToRecords(stream);
      expect(rows).toEqual([
        [index1.name, data1.name, index2.name, data2.name],
        ["10", "1", "", ""],
        ["11", "2", "", ""],
        ["12", "3", "", ""],
        ["13", "4", "13", "11"],
        ["14", "5", "14", "12"],
        ["15", "6", "15", "13"],
        ["", "", "16", "14"],
        ["", "", "17", "15"],
        ["", "", "18", "16"],
      ]);
    });
    it("should handle large dataset requiring multiple iterator calls", async () => {
      // Create 4 groups with different indexes at different rates
      const numGroups = 4;
      const samplesPerGroup = [3000, 2500, 2000, 1500]; // Different sample counts
      const channelsPerGroup = 3;

      interface GroupInfo {
        indexKey: number;
        dataKeys: number[];
        baseTime: TimeStamp;
        sampleCount: number;
        intervalMs: number;
      }

      const groups: GroupInfo[] = [];
      const allDataKeys: number[] = [];

      // Create channels for each group
      for (let g = 0; g < numGroups; g++) {
        const index = await client.channels.create({
          name: `stress_index_${id.create()}`,
          dataType: DataType.TIMESTAMP,
          isIndex: true,
        });

        const dataKeys: number[] = [];
        for (let c = 0; c < channelsPerGroup; c++) {
          const data = await client.channels.create({
            name: `stress_data_${id.create()}`,
            dataType: DataType.FLOAT64,
            index: index.key,
          });
          dataKeys.push(data.key);
          allDataKeys.push(data.key);
        }

        // Different base times and intervals to create interleaving
        const baseTime = TimeStamp.seconds(1000).add(TimeSpan.milliseconds(g * 7));
        const intervalMs = 10 + g * 3; // 10ms, 13ms, 16ms, 19ms intervals

        groups.push({
          indexKey: index.key,
          dataKeys,
          baseTime,
          sampleCount: samplesPerGroup[g],
          intervalMs,
        });
      }

      // Write data to each group in parallel using Promise.all
      await Promise.all(
        groups.map(async (group) => {
          const writer = await client.openWriter({
            start: group.baseTime,
            channels: [group.indexKey, ...group.dataKeys],
          });

          // Write in batches to avoid memory issues
          const batchSize = 500;
          for (
            let batchStart = 0;
            batchStart < group.sampleCount;
            batchStart += batchSize
          ) {
            const batchEnd = Math.min(batchStart + batchSize, group.sampleCount);
            const timestamps: TimeStamp[] = [];
            const dataArrays: number[][] = group.dataKeys.map(() => []);

            for (let i = batchStart; i < batchEnd; i++) {
              timestamps.push(
                group.baseTime.add(TimeSpan.milliseconds(i * group.intervalMs)),
              );
              group.dataKeys.forEach((_, c) => {
                dataArrays[c].push(i * 100 + c);
              });
            }

            const writeData: Record<number, unknown[]> = {
              [group.indexKey]: timestamps,
            };
            group.dataKeys.forEach((key, c) => {
              writeData[key] = dataArrays[c];
            });

            await writer.write(writeData);
          }
          await writer.commit();
          await writer.close();
        }),
      );
      // Calculate expected total samples across all groups
      const totalSamples = samplesPerGroup.reduce((a, b) => a + b, 0);

      // Export the data
      const stream = await client.read({
        channels: allDataKeys,
        timeRange: {
          start: TimeStamp.seconds(999),
          end: TimeStamp.seconds(1100),
        },
        responseType: "csv",
      });

      // Collect all chunks and track streaming behavior
      const reader = stream.getReader();
      const chunks: Uint8Array[] = [];
      let chunkCount = 0;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        chunks.push(value);
        chunkCount++;
      }

      // Verify multiple chunks were produced (proves streaming worked)
      expect(chunkCount).toBeGreaterThan(1);

      // Decode and parse the full CSV
      const decoder = new TextDecoder();
      const csv = chunks.map((c) => decoder.decode(c)).join("");
      const lines = csv.trim().split(delimiter);

      // Header + data rows (some timestamps may align, so rows <= totalSamples)
      expect(lines.length).toBeGreaterThan(1);
      expect(lines.length).toBeLessThanOrEqual(totalSamples + 1);

      // Verify header has correct number of columns
      // Each group has: 1 index + channelsPerGroup data channels
      const expectedColumns = numGroups * (1 + channelsPerGroup);
      const headerColumns = lines[0].split(",");
      expect(headerColumns).toHaveLength(expectedColumns);

      // Verify all data rows have correct column count
      for (let i = 1; i < lines.length; i++) {
        const cols = lines[i].split(",");
        expect(cols).toHaveLength(expectedColumns);
      }

      // Verify timestamps are in ascending order
      const rows = parseCSV(csv);
      let lastTimestamp: bigint | null = null;
      for (let i = 1; i < rows.length; i++)
        // Find the first non-empty timestamp in this row
        for (let g = 0; g < numGroups; g++) {
          const tsCol = g * (1 + channelsPerGroup);
          const tsStr = rows[i][tsCol];
          if (tsStr === "") continue;
          const ts = BigInt(tsStr);
          if (lastTimestamp !== null) expect(ts).toBeGreaterThanOrEqual(lastTimestamp);
          lastTimestamp = ts;
          break;
        }

      // Verify some specific data integrity
      // First row should have data from at least one group
      const firstDataRow = rows[1];
      const nonEmptyValues = firstDataRow.filter((v) => v !== "");
      expect(nonEmptyValues.length).toBeGreaterThan(0);
    });
    it(
      "should handle large dense and sparse indexes with correct ordering and merging",
      { timeout: 15_000 },
      async () => {
        const denseSamples = 100_000;
        const sparseStep = 1_000;
        const sparseSamples = denseSamples / sparseStep;

        const indexFast = await client.channels.create({
          name: `dense_index_${id.create()}`,
          dataType: DataType.TIMESTAMP,
          isIndex: true,
        });
        const dataFast = await client.channels.create({
          name: `dense_data_${id.create()}`,
          dataType: DataType.FLOAT64,
          index: indexFast.key,
        });

        const indexSlow = await client.channels.create({
          name: `sparse_index_${id.create()}`,
          dataType: DataType.TIMESTAMP,
          isIndex: true,
        });
        const dataSlow = await client.channels.create({
          name: `sparse_data_${id.create()}`,
          dataType: DataType.FLOAT64,
          index: indexSlow.key,
        });
        const start = TimeStamp.seconds(0);
        const denseWriter = await client.openWriter({
          start,
          channels: [indexFast.key, dataFast.key],
        });

        const maxBatchSize = 10_000;
        for (
          let batchStart = 1;
          batchStart <= denseSamples;
          batchStart += maxBatchSize
        ) {
          const batchEnd = Math.min(batchStart + maxBatchSize - 1, denseSamples);
          const batchSize = batchEnd - batchStart + 1;
          const times = secondsLinspace(batchStart, batchSize);
          const data = Array.from({ length: batchSize }, (_, i) => i + batchStart);
          await denseWriter.write({ [indexFast.key]: times, [dataFast.key]: data });
        }
        await denseWriter.commit();
        await denseWriter.close();

        const sparseWriter = await client.openWriter({
          start,
          channels: [indexSlow.key, dataSlow.key],
        });

        for (
          let batchStart = 1;
          batchStart < sparseSamples;
          batchStart += maxBatchSize
        ) {
          const batchEnd = Math.min(
            batchStart + (maxBatchSize - 1) * sparseStep,
            batchStart + sparseSamples * sparseStep,
          );
          const times: TimeStamp[] = [];
          const data: number[] = [];

          for (let j = batchStart; j < batchEnd; j += sparseStep) {
            times.push(start.add(TimeSpan.seconds(j)));
            data.push(j); // arbitrary data value
          }
          await sparseWriter.write({ [indexSlow.key]: times, [dataSlow.key]: data });
        }
        await sparseWriter.commit();
        await sparseWriter.close();

        const stream = await client.read({
          channels: [dataFast.key, dataSlow.key],
          timeRange: {
            start: TimeStamp.seconds(0),
            end: start.add(TimeSpan.seconds(denseSamples + 1)),
          },
          responseType: "csv",
        });

        const reader = stream.getReader();
        const decoder = new TextDecoder();

        let buffer = "";
        let chunkCount = 0;
        let isHeader = true;
        let totalRows = 0; // data rows only (exclude header)
        let sparseRows = 0;
        let lastTimestamp: bigint | null = null;

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          chunkCount++;
          buffer += decoder.decode(value);
          while (true) {
            const idx = buffer.indexOf(delimiter);
            if (idx === -1) break;
            const line = buffer.slice(0, idx);
            buffer = buffer.slice(idx + delimiter.length);
            if (line === "") continue;
            if (isHeader) {
              const headerCols = line.split(",");
              expect(headerCols).toEqual([
                indexFast.name,
                dataFast.name,
                indexSlow.name,
                dataSlow.name,
              ]);
              isHeader = false;
              continue;
            }

            totalRows++;
            const cols = line.split(",");
            expect(cols).toHaveLength(4);
            const [fastTsStr, fastValStr, slowTsStr, slowValStr] = cols;

            expect(fastTsStr).not.toBe("");
            expect(fastValStr).not.toBe("");

            const ts = BigInt(fastTsStr);
            if (lastTimestamp !== null) expect(ts).toBeGreaterThan(lastTimestamp);
            lastTimestamp = ts;

            if (slowValStr !== "") {
              sparseRows++;
              // When sparse has data, its timestamp should match dense's timestamp
              expect(slowTsStr).toBe(fastTsStr);
              expect(slowValStr).not.toBe("");
            }
          }
        }
        expect(chunkCount).toBeGreaterThan(1);
        expect(totalRows).toBe(denseSamples);
        expect(sparseRows).toBe(sparseSamples);
      },
    );
  });
});
