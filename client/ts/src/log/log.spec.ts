// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { NotFoundError } from "@/errors";
import { query } from "@/query";
import { createTestClient, expectDeleted } from "@/testutil";

const client = createTestClient();

describe("Log", () => {
  describe("create", () => {
    test("create one", async () => {
      const proj = await client.projects.create({ name: "Log", layout: { one: 1 } });
      const log = await client.logs.create(proj.key, { name: "Log" });
      expect(log.name).toEqual("Log");
      expect(log.key).not.toEqual(uuid.ZERO);
      expect(log.channels).toEqual([]);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const proj = await client.projects.create({ name: "Log", layout: { one: 1 } });
      const log = await client.logs.create(proj.key, { name: "Log" });
      await client.logs.rename(log.key, "Log2");
      const res = await client.logs.retrieve(log.key);
      expect(res.name).toEqual("Log2");
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const proj = await client.projects.create({ name: "Log", layout: { one: 1 } });
      const log = await client.logs.create(proj.key, { name: "Log" });
      await client.logs.delete(log.key);
      await expect(client.logs.retrieve(log.key)).rejects.toThrow(NotFoundError);
    });
  });
});

describe("store", () => {
  it("tombstones deletes from live delete signals", async () => {
    await client.connect();
    const project = await client.projects.create({ name: `log-${id.create()}` });
    const log = await client.logs.create(project.key, { name: `log-${id.create()}` });
    await client.logs.delete(log.key);
    await expect
      .poll(() => query.Deleted.matches(client.logs.getCached(log.key)))
      .toBe(true);
    const cached = expectDeleted(client.logs.getCached(log.key));
    expect(cached.corpse.name).toEqual(log.name);
  });
});
