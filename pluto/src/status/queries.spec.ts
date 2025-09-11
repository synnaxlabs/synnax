// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, ontology } from "@synnaxlabs/client";
import { TimeStamp } from "@synnaxlabs/x";
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
      const parentGroup = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "Parent Group",
      );

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
          parent: parentGroup.ontologyID,
        });
      });

      const resources = await client.ontology.retrieveChildren(parentGroup.ontologyID);

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
});
