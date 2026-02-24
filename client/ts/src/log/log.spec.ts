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
      const log = await client.logs.create(ws.key, {
        name: "Log",
        data: { one: 1 },
      });
      expect(log.name).toEqual("Log");
      expect(log.key).not.toEqual(uuid.ZERO);
      expect(log.data.one).toEqual(1);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({ name: "Log", layout: { one: 1 } });
      const log = await client.logs.create(ws.key, {
        name: "Log",
        data: { one: 1 },
      });
      await client.logs.rename(log.key, "Log2");
      const res = await client.logs.retrieve({ key: log.key });
      expect(res.name).toEqual("Log2");
    });
  });
  describe("setData", () => {
    test("set data", async () => {
      const ws = await client.workspaces.create({ name: "Log", layout: { one: 1 } });
      const log = await client.logs.create(ws.key, {
        name: "Log",
        data: { one: 1 },
      });
      await client.logs.setData(log.key, { two: 2 });
      const res = await client.logs.retrieve({ key: log.key });
      expect(res.data.two).toEqual(2);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({ name: "Log", layout: { one: 1 } });
      const log = await client.logs.create(ws.key, {
        name: "Log",
        data: { one: 1 },
      });
      await client.logs.delete(log.key);
      await expect(client.logs.retrieve({ key: log.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });
});
