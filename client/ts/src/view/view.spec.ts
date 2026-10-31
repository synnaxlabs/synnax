// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError, PathError } from "@/index";
import { createTestClient, waitForStreamLive } from "@/testutil";

describe("View", () => {
  describe("create", () => {
    it("should create a single view", async () => {
      const client = createTestClient();
      const label = uuid.create();
      const v = await client.views.create({
        name: "Test View",
        type: "range",
        where: { labels: [label] },
      });
      expect(v.name).toEqual("Test View");
      expect(v).toMatchObject({ type: "range", where: { labels: [label] } });
      expect(v.key).toBeDefined();
    });

    it("should store a where tree", async () => {
      const client = createTestClient();
      const label = uuid.create();
      const where = {
        or: [{ variant: { in: ["error" as const] } }, { not: { labels: [label] } }],
      };
      const v = await client.views.create({ name: "Statuses", type: "status", where });
      const retrieved = await createTestClient().views.retrieve({ key: v.key });
      expect(retrieved).toMatchObject({ type: "status", where });
    });

    it("should reject a where that the Core cannot run", async () => {
      const client = createTestClient();
      await expect(
        client.views.create({
          name: "Bad",
          type: "range",
          where: { name: { regex: "(" } },
        }),
      ).rejects.toSatisfy(
        (e) => PathError.matches(e) && e.path.join(".") === "where.name.regex",
      );
    });

    it("should create multiple views", async () => {
      const client = createTestClient();
      const views = await client.views.create([
        { name: "View 1", type: "range" },
        { name: "View 2", type: "status", where: { variant: { in: ["warning"] } } },
      ]);
      expect(views).toHaveLength(2);
      expect(views[0]).toMatchObject({ name: "View 1", type: "range" });
      expect(views[1]).toMatchObject({
        name: "View 2",
        type: "status",
        where: { variant: { in: ["warning"] } },
      });
    });
  });

  describe("retrieve", () => {
    it("should retrieve a view by key", async () => {
      const client = createTestClient();
      const created = await client.views.create({
        name: "Retrieve Test",
        type: "status",
        where: { variant: { in: ["success"] } },
      });
      const retrieved = await client.views.retrieve(created.key);
      expect(retrieved).toEqual(created);
    });

    it("should retrieve multiple views", async () => {
      const client = createTestClient();
      const created = await client.views.create([
        { name: "Multi 1", type: "range" },
        { name: "Multi 2", type: "status" },
      ]);
      const keys = created.map((v) => v.key);
      const retrieved = await client.views.retrieve(keys);
      expect(retrieved).toHaveLength(2);
      expect(retrieved.map((v) => v.name).sort()).toEqual(["Multi 1", "Multi 2"]);
    });

    it("should retrieve views by where", async () => {
      const client = createTestClient();
      const created = await client.views.create([
        { name: "Range", type: "range" },
        { name: "Status", type: "status" },
      ]);
      const keys = created.map((v) => v.key);
      const retrieved = await client.views.retrieve({
        where: { key: { in: keys }, type: { eq: "status" } },
      });
      expect(retrieved).toHaveLength(1);
      expect(retrieved[0]).toMatchObject({ name: "Status", type: "status" });
    });

    it("should search for views by name", async () => {
      const client = createTestClient();
      const prefix = id.create();
      const name = `${prefix} View`;
      await client.views.create({ name, type: "range" });
      await expect
        .poll(async () => {
          const results = await client.views.retrieve({ searchTerm: prefix });
          return results.find((v) => v.name === name);
        })
        .toMatchObject({ type: "range" });
    });
  });

  describe("rename", () => {
    it("should rename a view and keep its where", async () => {
      const client = createTestClient();
      const v = await client.views.create({
        name: "view-rename",
        type: "status",
        where: { variant: { eq: "error" } },
      });
      await client.views.rename(v.key, "view-renamed");
      const retrieved = await createTestClient().views.retrieve({ key: v.key });
      expect(retrieved.name).toBe("view-renamed");
      expect(retrieved).toMatchObject({
        type: "status",
        where: { variant: { eq: "error" } },
      });
    });
    it("should update the cache of another client", async () => {
      const client = createTestClient();
      const remote = createTestClient();
      const v = await client.views.create({
        name: "view-rename-remote",
        type: "range",
      });
      expect((await client.views.retrieve({ key: v.key })).name).toBe(
        "view-rename-remote",
      );
      const name = `view-renamed-${id.create()}`;
      await waitForStreamLive(client.connection);
      await remote.views.rename(v.key, name);
      await expect
        .poll(async () => (await client.views.retrieve({ key: v.key })).name)
        .toBe(name);
    });
    it("should reject an empty name", async () => {
      const client = createTestClient();
      const v = await client.views.create({
        name: "view-rename-empty",
        type: "range",
      });
      await expect(client.views.rename(v.key, "")).rejects.toThrow();
      expect((await createTestClient().views.retrieve({ key: v.key })).name).toBe(
        "view-rename-empty",
      );
    });
  });

  describe("delete", () => {
    it("should delete a single view", async () => {
      const client = createTestClient();
      const v = await client.views.create({
        name: "To Delete",
        type: "range",
      });
      await client.views.delete(v.key);
      await expect(client.views.retrieve(v.key)).rejects.toThrow(NotFoundError);
    });

    it("should delete multiple views", async () => {
      const client = createTestClient();
      const views = await client.views.create([
        { name: "Delete 1", type: "range" },
        { name: "Delete 2", type: "status" },
      ]);
      const keys = views.map((v) => v.key);
      await client.views.delete(keys);
      expect(await client.views.retrieve(keys)).toEqual([]);
    });
  });
});
