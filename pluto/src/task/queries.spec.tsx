// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, task } from "@synnaxlabs/client";
import { createTestClient, holdRequests } from "@synnaxlabs/client/testutil";
import { List } from "@synnaxlabs/lyra/list";
import { deep, id, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { act, render, renderHook, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, useEffect } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
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
    const setStatus = async (key: task.Key) =>
      await client.statuses.set(
        status.create<task.StatusDetailsZodObject>({
          owner: task.resourceID(key),
          variant: "error",
          message: "Task failed",
          details: { running: false, cmd: "", configHash: "", rack: 0, data: {} },
        }),
      );

    it("should render again only the item whose task status changed", async () => {
      const rack = await client.racks.create({ name: "renderRack" });
      const [a, b] = await Promise.all([
        rack.createTask({ name: "a", type: "pagerduty_alert", config: {} }),
        rack.createTask({ name: "b", type: "pagerduty_alert", config: {} }),
      ]);
      const renders = new Map<task.Key, number>();
      const Item = ({ itemKey }: { itemKey: task.Key }): ReactElement => {
        const t = List.useItem<task.Key, task.Task>(itemKey);
        renders.set(itemKey, (renders.get(itemKey) ?? 0) + 1);
        return <span>{`${t?.name} ${t?.status?.variant ?? "none"}`}</span>;
      };
      const renderItem = ({ key }: { key: task.Key }) => (
        <Item key={key} itemKey={key} />
      );
      const Tasks = (): ReactElement => {
        const { data, getItem, subscribe, more } = Task.useList({
          query: { where: { rack: { eq: rack.key } } },
        });
        useEffect(more, [more]);
        return (
          <List.Frame data={data} getItem={getItem} subscribe={subscribe}>
            <List.Scroll>
              <List.Items<task.Key>>{renderItem}</List.Items>
            </List.Scroll>
          </List.Frame>
        );
      };
      render(<Tasks />, { wrapper });
      await screen.findByText(/^b /);
      const rendersOfB = renders.get(b.key);
      await setStatus(a.key);
      await screen.findByText("a error");
      expect(renders.get(b.key)).toBe(rendersOfB);
    });

    it("should read the shown records while a search has no answer", async () => {
      const rack = await client.racks.create({ name: "searchRack" });
      const t = await rack.createTask({
        name: "shown",
        type: "pagerduty_alert",
        config: {},
      });
      await setStatus(t.key);
      const { result } = renderHook(
        () =>
          Task.useList({
            query: { where: { rack: { eq: rack.key } } },
            searchDebounce: 0,
          }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([t.key]));
      const shown = result.current.getItem(t.key);
      expect(shown?.status?.variant).toEqual("error");
      holdRequests(client, "/task/retrieve");
      act(() => result.current.search("shown"));
      await waitFor(() => expect(result.current.variant).toEqual("loading"));
      expect(result.current.data).toEqual([t.key]);
      expect(result.current.getItem(t.key)).toBe(shown);
    });

    it("should leave out internal tasks unless the query sets where.internal", async () => {
      const rack = await client.racks.create({ name: "testRack" });
      const [visible, internal] = await Promise.all([
        rack.createTask({ name: "visible", type: "pagerduty_alert", config: {} }),
        rack.createTask({
          name: "internal",
          type: "pagerduty_alert",
          config: {},
          internal: true,
        }),
      ]);
      const { result } = renderHook(
        () => ({
          visible: Task.useList({ query: { where: { rack: { eq: rack.key } } } }),
          internal: Task.useList({
            query: { where: { rack: { eq: rack.key }, internal: { eq: true } } },
          }),
        }),
        { wrapper },
      );
      act(() => {
        result.current.visible.more();
        result.current.internal.more();
      });
      await waitFor(() => {
        expect(result.current.visible.data).toEqual([visible.key]);
        expect(result.current.internal.data).toEqual([internal.key]);
      });
    });

    it("should update the status of a task in the list", async () => {
      const rack = await client.racks.create({ name: "testRack" });
      const t = await rack.createTask({
        name: "statusTask",
        type: "pagerduty_alert",
        config: {},
      });
      const { result } = renderHook(
        () => Task.useList({ query: { where: { rack: { eq: rack.key } } } }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([t.key]));
      await setStatus(t.key);
      await waitFor(() =>
        expect(result.current.getItem(t.key)?.status?.variant).toEqual("error"),
      );
    });

    it("should read the status of a task outside the list", async () => {
      const rack = await client.racks.create({ name: "testRack" });
      const t = await rack.createTask({
        name: "statusTask",
        type: "pagerduty_alert",
        config: {},
      });
      await setStatus(t.key);
      const missing = id.create();
      const { result } = renderHook(
        () => Task.useList({ query: { where: { name: { eq: missing } } } }),
        { wrapper },
      );
      const callback = vi.fn();
      const release = result.current.subscribe(callback, t.key);
      await waitFor(() => expect(callback).toHaveBeenCalled());
      expect(result.current.getItem(t.key)?.status?.variant).toEqual("error");
      release();
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
        owner: task.resourceID(testTask.key),
        variant: "success",
        message: "Task running",
        details: {
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
        owner: task.resourceID(testTask.key),
        variant: "error",
        message: "Task failed",
        details: {
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
    const createRange = async () =>
      await client.ranges.create({
        name: `snapshots-${id.create()}`,
        timeRange: new TimeRange(TimeStamp.now(), TimeStamp.MAX),
      });

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
      const rng = await createRange();

      const { result } = renderHook(() => Task.useCreateSnapshot(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          tasks: { key: originalTask.key, name: originalTask.name },
          range: rng.key,
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
      const rng = await createRange();

      const { result } = renderHook(() => Task.useCreateSnapshot(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          tasks: [
            { key: task1.key, name: task1.name },
            { key: task2.key, name: task2.name },
          ],
          range: rng.key,
        });
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      await waitFor(async () => {
        const firstSnapshotName = `${originalName1} (Snapshot)`;
        const secondSnapshotName = `${originalName2} (Snapshot)`;
        const snapshots = await client.tasks.retrieve({
          name: [firstSnapshotName, secondSnapshotName],
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

    it("should place each snapshot in the range", async () => {
      const rack = await client.racks.create({ name: "rangeRack" });
      const originalTask = await rack.createTask({
        name: id.create(),
        type: "pagerduty_alert",
        config: {},
      });
      const rng = await createRange();

      const { result } = renderHook(() => Task.useCreateSnapshot(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          tasks: { key: originalTask.key, name: originalTask.name },
          range: rng.key,
        });
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const inRange = await client.tasks.retrieve({ ranges: [rng.key] });
      expect(inRange.map(({ name }) => name)).toEqual([
        `${originalTask.name} (Snapshot)`,
      ]);
      expect(inRange[0].range).toEqual(rng.key);
    });

    it("should preserve task configuration in snapshots", async () => {
      const rack = await client.racks.create({
        name: "configRack",
      });
      await client.statuses.set({
        key: "st1",
        name: "st1",
        variant: "info",
        message: "st1",
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
      const rng = await createRange();

      const { result } = renderHook(() => Task.useCreateSnapshot(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          tasks: { key: originalTask.key, name: originalTask.name },
          range: rng.key,
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

  describe("useRange", () => {
    const renderRange = async (key: task.Key) =>
      await renderHookSuspended(
        () => {
          Task.useEnsure({ key });
          return Task.useRange({ key });
        },
        { wrapper },
      );

    it("should return the range of a snapshot", async () => {
      const rack = await client.racks.create({ name: "rangeRack" });
      const original = await rack.createTask({
        name: id.create(),
        type: "pagerduty_alert",
        config: { routingKey: "rk-range" },
      });
      const rng = await client.ranges.create({
        name: `range-${id.create()}`,
        timeRange: new TimeRange(TimeStamp.now(), TimeStamp.MAX),
      });
      const snapshot = await client.tasks.copy(original.key, "Snapshot", true, rng.key);
      const { result } = await renderRange(snapshot.key);
      await waitFor(() => expect(result.current).toEqual(rng.key));
    });

    it("should return nothing for a task in no range", async () => {
      const rack = await client.racks.create({ name: "noRangeRack" });
      const tsk = await rack.createTask({
        name: id.create(),
        type: "pagerduty_alert",
        config: { routingKey: "rk-no-range" },
      });
      const { result } = await renderRange(tsk.key);
      expect(result.current).toBeUndefined();
    });
  });

  describe("useMultiple", () => {
    it("should add a task on the rack created after the query resolves", async () => {
      const rack = await client.racks.create({ name: "testRack" });
      const type = "pagerduty_alert";
      const first = await rack.createTask({ name: "first", type, config: {} });
      const { result } = await renderHookSuspended(
        () =>
          Task.useMultiple({ where: { rack: { eq: rack.key }, type: { in: [type] } } }),
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
          const list = Task.useList({ query: { where: { rack: { eq: rack.key } } } });
          const del = Task.useDelete();
          return { list, del };
        },
        { wrapper },
      );
      act(() => result.current.list.more());
      await waitFor(() => expect(result.current.list.data).toEqual([testTask.key]));

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
          const list = Task.useList({ query: { where: { rack: { eq: rack.key } } } });
          const del = Task.useDelete();
          return { list, del };
        },
        { wrapper },
      );
      act(() => result.current.list.more());
      await waitFor(() => expect(result.current.list.data).toHaveLength(2));

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
    const schemas = {
      type: z.literal("pagerduty_alert"),
      config: z.object({ routingKey: z.string() }),
      statusData: statusData.nullish(),
    };
    const useForm = Task.createForm({ schemas });
    const path = deep.path<z.infer<Task.FormSchema<typeof schemas>>>();

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
      act(() => result.current.form.set(path.config.routingKey, "cat"));
      await act(async () => {
        expect(await result.current.flush()).toBe(true);
        await client.tasks.settled(t.key);
      });
      const stored = await createTestClient().tasks.retrieve(t.key);
      expect(stored.config).toMatchObject({ routingKey: "cat" });
      expect(stored.configHash).not.toBe(t.configHash);
      expect(result.current.form.get(path.configHash).value).toBe(stored.configHash);
    });

    it("should show a status update without touching a field", async () => {
      const t = await createTask();
      const { result } = await renderForm(t.key);
      const _errorStatusDetailsZ = task.statusDetailsZ(statusData);
      const taskStatus: task.Status = status.create<typeof _errorStatusDetailsZ>({
        owner: task.resourceID(t.key),
        variant: "error",
        message: "Task error",
        details: {
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
        const s = result.current.form.get(path.status).value;
        expect(s?.variant).toEqual("error");
        expect(s?.message).toEqual("Task error");
        expect(s?.details.data?.errorCode).toEqual(500);
      });
      expect(result.current.form.get(path.status).touched).toBe(false);
    });

    it("should show a name and config that another client sets", async () => {
      const t = await createTask();
      const { result } = await renderForm(t.key);
      const remote = createTestClient();
      await remote.tasks.retrieve(t.key);
      await act(async () => {
        await remote.tasks.dispatch(t.key, [
          task.set({ path: ["name"], value: "written" }),
          task.set({ path: ["config"], value: { routingKey: "remote" } }),
        ]);
      });
      await waitFor(() => {
        expect(result.current.form.get(path.name).value).toEqual("written");
        expect(result.current.form.get(path.config.routingKey).value).toEqual("remote");
      });
      expect(result.current.form.get(path.name).touched).toBe(false);
    });

    it("should keep a local edit over a rename from another client", async () => {
      const t = await createTask();
      const { result } = await renderForm(t.key);
      // The edit sends once its coalesce window ends, which can come before the remote
      // write on a slow machine. The hold keeps the Core from applying it first.
      const { release } = holdRequests(client, "/task/dispatch");
      act(() => result.current.form.set(path.config.routingKey, "local"));
      await act(async () => {
        await createTestClient().tasks.rename(t.key, "renamed");
      });
      await waitFor(() =>
        expect(result.current.form.get(path.name).value).toEqual("renamed"),
      );
      expect(result.current.form.get(path.config.routingKey).value).toEqual("local");
      release();
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
