// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

import { ontology } from "@/ontology";
import { task } from "@/task";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Task", async () => {
  const testRack = await client.racks.create({ name: "test" });
  describe("create", () => {
    it.only("should create a task on a rack", async () => {
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
    it("should create a task with a custom status", async () => {
      const customStatus: task.NewStatus = {
        key: "",
        name: "",
        variant: "success",
        message: "Custom task status",
        description: "Task is running",
        time: TimeStamp.now(),
        details: { running: true, data: { customData: true } },
      };
      const m = await testRack.createTask({
        name: "task-with-status",
        config: { test: true },
        type: "ni",
        status: customStatus,
      });
      expect(m.key).not.toHaveLength(0);
      const retrieved = await client.tasks.retrieve({
        key: m.key,
        includeStatus: true,
      });
      expect(retrieved.status).toBeDefined();
      expect(retrieved.status?.variant).toBe("success");
      expect(retrieved.status?.message).toBe("Custom task status");
      expect(retrieved.status?.description).toBe("Task is running");
      expect(retrieved.status?.details?.task).toBe(m.key);
      expect(retrieved.status?.details?.running).toBe(true);
    });
  });
  describe("update", () => {
    it("should update a task if the key is provided", async () => {
      const m = await testRack.createTask({
        name: "test",
        config: { a: "dog" },
        type: "ni",
      });
      // Exclude status, internal, snapshot when updating - these have different input/output types
      const { status: _, internal: __, snapshot: ___, ...taskFields } = m;
      const updated = await client.tasks.create({
        ...taskFields,
        name: "updated",
      });
      expect(updated.name).toBe("updated");
      const retrieved = await client.tasks.retrieve({ key: m.key });
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
      const retrieved = await client.tasks.retrieve({ key: m.key });
      expect(retrieved.key).toBe(m.key);
      expect(retrieved.name).toBe("test");
      expect(retrieved.config).toStrictEqual({ a: "dog" });
      expect(retrieved.type).toBe("ni");
    });

    it("should retrieve a task by its name", async () => {
      const name = `test-${Date.now()}-${Math.random()}`;
      const m = await testRack.createTask({ name, config: { a: "dog" }, type: "ni" });
      const retrieved = await client.tasks.retrieve({ name });
      expect(retrieved.key).toBe(m.key);
    });

    describe("status", () => {
      it("should include task status when requested", async () => {
        const t = await testRack.createTask({
          name: "test",
          config: { a: "dog" },
          type: "ni",
        });
        const communicatedStatus: task.Status = {
          key: ontology.idToString(task.ontologyID(t.key)),
          name: "test",
          variant: "success",
          details: { task: t.key, running: false, data: {} },
          message: "test",
          time: TimeStamp.now(),
        };
        await client.statuses.set(communicatedStatus);
        await expect
          .poll(async () => {
            const retrieved = await client.tasks.retrieve({
              key: t.key,
              includeStatus: true,
            });
            return retrieved.status?.variant === communicatedStatus.variant;
          })
          .toBe(true);
      });
    });

    describe("request object format", () => {
      const testTasks: Array<{ key: string; name: string; type: string }> = [];
      let secondRack: any;

      beforeAll(async () => {
        secondRack = await client.racks.create({ name: "test-rack-2" });

        const taskConfigs = [
          { name: "sensor_task1", type: "ni", rack: testRack },
          { name: "sensor_task2", type: "ni", rack: testRack },
          { name: "actuator_task1", type: "labjack", rack: testRack },
          { name: "actuator_task2", type: "labjack", rack: secondRack },
          { name: "controller_task", type: "opc", rack: secondRack },
        ];

        for (const config of taskConfigs) {
          const task = await config.rack.createTask({
            name: config.name,
            type: config.type,
            config: { test: true },
          });
          testTasks.push({ key: task.key, name: config.name, type: config.type });
        }
      });

      it("should retrieve tasks by rack", async () => {
        const result = await client.tasks.retrieve({
          rack: testRack.key,
        });
        expect(result.length).toBeGreaterThanOrEqual(3);
        expect(result.every((t) => task.rackKey(t.key) === testRack.key)).toBe(true);
      });

      it("should retrieve tasks by multiple keys", async () => {
        const keysToQuery = testTasks.slice(0, 2).map((t) => t.key);
        const result = await client.tasks.retrieve({
          keys: keysToQuery,
        });
        expect(result).toHaveLength(2);
        expect(result.map((t) => t.key).sort()).toEqual(keysToQuery.sort());
      });

      it("should retrieve tasks by multiple names", async () => {
        const namesToQuery = ["sensor_task1", "actuator_task1"];
        const result = await client.tasks.retrieve({
          names: namesToQuery,
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((t) => namesToQuery.includes(t.name))).toBe(true);
      });

      it("should retrieve tasks by types", async () => {
        const result = await client.tasks.retrieve({
          types: ["ni"],
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((t) => t.type === "ni")).toBe(true);
      });

      it("should retrieve tasks by multiple types", async () => {
        const typesToQuery = ["ni", "labjack"];
        const result = await client.tasks.retrieve({
          types: typesToQuery,
        });
        expect(result.length).toBeGreaterThanOrEqual(4);
        expect(result.every((t) => typesToQuery.includes(t.type))).toBe(true);
      });

      it("should support pagination with limit and offset", async () => {
        const firstPage = await client.tasks.retrieve({
          rack: testRack.key,
          limit: 2,
          offset: 0,
        });
        expect(firstPage.length).toBeLessThanOrEqual(2);

        if (firstPage.length === 2) {
          const secondPage = await client.tasks.retrieve({
            rack: testRack.key,
            limit: 2,
            offset: 2,
          });

          const firstPageKeys = firstPage.map((t) => t.key);
          const secondPageKeys = secondPage.map((t) => t.key);
          expect(firstPageKeys.every((key) => !secondPageKeys.includes(key))).toBe(
            true,
          );
        }
      });

      it("should support combined filters", async () => {
        const result = await client.tasks.retrieve({
          rack: testRack.key,
          types: ["ni"],
          includeStatus: true,
        });
        expect(result.length).toBeGreaterThanOrEqual(1);
        expect(result.every((t) => t.type === "ni")).toBe(true);

        await expect
          .poll(async () => {
            const tasks = await client.tasks.retrieve({
              rack: testRack.key,
              types: ["ni"],
              includeStatus: true,
            });
            return tasks.every((t) => t.status !== undefined);
          })
          .toBe(true);
      });

      it("should handle empty results gracefully", async () => {
        const result = await client.tasks.retrieve({
          types: ["nonexistent_type"],
        });
        expect(result).toEqual([]);
      });

      it("should combine rack and type filters", async () => {
        const result = await client.tasks.retrieve({
          rack: secondRack.key,
          types: ["labjack"],
        });
        expect(result.length).toBeGreaterThanOrEqual(1);
        expect(result.every((t) => t.type === "labjack")).toBe(true);
      });

      it("should handle limit without offset", async () => {
        const result = await client.tasks.retrieve({
          limit: 1,
        });
        expect(result).toHaveLength(1);
      });

      it("should retrieve tasks with includeStatus in request object", async () => {
        const result = await client.tasks.retrieve({
          rack: testRack.key,
          includeStatus: true,
        });
        expect(result.length).toBeGreaterThanOrEqual(1);

        await expect
          .poll(async () => {
            const tasks = await client.tasks.retrieve({
              rack: testRack.key,
              includeStatus: true,
            });
            return tasks.every((t) => t.status !== undefined);
          })
          .toBe(true);
      });

      it("should filter tasks by snapshot parameter", async () => {
        const regularTask = await testRack.createTask({
          name: "regular_test_task",
          type: "ni",
          config: { test: true },
        });

        const snapshotTask = await client.tasks.copy(
          regularTask.key,
          "snapshot_test_task",
          true,
        );

        const snapshotOnlyResult = await client.tasks.retrieve({
          snapshot: true,
        });
        expect(snapshotOnlyResult.some((t) => t.key === snapshotTask.key)).toBe(true);
        expect(snapshotOnlyResult.every((t) => t.snapshot === true)).toBe(true);

        const regularOnlyResult = await client.tasks.retrieve({
          snapshot: false,
        });
        expect(regularOnlyResult.some((t) => t.key === regularTask.key)).toBe(true);
        expect(regularOnlyResult.every((t) => t.snapshot === false)).toBe(true);
      });

      it("should combine snapshot filter with other filters", async () => {
        const task1 = await testRack.createTask({
          name: "combined_filter_task",
          type: "ni",
          config: { test: true },
        });

        const result = await client.tasks.retrieve({
          rack: testRack.key,
          types: ["ni"],
          snapshot: false,
        });

        expect(result.some((t) => t.key === task1.key)).toBe(true);
        expect(result.every((t) => t.type === "ni")).toBe(true);
        expect(result.every((t) => t.snapshot === false)).toBe(true);
      });
    });
  });

  describe("list", () => {
    it("should list all tasks excluding internal tasks", async () => {
      const tasks = await client.tasks.list();
      expect(Array.isArray(tasks)).toBe(true);
      expect(tasks.every((t) => t.internal === false)).toBe(true);
    });

    it("should list tasks on a specific rack", async () => {
      const task1 = await testRack.createTask({
        name: `list-test-${Date.now()}`,
        config: { test: true },
        type: "ni",
      });
      const tasks = await client.tasks.list(testRack.key);
      expect(tasks.some((t) => t.key === task1.key)).toBe(true);
      expect(tasks.every((t) => task.rackKey(t.key) === testRack.key)).toBe(true);
    });

    it("should exclude internal tasks by default", async () => {
      const allTasks = await client.tasks.list();
      const internalTasks = await client.tasks.retrieve({
        internal: true,
      });
      const allTaskKeys = allTasks.map((t) => t.key);
      const internalTaskKeys = internalTasks.map((t) => t.key);
      expect(internalTaskKeys.every((key) => !allTaskKeys.includes(key))).toBe(true);
    });
  });

  describe("copy", () => {
    it("should correctly copy the task", async () => {
      const m = await testRack.createTask({
        name: "test",
        config: { a: "dog" },
        type: "ni",
      });
      const copy = await client.tasks.copy(m.key, "New Name", false);
      expect(copy.name).toBe("New Name");
      expect(copy.config).toStrictEqual({ a: "dog" });
    });
  });

  describe("lifecycle methods", () => {
    it("should start a task", async () => {
      const t = await testRack.createTask({
        name: "lifecycle-start-test",
        config: {},
        type: "ni",
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      await t.start();
      await expect
        .poll<Promise<task.Command>>(async () => {
          const fr = await streamer.read();
          const sample = fr.at(-1)[task.COMMAND_CHANNEL_NAME];
          return task.commandZ.parse(sample);
        })
        .toMatchObject({ task: t.key, type: "start" });
      streamer.close();
    });

    it("should stop a task", async () => {
      const t = await testRack.createTask({
        name: "lifecycle-stop-test",
        config: {},
        type: "ni",
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      await t.stop();
      await expect
        .poll<Promise<task.Command>>(async () => {
          const fr = await streamer.read();
          const sample = fr.at(-1)[task.COMMAND_CHANNEL_NAME];
          return task.commandZ.parse(sample);
        })
        .toMatchObject({ task: t.key, type: "stop" });
      streamer.close();
    });

    it("should run a task with automatic start and stop", async () => {
      const t = await testRack.createTask({
        name: "lifecycle-run-test",
        config: {},
        type: "ni",
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      let executedCallback = false;

      await t.run(async () => {
        executedCallback = true;
      });

      expect(executedCallback).toBe(true);

      // Should have received both start and stop commands
      const commands: task.Command[] = [];
      await expect
        .poll(async () => {
          try {
            const fr = await streamer.read();
            const samples = fr.get(task.COMMAND_CHANNEL_NAME);
            for (const sample of samples) commands.push(task.commandZ.parse(sample));

            return (
              commands.some((c) => c.task === t.key && c.type === "start") &&
              commands.some((c) => c.task === t.key && c.type === "stop")
            );
          } catch {
            return false;
          }
        })
        .toBe(true);

      streamer.close();
    });

    it("should stop task even if callback throws error", async () => {
      const t = await testRack.createTask({
        name: "lifecycle-run-error-test",
        config: {},
        type: "ni",
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);

      await expect(
        t.run(async () => {
          throw new Error("Test error");
        }),
      ).rejects.toThrow("Test error");

      // Should still have received stop command
      const stopCommands: task.Command[] = [];
      await expect
        .poll(async () => {
          try {
            const fr = await streamer.read();
            const samples = fr.get(task.COMMAND_CHANNEL_NAME);
            for (const sample of samples) {
              const cmd = task.commandZ.parse(sample);
              if (cmd.task === t.key && cmd.type === "stop") stopCommands.push(cmd);
            }
            return stopCommands.length > 0;
          } catch {
            return false;
          }
        })
        .toBe(true);

      streamer.close();
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
      const key = await client.tasks.executeCommand({
        task: t.key,
        type,
        args,
      });
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
      await expect(t.executeCommandSync({ type: "test", timeout: 0 })).rejects.toThrow(
        "timed out",
      );
    });
  });
});
