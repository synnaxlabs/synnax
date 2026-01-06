// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status as xStatus, TimeStamp, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import z from "zod";

import { ontology } from "@/ontology";
import { group } from "@/group";
import { status } from "@/status";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Status", () => {
  describe("set", () => {
    it("should create a new status", async () => {
      const s = await client.statuses.set({
        name: "Test Status",
        key: "test-status-1",
        variant: "info",
        message: "This is a test status",
        time: TimeStamp.now(),
      });
      expect(s.key).toBe("test-status-1");
      expect(s.name).toBe("Test Status");
      expect(s.variant).toBe("info");
      expect(s.message).toBe("This is a test status");
    });

    it("should update an existing status", async () => {
      const key = "test-status-update";
      await client.statuses.set({
        name: "Original Status",
        key,
        variant: "info",
        message: "Original message",
        time: TimeStamp.now(),
      });

      const updated = await client.statuses.set({
        name: "Updated Status",
        key,
        variant: "warning",
        message: "Updated message",
        time: TimeStamp.now(),
      });

      expect(updated.key).toBe(key);
      expect(updated.name).toBe("Updated Status");
      expect(updated.variant).toBe("warning");
      expect(updated.message).toBe("Updated message");
    });

    it("should create multiple statuses at once", async () => {
      const statuses = await client.statuses.set([
        {
          name: "Status 1",
          key: "batch-1",
          variant: "success",
          message: "First batch status",
          time: TimeStamp.now(),
        },
        {
          name: "Status 2",
          key: "batch-2",
          variant: "error",
          message: "Second batch status",
          time: TimeStamp.now(),
        },
      ]);

      expect(statuses).toHaveLength(2);
      expect(statuses[0].key).toBe("batch-1");
      expect(statuses[1].key).toBe("batch-2");
    });

    it("should set a status with a parent", async () => {
      const parentGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "Parent Group",
      });
      const parentOntologyID = group.ontologyID(parentGroup.key);
      const s = await client.statuses.set(
        {
          name: "Child Status",
          key: "child-status",
          variant: "info",
          message: "Status with parent",
          time: TimeStamp.now(),
        },
        { parent: parentOntologyID },
      );

      expect(s.key).toBe("child-status");

      const resources = await client.ontology.retrieveChildren(parentOntologyID);

      const statusResource = resources.find((r) => r.id.key === "child-status");
      expect(statusResource).toBeDefined();
    });
  });

  describe("retrieve", () => {
    it("should retrieve a status by key", async () => {
      const created = await client.statuses.set({
        name: "Retrieve Test",
        key: "retrieve-test",
        variant: "loading",
        message: "Test retrieve",
        time: TimeStamp.now(),
      });

      const retrieved = await client.statuses.retrieve({ key: "retrieve-test" });
      expect(retrieved.key).toBe(created.key);
      expect(retrieved.name).toBe(created.name);
      expect(retrieved.variant).toBe(created.variant);
    });

    it("should retrieve multiple statuses by keys", async () => {
      await client.statuses.set([
        {
          name: "Multi 1",
          key: "multi-1",
          variant: "info",
          message: "First",
          time: TimeStamp.now(),
        },
        {
          name: "Multi 2",
          key: "multi-2",
          variant: "warning",
          message: "Second",
          time: TimeStamp.now(),
        },
      ]);

      const statuses = await client.statuses.retrieve({
        keys: ["multi-1", "multi-2"],
      });

      expect(statuses).toHaveLength(2);
      const keys = statuses.map((s) => s.key);
      expect(keys).toContain("multi-1");
      expect(keys).toContain("multi-2");
    });

    it("should search for statuses", async () => {
      const uniqueName = `SearchableStatus_${Date.now()}`;
      await client.statuses.set({
        name: uniqueName,
        key: `searchable-${Date.now()}`,
        variant: "info",
        message: "Searchable status",
        time: TimeStamp.now(),
      });

      const results = await client.statuses.retrieve({
        searchTerm: uniqueName,
      });

      expect(results.length).toBeGreaterThanOrEqual(1);
      expect(results.some((s) => s.name === uniqueName)).toBe(true);
    });

    it("should paginate results", async () => {
      // Create several statuses
      const keys = [];
      for (let i = 0; i < 5; i++) {
        const key = `paginate-${i}-${Date.now()}`;
        keys.push(key);
        await client.statuses.set({
          name: `Paginate ${i}`,
          key,
          variant: "info",
          message: `Message ${i}`,
          time: TimeStamp.now(),
        });
      }

      // Retrieve with limit
      const page1 = await client.statuses.retrieve({
        keys,
        limit: 2,
        offset: 0,
      });

      const page2 = await client.statuses.retrieve({
        keys,
        limit: 2,
        offset: 2,
      });

      expect(page1).toHaveLength(2);
      expect(page2).toHaveLength(2);

      // Ensure no overlap
      const page1Keys = page1.map((s) => s.key);
      const page2Keys = page2.map((s) => s.key);
      expect(page1Keys.some((k) => page2Keys.includes(k))).toBe(false);
    });

    it("should retrieve a status with a details schema", async () => {
      const detailsSchema = z.object({
        name: z.string(),
        key: z.string(),
      });
      const s = await client.statuses.set<typeof detailsSchema>({
        name: "Details Schema",
        key: "details-schema",
        variant: "info",
        message: "Test",
        time: TimeStamp.now(),
        details: {
          name: "Details Schema",
          key: "details-schema",
        },
      });
      const retrieved = await client.statuses.retrieve<typeof detailsSchema>({
        key: s.key,
        detailsSchema,
      });
      expect(retrieved.key).toBe(s.key);
      expect(retrieved.name).toBe(s.name);
      expect(retrieved.details).toBeDefined();
      expect(retrieved.details.name).toBe(s.details.name);
      expect(retrieved.details.key).toBe(s.details.key);
    });
  });

  describe("delete", () => {
    it("should delete a status by key", async () => {
      const s = await client.statuses.set({
        name: "To Delete",
        key: "delete-me",
        variant: "error",
        message: "Will be deleted",
        time: TimeStamp.now(),
      });

      await client.statuses.delete(s.key);

      await expect(
        async () => await client.statuses.retrieve({ key: s.key }),
      ).rejects.toThrow();
    });

    it("should delete multiple statuses", async () => {
      const keys = ["del-1", "del-2", "del-3"];
      await client.statuses.set(
        keys.map((key) =>
          xStatus.create({
            name: `Delete ${key}`,
            key,
            variant: "info",
            message: "To be deleted",
            time: TimeStamp.now(),
          }),
        ),
      );

      await client.statuses.delete(keys);

      // Try to retrieve them - should get empty or error
      const results = await client.statuses.retrieve({ keys }).catch(() => []);
      expect(results).toHaveLength(0);
    });

    it("should be idempotent", async () => {
      const key = "idempotent-delete";

      // Delete a non-existent status - should not throw
      await expect(client.statuses.delete(key)).resolves.not.toThrow();

      // Create and delete
      await client.statuses.set({
        name: "Idempotent",
        key,
        variant: "info",
        message: "Test",
        time: TimeStamp.now(),
      });

      await client.statuses.delete(key);

      // Delete again - should not throw
      await expect(client.statuses.delete(key)).resolves.not.toThrow();
    });
  });

  describe("with labels", () => {
    it("should correctly retrieve a status with labels attached", async () => {
      const label1 = await client.labels.create({
        name: "Label 1",
        color: "#0000FF",
      });
      const label2 = await client.labels.create({
        name: "Label 2",
        color: "#FF0000",
      });
      const stat = await client.statuses.set({
        name: "Idempotent",
        key: uuid.create(),
        variant: "info",
        message: "Test",
        time: TimeStamp.now(),
      });
      await client.labels.label(status.ontologyID(stat.key), [label1.key, label2.key], {
        replace: true,
      });

      const retrievedStat = await client.statuses.retrieve({
        key: stat.key,
        includeLabels: true,
      });
      expect(retrievedStat.key).toEqual(stat.key);
      expect(retrievedStat.labels).toHaveLength(2);
    });
  });

  describe("status variants", () => {
    it("should support all status variants", async () => {
      const variants: xStatus.Variant[] = [
        "success",
        "info",
        "warning",
        "error",
        "loading",
        "disabled",
      ];

      const statuses = await client.statuses.set(
        variants.map((variant) => ({
          name: `Variant ${variant}`,
          key: `variant-${variant}-${Date.now()}`,
          variant,
          message: `Testing ${variant} variant`,
          time: TimeStamp.now(),
        })),
      );

      expect(statuses).toHaveLength(variants.length);
      statuses.forEach((s, i) => {
        expect(s.variant).toBe(variants[i]);
      });
    });
  });
});
