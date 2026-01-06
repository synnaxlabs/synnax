// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, group, ontology, task } from "@synnaxlabs/client";
import { id, status, TimeStamp } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import z from "zod";

import { Task } from "@/task";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  const abortController = new AbortController();
  let wrapper: React.FC<PropsWithChildren>;

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should return a list of task keys", async () => {
      const rack = await client.racks.create({
        name: "testRack",
      });
      const task1 = await rack.createTask({
        name: "task1",
        type: "testType",
        config: {},
      });
      const task2 = await rack.createTask({
        name: "task2",
        type: "testType",
        config: {},
      });

      const { result } = renderHook(() => Task.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(task1.key);
      expect(result.current.data).toContain(task2.key);
    });

    it("should get individual tasks using getItem", async () => {
      const rack = await client.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "testTask",
        type: "testType",
        config: {},
      });

      const { result } = renderHook(() => Task.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedTask = result.current.getItem(testTask.key);
      expect(retrievedTask?.key).toEqual(testTask.key);
      expect(retrievedTask?.name).toEqual("testTask");
    });

    it("should filter tasks by search term", async () => {
      const rack = await client.racks.create({
        name: "testRack",
      });
      await rack.createTask({
        name: "ordinary",
        type: "testType",
        config: {},
      });
      await rack.createTask({
        name: "special",
        type: "testType",
        config: {},
      });

      const { result } = renderHook(() => Task.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ searchTerm: "special" });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(1);
      expect(
        result.current.data
          .map((key: task.Key) => result.current.getItem(key)?.name)
          .includes("special"),
      ).toBe(true);
    });

    it("should handle pagination with limit and offset", async () => {
      const rack = await client.racks.create({
        name: "testRack",
      });
      for (let i = 0; i < 5; i++)
        await rack.createTask({
          name: `paginationTask${i}`,
          type: "testType",
          config: {},
        });

      const { result } = renderHook(() => Task.useList({ useCachedList: false }), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ limit: 2, offset: 1 });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
    });

    it("should update the list when a task is created", async () => {
      const rack = await client.racks.create({
        name: "testRack",
      });

      const { result } = renderHook(() => Task.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const initialLength = result.current.data.length;

      const newTask = await rack.createTask({
        name: "newTask",
        type: "testType",
        config: {},
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(initialLength + 1);
        expect(result.current.data).toContain(newTask.key);
      });
    });

    it("should update the list when a task is updated", async () => {
      const rack = await client.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "original",
        type: "testType",
        config: {},
      });

      const { result } = renderHook(() => Task.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.getItem(testTask.key)?.name).toEqual("original");

      await client.tasks.create({
        ...testTask.payload,
        name: "updated",
      });

      await waitFor(() => {
        expect(result.current.getItem(testTask.key)?.name).toEqual("updated");
      });
    });

    it("should remove task from list when deleted", async () => {
      const rack = await client.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "toDelete",
        type: "testType",
        config: {},
      });

      const { result } = renderHook(() => Task.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(testTask.key);

      await client.tasks.delete([testTask.key]);

      await waitFor(() => {
        expect(result.current.data).not.toContain(testTask.key);
      });
    });

    it("should update task status in the list", async () => {
      const rack = await client.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "statusTask",
        type: "testType",
        config: {},
      });

      const { result } = renderHook(() => Task.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const taskStatus: task.Status = status.create<
        ReturnType<typeof task.statusDetailsZ>
      >({
        key: id.create(),
        variant: "error",
        message: "Task failed",
        details: {
          task: testTask.key,
          running: false,
          data: {},
        },
      });

      await act(async () => {
        await client.statuses.set(taskStatus);
      });

      await waitFor(() => {
        const taskInList = result.current.getItem(testTask.key);
        expect(taskInList?.status?.variant).toEqual("error");
        expect(taskInList?.status?.message).toEqual("Task failed");
      });
    });

    it("should update task status when a command is executed", async () => {
      const rack = await client.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "commandTask",
        type: "testType",
        config: {},
      });

      const { result } = renderHook(() => Task.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const command: task.Command = {
        key: id.create(),
        task: testTask.key,
        type: "start",
        args: {},
      };

      await act(async () => {
        const writer = await client.openWriter([task.COMMAND_CHANNEL_NAME]);
        await writer.write(task.COMMAND_CHANNEL_NAME, [command]);
        await writer.close();
      });

      await waitFor(() => {
        const taskInList = result.current.getItem(testTask.key);
        expect(taskInList?.status?.variant).toEqual("loading");
        expect(taskInList?.status?.message).toEqual("Running start command...");
        expect(taskInList?.status?.details.running).toBe(true);
      });
    });
  });

  describe("useRetrieve", () => {
    it("should retrieve a task by key", async () => {
      const rack = await client.racks.create({
        name: "retrieveRack",
      });
      const testTask = await rack.createTask({
        name: "retrieve_test",
        type: "testType",
        config: { value: "test" },
      });

      const { result } = renderHook(() => Task.useRetrieve({ key: testTask.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.key).toEqual(testTask.key);
      expect(result.current.data?.name).toEqual("retrieve_test");
      expect(result.current.data?.type).toEqual("testType");
      expect(result.current.data?.config).toEqual({ value: "test" });
    });

    it("should retrieve task with status", async () => {
      const rack = await client.racks.create({
        name: "statusRack",
      });
      const testTask = await rack.createTask({
        name: "status_task",
        type: "testType",
        config: {},
      });

      const taskStatus: task.Status = status.create<
        ReturnType<typeof task.statusDetailsZ>
      >({
        key: task.statusKey(testTask.key),
        variant: "success",
        message: "Task running",
        details: {
          task: testTask.key,
          running: true,
          data: {},
        },
      });

      await act(async () => {
        await client.statuses.set(taskStatus);
      });

      const { result } = renderHook(
        () => Task.useRetrieve({ key: testTask.key, includeStatus: true }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data?.status?.variant).toEqual("success");
        expect(result.current.data?.status?.message).toEqual("Task running");
      });
    });

    it("should update when task is renamed", async () => {
      const rack = await client.racks.create({
        name: "renameRack",
      });
      const testTask = await rack.createTask({
        name: "original_retrieve",
        type: "testType",
        config: {},
      });

      const { result } = renderHook(
        () => {
          const retrieve = Task.useRetrieve({ key: testTask.key });
          const rename = Task.useRename();
          return { retrieve, rename };
        },
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      expect(result.current.retrieve.data?.name).toEqual("original_retrieve");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: testTask.key,
          name: "renamed_retrieve",
        });
      });
      await waitFor(() => {
        expect(result.current.retrieve.data?.name).toEqual("renamed_retrieve");
      });
    });

    it("should update when task status changes", async () => {
      const rack = await client.racks.create({
        name: "statusUpdateRack",
      });
      const testTask = await rack.createTask({
        name: "status_update_task",
        type: "testType",
        config: {},
      });

      const { result } = renderHook(() => Task.useRetrieve({ key: testTask.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const newStatus: task.Status = status.create<
        ReturnType<typeof task.statusDetailsZ>
      >({
        key: task.statusKey(testTask.key),
        variant: "error",
        message: "Task failed",
        details: {
          task: testTask.key,
          running: false,
          data: { error: "Test error" },
        },
      });

      await act(async () => {
        await client.statuses.set(newStatus);
      });

      await waitFor(() => {
        expect(result.current.data?.status?.variant).toEqual("error");
        expect(result.current.data?.status?.message).toEqual("Task failed");
        expect(result.current.data?.status?.details.data).toEqual({
          error: "Test error",
        });
      });
    });

    it("should handle retrieval with schemas", async () => {
      const rack = await client.racks.create({
        name: "schemaRack",
      });
      const testTask = await rack.createTask({
        name: "schema_task",
        type: "typedTask",
        config: { port: 8080, host: "localhost" },
      });

      const schemas = {
        type: z.literal("typedTask"),
        config: z.object({
          port: z.number(),
          host: z.string(),
        }),
        statusData: z.any(),
      };

      const { useRetrieve } = Task.createRetrieve(schemas);

      const { result } = renderHook(() => useRetrieve({ key: testTask.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.type).toEqual("typedTask");
      expect(result.current.data?.config.port).toEqual(8080);
      expect(result.current.data?.config.host).toEqual("localhost");
    });
  });

  describe("useCreateSnapshot", () => {
    it("should create a snapshot of a single task", async () => {
      const rack = await client.racks.create({
        name: "snapshotRack",
      });
      const originalName = id.create();
      const originalTask = await rack.createTask({
        name: originalName,
        type: "testType",
        config: { value: "original" },
      });
      const parentGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "snapshot_parent",
      });

      const { result } = renderHook(() => Task.useCreateSnapshot(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          tasks: { key: originalTask.key, name: originalTask.name },
          parentID: group.ontologyID(parentGroup.key),
        });
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const snapshot = await client.tasks.retrieve({
        name: `${originalName} (Snapshot)`,
      });
      expect(snapshot.name).toEqual(`${originalName} (Snapshot)`);
      expect(snapshot.snapshot).toBe(true);
      expect(snapshot.config).toEqual({ value: "original" });
    });

    it("should create snapshots of multiple tasks", async () => {
      const rack = await client.racks.create({
        name: "multiSnapshotRack",
      });
      const originalName1 = id.create();
      const originalName2 = id.create();
      const task1 = await rack.createTask({
        name: originalName1,
        type: "testType",
        config: { id: 1 },
      });
      const task2 = await rack.createTask({
        name: originalName2,
        type: "testType",
        config: { id: 2 },
      });
      const parentGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "multi_snapshot_parent",
      });

      const { result } = renderHook(() => Task.useCreateSnapshot(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          tasks: [
            { key: task1.key, name: task1.name },
            { key: task2.key, name: task2.name },
          ],
          parentID: group.ontologyID(parentGroup.key),
        });
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      await waitFor(async () => {
        const firstSnapshotName = `${originalName1} (Snapshot)`;
        const secondSnapshotName = `${originalName2} (Snapshot)`;
        const snapshots = await client.tasks.retrieve({
          names: [firstSnapshotName, secondSnapshotName],
        });
        const snapshot1 = snapshots.find((s) => s.name === firstSnapshotName);
        const snapshot2 = snapshots.find((s) => s.name === secondSnapshotName);
        expect(snapshot1).toBeDefined();
        expect(snapshot2).toBeDefined();
        expect(snapshot1?.snapshot).toBe(true);
        expect(snapshot2?.snapshot).toBe(true);
        expect(snapshot1?.config).toEqual({ id: 1 });
        expect(snapshot2?.config).toEqual({ id: 2 });
      });
    });

    it("should add snapshots to parent ontology group", async () => {
      const rack = await client.racks.create({
        name: "ontologyRack",
      });
      const originalName = id.create();
      const originalTask = await rack.createTask({
        name: originalName,
        type: "testType",
        config: {},
      });
      const parentGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "ontology_parent",
      });

      const { result } = renderHook(() => Task.useCreateSnapshot(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          tasks: { key: originalTask.key, name: originalTask.name },
          parentID: group.ontologyID(parentGroup.key),
        });
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const children = await client.ontology.retrieveChildren(
        group.ontologyID(parentGroup.key),
      );
      expect(children.length).toBeGreaterThan(0);

      const snapshotChild = children.find(
        (c) => c.name === `${originalTask.name} (Snapshot)`,
      );
      expect(snapshotChild).toBeDefined();
    });

    it("should preserve task configuration in snapshots", async () => {
      const rack = await client.racks.create({
        name: "configRack",
      });
      const complexConfig = {
        host: "localhost",
        port: 8080,
        settings: {
          timeout: 5000,
          retries: 3,
        },
      };
      const originalName = id.create();
      const originalTask = await rack.createTask({
        name: originalName,
        type: "complexType",
        config: complexConfig,
      });
      const parentGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "config_parent",
      });

      const { result } = renderHook(() => Task.useCreateSnapshot(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          tasks: { key: originalTask.key, name: originalTask.name },
          parentID: group.ontologyID(parentGroup.key),
        });
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const snapshot = await client.tasks.retrieve({
        name: `${originalName} (Snapshot)`,
      });
      expect(snapshot).toBeDefined();
      expect(snapshot.config).toEqual(complexConfig);
      expect(snapshot.type).toEqual("complexType");
    });
  });

  describe("useDelete", () => {
    it("should delete a single task", async () => {
      const rack = await client.racks.create({
        name: "deleteRack",
      });
      const testTask = await rack.createTask({
        name: "delete_single",
        type: "testType",
        config: {},
      });

      const { result } = renderHook(
        () => {
          const list = Task.useList();
          const del = Task.useDelete();
          return { list, del };
        },
        { wrapper },
      );
      act(() => {
        result.current.list.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.list.variant).toEqual("success");
      });
      expect(result.current.list.data).toContain(testTask.key);

      await act(async () => {
        await result.current.del.updateAsync(testTask.key);
      });
      await waitFor(() => {
        expect(result.current.list.data).not.toContain(testTask.key);
      });
    });

    it("should delete multiple tasks", async () => {
      const rack = await client.racks.create({
        name: "deleteMultiRack",
      });
      const task1 = await rack.createTask({
        name: "delete_multi_1",
        type: "testType",
        config: {},
      });
      const task2 = await rack.createTask({
        name: "delete_multi_2",
        type: "testType",
        config: {},
      });

      const { result } = renderHook(
        () => {
          const list = Task.useList();
          const del = Task.useDelete();
          return { list, del };
        },
        { wrapper },
      );
      act(() => {
        result.current.list.retrieve({});
      });
      await waitFor(() => expect(result.current.list.variant).toEqual("success"));
      expect(result.current.list.data).toContain(task1.key);
      expect(result.current.list.data).toContain(task2.key);

      await act(async () => {
        await result.current.del.updateAsync([task1.key, task2.key]);
      });
      await waitFor(() => {
        expect(result.current.list.data).not.toContain(task1.key);
        expect(result.current.list.data).not.toContain(task2.key);
      });
    });
  });

  describe("useForm", async () => {
    const testRack = await client.racks.create({ name: "testRack" });
    it("should initialize a form with default values", async () => {
      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({}),
          statusData: z.any(),
        },
        initialValues: {
          key: "123",
          name: "testTask",
          type: "testType",
          config: {},
        },
      });
      const { result } = renderHook(() => useForm({ query: {} }), {
        wrapper,
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.form.get("key").value).toEqual("123");
      expect(result.current.form.get("name").value).toEqual("testTask");
      expect(result.current.form.get("type").value).toEqual("testType");
      expect(result.current.form.get("config").value).toEqual({});
      expect(result.current.form.get("rackKey").value).toEqual(0);
      expect(result.current.form.get("snapshot").value).toEqual(false);
    });

    it("should retrieve and populate form with existing task", async () => {
      const testTask = await testRack.createTask({
        name: "existingTask",
        type: "testType",
        config: { setting: "value" },
      });

      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({ setting: z.string() }),
          statusData: z.any().or(z.null()),
        },
        initialValues: {
          key: "0",
          name: "",
          type: "testType",
          config: { setting: "" },
        },
      });

      const { result } = renderHook(() => useForm({ query: { key: testTask.key } }), {
        wrapper,
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.form.get("key").value).toEqual(testTask.key);
        expect(result.current.form.get("name").value).toEqual("existingTask");
        expect(result.current.form.get("config").value).toEqual({ setting: "value" });
      });
    });

    it("should save form changes when save is called", async () => {
      const testTask = await testRack.createTask({
        name: "taskToUpdate",
        type: "testType",
        config: {},
      });

      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({}),
          statusData: z.any(),
        },
        initialValues: {
          key: testTask.key,
          name: "taskToUpdate",
          type: "testType",
          config: {},
        },
      });

      const { result } = renderHook(() => useForm({ query: { key: testTask.key } }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      act(() => {
        result.current.form.set("name", "updatedTaskName");
      });

      await act(async () => {
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.form.get("name").value).toEqual("updatedTaskName");
      });

      const updatedTask = await client.tasks.retrieve({
        key: testTask.key,
      });
      expect(updatedTask.name).toEqual("updatedTaskName");
    });

    it("should validate form fields according to schema", async () => {
      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({
            port: z.number().min(1).max(65535),
            host: z.string().min(1),
          }),
          statusData: z.any(),
        },
        initialValues: {
          key: "0",
          name: "test",
          type: "testType",
          config: { port: 0, host: "" },
        },
      });

      const { result } = renderHook(() => useForm({ query: {} }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      act(() => {
        result.current.form.set("config.port", 70000);
      });

      await act(async () => {
        const isValid = await result.current.form.validateAsync();
        expect(isValid).toBe(false);
      });

      const portField = result.current.form.get("config.port");
      expect(portField.status.variant).toBe("error");

      act(() => {
        result.current.form.set("config.port", 8080);
        result.current.form.set("config.host", "localhost");
      });

      await act(async () => {
        const isValid = await result.current.form.validateAsync();
        expect(isValid).toBe(true);
      });
    });

    it("should handle beforeSave callback", async () => {
      const testTask = await testRack.createTask({
        name: "taskWithBeforeSave",
        type: "testType",
        config: {},
      });

      const beforeSave = vi.fn().mockResolvedValue(true);

      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({}),
          statusData: z.any(),
        },
        initialValues: {
          key: testTask.key,
          name: "taskWithBeforeSave",
          type: "testType",
          config: {},
        },
      });

      const { result } = renderHook(
        () =>
          useForm({
            query: { key: testTask.key },
            beforeSave,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      act(() => {
        result.current.form.set("name", "modifiedName");
      });

      act(() => {
        result.current.save({ signal: abortController.signal });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(beforeSave).toHaveBeenCalledWith(
        expect.objectContaining({
          query: { key: testTask.key },
        }),
      );
    });

    it("should handle afterSave callback", async () => {
      const testTask = await testRack.createTask({
        name: "taskWithAfterSave",
        type: "testType",
        config: {},
      });

      const afterSave = vi.fn();

      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({}),
          statusData: z.any(),
        },
        initialValues: {
          key: testTask.key,
          name: "taskWithAfterSave",
          type: "testType",
          config: {},
        },
      });

      const { result } = renderHook(
        () =>
          useForm({
            query: { key: testTask.key },
            afterSave,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      act(() => {
        result.current.form.set("name", "savedName");
      });

      await act(async () => {
        result.current.save({ signal: abortController.signal });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.form.get("name").value).toEqual("savedName");
      });

      expect(afterSave).toHaveBeenCalledWith(
        expect.objectContaining({
          query: { key: testTask.key },
        }),
      );
    });

    it("should update form when task status changes", async () => {
      const testTask = await testRack.createTask({
        name: "statusTask",
        type: "testType",
        config: {},
      });

      const statusDataSchema = z.object({ errorCode: z.number().optional() });

      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({}),
          statusData: statusDataSchema.or(z.null()),
        },
        initialValues: {
          key: testTask.key,
          name: "statusTask",
          type: "testType",
          config: {},
        },
      });

      const { result } = renderHook(() => useForm({ query: { key: testTask.key } }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const taskStatus: task.Status = status.create<
        ReturnType<typeof task.statusDetailsZ>
      >({
        key: task.statusKey(testTask.key),
        variant: "error",
        message: "Task error",
        details: {
          task: testTask.key,
          running: false,
          data: { errorCode: 500 },
        },
      });

      await act(async () => {
        await client.statuses.set(taskStatus);
      });

      await waitFor(() => {
        const status =
          result.current.form.get<task.Status<typeof statusDataSchema>>("status").value;
        expect(status?.variant).toEqual("error");
        expect(status?.message).toEqual("Task error");
        expect(status?.details.data.errorCode).toEqual(500);
      });
    });

    it("should handle field changes and mark form as touched", async () => {
      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({}),
          statusData: z.any(),
        },
        initialValues: {
          key: "123",
          name: "originalName",
          type: "testType",
          config: {},
        },
      });

      const { result } = renderHook(() => useForm({ query: {} }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.get("name").touched).toBe(false);

      act(() => {
        result.current.form.set("name", "modifiedName");
      });

      expect(result.current.form.get("name").touched).toBe(true);
      expect(result.current.form.get("name").value).toEqual("modifiedName");
    });

    it("should not mark form as touched after saving", async () => {
      const testTask = await testRack.createTask({
        name: "touchedAfterSaveTask",
        type: "testType",
        config: { value: "initial" },
      });

      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({ value: z.string() }),
          statusData: z.any(),
        },
        initialValues: {
          key: testTask.key,
          name: "touchedAfterSaveTask",
          type: "testType",
          config: { value: "initial" },
        },
      });

      const { result } = renderHook(() => useForm({ query: { key: testTask.key } }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.get("name").touched).toBe(false);

      act(() => {
        result.current.form.set("name", "updatedName");
        result.current.form.set("config.value", "updated");
      });

      expect(result.current.form.get("name").touched).toBe(true);
      expect(result.current.form.get("config.value").touched).toBe(true);

      await act(async () => {
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.form.get("name").value).toEqual("updatedName");
      });
      expect(result.current.form.get("name").touched).toBe(false);
      expect(result.current.form.get("config.value").touched).toBe(false);
    });

    it("should not mark form as touched when status updates from server", async () => {
      const testTask = await testRack.createTask({
        name: "statusTouchedTask",
        type: "testType",
        config: {},
      });

      const statusDataSchema = z.object({ errorCode: z.number().optional() });

      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({}),
          statusData: statusDataSchema.or(z.null()),
        },
        initialValues: {
          key: testTask.key,
          name: "statusTouchedTask",
          type: "testType",
          config: {},
        },
      });

      const { result } = renderHook(() => useForm({ query: { key: testTask.key } }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.get("name").touched).toBe(false);

      const taskStatus: task.Status = status.create<
        ReturnType<typeof task.statusDetailsZ>
      >({
        key: task.statusKey(testTask.key),
        variant: "error",
        message: "Task error from server",
        details: {
          task: testTask.key,
          running: false,
          data: { errorCode: 500 },
        },
      });

      await act(async () => {
        await client.statuses.set(taskStatus);
      });

      await waitFor(() => {
        const statusField =
          result.current.form.get<task.Status<typeof statusDataSchema>>("status").value;
        expect(statusField?.variant).toEqual("error");
      });
      expect(result.current.form.get("status").touched).toBe(false);
      expect(result.current.form.get("name").touched).toBe(false);
    });

    it("should not mark form as touched when task data updates from server listener", async () => {
      const testTask = await testRack.createTask({
        name: "serverUpdateTask",
        type: "testType",
        config: { setting: "original" },
      });

      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({ setting: z.string() }),
          statusData: z.any(),
        },
        initialValues: {
          key: testTask.key,
          name: "serverUpdateTask",
          type: "testType",
          config: { setting: "original" },
        },
      });

      const { result } = renderHook(() => useForm({ query: { key: testTask.key } }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.get("name").touched).toBe(false);
      expect(result.current.form.get("config.setting").touched).toBe(false);
      await act(async () => {
        await client.tasks.create({
          ...testTask.payload,
          name: "serverUpdatedName",
          config: { setting: "serverUpdated" },
        });
      });

      await waitFor(() => {
        expect(result.current.form.get("name").value).toEqual("serverUpdatedName");
        expect(result.current.form.get("config.setting").value).toEqual(
          "serverUpdated",
        );
      });
      expect(result.current.form.get("name").touched).toBe(false);
      expect(result.current.form.get("config.setting").touched).toBe(false);
    });

    it("should allow new changes after save to mark form as touched again", async () => {
      const testTask = await testRack.createTask({
        name: "reTouchedTask",
        type: "testType",
        config: {},
      });

      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({}),
          statusData: z.any(),
        },
        initialValues: {
          key: testTask.key,
          name: "reTouchedTask",
          type: "testType",
          config: {},
        },
      });

      const { result } = renderHook(() => useForm({ query: { key: testTask.key } }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      act(() => {
        result.current.form.set("name", "firstUpdate");
      });
      expect(result.current.form.get("name").touched).toBe(true);

      await act(async () => {
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.form.get("name").touched).toBe(false);
      });
      act(() => {
        result.current.form.set("name", "secondUpdate");
      });

      expect(result.current.form.get("name").touched).toBe(true);
      expect(result.current.form.get("name").value).toEqual("secondUpdate");
    });

    it("should reset form to initial values", async () => {
      const testTask = await testRack.createTask({
        name: "resetTask",
        type: "testType",
        config: { value: "initial" },
      });

      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({ value: z.string() }),
          statusData: z.any(),
        },
        initialValues: {
          key: testTask.key,
          name: "resetTask",
          type: "testType",
          config: { value: "initial" },
        },
      });

      const { result } = renderHook(() => useForm({ query: { key: testTask.key } }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      act(() => {
        result.current.form.set("name", "changedName");
        result.current.form.set("config.value", "changed");
      });

      expect(result.current.form.get("name").value).toEqual("changedName");
      expect(result.current.form.get("config.value").value).toEqual("changed");

      act(() => {
        result.current.form.reset();
      });

      expect(result.current.form.get("name").value).toEqual("resetTask");
      expect(result.current.form.get("config.value").value).toEqual("initial");
    });

    it("should handle error states in form operations", async () => {
      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({}),
          statusData: z.any(),
        },
        initialValues: {
          key: "999999",
          name: "errorTask",
          type: "testType",
          config: {},
        },
      });

      const { result } = renderHook(
        () =>
          useForm({
            query: { key: "999999" },
          }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("error");
        expect(result.current.status.message).toEqual("Failed to retrieve task");
      });
    });

    it("should handle autoSave functionality", async () => {
      const testTask = await testRack.createTask({
        name: "autoSaveTask",
        type: "testType",
        config: {},
      });

      const useForm = Task.createForm({
        schemas: {
          type: z.literal("testType"),
          config: z.object({}),
          statusData: z.any(),
        },
        initialValues: {
          key: testTask.key,
          name: "autoSaveTask",
          type: "testType",
          config: {},
        },
      });

      const { result } = renderHook(
        () =>
          useForm({
            query: { key: testTask.key },
            autoSave: true,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      act(() => {
        result.current.form.set("name", "autoSavedName");
      });

      await waitFor(
        async () => {
          const updatedTask = await client.tasks.retrieve({
            key: testTask.key,
          });
          expect(updatedTask.name).toEqual("autoSavedName");
        },
        { timeout: 3000 },
      );
    });

    it("should handle complex config schemas with nested objects", async () => {
      const complexConfig = {
        connection: {
          host: "localhost",
          port: 8080,
          secure: true,
        },
        settings: {
          timeout: 5000,
          retryCount: 3,
        },
      };

      const testTask = await testRack.createTask({
        name: "complexTask",
        type: "complexType",
        config: complexConfig,
      });

      const typeSchema = z.literal("complexType");
      const configSchema = z.object({
        connection: z.object({
          host: z.string(),
          port: z.number(),
          secure: z.boolean(),
        }),
        settings: z.object({
          timeout: z.number(),
          retryCount: z.number(),
        }),
      });
      const statusDataSchema = z.any();
      const schemas = {
        type: typeSchema,
        config: configSchema,
        statusData: statusDataSchema,
      };

      const useForm = Task.createForm({
        schemas,
        initialValues: {
          key: testTask.key,
          name: "complexTask",
          type: "complexType",
          config: complexConfig,
        },
      });

      const { result } = renderHook(() => useForm({ query: { key: testTask.key } }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.get("config.connection.host").value).toEqual(
        "localhost",
      );
      expect(result.current.form.get("config.settings.timeout").value).toEqual(5000);

      act(() => {
        result.current.form.set("config.connection.port", 9090);
      });

      act(() => {
        result.current.save({ signal: abortController.signal });
      });

      await waitFor(
        async () => {
          const updatedTask = await client.tasks.retrieve<
            typeof typeSchema,
            typeof configSchema,
            typeof statusDataSchema
          >({
            key: testTask.key,
          });
          expect(updatedTask.config.connection.port).toEqual(9090);
        },
        { timeout: 3000 },
      );
    });
  });

  describe("useCommand", () => {
    it("should execute a command on a task", async () => {
      const type = "start";
      const args = { a: "dog" };
      const testRack = await client.racks.create({ name: "test" });
      const t = await testRack.createTask({
        name: "test",
        config: { a: "dog" },
        type: "ni",
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);

      const { result } = renderHook(() => Task.useCommand(), { wrapper });

      await act(async () => {
        result.current.update([{ task: t.key, type, args }]);
      });

      await waitFor(async () => {
        const fr = await streamer.read();
        const sample = fr.at(-1)[task.COMMAND_CHANNEL_NAME];
        const parsed = task.commandZ.parse(sample);
        const stat: task.Status = {
          key: parsed.key,
          name: "Task Status",
          variant: "success",
          message: "Command executed successfully",
          time: TimeStamp.now(),
          details: { task: t.key, running: true, data: {} },
        };
        await client.statuses.set(stat);
      });
      streamer.close();
      await waitFor(async () => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(1);
      });
    });
  });
});
