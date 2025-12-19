// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError } from "@/errors";
import { createTestClient } from "@/testutil/client";

describe("View", () => {
  describe("create", () => {
    it("should create a single view", async () => {
      const client = createTestClient();
      const v = await client.views.create({
        name: "Test View",
        type: "lineplot",
        query: { channels: ["test-1", "test-2"] },
      });
      expect(v.name).toEqual("Test View");
      expect(v.type).toEqual("lineplot");
      expect(v.key).toBeDefined();
    });

    it("should create multiple views", async () => {
      const client = createTestClient();
      const views = await client.views.create([
        {
          name: "View 1",
          type: "table",
          query: { channels: ["ch-1"] },
        },
        {
          name: "View 2",
          type: "lineplot",
          query: { channels: ["ch-2"] },
        },
      ]);
      expect(views).toHaveLength(2);
      expect(views[0].name).toEqual("View 1");
      expect(views[0].query).toEqual({ channels: ["ch-1"] });
      expect(views[1].name).toEqual("View 2");
      expect(views[1].query).toEqual({ channels: ["ch-2"] });
    });
  });

  describe("retrieve", () => {
    it("should retrieve a view by key", async () => {
      const client = createTestClient();
      const created = await client.views.create({
        name: "Retrieve Test",
        type: "table",
        query: { channels: ["test"] },
      });
      const retrieved = await client.views.retrieve({ key: created.key });
      expect(retrieved.key).toEqual(created.key);
      expect(retrieved.name).toEqual("Retrieve Test");
      expect(retrieved.query).toEqual({ channels: ["test"] });
    });

    it("should retrieve multiple views", async () => {
      const client = createTestClient();
      const created = await client.views.create([
        {
          name: "Multi 1",
          type: "table",
          query: { channels: ["ch-1"] },
        },
        {
          name: "Multi 2",
          type: "lineplot",
          query: { channels: ["ch-2"] },
        },
      ]);
      const keys = created.map((v) => v.key);
      const retrieved = await client.views.retrieve({ keys });
      expect(retrieved).toHaveLength(2);
      expect(retrieved[0].name).toEqual("Multi 1");
      expect(retrieved[0].query).toEqual({ channels: ["ch-1"] });
      expect(retrieved[1].name).toEqual("Multi 2");
      expect(retrieved[1].query).toEqual({ channels: ["ch-2"] });
    });

    it("should retrieve views by type", async () => {
      const client = createTestClient();
      const type = id.create();
      await client.views.create([
        {
          name: "Type 1",
          type,
          query: { channels: ["ch-1"] },
        },
        {
          name: "Type 2",
          type,
          query: { channels: ["ch-2"] },
        },
      ]);
      const retrieved = await client.views.retrieve({ types: [type] });
      expect(retrieved).toHaveLength(2);
      retrieved.forEach((v) => expect(v.type).toEqual(type));
      retrieved.sort((a, b) => a.name.localeCompare(b.name));
      expect(retrieved[0].name).toEqual("Type 1");
      expect(retrieved[0].query).toEqual({ channels: ["ch-1"] });
      expect(retrieved[1].name).toEqual("Type 2");
      expect(retrieved[1].query).toEqual({ channels: ["ch-2"] });
    });

    it("should search for views by name", async () => {
      const client = createTestClient();
      await client.views.create({
        name: "Searchable View",
        type: "lineplot",
        query: { channels: ["search"] },
      });
      const results = await client.views.retrieve({ searchTerm: "Searchable" });
      expect(results.length).toBeGreaterThan(0);
      expect(results[0].name).toContain("Searchable");
      expect(results[0].query).toEqual({ channels: ["search"] });
    });
  });

  describe("delete", () => {
    it("should delete a single view", async () => {
      const client = createTestClient();
      const v = await client.views.create({
        name: "To Delete",
        type: "table",
        query: { channels: ["delete"] },
      });
      await client.views.delete(v.key);
      await expect(client.views.retrieve({ key: v.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should delete multiple views", async () => {
      const client = createTestClient();
      const views = await client.views.create([
        {
          name: "Delete 1",
          type: "table",
          query: { channels: ["d1"] },
        },
        {
          name: "Delete 2",
          type: "lineplot",
          query: { channels: ["d2"] },
        },
      ]);
      const keys = views.map((v) => v.key);
      await client.views.delete(keys);
      await expect(client.views.retrieve({ keys })).rejects.toThrow(NotFoundError);
    });
  });
});
