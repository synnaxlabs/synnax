// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeStamp, uuid } from "@synnaxlabs/x";
import { assert, beforeAll, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { ontology } from "@/ontology";
import { query } from "@/query";
import { task } from "@/task";
import { createTestClient } from "@/testutil";

const client = createTestClient();

describe("Task", async () => {
  const testRack = await client.racks.create({ name: "test" });
  describe("create", () => {
    it("should create a task on a rack", async () => {
      const m = await testRack.createTask({
        name: "test",
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      expect(m.key).not.toHaveLength(0);
      expect(m.rack).toBe(testRack.key);
    });
    it("should reject a task with an unknown type", async () => {
      await expect(
        testRack.createTask({ name: "test", config: {}, type: "made_up_type" }),
      ).rejects.toThrow("unknown task type");
    });
    it("should create a task with a config", async () => {
      const config = {
        routingKey: "rk-create",
        autoStart: true,
        alerts: [{ key: "a1", status: "st1" }],
      };
      const m = await testRack.createTask({
        name: "test",
        config,
        type: "pagerduty_alert",
      });
      expect(m.config).toMatchObject({
        routingKey: "rk-create",
        autoStart: true,
        alerts: [{ key: "a1", status: "st1" }],
      });
      expect(m.config.key).toBeTypeOf("string");
    });
    it("should create a task with a custom status", async () => {
      const key = uuid.create();
      const customStatus: task.New["status"] = {
        name: "Status",
        variant: "success",
        message: "Custom task status",
        description: "Task is running",
        details: { task: key, running: true, data: { customData: true } },
      };
      const m = await testRack.createTask({
        key,
        name: "task-with-status",
        config: {},
        type: "pagerduty_alert",
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
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      // Exclude status, internal, snapshot when updating - these have different input/output types
      const { status: _, internal: __, snapshot: ___, ...taskFields } = m;
      const updated = await client.tasks.create({
        ...taskFields,
        name: "updated",
      });
      expect(updated.name).toBe("updated");
      const retrieved = await client.tasks.retrieve(m.key);
      expect(retrieved.name).toBe("updated");
    });
  });
  describe("retrieve", () => {
    it("should retrieve a task by its key", async () => {
      const m = await testRack.createTask({
        name: "test",
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const retrieved = await client.tasks.retrieve(m.key);
      expect(retrieved.key).toBe(m.key);
      expect(retrieved.name).toBe("test");
      expect(retrieved.config).toMatchObject({ routingKey: "dog" });
      expect(retrieved.type).toBe("pagerduty_alert");
    });

    it("should retrieve a task by its name", async () => {
      const name = `test-${Date.now()}-${Math.random()}`;
      const m = await testRack.createTask({
        name,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const retrieved = await client.tasks.retrieve({ name });
      expect(retrieved.key).toBe(m.key);
    });

    it("should retrieve tasks by search term", async () => {
      const prefix = `searchable-task-${id.create()}`;
      const names = [`${prefix}-1`, `${prefix}-2`];
      await Promise.all(
        names.map((name) =>
          testRack.createTask({ name, config: {}, type: "pagerduty_alert" }),
        ),
      );
      await expect
        .poll(async () => {
          const results = await client.tasks.retrieve({ searchTerm: prefix });
          return results.map((t) => t.name).sort();
        })
        .toEqual(names);
    });

    describe("status", () => {
      it("should include task status when requested", async () => {
        const t = await testRack.createTask({
          name: "test",
          config: { routingKey: "dog" },
          type: "pagerduty_alert",
        });
        const communicatedStatus: task.Status = {
          key: ontology.idToString(task.ontologyID(t.key)),
          name: "test",
          variant: "success",
          details: {
            task: t.key,
            running: false,
            cmd: "",
            configHash: "",
            rack: testRack.key,
            data: undefined,
          },
          message: "test",
          description: "",
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
          { name: "sensor_task1", type: "pagerduty_alert", rack: testRack },
          { name: "sensor_task2", type: "pagerduty_alert", rack: testRack },
          { name: "actuator_task1", type: "labjack_scan", rack: testRack },
          { name: "actuator_task2", type: "labjack_scan", rack: secondRack },
          { name: "controller_task", type: "opc_scan", rack: secondRack },
        ];

        for (const config of taskConfigs) {
          const task = await config.rack.createTask({
            name: config.name,
            type: config.type,
            config: {},
          });
          testTasks.push({ key: task.key, name: config.name, type: config.type });
        }
      });

      it("should retrieve tasks by rack", async () => {
        const result = await client.tasks.retrieve({
          rack: testRack.key,
        });
        expect(result.length).toBeGreaterThanOrEqual(3);
        expect(result.every((t) => t.rack === testRack.key)).toBe(true);
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
          types: ["pagerduty_alert"],
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((t) => t.type === "pagerduty_alert")).toBe(true);
      });

      it("should retrieve tasks by multiple types", async () => {
        const typesToQuery = ["pagerduty_alert", "labjack_scan"];
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
          types: ["pagerduty_alert"],
          includeStatus: true,
        });
        expect(result.length).toBeGreaterThanOrEqual(1);
        expect(result.every((t) => t.type === "pagerduty_alert")).toBe(true);

        await expect
          .poll(async () => {
            const tasks = await client.tasks.retrieve({
              rack: testRack.key,
              types: ["pagerduty_alert"],
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
          types: ["labjack_scan"],
        });
        expect(result.length).toBeGreaterThanOrEqual(1);
        expect(result.every((t) => t.type === "labjack_scan")).toBe(true);
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
          type: "pagerduty_alert",
          config: {},
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
          type: "pagerduty_alert",
          config: {},
        });

        const result = await client.tasks.retrieve({
          rack: testRack.key,
          types: ["pagerduty_alert"],
          snapshot: false,
        });

        expect(result.some((t) => t.key === task1.key)).toBe(true);
        expect(result.every((t) => t.type === "pagerduty_alert")).toBe(true);
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
        config: {},
        type: "pagerduty_alert",
      });
      const tasks = await client.tasks.list(testRack.key);
      expect(tasks.some((t) => t.key === task1.key)).toBe(true);
      expect(tasks.every((t) => t.rack === testRack.key)).toBe(true);
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
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const copy = await client.tasks.copy(m.key, "New Name", false);
      expect(copy.name).toBe("New Name");
      expect(copy.config).toMatchObject({ routingKey: "dog" });
    });
  });

  describe("lifecycle methods", () => {
    it("should start a task", async () => {
      const t = await testRack.createTask({
        name: "lifecycle-start-test",
        config: {},
        type: "pagerduty_alert",
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
        type: "pagerduty_alert",
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
        type: "pagerduty_alert",
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
        type: "pagerduty_alert",
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
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
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
    it("should stamp the config hash from the task instance", async () => {
      const t = await testRack.createTask({
        name: "test",
        config: { a: "dog" },
        type: "ni",
      });
      expect(t.configHash).not.toEqual("");
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      const key = await t.executeCommand({ type: "start" });
      await expect
        .poll<Promise<task.Command>>(async () => {
          const fr = await streamer.read();
          const sample = fr.at(-1)[task.COMMAND_CHANNEL_NAME];
          return task.commandZ.parse(sample);
        })
        .toMatchObject({ key, task: t.key, configHash: t.configHash });
      streamer.close();
    });
    it("should stamp the config hash from the cached task row", async () => {
      const t = await testRack.createTask({
        name: "test",
        config: { a: "dog" },
        type: "ni",
      });
      await client.tasks.retrieve(t.key);
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      const key = await client.tasks.executeCommand({ task: t.key, type: "start" });
      await expect
        .poll<Promise<task.Command>>(async () => {
          const fr = await streamer.read();
          const sample = fr.at(-1)[task.COMMAND_CHANNEL_NAME];
          return task.commandZ.parse(sample);
        })
        .toMatchObject({ key, task: t.key, configHash: t.configHash });
      streamer.close();
    });
    it("should timeout on a synchronously executed command", async () => {
      const t = await testRack.createTask({
        name: "test",
        config: {},
        type: "pagerduty_alert",
      });
      await expect(t.executeCommandSync({ type: "test", timeout: 0 })).rejects.toThrow(
        "timed out",
      );
    });
  });

  describe("with schemas", () => {
    const schemas = {
      type: z.literal("pagerduty_alert"),
      config: z.object({
        routingKey: z.string(),
        autoStart: z.boolean(),
        alerts: z.array(z.object({ key: z.string(), status: z.string() })),
      }),
      statusData: z.unknown().optional(),
    };

    it("should create and retrieve a task with typed config", async () => {
      const config = {
        routingKey: "rk-typed",
        autoStart: true,
        alerts: [{ key: "a1", status: "st1" }],
      };
      const t = await testRack.createTask(
        {
          name: "typed-config-task",
          type: "pagerduty_alert",
          config,
        },
        schemas,
      );
      expect(t.config.routingKey).toBe("rk-typed");
      expect(t.config.autoStart).toBe(true);

      const retrieved = await client.tasks.retrieve({
        key: t.key,
        schemas,
      });
      expect(retrieved.config.routingKey).toBe("rk-typed");
      expect(retrieved.config.alerts).toMatchObject([{ key: "a1", status: "st1" }]);
      expect(retrieved.type).toBe("pagerduty_alert");
    });

    it("should retrieve multiple tasks with schemas", async () => {
      const t1 = await testRack.createTask(
        {
          name: "multi-schema-1",
          type: "pagerduty_alert",
          config: { routingKey: "rk-100", autoStart: false, alerts: [] },
        },
        schemas,
      );
      const t2 = await testRack.createTask(
        {
          name: "multi-schema-2",
          type: "pagerduty_alert",
          config: { routingKey: "rk-200", autoStart: false, alerts: [] },
        },
        schemas,
      );

      const retrieved = await client.tasks.retrieve({
        keys: [t1.key, t2.key],
        schemas,
      });
      expect(retrieved).toHaveLength(2);
      expect(retrieved[0].config.routingKey).toBe("rk-100");
      expect(retrieved[1].config.routingKey).toBe("rk-200");
    });

    it("should retrieve task by name with schemas", async () => {
      const uniqueName = `schema-name-test-${Date.now()}`;
      await testRack.createTask(
        {
          name: uniqueName,
          type: "pagerduty_alert",
          config: { routingKey: "rk-500", autoStart: false, alerts: [] },
        },
        schemas,
      );

      const retrieved = await client.tasks.retrieve({
        name: uniqueName,
        schemas,
      });
      expect(retrieved.name).toBe(uniqueName);
      expect(retrieved.config.routingKey).toBe("rk-500");
    });

    it("should retrieve task by type with schemas", async () => {
      const customSchemas = {
        type: z.literal("modbus_scan"),
        config: z.object({ rate: z.number() }),
        statusData: z.unknown(),
      };

      await testRack.createTask(
        {
          name: "type-filter-test",
          type: "modbus_scan",
          config: { rate: 42 },
        },
        customSchemas,
      );

      const retrieved = await client.tasks.retrieve({
        type: "modbus_scan",
        rack: testRack.key,
        schemas: customSchemas,
      });
      expect(retrieved.type).toBe("modbus_scan");
      expect(retrieved.config.rate).toBe(42);
    });
  });

  describe("onChange", () => {
    it("merges a status keyed by details.task into a subscribed single query", async () => {
      const t = await testRack.createTask({
        name: `status-details-single-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const params = { key: t.key };
      const off = client.tasks.onChange(params, vi.fn());
      try {
        await client.tasks.retrieve(params);
        await client.statuses.set({
          key: id.create(),
          name: "Task Status",
          variant: "error",
          message: "task failed",
          time: TimeStamp.now(),
          details: { task: t.key, running: false, cmd: "", data: {} },
        });
        await expect
          .poll(() => {
            const cached = client.tasks.getCached(params);
            if (!query.isLive(cached)) return undefined;
            return cached.status?.message;
          })
          .toBe("task failed");
      } finally {
        off();
      }
    });

    it("merges a status keyed by details.task into a subscribed request query", async () => {
      const t = await testRack.createTask({
        name: `status-details-request-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const params = { keys: [t.key] };
      const off = client.tasks.onChange(params, vi.fn());
      try {
        await client.tasks.retrieve(params);
        await client.statuses.set({
          key: id.create(),
          name: "Task Status",
          variant: "warning",
          message: "task degraded",
          time: TimeStamp.now(),
          details: { task: t.key, running: true, cmd: "", data: {} },
        });
        await expect
          .poll(() => {
            const cached = client.tasks.getCached(params);
            if (!query.isLive(cached)) return undefined;
            return cached.find((v) => v.key === t.key)?.status?.message;
          })
          .toBe("task degraded");
      } finally {
        off();
      }
    });

    it("preserves deploy info on the optimistic command-loading status", async () => {
      const t = await testRack.createTask({
        name: `status-loading-carry-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const params = { key: t.key };
      const off = client.tasks.onChange(params, vi.fn());
      try {
        await client.tasks.retrieve(params);
        await client.statuses.set({
          key: id.create(),
          name: "Task Status",
          variant: "success",
          message: "task started",
          time: TimeStamp.now(),
          details: {
            task: t.key,
            running: true,
            cmd: "",
            configHash: "deadbeef",
            rack: testRack.key,
            data: {},
          },
        });
        await expect
          .poll(() => {
            const cached = client.tasks.getCached(params);
            if (!query.isLive(cached)) return undefined;
            return cached.status?.details.configHash;
          })
          .toBe("deadbeef");
        await client.tasks.executeCommand({ task: t.key, type: "stop" });
        await expect
          .poll(() => {
            const cached = client.tasks.getCached(params);
            if (!query.isLive(cached)) return undefined;
            return cached.status?.variant;
          })
          .toBe("loading");
        const cached = client.tasks.getCached(params);
        if (!query.isLive(cached)) throw new Error("expected live cached task");
        expect(cached.status?.details.configHash).toBe("deadbeef");
        expect(cached.status?.details.rack).toBe(testRack.key);
      } finally {
        off();
      }
    });

    it("refetches a cached task whose config was changed by another client", async () => {
      const t = await testRack.createTask({
        name: `set-config-${id.create()}`,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const remote = createTestClient();
      await remote.connect();
      const params = { key: t.key };
      const off = remote.tasks.onChange(params, vi.fn());
      try {
        expect((await remote.tasks.retrieve(params)).config).toMatchObject({
          routingKey: "dog",
        });
        await client.tasks.create({
          ...t.payload,
          config: { ...(t.payload.config as object), routingKey: "cat" },
        });
        await expect
          .poll(() => {
            const cached = remote.tasks.getCached(params);
            return query.isLive(cached) ? cached.config : undefined;
          })
          .toMatchObject({ routingKey: "cat" });
      } finally {
        off();
      }
    });

    it("merges a metadata-only set signal into a cached task", async () => {
      const remote = createTestClient();
      await remote.connect();
      const t = await testRack.createTask({
        name: `set-merge-${id.create()}`,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const params = { key: t.key };
      const off = remote.tasks.onChange(params, vi.fn());
      try {
        await remote.tasks.retrieve(params);
        const name = `set-merge-renamed-${id.create()}`;
        await client.tasks.create({ ...t.payload, name });
        await expect
          .poll(() => {
            const cached = remote.tasks.getCached(params);
            return query.isLive(cached) ? cached.name : undefined;
          })
          .toBe(name);
        const cached = remote.tasks.getCached(params);
        assert(query.isLive(cached));
        expect(cached.config).toMatchObject({ routingKey: "dog" });
      } finally {
        off();
      }
    });

    it("fetches an uncached task moved onto a subscribed rack", async () => {
      const remote = createTestClient();
      await remote.connect();
      const source = await client.racks.create({ name: `set-move-src-${id.create()}` });
      const dest = await client.racks.create({ name: `set-move-dest-${id.create()}` });
      const t = await source.createTask({
        name: `set-move-${id.create()}`,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const params = { rack: dest.key };
      const off = remote.tasks.onChange(params, vi.fn());
      try {
        expect(await remote.tasks.retrieve(params)).toHaveLength(0);
        await client.tasks.create({ ...t.payload, rack: dest.key });
        await expect
          .poll(() => {
            const cached = remote.tasks.getCached(params);
            if (!query.isLive(cached)) return undefined;
            return cached.find((v) => v.key === t.key)?.config;
          })
          .toMatchObject({ routingKey: "dog" });
      } finally {
        off();
      }
    });
  });
});

describe("drifted", () => {
  // Both hashes are server-assigned and opaque here; the client only compares them.
  const DEPLOYED = "2de66015b3bdded8";
  const EDITED = "0000000000000000";
  const newPayload = (overrides: {
    running?: boolean;
    taskHash?: string;
    statusHash?: string;
    statusRack?: number;
    hasStatus?: boolean;
  }): task.Payload => {
    const key = uuid.create();
    const {
      running = true,
      taskHash = DEPLOYED,
      statusHash = DEPLOYED,
      statusRack = 1,
      hasStatus = true,
    } = overrides;
    return {
      key,
      rack: 1,
      name: "test",
      type: "test",
      config: { rate: 50, port: 8080, host: "localhost" },
      configHash: taskHash,
      internal: false,
      snapshot: false,
      status: hasStatus
        ? task.statusZ().parse({
            message: "running",
            variant: "success",
            details: { task: key, running, configHash: statusHash, rack: statusRack },
          })
        : undefined,
    };
  };

  it("should not drift when the deployed hash and rack match the task", () => {
    expect(task.drifted(newPayload({}))).toBe(false);
  });

  it("should drift when the stored hash differs from the deployed hash", () => {
    expect(task.drifted(newPayload({ taskHash: EDITED }))).toBe(true);
  });

  it("should drift when the task moved racks while running", () => {
    expect(task.drifted(newPayload({ statusRack: 2 }))).toBe(true);
  });

  it("should never drift when the task is not running", () => {
    expect(task.drifted(newPayload({ running: false, taskHash: EDITED }))).toBe(false);
  });

  it("should never drift without a status", () => {
    expect(task.drifted(newPayload({ hasStatus: false }))).toBe(false);
  });

  it("should never drift when the deployed hash is unknown", () => {
    // The optimistic command-loading status reports running with an empty hash.
    expect(task.drifted(newPayload({ statusHash: "", taskHash: EDITED }))).toBe(false);
    expect(task.drifted(newPayload({ statusHash: "", statusRack: 2 }))).toBe(false);
  });
});
