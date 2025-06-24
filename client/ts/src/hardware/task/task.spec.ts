// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { task } from "@/hardware/task";
import { newTestClient } from "@/testutil/client";

const client = newTestClient();

describe("Task", async () => {
  const testRack = await client.hardware.racks.create({ name: "test" });
  describe("create", () => {
    it("should create a task on a rack", async () => {
      const m = await testRack.createTask({
        name: "test",
        config: { a: "dog" },
        type: "ni",
      });
      expect(m.key).not.toHaveLength(0);
      const rackKey = BigInt(m.key) >> 32n;
      expect(Number(rackKey)).toBe(testRack.key);
    });
    it("should create a task with a config", async () => {
      const config = {
        stateRate: 100,
        inputChannels: [
          { port: "AIN0", enabled: true },
          { port: "DAC2", enabled: false },
        ],
        dataSaving: false,
      };
      const m = await testRack.createTask({
        name: "test",
        config,
        type: "ni",
      });
      expect(m.config).toStrictEqual(config);
    });
  });
  describe("update", () => {
    it("should update a task if the key is provided", async () => {
      const m = await testRack.createTask({
        name: "test",
        config: { a: "dog" },
        type: "ni",
      });
      const updated = await client.hardware.tasks.create({
        ...m,
        name: "updated",
      });
      expect(updated.name).toBe("updated");
      const retrieved = await client.hardware.tasks.retrieve({ key: m.key });
      expect(retrieved.name).toBe("updated");
    });
  });
  describe("retrieve", () => {
    it("should retrieve a task by its key", async () => {
      const m = await testRack.createTask({
        name: "test",
        config: { a: "dog" },
        type: "ni",
      });
      const retrieved = await client.hardware.tasks.retrieve({ key: m.key });
      expect(retrieved.key).toBe(m.key);
      expect(retrieved.name).toBe("test");
      expect(retrieved.config).toStrictEqual({ a: "dog" });
      expect(retrieved.type).toBe("ni");
    });
    describe("retrieveByName", () => {
      it("should retrieve a task by its name", async () => {
        const name = `test-${Date.now()}-${Math.random()}`;
        const m = await testRack.createTask({ name, config: { a: "dog" }, type: "ni" });
        const retrieved = await client.hardware.tasks.retrieve({ name });
        expect(retrieved.key).toBe(m.key);
      });
    });
    describe("retrieve with state", () => {
      it("should also send the tasks state", async () => {
        const t = await testRack.createTask({
          name: "test",
          config: { a: "dog" },
          type: "ni",
        });
        const w = await client.openWriter(["sy_task_status"]);
        const communicatedStatus: task.Status = {
          key: id.create(),
          variant: "success",
          details: { task: t.key, running: false, data: {} },
          message: "test",
          time: TimeStamp.now(),
        };
        await w.write(task.STATUS_CHANNEL_NAME, [communicatedStatus]);
        await w.close();
        await expect
          .poll(async () => {
            const retrieved = await client.hardware.tasks.retrieve({
              key: t.key,
              includeStatus: true,
            });
            return retrieved.status?.variant === communicatedStatus.variant;
          })
          .toBeTruthy();
      });
    });
  });

  describe("copy", () => {
    it("should correctly copy the task", async () => {
      const m = await testRack.createTask({
        name: "test",
        config: { a: "dog" },
        type: "ni",
      });
      const copy = await client.hardware.tasks.copy(m.key, "New Name", false);
      expect(copy.name).toBe("New Name");
      expect(copy.config).toStrictEqual({ a: "dog" });
    });
  });

  describe("list", () => {
    it("should list all tasks", async () => {
      await testRack.createTask({ name: "test", config: { a: "dog" }, type: "ni" });
      const tasks = await client.hardware.tasks.list();
      expect(tasks.length).toBeGreaterThan(0);
    });
  });

  describe("executeCommand", () => {
    it("should execute a command on a task", async () => {
      const type = "testCmd";
      const args = { a: "dog" };
      const t = await testRack.createTask({
        name: "test",
        config: { a: "dog" },
        type: "ni",
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      const key = await client.hardware.tasks.executeCommand(t.key, type, args);
      await expect
        .poll<Promise<task.Command>>(async () => {
          const fr = await streamer.read();
          const sample = fr.at(-1)[task.COMMAND_CHANNEL_NAME];
          return task.commandZ.parse(sample);
        })
        .toMatchObject({ key, task: t.key, type, args });
      streamer.close();
    });
    it("should timeout on a synchronously executed command", async () => {
      const t = await testRack.createTask({
        name: "test",
        config: {},
        type: "ni",
      });
      await expect(t.executeCommandSync("test", 0)).rejects.toThrow("timed out");
    });
    it("should execute synchronously if timeout is large enough", async () => {
      const t = await testRack.createTask({
        name: "test",
        config: {},
        type: "ni",
      });
      const commandObs = await t.openCommandObserver();
      const w = await client.openWriter([task.STATUS_CHANNEL_NAME]);
      commandObs.onChange((cmd) => {
        void (async () => {
          const status: task.Status = {
            key: cmd.key,
            variant: "success",
            details: { task: cmd.task, running: false, data: { beacons: "lit" } },
            message: "test",
            time: TimeStamp.now(),
          };
          await w.write(task.STATUS_CHANNEL_NAME, [status]);
        })();
      });
      const status = await t.executeCommandSync("test", TimeSpan.fromSeconds(1));
      expect(status.variant).toBe("success");
      expect(status.details).toMatchObject({
        data: { beacons: "lit" },
        running: false,
      });
      await w.close();
      await commandObs.close();
    });
  });
});
