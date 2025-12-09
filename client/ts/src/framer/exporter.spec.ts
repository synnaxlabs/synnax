// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

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

/** Parse CSV string into rows */
const parseCSV = (csv: string): string[][] => {
  const lines = csv.trim().split("\n");
  return lines.map((line) => line.split(","));
};

describe("Exporter", () => {
  describe("exportCSV", () => {
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
        [index.key]: [
          TimeStamp.seconds(1),
          TimeStamp.seconds(2),
          TimeStamp.seconds(3),
        ],
        [data1.key]: [10, 20, 30],
        [data2.key]: [100, 200, 300],
      });
      await writer.commit();
      await writer.close();

      const stream = await client.exportCSV({
        channels: [index.key, data1.key, data2.key],
        timeRange: { start: TimeStamp.seconds(0), end: TimeStamp.seconds(10) },
        headers: new Map([
          [index.key, "Time"],
          [data1.key, "Sensor1"],
          [data2.key, "Sensor2"],
        ]),
      });

      const csv = await streamToString(stream);

      // Check the full CSV structure
      const lines = csv.trim().split("\n");
      expect(lines).toHaveLength(4);
      expect(lines[0]).toBe("Time,Sensor1,Sensor2");

      // Data rows - timestamps are raw int64 nanoseconds
      const row1 = lines[1].split(",");
      const row2 = lines[2].split(",");
      const row3 = lines[3].split(",");

      // TimeStamp.seconds(1) = 1e9 nanoseconds
      expect(row1[0]).toBe("1000000000");
      expect(row1[1]).toBe("10");
      expect(row1[2]).toBe("100");

      expect(row2[0]).toBe("2000000000");
      expect(row2[1]).toBe("20");
      expect(row2[2]).toBe("200");

      expect(row3[0]).toBe("3000000000");
      expect(row3[1]).toBe("30");
      expect(row3[2]).toBe("300");
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

      const stream = await client.exportCSV({
        channels: [data1.key, data2.key], // Just data channels - indexes auto-included
        timeRange: { start: TimeStamp.seconds(0), end: TimeStamp.seconds(10) },
        headers: new Map([
          [index1.key, "Time1"],
          [data1.key, "Data1"],
          [index2.key, "Time2"],
          [data2.key, "Data2"],
        ]),
      });

      const csv = await streamToString(stream);
      const rows = parseCSV(csv);

      // Header row
      expect(rows[0]).toEqual(["Time1", "Data1", "Time2", "Data2"]);

      // Should have 6 data rows (timestamps 1,2,3,4,5,6)
      expect(rows).toHaveLength(7);

      // Verify rows are in timestamp order and properly interleaved
      // Row 1 (t=1): group1 has data, group2 empty
      expect(rows[1][1]).toBe("100");
      expect(rows[1][3]).toBe("");

      // Row 2 (t=2): group1 empty, group2 has data
      expect(rows[2][1]).toBe("");
      expect(rows[2][3]).toBe("200");

      // Row 3 (t=3): group1 has data, group2 empty
      expect(rows[3][1]).toBe("300");
      expect(rows[3][3]).toBe("");

      // Row 4 (t=4): group1 empty, group2 has data
      expect(rows[4][1]).toBe("");
      expect(rows[4][3]).toBe("400");
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

      const baseTime = TimeStamp.seconds(1000);

      // Write fast data: 0ms, 100ms, 200ms, 300ms, 400ms
      const writerFast = await client.openWriter({
        start: baseTime,
        channels: [indexFast.key, dataFast.key],
      });
      await writerFast.write({
        [indexFast.key]: [
          baseTime,
          baseTime.add(TimeSpan.milliseconds(100)),
          baseTime.add(TimeSpan.milliseconds(200)),
          baseTime.add(TimeSpan.milliseconds(300)),
          baseTime.add(TimeSpan.milliseconds(400)),
        ],
        [dataFast.key]: [1.0, 1.1, 1.2, 1.3, 1.4],
      });
      await writerFast.commit();
      await writerFast.close();

      // Write slow data: 0ms, 500ms
      const writerSlow = await client.openWriter({
        start: baseTime,
        channels: [indexSlow.key, dataSlow.key],
      });
      await writerSlow.write({
        [indexSlow.key]: [baseTime, baseTime.add(TimeSpan.milliseconds(500))],
        [dataSlow.key]: [2.0, 2.5],
      });
      await writerSlow.commit();
      await writerSlow.close();

      const stream = await client.exportCSV({
        channels: [dataFast.key, dataSlow.key],
        timeRange: {
          start: baseTime.sub(TimeSpan.seconds(1)),
          end: baseTime.add(TimeSpan.seconds(2)),
        },
      });

      const csv = await streamToString(stream);
      const rows = parseCSV(csv);

      // Should have 6 unique timestamps: 0, 100, 200, 300, 400, 500ms
      expect(rows).toHaveLength(7);

      // First row (t=0): both groups have data
      expect(rows[1][1]).toBe("1");
      expect(rows[1][3]).toBe("2");

      // Second row (t=100ms): only fast group
      expect(rows[2][1]).toBe("1.1");
      expect(rows[2][3]).toBe("");

      // Last row (t=500ms): only slow group
      expect(rows[6][1]).toBe("");
      expect(rows[6][3]).toBe("2.5");
    });

    it("should handle channels with aligned timestamps from different indexes", async () => {
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

      // Both write at timestamps 1, 2, 3
      const writer1 = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels: [index1.key, data1.key],
      });
      await writer1.write({
        [index1.key]: [
          TimeStamp.seconds(1),
          TimeStamp.seconds(2),
          TimeStamp.seconds(3),
        ],
        [data1.key]: [10, 20, 30],
      });
      await writer1.commit();
      await writer1.close();

      const writer2 = await client.openWriter({
        start: TimeStamp.seconds(1),
        channels: [index2.key, data2.key],
      });
      await writer2.write({
        [index2.key]: [
          TimeStamp.seconds(1),
          TimeStamp.seconds(2),
          TimeStamp.seconds(3),
        ],
        [data2.key]: [100, 200, 300],
      });
      await writer2.commit();
      await writer2.close();

      const stream = await client.exportCSV({
        channels: [data1.key, data2.key],
        timeRange: { start: TimeStamp.seconds(0), end: TimeStamp.seconds(10) },
      });

      const csv = await streamToString(stream);
      const rows = parseCSV(csv);

      // 3 data rows (aligned timestamps produce one row each)
      expect(rows).toHaveLength(4);

      // Each row should have data from both groups
      expect(rows[1][1]).toBe("10");
      expect(rows[1][3]).toBe("100");
      expect(rows[2][1]).toBe("20");
      expect(rows[2][3]).toBe("200");
      expect(rows[3][1]).toBe("30");
      expect(rows[3][3]).toBe("300");
    });

    it("should handle large amounts of channels", async () => {
      const numGroups = 5;
      const channelsPerGroup = 3;
      const dataKeys: channel.Keys = [];

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
        const writeData: Record<number, unknown[]> = {
          [index.key]: [TimeStamp.seconds(g + 1), TimeStamp.seconds(g + 2)],
        };
        for (let c = 0; c < channelsPerGroup; c++)
          writeData[groupChannels[c + 1]] = [g * 10 + c, g * 10 + c + 1];
        await writer.write(writeData);
        await writer.commit();
        await writer.close();
      }

      const stream = await client.exportCSV({
        channels: dataKeys,
        timeRange: { start: TimeStamp.seconds(0), end: TimeStamp.seconds(20) },
      });

      const csv = await streamToString(stream);
      const rows = parseCSV(csv);

      expect(rows.length).toBeGreaterThan(1);

      // Each row should have columns for all groups (index + data channels each)
      const expectedColumns = numGroups * (1 + channelsPerGroup);
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

      // No data written

      const stream = await client.exportCSV({
        channels: [data.key],
        timeRange: { start: TimeStamp.seconds(0), end: TimeStamp.seconds(10) },
      });

      const csv = await streamToString(stream);
      const rows = parseCSV(csv);

      // Should just have header
      expect(rows).toHaveLength(1);
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
        start: TimeStamp.seconds(1),
        channels: [index.key, data.key],
      });
      await writer.write({
        [index.key]: [TimeStamp.seconds(1)],
        [data.key]: [42],
      });
      await writer.commit();
      await writer.close();

      const stream = await client.exportCSV({
        channels: [data.key],
        timeRange: { start: TimeStamp.seconds(0), end: TimeStamp.seconds(10) },
      });

      const csv = await streamToString(stream);

      // Check full CSV output
      const lines = csv.trim().split("\n");
      expect(lines).toHaveLength(2);
      expect(lines[0]).toBe(`${indexName},${dataName}`);

      const dataRow = lines[1].split(",");
      // TimeStamp.seconds(1) = 1e9 nanoseconds
      expect(dataRow[0]).toBe("1000000000");
      expect(dataRow[1]).toBe("42");
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

      // Write data to each group
      for (const group of groups) {
        const writer = await client.openWriter({
          start: group.baseTime,
          channels: [group.indexKey, ...group.dataKeys],
        });

        // Write in batches to avoid memory issues
        const batchSize = 500;
        for (let batchStart = 0; batchStart < group.sampleCount; batchStart += batchSize) {
          const batchEnd = Math.min(batchStart + batchSize, group.sampleCount);
          const timestamps: TimeStamp[] = [];
          const dataArrays: number[][] = group.dataKeys.map(() => []);

          for (let i = batchStart; i < batchEnd; i++) {
            timestamps.push(group.baseTime.add(TimeSpan.milliseconds(i * group.intervalMs)));
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
      }

      // Calculate expected total samples across all groups
      const totalSamples = samplesPerGroup.reduce((a, b) => a + b, 0);

      // Export the data
      const stream = await client.exportCSV({
        channels: allDataKeys,
        timeRange: {
          start: TimeStamp.seconds(999),
          end: TimeStamp.seconds(1100),
        },
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
      const lines = csv.trim().split("\n");

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
  });
});
