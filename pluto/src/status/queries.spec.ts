// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, group, ontology, status } from "@synnaxlabs/client";
import { TimeStamp, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Ontology } from "@/ontology";
import { Status } from "@/status";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("Status queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({
      client,
      excludeFluxStores: [Ontology.RESOURCES_FLUX_STORE_KEY],
    });
  });

  describe("useList", () => {
    it("should return a list of status keys", async () => {
      const status1 = await client.statuses.set({
        name: "Status 1",
        key: "list-test-1",
        variant: "info",
        message: "First status",
        time: TimeStamp.now(),
      });
      const status2 = await client.statuses.set({
        name: "Status 2",
        key: "list-test-2",
        variant: "success",
        message: "Second status",
        time: TimeStamp.now(),
      });

      const { result } = renderHook(() => Status.useList(), { wrapper });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(status1.key);
      expect(result.current.data).toContain(status2.key);
    });

    it("should get individual statuses using getItem", async () => {
      const testStatus = await client.statuses.set({
        name: "Test Status",
        key: "item-test",
        variant: "warning",
        message: "Test message",
        description: "Test description",
        time: TimeStamp.now(),
      });

      const { result } = renderHook(() => Status.useList(), { wrapper });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedStatus = result.current.getItem(testStatus.key);
      expect(retrievedStatus?.key).toEqual(testStatus.key);
      expect(retrievedStatus?.name).toEqual("Test Status");
      expect(retrievedStatus?.variant).toEqual("warning");
      expect(retrievedStatus?.message).toEqual("Test message");
      expect(retrievedStatus?.description).toEqual("Test description");
    });

    it("should filter statuses by specific keys", async () => {
      const keys = ["filter-1", "filter-2", "filter-3"];
      await Promise.all(
        keys.map((key, i) =>
          client.statuses.set({
            name: `Filter ${i}`,
            key,
            variant: "info",
            message: `Message ${i}`,
            time: TimeStamp.now(),
          }),
        ),
      );

      const { result } = renderHook(() => Status.useList(), { wrapper });

      act(() => {
        result.current.retrieve({ keys: keys.slice(0, 2) });
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
      expect(result.current.data).toContain(keys[0]);
      expect(result.current.data).toContain(keys[1]);
      expect(result.current.data).not.toContain(keys[2]);
    });

    it("should handle pagination with limit and offset", async () => {
      await Promise.all(
        Array.from({ length: 5 }).map((_, i) =>
          client.statuses.set({
            name: `Page Status ${i}`,
            key: `page-${i}-${Date.now()}`,
            variant: "info",
            message: `Message ${i}`,
            time: TimeStamp.now(),
          }),
        ),
      );

      const { result } = renderHook(() => Status.useList(), { wrapper });

      act(() => {
        result.current.retrieve({ limit: 2 });
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeLessThanOrEqual(2);
    });
  });

  describe("useDelete", () => {
    it("should delete a status", async () => {
      const statusToDelete = await client.statuses.set({
        name: "To Delete",
        key: "delete-hook-test",
        variant: "error",
        message: "Will be deleted",
        time: TimeStamp.now(),
      });

      const { result } = renderHook(() => Status.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(statusToDelete.key);
      });

      await expect(
        client.statuses.retrieve({ key: statusToDelete.key }),
      ).rejects.toThrow();
    });
  });

  describe("useSet", () => {
    it("should set a single status", async () => {
      const { result } = renderHook(() => Status.useSet(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          statuses: {
            name: "Set Hook Test",
            key: "set-hook-test",
            variant: "loading",
            message: "Testing set hook",
            time: TimeStamp.now(),
          },
        });
      });

      const created = await client.statuses.retrieve({ key: "set-hook-test" });
      expect(created.name).toEqual("Set Hook Test");
      expect(created.variant).toEqual("loading");
    });

    it("should set multiple statuses", async () => {
      const { result } = renderHook(() => Status.useSet(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          statuses: [
            {
              name: "Batch 1",
              key: "batch-hook-1",
              variant: "info",
              message: "First",
              time: TimeStamp.now(),
            },
            {
              name: "Batch 2",
              key: "batch-hook-2",
              variant: "success",
              message: "Second",
              time: TimeStamp.now(),
            },
          ],
        });
      });

      const statuses = await client.statuses.retrieve({
        keys: ["batch-hook-1", "batch-hook-2"],
      });
      expect(statuses).toHaveLength(2);
    });

    it("should set status with parent", async () => {
      const parentGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "Parent Group",
      });
      const parentOntologyID = group.ontologyID(parentGroup.key);

      const { result } = renderHook(() => Status.useSet(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          statuses: {
            name: "Child Status",
            key: "child-hook-test",
            variant: "info",
            message: "Has parent",
            time: TimeStamp.now(),
          },
          parent: parentOntologyID,
        });
      });

      const resources = await client.ontology.retrieveChildren(parentOntologyID);

      const statusResource = resources.find((r) => r.id.key === "child-hook-test");
      expect(statusResource).toBeDefined();
    });
  });

  describe("useStatus", () => {
    it("should retrieve a single status", async () => {
      const testStatus = await client.statuses.set({
        name: "Single Retrieve",
        key: "single-retrieve-test",
        variant: "disabled",
        message: "Test single retrieve",
        time: TimeStamp.now(),
      });

      const { result } = renderHook(() => Status.useRetrieve({ key: testStatus.key }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.key).toEqual(testStatus.key);
      expect(result.current.data?.name).toEqual("Single Retrieve");
      expect(result.current.data?.variant).toEqual("disabled");
    });

    it("should update when status changes", async () => {
      const key = "reactive-test";
      await client.statuses.set({
        name: "Original",
        key,
        variant: "info",
        message: "Original",
        time: TimeStamp.now(),
      });

      const { result } = renderHook(() => Status.useRetrieve({ key }), { wrapper });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.name).toEqual("Original");

      // Update the status
      await act(async () => {
        await client.statuses.set({
          name: "Updated",
          key,
          variant: "warning",
          message: "Updated",
          time: TimeStamp.now(),
        });
      });

      await waitFor(() => expect(result.current.data?.name).toEqual("Updated"));
      expect(result.current.data?.variant).toEqual("warning");
    });
  });

  describe("real-time updates", () => {
    it("should update list when new status is created", async () => {
      const { result } = renderHook(() => Status.useList(), { wrapper });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const initialCount = result.current.data.length;

      const newStatus = await client.statuses.set({
        name: "Real-time Test",
        key: `realtime-${Date.now()}`,
        variant: "success",
        message: "Created after hook",
        time: TimeStamp.now(),
      });

      await waitFor(() => expect(result.current.data).toContain(newStatus.key));
      expect(result.current.data.length).toBeGreaterThan(initialCount);
    });

    it("should remove from list when status is deleted", async () => {
      const toDelete = await client.statuses.set({
        name: "Will be deleted",
        key: `delete-realtime-${Date.now()}`,
        variant: "error",
        message: "Delete me",
        time: TimeStamp.now(),
      });

      const { result } = renderHook(() => Status.useList(), { wrapper });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => expect(result.current.data).toContain(toDelete.key));

      await act(async () => {
        await client.statuses.delete(toDelete.key);
      });

      await waitFor(() => expect(result.current.data).not.toContain(toDelete.key));
    });
  });

  describe("useForm", () => {
    it("should initialize with default values for new status", async () => {
      const { result } = renderHook(() => Status.useForm({ query: {} }), { wrapper });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const v = result.current.form.value();
      expect(v.key).toEqual("");
      expect(v.variant).toEqual("success");
      expect(v.message).toEqual("");
      expect(v.name).toEqual("");
      expect(v.description).toEqual("");
      expect(v.labels).toEqual([]);
    });

    it("should create a new status on save", async () => {
      const { result } = renderHook(() => Status.useForm({ query: {} }), { wrapper });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      act(() => {
        result.current.form.set("name", "Form Test Status");
        result.current.form.set("variant", "info");
        result.current.form.set("message", "Created via form");
        result.current.form.set("description", "Test description");
      });
      act(() => {
        result.current.save();
      });
      await waitFor(() => {
        const v = result.current.form.value();
        expect(v.key).not.toEqual("");
        expect(v.name).toEqual("Form Test Status");
        expect(v.variant).toEqual("info");
        expect(v.message).toEqual("Created via form");
        expect(v.description).toEqual("Test description");
      });
    });

    it("should retrieve and populate existing status", async () => {
      const key = uuid.create();
      await client.statuses.set({
        name: "Existing Status",
        key,
        variant: "warning",
        message: "Existing message",
        description: "Existing description",
        time: TimeStamp.now(),
      });
      const { result } = renderHook(() => Status.useForm({ query: { key } }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const v = result.current.form.value();
      expect(v.key).toEqual(key);
      expect(v.name).toEqual("Existing Status");
      expect(v.variant).toEqual("warning");
      expect(v.message).toEqual("Existing message");
      expect(v.description).toEqual("Existing description");
    });

    it("should update existing status on save", async () => {
      const key = uuid.create();
      await client.statuses.set({
        name: "Original Name",
        key,
        variant: "error",
        message: "Original",
        time: TimeStamp.now(),
      });
      const { result } = renderHook(
        () => ({
          form: Status.useForm({ query: { key } }),
          retrieve: Status.useRetrieve({ key }),
        }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current.form.variant).toEqual("success"));
      act(() => {
        result.current.form.form.set("name", "Updated Name");
        result.current.form.form.set("variant", "success");
        result.current.form.form.set("message", "Updated message");
      });
      act(() => {
        result.current.form.save();
      });
      await waitFor(() => {
        expect(result.current.form.variant).toEqual("success");
        expect(result.current.form.form.value().name).toEqual("Updated Name");
        expect(result.current.form.form.value().variant).toEqual("success");
        expect(result.current.form.form.value().message).toEqual("Updated message");
        expect(result.current.retrieve.data?.name).toEqual("Updated Name");
        expect(result.current.retrieve.data?.variant).toEqual("success");
        expect(result.current.retrieve.data?.message).toEqual("Updated message");
      });
    });

    it("should handle status with labels", async () => {
      const label1 = await client.labels.create({ name: "Label 1", color: "#FF0000" });
      const label2 = await client.labels.create({ name: "Label 2", color: "#00FF00" });
      const { result } = renderHook(() => Status.useForm({ query: {} }), { wrapper });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      act(() => {
        result.current.form.set("name", "Status with Labels");
        result.current.form.set("variant", "info");
        result.current.form.set("message", "Has labels");
        result.current.form.set("labels", [label1.key, label2.key]);
      });
      act(() => {
        result.current.save();
      });
      await waitFor(async () => {
        expect(result.current.form.value().key).not.toEqual("");
        expect(result.current.variant).toEqual("success");
      });
      const q = renderHook(
        () => Status.useRetrieve({ key: result.current.form.value().key }),
        { wrapper },
      );
      await waitFor(() => expect(q.result.current.variant).toEqual("success"));
      expect(q.result.current.data?.labels).toEqual([label1, label2]);
    });

    it("should update form when status changes externally", async () => {
      const key = uuid.create();
      await client.statuses.set({
        name: "Initial",
        key,
        variant: "info",
        message: "Initial",
        time: TimeStamp.now(),
      });
      const { result } = renderHook(() => Status.useForm({ query: { key } }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value().name).toEqual("Initial");
      await act(async () => {
        await client.statuses.set({
          name: "External Update",
          key,
          variant: "warning",
          message: "Updated externally",
          time: TimeStamp.now(),
        });
      });
      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("External Update");
        expect(result.current.form.value().variant).toEqual("warning");
        expect(result.current.form.value().message).toEqual("Updated externally");
      });
    });

    it("should generate UUID for new status if key not provided", async () => {
      const { result } = renderHook(() => Status.useForm({ query: {} }), { wrapper });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      act(() => {
        result.current.form.set("name", "Auto UUID Status");
        result.current.form.set("variant", "success");
      });
      act(() => {
        result.current.save();
      });
      await waitFor(() => {
        const key = result.current.form.value().key;
        expect(key).not.toEqual("");
        expect(key).toHaveLength(36);
      });
    });

    it("should update timestamp on save", async () => {
      const { result } = renderHook(() => Status.useForm({ query: {} }), { wrapper });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const initialTime = result.current.form.value().time;
      act(() => {
        result.current.form.set("name", "Time Test");
        result.current.form.set("variant", "info");
      });
      await new Promise((r) => setTimeout(r, 10));
      act(() => {
        result.current.save();
      });
      await waitFor(() => {
        const newTime = result.current.form.value().time;
        expect(newTime.afterEq(initialTime)).toBe(true);
      });
    });

    it("should handle all status variants", async () => {
      const variants = [
        "success",
        "error",
        "warning",
        "info",
        "loading",
        "disabled",
      ] as const;
      for (const variant of variants) {
        const { result } = renderHook(() => Status.useForm({ query: {} }), {
          wrapper,
        });
        await waitFor(() => expect(result.current.variant).toEqual("success"));
        act(() => {
          result.current.form.set("name", `${variant} Status`);
          result.current.form.set("variant", variant);
          result.current.form.set("message", `Test ${variant}`);
        });
        act(() => {
          result.current.save();
        });
        await waitFor(() => {
          const v = result.current.form.value();
          expect(v.variant).toEqual(variant);
          expect(v.key).not.toEqual("");
        });
      }
    });

    it("should handle adding and removing labels", async () => {
      const label1 = await client.labels.create({
        name: "Add Label 1",
        color: "#FF0000",
      });
      const label2 = await client.labels.create({
        name: "Add Label 2",
        color: "#00FF00",
      });
      const label3 = await client.labels.create({
        name: "Add Label 3",
        color: "#0000FF",
      });
      const key = uuid.create();
      await client.statuses.set({
        name: "Label Test Status",
        key,
        variant: "info",
        message: "Testing labels",
        time: TimeStamp.now(),
      });
      await client.labels.label(status.ontologyID(key), [label1.key]);
      const { result } = renderHook(() => Status.useForm({ query: { key } }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      await waitFor(() =>
        expect(result.current.form.value().labels).toContain(label1.key),
      );
      act(() => {
        result.current.form.set("labels", [label2.key, label3.key]);
      });
      act(() => {
        result.current.save();
      });
      await waitFor(async () => {
        const labels = await client.labels.retrieve({
          for: status.ontologyID(key),
        });
        const labelKeys = labels.map((l) => l.key);
        expect(labelKeys).not.toContain(label1.key);
        expect(labelKeys).toContain(label2.key);
        expect(labelKeys).toContain(label3.key);
      });
    });

    it("should sync label changes from external updates", async () => {
      const label1 = await client.labels.create({
        name: "Sync Label 1",
        color: "#FF0000",
      });
      const label2 = await client.labels.create({
        name: "Sync Label 2",
        color: "#00FF00",
      });
      const key = uuid.create();
      await client.statuses.set({
        name: "Label Sync Test",
        key,
        variant: "info",
        message: "Testing label sync",
        time: TimeStamp.now(),
      });
      const { result } = renderHook(() => Status.useForm({ query: { key } }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value().labels).toEqual([]);
      await act(async () => {
        await client.labels.label(status.ontologyID(key), [label1.key]);
      });
      await waitFor(() =>
        expect(result.current.form.value().labels).toContain(label1.key),
      );
      await act(async () => {
        await client.labels.label(status.ontologyID(key), [label2.key]);
      });
      await waitFor(() => {
        const labels = result.current.form.value().labels;
        expect(labels).toContain(label1.key);
        expect(labels).toContain(label2.key);
      });
      await act(async () => {
        await client.labels.remove(status.ontologyID(key), [label1.key]);
      });
      await waitFor(() => {
        const labels = result.current.form.value().labels;
        expect(labels).not.toContain(label1.key);
        expect(labels).toContain(label2.key);
      });
    });

    it("should preserve key when updating existing status", async () => {
      const key = uuid.create();
      await client.statuses.set({
        name: "Original",
        key,
        variant: "info",
        message: "Original",
        time: TimeStamp.now(),
      });
      const { result } = renderHook(
        () => ({
          form: Status.useForm({ query: { key } }),
          retrieve: Status.useRetrieve({ key }),
        }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current.form.variant).toEqual("success"));
      act(() => {
        result.current.form.form.set("name", "Updated");
        result.current.form.form.set("message", "Updated message");
      });
      act(() => {
        result.current.form.save();
      });
      await waitFor(() => {
        expect(result.current.form.variant).toEqual("success");
        expect(result.current.form.form.value().key).toEqual(key);
        expect(result.current.retrieve.data?.key).toEqual(key);
        expect(result.current.retrieve.data?.name).toEqual("Updated");
      });
    });
  });
});
