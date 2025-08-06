// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, TimeRange } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { newTestClient } from "@/testutil/client";

const client = newTestClient();

describe.skip("export", async () => {
  const indexChannel = await client.channels.create({
    name: "export-csv-ts-test-index",
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });
  const dataChannel = await client.channels.create({
    name: "export-csv-ts-test-data",
    dataType: DataType.FLOAT64,
    index: indexChannel.key,
  });
  const testTimeRange = new TimeRange(1, 6);
  await client.write(testTimeRange.start, {
    [indexChannel.key]: [1, 2, 3, 4, 5],
    [dataChannel.key]: [10, 20, 30, 40, 50],
  });
  describe("csv", () => {
    test("should export csv", async () => {
      const res = await client.export.csv({
        keys: [indexChannel.key, dataChannel.key],
        timeRange: testTimeRange,
      });
      const body = await res.text();
      expect(body).toContain(`${indexChannel.name},${dataChannel.name}`);
      expect(body).toContain("1,10");
      expect(body).toContain("2,20");
      expect(body).toContain("3,30");
      expect(body).toContain("4,40");
      expect(body).toContain("5,50");
    });
    test("should allow name remapping", async () => {
      const res = await client.export.csv({
        keys: [indexChannel.key, dataChannel.key],
        timeRange: testTimeRange,
        channelNames: {
          [indexChannel.key]: "index",
          [`${dataChannel.key}`]: "data",
        },
      });
      const body = await res.text();
      expect(body).toContain(`index,data`);
      expect(body).toContain("1,10");
      expect(body).toContain("2,20");
      expect(body).toContain("3,30");
      expect(body).toContain("4,40");
      expect(body).toContain("5,50");
    });
    test("should default to all data", async () => {
      const res = await client.export.csv({
        keys: [indexChannel.key, dataChannel.key],
      });
      const body = await res.text();
      expect(body).toContain(`${indexChannel.name},${dataChannel.name}`);
      expect(body).toContain("1,10");
      expect(body).toContain("2,20");
      expect(body).toContain("3,30");
      expect(body).toContain("4,40");
      expect(body).toContain("5,50");
    });
    test("should allow a partial time range", async () => {
      const res = await client.export.csv({
        keys: [indexChannel.key, dataChannel.key],
        timeRange: new TimeRange(2, 4),
      });
      const body = await res.text();
      expect(body).toContain(`${indexChannel.name},${dataChannel.name}`);
      expect(body).toContain("2,20");
      expect(body).toContain("3,30");
      expect(body).not.toContain("4,40");
      expect(body).not.toContain("1,10");
      expect(body).not.toContain("5,50");
    });
    test("should grab the index channel even if not specified", async () => {
      const res = await client.export.csv({
        keys: [dataChannel.key],
        timeRange: testTimeRange,
      });
      const body = await res.text();
      expect(body).toContain(`${dataChannel.name},${indexChannel.name}`);
      expect(body).toContain("10,1");
      expect(body).toContain("20,2");
      expect(body).toContain("30,3");
      expect(body).toContain("40,4");
      expect(body).toContain("50,5");
    });
    test("should throw an error if no keys are provided", async () => {
      const res = await client.export.csv({ keys: [] });
      await expect(res.text()).rejects.toThrow();
    });
  });
});
