import { newTestClient, task } from "@synnaxlabs/client";
import { id, status } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Task } from "@/hardware/task";
import { newSynnaxWrapperWithAwait } from "@/testutil/Synnax";

const client = newTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await newSynnaxWrapperWithAwait(client);
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
});
