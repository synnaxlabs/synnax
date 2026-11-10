// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, uuid } from "@synnaxlabs/x";
import { describe, expect, it, test, vi } from "vitest";

import { NotFoundError } from "@/index";
import { query } from "@/query";
import { createTestClient, expectDeleted, expectLive } from "@/testutil";

const client = createTestClient();

describe("Log", () => {
  describe("create", () => {
    test("create one", async () => {
      const proj = await client.projects.create({ name: "Log" });
      const log = await client.logs.create({ project: proj.key, name: "Log" });
      expect(log.name).toEqual("Log");
      expect(log.key).not.toEqual(uuid.ZERO);
      expect(log.channels).toEqual([]);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const proj = await client.projects.create({ name: "Log" });
      const log = await client.logs.create({ project: proj.key, name: "Log" });
      await client.logs.rename(log.key, "Log2");
      const res = await client.logs.retrieve(log.key);
      expect(res.name).toEqual("Log2");
    });
  });
  describe("project", () => {
    test("retrieve the logs of a project", async () => {
      const proj = await client.projects.create({ name: "Log" });
      const other = await client.projects.create({ name: "Log" });
      const v = await client.logs.create({ project: proj.key, name: "Log" });
      expect(v.project).toEqual(proj.key);
      const owned = await client.logs.retrieve({ projects: [proj.key] });
      expect(owned.map(({ key }) => key)).toEqual([v.key]);
      const unowned = await client.logs.retrieve({ projects: [other.key] });
      expect(unowned).toEqual([]);
    });

    test("move to another project", async () => {
      const proj = await client.projects.create({ name: "Log" });
      const dest = await client.projects.create({ name: "Log" });
      const v = await client.logs.create({ project: proj.key, name: "Log" });
      await client.batch.send(client.logs.setProjectRequest(v.key, dest.key));
      const res = await client.logs.retrieve([v.key]);
      expect(res[0].project).toEqual(dest.key);
    });

    test("delete with its project", async () => {
      const proj = await client.projects.create({ name: "Log" });
      const v = await client.logs.create({ project: proj.key, name: "Log" });
      await client.projects.delete(proj.key);
      await vi.waitFor(
        async () =>
          await expect(client.logs.retrieve(v.key)).rejects.toThrow(NotFoundError),
      );
    });
  });

  describe("delete", () => {
    test("delete one", async () => {
      const proj = await client.projects.create({ name: "Log" });
      const log = await client.logs.create({ project: proj.key, name: "Log" });
      await client.logs.delete(log.key);
      await expect(client.logs.retrieve(log.key)).rejects.toThrow(NotFoundError);
    });
  });
});

describe("store", () => {
  it("tombstones deletes from live delete signals", async () => {
    await client.connect();
    const project = await client.projects.create({ name: `log-${id.create()}` });
    const log = await client.logs.create({
      project: project.key,
      name: `log-${id.create()}`,
    });
    await client.logs.delete(log.key);
    await expect
      .poll(() => query.Deleted.matches(client.logs.watch(log.key).getCached()))
      .toBe(true);
    const cached = expectDeleted(client.logs.watch(log.key).getCached());
    expect(cached.corpse.name).toEqual(log.name);
  });

  it("drops a deleted channel from the cached document", async () => {
    await client.connect();
    const [kept, deleted] = await client.channels.create([
      { name: `log_${id.create()}`, dataType: DataType.FLOAT32, virtual: true },
      { name: `log_${id.create()}`, dataType: DataType.FLOAT32, virtual: true },
    ]);
    const project = await client.projects.create({ name: `log-${id.create()}` });
    const log = await client.logs.create({
      project: project.key,
      name: `log-${id.create()}`,
      channels: [
        { channel: kept.key, color: "#000000" },
        { channel: deleted.key, color: "#000000" },
      ],
    });
    await client.channels.delete(deleted.key);
    await expect
      .poll(() =>
        expectLive(client.logs.watch(log.key).getCached()).channels.map(
          ({ channel }) => channel,
        ),
      )
      .toEqual([kept.key]);
  });
});
