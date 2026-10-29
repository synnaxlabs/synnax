// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology, rack, status, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";
import z from "zod";

import { Task } from "@/task";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
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
        type: "pagerduty_alert",
        config: {},
      });
      const task2 = await rack.createTask({
        name: "task2",
        type: "pagerduty_alert",
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
        type: "pagerduty_alert",
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
        type: "pagerduty_alert",
        config: {},
      });
      await rack.createTask({
        name: "special",
        type: "pagerduty_alert",
        config: {},
      });

      const { result } = renderHook(() => Task.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ searchTerm: "special" });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data.length).toBeGreaterThanOrEqual(1);
        expect(
          result.current.data
            .map((key: task.Key) => result.current.getItem(key)?.name)
            .includes("special"),
        ).toBe(true);
      });
    });

    it("should handle pagination with limit and offset", async () => {
      const rack = await client.racks.create({
        name: "testRack",
      });
      for (let i = 0; i < 5; i++)
        await rack.createTask({
          name: `paginationTask${i}`,
          type: "pagerduty_alert",
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

      const newTask = await rack.createTask({
        name: "newTask",
        type: "pagerduty_alert",
        config: {},
      });

      await waitFor(() => expect(result.current.data).toContain(newTask.key));
    });

    it("should update the list when a task is updated", async () => {
      const rack = await client.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "original",
        type: "pagerduty_alert",
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
        type: "pagerduty_alert",
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
        type: "pagerduty_alert",
        config: {},
      });

      const { result } = renderHook(() => Task.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const taskStatus: task.Status = status.create<task.StatusDetailsZodObject>({
        key: id.create(),
        variant: "error",
        message: "Task failed",
        details: {
          task: testTask.key,
          running: false,
          cmd: "",
          configHash: "",
          rack: 0,
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
      const testRack = await client.racks.create({
        name: "testRack",
      });
      const testTask = await testRack.createTask({
        name: "commandTask",
        type: "pagerduty_alert",
        config: {},
      });

      const { result } = renderHook(() => Task.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      // A rack with no Driver goes down within seconds of its creation, and a
      // command on a down rack shows its warning rather than a wait for an answer.
      await act(async () => {
        await client.statuses.set(
          status.create<typeof rack.statusDetailsZ>({
            key: rack.statusKey(testRack.key),
            variant: "success",
            message: "Driver is running",
            details: { rack: testRack.key },
          }),
        );
      });

      const command: task.Command = {
        key: id.create(),
        task: testTask.key,
        type: "start",
        configHash: "",
        args: {},
        pinned: false,
      };

      await act(async () => {
        const writer = await client.openWriter([task.COMMAND_CHANNEL_NAME]);
        await writer.write(task.COMMAND_CHANNEL_NAME, [command]);
        await writer.close();
      });

      await waitFor(() => {
        const taskInList = result.current.getItem(testTask.key);
        expect(taskInList?.status?.variant).toEqual("loading");
        expect(taskInList?.status?.message).toEqual("Sending start command");
        expect(taskInList?.status?.details.running).toBe(true);
      });
    });
  });

  describe("use", () => {
    it("should retrieve a task by key", async () => {
      const rack = await client.racks.create({
        name: "retrieveRack",
      });
      const testTask = await rack.createTask({
        name: "retrieve_test",
        type: "pagerduty_alert",
        config: { routingKey: "rk-test" },
      });

      const { result } = await renderHookSuspended(
        () => Task.use({ key: testTask.key }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current?.key).toEqual(testTask.key);
      expect(result.current?.name).toEqual("retrieve_test");
      expect(result.current?.type).toEqual("pagerduty_alert");
      expect(result.current?.config).toMatchObject({ routingKey: "rk-test" });
    });

    it("should retrieve task with status", async () => {
      const rack = await client.racks.create({
        name: "statusRack",
      });
      const testTask = await rack.createTask({
        name: "status_task",
        type: "pagerduty_alert",
        config: {},
      });

      const taskStatus: task.Status = status.create<task.StatusDetailsZodObject>({
        key: task.statusKey(testTask.key),
        variant: "success",
        message: "Task running",
        details: {
          task: testTask.key,
          running: true,
          cmd: "",
          configHash: "",
          rack: 0,
          data: {},
        },
      });

      await act(async () => {
        await client.statuses.set(taskStatus);
      });

      const { result } = await renderHookSuspended(
        () => Task.use({ key: testTask.key, includeStatus: true }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current).not.toBeNull();
        expect(result.current?.status?.variant).toEqual("success");
        expect(result.current?.status?.message).toEqual("Task running");
      });
    });

    it("should update when task is renamed", async () => {
      const rack = await client.racks.create({
        name: "renameRack",
      });
      const testTask = await rack.createTask({
        name: "original_retrieve",
        type: "pagerduty_alert",
        config: {},
      });

      const { result } = await renderHookSuspended(
        () => {
          const retrieve = Task.use({ key: testTask.key });
          const rename = Task.useRename();
          return { retrieve, rename };
        },
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve).not.toBeNull());
      expect(result.current.retrieve?.name).toEqual("original_retrieve");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: testTask.key,
          name: "renamed_retrieve",
        });
      });
      await waitFor(() => {
        expect(result.current.retrieve?.name).toEqual("renamed_retrieve");
      });
    });

    it("should update when task status changes", async () => {
      const rack = await client.racks.create({
        name: "statusUpdateRack",
      });
      const testTask = await rack.createTask({
        name: "status_update_task",
        type: "pagerduty_alert",
        config: {},
      });

      const { result } = await renderHookSuspended(
        () => Task.use({ key: testTask.key }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current).not.toBeNull());

      const _errorStatusDetailsZ = task.statusDetailsZ(z.object({ error: z.string() }));
      const newStatus = status.create<typeof _errorStatusDetailsZ>({
        key: task.statusKey(testTask.key),
        variant: "error",
        message: "Task failed",
        details: {
          task: testTask.key,
          running: false,
          cmd: "",
          configHash: "",
          rack: 0,
          data: { error: "Test error" },
        },
      });

      await act(async () => {
        await client.statuses.set(newStatus);
      });

      await waitFor(() => {
        expect(result.current?.status?.variant).toEqual("error");
        expect(result.current?.status?.message).toEqual("Task failed");
        expect(result.current?.status?.details.data).toEqual({
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
        type: "pagerduty_alert",
        config: { routingKey: "rk-schema", autoStart: true },
      });

      const schemas = {
        type: z.literal("pagerduty_alert"),
        config: z.object({
          routingKey: z.string(),
          autoStart: z.boolean(),
        }),
        statusData: z.any().optional(),
      };

      const { use } = Task.createRetrieve(schemas);

      const { result } = await renderHookSuspended(() => use({ key: testTask.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current?.type).toEqual("pagerduty_alert");
      expect(result.current?.config.routingKey).toEqual("rk-schema");
      expect(result.current?.config.autoStart).toEqual(true);
    });
  });

  // Other consumers write untyped tasks into the client cache. A typed hook must parse
  // them, or a field the schema defaults and the Core does not store reads undefined.
  describe("untyped tasks in the cache", () => {
    const schemas = {
      type: z.literal("pagerduty_alert"),
      config: z.object({
        routingKey: z.string(),
        autoStart: z.boolean(),
        severity: z.string().default("critical"),
      }),
      statusData: z.any().optional(),
    };
    const createUntyped = async () => {
      const rack = await client.racks.create({ name: "untypedRack" });
      const created = await rack.createTask({
        name: "untyped_task",
        type: "pagerduty_alert",
        config: { routingKey: "rk-untyped", autoStart: true },
      });
      await client.tasks.retrieve(created.key);
      return created;
    };

    it("should fill schema defaults on a task cached untyped", async () => {
      const created = await createUntyped();
      const { use } = Task.createRetrieve(schemas);
      const { result } = await renderHookSuspended(() => use({ key: created.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current?.config.severity).toBe("critical"));
      expect(result.current?.config.routingKey).toBe("rk-untyped");
    });

    it("should keep schema defaults across an untyped streamed update", async () => {
      const created = await createUntyped();
      const { use } = Task.createRetrieve(schemas);
      const { result } = await renderHookSuspended(() => use({ key: created.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current?.config.severity).toBe("critical"));
      await act(async () => {
        await client.tasks.rename(created.key, "renamed_task");
      });
      await waitFor(() => expect(result.current?.name).toBe("renamed_task"));
      expect(result.current?.config.severity).toBe("critical");
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
        type: "pagerduty_alert",
        config: { routingKey: "rk-original" },
      });
      const parentGroup = await client.groups.create({
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
      expect(snapshot.config).toMatchObject({ routingKey: "rk-original" });
    });

    it("should create snapshots of multiple tasks", async () => {
      const rack = await client.racks.create({
        name: "multiSnapshotRack",
      });
      const originalName1 = id.create();
      const originalName2 = id.create();
      const task1 = await rack.createTask({
        name: originalName1,
        type: "pagerduty_alert",
        config: { routingKey: "rk-1" },
      });
      const task2 = await rack.createTask({
        name: originalName2,
        type: "pagerduty_alert",
        config: { routingKey: "rk-2" },
      });
      const parentGroup = await client.groups.create({
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
        expect(snapshot1?.config).toMatchObject({ routingKey: "rk-1" });
        expect(snapshot2?.config).toMatchObject({ routingKey: "rk-2" });
      });
    });

    it("should add snapshots to parent ontology group", async () => {
      const rack = await client.racks.create({
        name: "ontologyRack",
      });
      const originalName = id.create();
      const originalTask = await rack.createTask({
        name: originalName,
        type: "pagerduty_alert",
        config: {},
      });
      const parentGroup = await client.groups.create({
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

      const children = await client.ontology.children.retrieve({
        ids: group.ontologyID(parentGroup.key),
      });
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
        routingKey: "rk-preserved",
        autoStart: true,
        alerts: [{ key: "a1", status: "st1" }],
      };
      const originalName = id.create();
      const originalTask = await rack.createTask({
        name: originalName,
        type: "pagerduty_alert",
        config: complexConfig,
      });
      const parentGroup = await client.groups.create({
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
      expect(snapshot.config).toMatchObject(complexConfig);
      expect(snapshot.type).toEqual("pagerduty_alert");
    });
  });

  describe("useMultiple", () => {
    it("should add a task on the rack created after the query resolves", async () => {
      const rack = await client.racks.create({ name: "testRack" });
      const type = "pagerduty_alert";
      const first = await rack.createTask({ name: "first", type, config: {} });
      const { result } = await renderHookSuspended(
        () => Task.useMultiple({ rack: rack.key, types: [type] }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current?.map((t) => t.key)).toEqual([first.key]),
      );
      const second = await rack.createTask({ name: "second", type, config: {} });
      await waitFor(() =>
        expect(result.current?.map((t) => t.key).sort()).toEqual(
          [first.key, second.key].sort(),
        ),
      );
    });
  });

  describe("useDelete", () => {
    it("should delete a single task", async () => {
      const rack = await client.racks.create({
        name: "deleteRack",
      });
      const testTask = await rack.createTask({
        name: "delete_single",
        type: "pagerduty_alert",
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
        type: "pagerduty_alert",
        config: {},
      });
      const task2 = await rack.createTask({
        name: "delete_multi_2",
        type: "pagerduty_alert",
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

  describe("createForm", async () => {
    const testRack = await client.racks.create({ name: "testRack" });
    const statusData = z.object({ errorCode: z.number().optional() });
    const useForm = Task.createForm({
      schemas: {
        type: z.literal("pagerduty_alert"),
        config: z.object({ routingKey: z.string() }),
        statusData: statusData.nullish(),
      },
    });

    const createTask = async (): Promise<task.Task> =>
      await testRack.createTask({
        name: `form-${id.create()}`,
        type: "pagerduty_alert",
        config: { routingKey: "dog" },
      });

    const renderForm = async (key: task.Key) =>
      await renderHookSuspended(() => useForm({ query: { key } }), { wrapper });

    it("should show the cached task", async () => {
      const t = await createTask();
      const { result } = await renderForm(t.key);
      expect(result.current.form.value()).toMatchObject({
        key: t.key,
        name: t.name,
        rack: testRack.key,
        type: "pagerduty_alert",
        config: { routingKey: "dog" },
        configHash: t.configHash,
        snapshot: false,
      });
    });

    it("should show the config hash of an edit once the Core applies it", async () => {
      const t = await createTask();
      const { result } = await renderForm(t.key);
      act(() => result.current.form.set("config.routingKey", "cat"));
      await act(async () => {
        expect(await result.current.flush()).toBe(true);
        await client.tasks.settled(t.key);
      });
      const stored = await createTestClient().tasks.retrieve(t.key);
      expect(stored.config).toMatchObject({ routingKey: "cat" });
      expect(stored.configHash).not.toBe(t.configHash);
      expect(result.current.form.get("configHash").value).toBe(stored.configHash);
    });

    it("should show a status update without touching a field", async () => {
      const t = await createTask();
      const { result } = await renderForm(t.key);
      const _errorStatusDetailsZ = task.statusDetailsZ(statusData);
      const taskStatus: task.Status = status.create<typeof _errorStatusDetailsZ>({
        key: task.statusKey(t.key),
        variant: "error",
        message: "Task error",
        details: {
          task: t.key,
          running: false,
          cmd: "",
          configHash: "",
          rack: 0,
          data: { errorCode: 500 },
        },
      });
      await act(async () => {
        await client.statuses.set(taskStatus);
      });
      await waitFor(() => {
        const s =
          result.current.form.get<task.Status<typeof statusData>>("status").value;
        expect(s?.variant).toEqual("error");
        expect(s?.message).toEqual("Task error");
        expect(s?.details.data?.errorCode).toEqual(500);
      });
      expect(result.current.form.get("status").touched).toBe(false);
    });

    it("should show a task that another client writes whole", async () => {
      const t = await createTask();
      const { result } = await renderForm(t.key);
      await act(async () => {
        await createTestClient().tasks.create({
          ...t.payload,
          name: "written",
          config: { routingKey: "remote" },
        });
      });
      await waitFor(() => {
        expect(result.current.form.get("name").value).toEqual("written");
        expect(result.current.form.get("config.routingKey").value).toEqual("remote");
      });
      expect(result.current.form.get("name").touched).toBe(false);
    });

    it("should keep a local edit over a whole write from another client", async () => {
      const t = await createTask();
      const { result } = await renderForm(t.key);
      act(() => result.current.form.set("config.routingKey", "local"));
      await act(async () => {
        await createTestClient().tasks.create({ ...t.payload, name: "renamed" });
      });
      await waitFor(() =>
        expect(result.current.form.get("name").value).toEqual("renamed"),
      );
      expect(result.current.form.get("config.routingKey").value).toEqual("local");
      await act(async () => expect(await result.current.flush()).toBe(true));
      const stored = await createTestClient().tasks.retrieve(t.key);
      expect(stored.name).toEqual("renamed");
      expect(stored.config).toMatchObject({ routingKey: "local" });
    });
  });

  describe("useCommand", () => {
    it("should execute a command on a task", async () => {
      const type = "start";
      const args = { a: "dog" };
      const testRack = await client.racks.create({ name: "test" });
      const t = await testRack.createTask({
        name: "test",
        config: { routingKey: "rk" },
        type: "pagerduty_alert",
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
        const stat: task.Status = status.create<task.StatusDetailsZodObject>({
          key: parsed.key,
          name: "Task Status",
          variant: "success",
          message: "Command executed successfully",
          details: {
            task: t.key,
            running: true,
            cmd: "",
            configHash: "",
            rack: 0,
            data: {},
          },
        });
        await client.statuses.set(stat);
      });
      streamer.close();
      await waitFor(async () => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.status.message).toEqual("Sent task commands");
        expect(result.current.data).toHaveLength(1);
      });
    });
  });
});
