// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { NotFoundError } from "@/errors";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Log", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({ name: "Log", layout: { one: 1 } });
      const log = await client.logs.create(ws.key, { name: "Log" });
      expect(log.name).toEqual("Log");
      expect(log.key).not.toEqual(uuid.ZERO);
      expect(log.channels).toEqual([]);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({ name: "Log", layout: { one: 1 } });
      const log = await client.logs.create(ws.key, { name: "Log" });
      await client.logs.rename(log.key, "Log2");
      const res = await client.logs.retrieve({ key: log.key });
      expect(res.name).toEqual("Log2");
    });
  });
  describe("setData", () => {
    test("replaces typed body while preserving key and name", async () => {
      const ws = await client.workspaces.create({ name: "Log", layout: { one: 1 } });
      const log = await client.logs.create(ws.key, { name: "Log" });
      await client.logs.setData(log.key, {
        channels: [
          {
            channel: 1,
            color: "#ff0000",
            notation: "scientific",
            precision: 2,
            alias: "temp",
            timestamp: { format: "ISO", tz: "UTC" },
          },
        ],
        remoteCreated: true,
        timestampPrecision: 3,
        showChannelNames: false,
        showReceiptTimestamp: true,
      });
      const res = await client.logs.retrieve({ key: log.key });
      expect(res.name).toEqual("Log");
      expect(res.channels).toHaveLength(1);
      expect(res.channels[0].alias).toEqual("temp");
      expect(res.channels[0].timestamp.format).toEqual("ISO");
      expect(res.timestampPrecision).toEqual(3);
      expect(res.showChannelNames).toBe(false);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({ name: "Log", layout: { one: 1 } });
      const log = await client.logs.create(ws.key, { name: "Log" });
      await client.logs.delete(log.key);
      await expect(client.logs.retrieve({ key: log.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });
});
