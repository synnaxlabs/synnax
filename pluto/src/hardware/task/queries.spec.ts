import { createTestClient, task } from "@synnaxlabs/client";
import { id, status } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import z from "zod";

import { Task } from "@/hardware/task";
import { Ontology } from "@/ontology";
import { createSynnaxWraperWithAwait } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createSynnaxWraperWithAwait({
      client,
      excludeFluxStores: [Ontology.RESOURCES_FLUX_STORE_KEY],
    });
  });

  describe("useList", () => {
    it("should return a list of task keys", async () => {
      const rack = await client.hardware.racks.create({
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
      const rack = await client.hardware.racks.create({
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
      const rack = await client.hardware.racks.create({
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
        result.current.retrieve({ term: "special" });
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
      const rack = await client.hardware.racks.create({
        name: "testRack",
      });
      for (let i = 0; i < 5; i++)
        await rack.createTask({
          name: `paginationTask${i}`,
          type: "testType",
          config: {},
        });

      const { result } = renderHook(() => Task.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ limit: 2, offset: 1 });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
    });

    it("should update the list when a task is created", async () => {
      const rack = await client.hardware.racks.create({
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
      const rack = await client.hardware.racks.create({
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

      await client.hardware.tasks.create({
        ...testTask.payload,
        name: "updated",
      });

      await waitFor(() => {
        expect(result.current.getItem(testTask.key)?.name).toEqual("updated");
      });
    });

    it("should remove task from list when deleted", async () => {
      const rack = await client.hardware.racks.create({
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

      await client.hardware.tasks.delete([testTask.key]);

      await waitFor(() => {
        expect(result.current.data).not.toContain(testTask.key);
      });
    });

    it("should update task status in the list", async () => {
      const rack = await client.hardware.racks.create({
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

      const taskStatus: task.Status = status.create({
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
        const writer = await client.openWriter([task.STATUS_CHANNEL_NAME]);
        await writer.write(task.STATUS_CHANNEL_NAME, [taskStatus]);
        await writer.close();
      });

      await waitFor(() => {
        const taskInList = result.current.getItem(testTask.key);
        expect(taskInList?.status?.variant).toEqual("error");
        expect(taskInList?.status?.message).toEqual("Task failed");
      });
    });

    it("should update task status when a command is executed", async () => {
      const rack = await client.hardware.racks.create({
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

  describe("useForm", () => {
    it("should initialize a form with default values", async () => {
      const useForm = Task.createForm({
        schemas: {
          typeSchema: z.literal("testType"),
          configSchema: z.object({}),
          statusDataSchema: z.any(),
        },
        initialValues: {
          key: "123",
          name: "testTask",
          type: "testType",
          config: {},
        },
      });
      const { result } = renderHook(() => useForm({ params: {} }), {
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
      const rack = await client.hardware.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "existingTask",
        type: "testType",
        config: { setting: "value" },
      });

      const useForm = Task.createForm({
        schemas: {
          typeSchema: z.literal("testType"),
          configSchema: z.object({ setting: z.string() }),
          statusDataSchema: z.any().or(z.null()),
        },
        initialValues: {
          key: "0",
          name: "",
          type: "testType",
          config: { setting: "" },
        },
      });

      const { result } = renderHook(() => useForm({ params: { key: testTask.key } }), {
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
      const rack = await client.hardware.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "taskToUpdate",
        type: "testType",
        config: {},
      });

      const useForm = Task.createForm({
        schemas: {
          typeSchema: z.literal("testType"),
          configSchema: z.object({}),
          statusDataSchema: z.any(),
        },
        initialValues: {
          key: testTask.key,
          name: "taskToUpdate",
          type: "testType",
          config: {},
        },
      });

      const { result } = renderHook(() => useForm({ params: { key: testTask.key } }), {
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

      const updatedTask = await client.hardware.tasks.retrieve({
        key: testTask.key,
      });
      expect(updatedTask.name).toEqual("updatedTaskName");
    });

    it("should validate form fields according to schema", async () => {
      const useForm = Task.createForm({
        schemas: {
          typeSchema: z.literal("testType"),
          configSchema: z.object({
            port: z.number().min(1).max(65535),
            host: z.string().min(1),
          }),
          statusDataSchema: z.any(),
        },
        initialValues: {
          key: "0",
          name: "test",
          type: "testType",
          config: { port: 0, host: "" },
        },
      });

      const { result } = renderHook(() => useForm({ params: {} }), {
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
      const rack = await client.hardware.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "taskWithBeforeSave",
        type: "testType",
        config: {},
      });

      const beforeSave = vi.fn().mockResolvedValue(true);

      const useForm = Task.createForm({
        schemas: {
          typeSchema: z.literal("testType"),
          configSchema: z.object({}),
          statusDataSchema: z.any(),
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
            params: { key: testTask.key },
            beforeSave,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      act(() => {
        result.current.form.set("name", "modifiedName");
      });

      act(() => {
        result.current.save();
      });

      expect(beforeSave).toHaveBeenCalledWith(
        expect.objectContaining({
          params: { key: testTask.key },
        }),
      );
    });

    it("should handle afterSave callback", async () => {
      const rack = await client.hardware.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "taskWithAfterSave",
        type: "testType",
        config: {},
      });

      const afterSave = vi.fn();

      const useForm = Task.createForm({
        schemas: {
          typeSchema: z.literal("testType"),
          configSchema: z.object({}),
          statusDataSchema: z.any(),
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
            params: { key: testTask.key },
            afterSave,
          }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      act(() => {
        result.current.form.set("name", "savedName");
      });

      await act(async () => {
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.form.get("name").value).toEqual("savedName");
      });

      expect(afterSave).toHaveBeenCalledWith(
        expect.objectContaining({
          params: { key: testTask.key },
        }),
      );
    });

    it("should update form when task status changes", async () => {
      const rack = await client.hardware.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "statusTask",
        type: "testType",
        config: {},
      });

      const statusDataSchema = z.object({ errorCode: z.number().optional() });

      const useForm = Task.createForm({
        schemas: {
          typeSchema: z.literal("testType"),
          configSchema: z.object({}),
          statusDataSchema: statusDataSchema.or(z.null()),
        },
        initialValues: {
          key: testTask.key,
          name: "statusTask",
          type: "testType",
          config: {},
        },
      });

      const { result } = renderHook(() => useForm({ params: { key: testTask.key } }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const taskStatus: task.Status = status.create({
        key: id.create(),
        variant: "error",
        message: "Task error",
        details: {
          task: testTask.key,
          running: false,
          data: { errorCode: 500 },
        },
      });

      await act(async () => {
        const writer = await client.openWriter([task.STATUS_CHANNEL_NAME]);
        await writer.write(task.STATUS_CHANNEL_NAME, [taskStatus]);
        await writer.close();
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
          typeSchema: z.literal("testType"),
          configSchema: z.object({}),
          statusDataSchema: z.any(),
        },
        initialValues: {
          key: "123",
          name: "originalName",
          type: "testType",
          config: {},
        },
      });

      const { result } = renderHook(() => useForm({ params: {} }), {
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

    it("should reset form to initial values", async () => {
      const rack = await client.hardware.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "resetTask",
        type: "testType",
        config: { value: "initial" },
      });

      const useForm = Task.createForm({
        schemas: {
          typeSchema: z.literal("testType"),
          configSchema: z.object({ value: z.string() }),
          statusDataSchema: z.any(),
        },
        initialValues: {
          key: testTask.key,
          name: "resetTask",
          type: "testType",
          config: { value: "initial" },
        },
      });

      const { result } = renderHook(() => useForm({ params: { key: testTask.key } }), {
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
          typeSchema: z.literal("testType"),
          configSchema: z.object({}),
          statusDataSchema: z.any(),
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
            params: { key: "999999" },
          }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("error");
        expect(result.current.status.message).toEqual("Failed to retrieve Task");
      });
    });

    it("should handle autoSave functionality", async () => {
      const rack = await client.hardware.racks.create({
        name: "testRack",
      });
      const testTask = await rack.createTask({
        name: "autoSaveTask",
        type: "testType",
        config: {},
      });

      const useForm = Task.createForm({
        schemas: {
          typeSchema: z.literal("testType"),
          configSchema: z.object({}),
          statusDataSchema: z.any(),
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
            params: { key: testTask.key },
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
          const updatedTask = await client.hardware.tasks.retrieve({
            key: testTask.key,
          });
          expect(updatedTask.name).toEqual("autoSavedName");
        },
        { timeout: 3000 },
      );
    });

    it("should handle complex config schemas with nested objects", async () => {
      const rack = await client.hardware.racks.create({
        name: "testRack",
      });

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

      const testTask = await rack.createTask({
        name: "complexTask",
        type: "complexType",
        config: complexConfig,
      });

      const useForm = Task.createForm({
        schemas: {
          typeSchema: z.literal("complexType"),
          configSchema: z.object({
            connection: z.object({
              host: z.string(),
              port: z.number(),
              secure: z.boolean(),
            }),
            settings: z.object({
              timeout: z.number(),
              retryCount: z.number(),
            }),
          }),
          statusDataSchema: z.any(),
        },
        initialValues: {
          key: testTask.key,
          name: "complexTask",
          type: "complexType",
          config: complexConfig,
        },
      });

      const { result } = renderHook(() => useForm({ params: { key: testTask.key } }), {
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
        result.current.save();
      });

      // const updatedTask = await client.hardware.tasks.retrieve<({
      //   key: testTask.key,
      // });
      // expect(updatedTask.config.connection.port).toEqual(9090);
    });
  });
});
