// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, sleep, TimeRange, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, onTestFinished, vi } from "vitest";
import { z } from "zod";

import { channel } from "@/channel";
import { framer } from "@/framer";
import { MultipleFoundError, NotFoundError, ValidationError } from "@/index";
import { query } from "@/query";
import { rack } from "@/rack";
import { resource } from "@/resource";
import { type status } from "@/status";
import { task } from "@/task";
import {
  createTestClient,
  createTestClientWithPolicy,
  expectLive,
  holdResponses,
  recordResponses,
  spyOnSend,
  waitForStreamLive,
} from "@/testutil";

const client = createTestClient();
const writer = createTestClient();

const statusKeyOf = (owner: resource.ID): status.Key => {
  const held = client.statuses.ownedBy(owner);
  if (held == null) throw new Error(`${resource.idToString(owner)} holds no status`);
  return held.key;
};

// Mirrors the deadline the task client gives an unanswered command.
const COMMAND_DEADLINE = TimeSpan.seconds(10);

describe("Task", async () => {
  const testRack = await client.racks.create({ name: "test" });
  // The Core refuses an alert config that names a status it does not hold.
  await client.statuses.set(
    ["st1", "st-a", "st-b"].map((key) => ({
      key,
      name: key,
      variant: "info" as const,
      message: key,
    })),
  );
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
        details: { running: true, data: { customData: true } },
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
      expect(retrieved.status?.owner).toEqual(task.resourceID(m.key));
      expect(retrieved.status?.details?.running).toBe(true);
    });
  });
  describe("update", () => {
    it("should refuse a task whose key exists and keep the stored task", async () => {
      const m = await testRack.createTask({
        name: "test",
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      // Exclude status, internal, snapshot - these have different input/output types
      const { status: _, internal: __, snapshot: ___, ...taskFields } = m;
      const create = client.tasks.create({ ...taskFields, name: "updated" });
      await expect(create).rejects.toThrow(ValidationError);
      await expect(create).rejects.toThrow(`task ${m.key} already exists`);
      const retrieved = await createTestClient().tasks.retrieve(m.key);
      expect(retrieved.name).toBe("test");
    });
  });
  describe("rename", () => {
    it("should keep a rename that lands before an older fetch by name", async () => {
      const m = await testRack.createTask({
        name: `task-${id.create()}`,
        config: { routingKey: "rk" },
        type: "pagerduty_alert",
      });
      const local = createTestClient();
      await local.connect();
      await waitForStreamLive(local.connection);
      await local.tasks.retrieve(m.key);
      local.statuses.store.delete(statusKeyOf(task.resourceID(m.key)));
      const { responded, release } = holdResponses(local, "/task/retrieve");
      const fetching = local.tasks.retrieve({ name: m.name });
      await responded;
      const name = `task-renamed-${id.create()}`;
      await client.tasks.rename(m.key, name);
      await expect
        .poll(() => expectLive(local.tasks.watch(m.key).getCached()).name)
        .toBe(name);
      release();
      await fetching;
      expect(expectLive(local.tasks.watch(m.key).getCached()).name).toBe(name);
    });
    it("should rename a task and keep its config", async () => {
      const m = await testRack.createTask({
        name: "task-rename",
        config: { routingKey: "rk-rename" },
        type: "pagerduty_alert",
      });
      await client.tasks.rename(m.key, "task-renamed");
      const retrieved = await createTestClient().tasks.retrieve(m.key);
      expect(retrieved.name).toBe("task-renamed");
      expect(retrieved.type).toBe("pagerduty_alert");
      expect(retrieved.config).toEqual(m.config);
    });
    it("should update the cache of another client", async () => {
      const m = await testRack.createTask({
        name: "task-rename-remote",
        config: { routingKey: "rk" },
        type: "pagerduty_alert",
      });
      expect((await client.tasks.retrieve(m.key)).name).toBe("task-rename-remote");
      const name = `task-renamed-${id.create()}`;
      await waitForStreamLive(client.connection);
      await writer.tasks.rename(m.key, name);
      await expect
        .poll(async () => (await client.tasks.retrieve(m.key)).name)
        .toBe(name);
    });
    it("should reject an empty name", async () => {
      const m = await testRack.createTask({
        name: "task-rename-empty",
        config: { routingKey: "rk" },
        type: "pagerduty_alert",
      });
      await expect(client.tasks.rename(m.key, "")).rejects.toThrow();
      expect((await createTestClient().tasks.retrieve(m.key)).name).toBe(
        "task-rename-empty",
      );
    });
  });
  describe("retrieve", () => {
    it("retrieves a task by its name", async () => {
      const created = await testRack.createTask({
        name: `named-task-${id.create()}`,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const retrieved = await client.tasks.retrieve(created.name);
      expect(retrieved.key).toEqual(created.key);
    });

    it("coalesces concurrent single retrieves into one request", async () => {
      const created = [
        await testRack.createTask({
          name: "coalesce-a",
          config: { routingKey: "dog" },
          type: "pagerduty_alert",
        }),
        await testRack.createTask({
          name: "coalesce-b",
          config: { routingKey: "dog" },
          type: "pagerduty_alert",
        }),
      ];
      const local = createTestClient();
      await local.connect();
      const send = spyOnSend(local);
      const res = await Promise.all(
        created.map(async ({ key }) => await local.tasks.retrieve(key)),
      );
      expect(res.map(({ key }) => key)).toEqual(created.map(({ key }) => key));
      expect(
        send.mock.calls.filter(([target]) => target === "/task/retrieve"),
      ).toHaveLength(1);
    });

    it("does not reject concurrent retrieves when a key in the window is missing", async () => {
      const created = await testRack.createTask({
        name: "isolation",
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const local = createTestClient();
      await local.connect();
      const [ok, missing] = await Promise.allSettled([
        local.tasks.retrieve(created.key),
        local.tasks.retrieve(uuid.create()),
      ]);
      expect(ok.status).toEqual("fulfilled");
      expect(missing.status).toEqual("rejected");
      if (missing.status === "rejected")
        expect(NotFoundError.matches(missing.reason)).toBe(true);
    });

    it("should treat a missing key as a non-match", async () => {
      const created = await testRack.createTask({
        name: "present",
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const retrieved = await createTestClient().tasks.retrieve({
        key: [created.key, uuid.create()],
      });
      expect(retrieved.map(({ key }) => key)).toEqual([created.key]);
    });

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

    it("should retrieve tasks by a key list in key order", async () => {
      const t1 = await testRack.createTask({
        name: "t1",
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const t2 = await testRack.createTask({
        name: "t2",
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const retrieved = await client.tasks.retrieve([t2.key, t1.key]);
      expect(retrieved.map(({ key }) => key)).toEqual([t2.key, t1.key]);
    });

    it("should throw a NotFoundError for a missing key in a key list", async () => {
      const t = await testRack.createTask({
        name: "t",
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const missing = uuid.create();
      await expect(client.tasks.retrieve([t.key, missing])).rejects.toThrow(
        new NotFoundError(`task with key ${missing} not found`),
      );
    });

    it("should return one task for a plain key beside other fields", async () => {
      const t = await testRack.createTask({
        name: "t",
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const retrieved: task.Task = await client.tasks.retrieve({
        key: t.key,
        internal: false,
      });
      expect(retrieved.key).toBe(t.key);
      await expect(
        client.tasks.retrieve({ key: t.key, internal: true }),
      ).rejects.toThrow(NotFoundError);
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

    it("should reject a name that two tasks share", async () => {
      const name = `test-${id.create()}`;
      for (let i = 0; i < 2; i++)
        await testRack.createTask({
          name,
          config: { routingKey: "dog" },
          type: "pagerduty_alert",
        });
      await expect(client.tasks.retrieve({ name })).rejects.toThrow(MultipleFoundError);
    });

    it("should name the operator at fault in a where tree", async () => {
      await expect(
        // @ts-expect-error - a name operator takes a string.
        client.tasks.retrieve({ where: { name: { eq: 5 } } }),
      ).rejects.toThrow(/at where\.name\n.*\n\s*✗ "eq": 5 {2}× expected string/);
    });

    it("should name the filter field at fault", async () => {
      await expect(
        // @ts-expect-error - internal takes a boolean.
        client.tasks.retrieve({ internal: "yes" }),
      ).rejects.toThrow('✗ "internal": "yes"  × expected boolean');
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
          key: statusKeyOf(task.resourceID(t.key)),
          owner: task.resourceID(t.key),
          name: "test",
          variant: "success",
          labels: [],
          details: {
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
        const result = await client.tasks.retrieve({ rack: testRack.key });
        expect(result.length).toBeGreaterThanOrEqual(3);
        expect(result.every((t) => t.rack === testRack.key)).toBe(true);
      });

      it("should retrieve tasks by multiple keys", async () => {
        const keysToQuery = testTasks.slice(0, 2).map((t) => t.key);
        const result = await client.tasks.retrieve({ key: keysToQuery });
        expect(result).toHaveLength(2);
        expect(result.map((t) => t.key).sort()).toEqual(keysToQuery.sort());
      });

      it("should retrieve tasks by multiple names", async () => {
        const namesToQuery = ["sensor_task1", "actuator_task1"];
        const result = await client.tasks.retrieve({ name: namesToQuery });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((t) => namesToQuery.includes(t.name))).toBe(true);
      });

      it("should retrieve tasks by types", async () => {
        const result = await client.tasks.retrieve({ type: ["pagerduty_alert"] });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((t) => t.type === "pagerduty_alert")).toBe(true);
      });

      it("should retrieve tasks by multiple types", async () => {
        const typesToQuery = ["pagerduty_alert", "labjack_scan"];
        const result = await client.tasks.retrieve({ type: typesToQuery });
        expect(result.length).toBeGreaterThanOrEqual(4);
        expect(result.every((t) => typesToQuery.includes(t.type))).toBe(true);
      });

      it("should page with a limit and a cursor", async () => {
        const firstPage = await client.tasks.retrieve({
          rack: testRack.key,
          limit: 2,
        });
        expect(firstPage.length).toBeLessThanOrEqual(2);

        if (firstPage.length === 2) {
          const secondPage = await client.tasks.retrieve({
            rack: testRack.key,
            limit: 2,
            after: { key: firstPage[1].key },
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
          type: ["pagerduty_alert"],
          includeStatus: true,
        });
        expect(result.length).toBeGreaterThanOrEqual(1);
        expect(result.every((t) => t.type === "pagerduty_alert")).toBe(true);

        await expect
          .poll(async () => {
            const tasks = await client.tasks.retrieve({
              rack: testRack.key,
              type: ["pagerduty_alert"],
              includeStatus: true,
            });
            return tasks.every((t) => t.status !== undefined);
          })
          .toBe(true);
      });

      it("should handle empty results gracefully", async () => {
        const result = await client.tasks.retrieve({ type: ["nonexistent_type"] });
        expect(result).toEqual([]);
      });

      it("should combine rack and type filters", async () => {
        const result = await client.tasks.retrieve({
          rack: secondRack.key,
          type: ["labjack_scan"],
        });
        expect(result.length).toBeGreaterThanOrEqual(1);
        expect(result.every((t) => t.type === "labjack_scan")).toBe(true);
      });

      it("should combine rack and type in a flat filter", async () => {
        const result = await client.tasks.retrieve({
          rack: secondRack.key,
          type: ["labjack_scan"],
        });
        expect(result.length).toBeGreaterThanOrEqual(1);
        expect(result.every((t) => t.type === "labjack_scan")).toBe(true);
      });

      it("should handle a limit", async () => {
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

        const snapshotOnlyResult = await client.tasks.retrieve({ snapshot: true });
        expect(snapshotOnlyResult.some((t) => t.key === snapshotTask.key)).toBe(true);
        expect(snapshotOnlyResult.every((t) => t.snapshot === true)).toBe(true);

        const regularOnlyResult = await client.tasks.retrieve({ snapshot: false });
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
          type: ["pagerduty_alert"],
          snapshot: false,
        });

        expect(result.some((t) => t.key === task1.key)).toBe(true);
        expect(result.every((t) => t.type === "pagerduty_alert")).toBe(true);
        expect(result.every((t) => t.snapshot === false)).toBe(true);
      });
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

    it("should place a snapshot in a range", async () => {
      const m = await testRack.createTask({
        name: "test",
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const rng = await client.ranges.create({
        name: `snapshot-${id.create()}`,
        timeRange: new TimeRange(TimeStamp.now(), TimeStamp.MAX),
      });
      const snap = await client.tasks.copy(m.key, "Snapshot", true, rng.key);
      expect(snap.range).toEqual(rng.key);
      expect((await snap.snapshottedTo())?.key).toEqual(rng.key);
      const inRange = await client.tasks.retrieve({ ranges: [rng.key] });
      expect(inRange.map(({ key }) => key)).toEqual([snap.key]);
    });
  });

  describe("dispatch", () => {
    const routingKey = (value: string) =>
      task.set({ path: ["config", "routingKey"], value });

    it("should store the actions and settle the cached task on the echo", async () => {
      await waitForStreamLive(client.connection);
      const t = await testRack.createTask({
        name: `dispatch-${id.create()}`,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      expect(await client.tasks.dispatch(t.key, routingKey("cat"))).toBe(true);
      const stored = await createTestClient().tasks.retrieve(t.key);
      expect(stored.config).toMatchObject({ routingKey: "cat" });
      expect(stored.configHash).not.toBe(t.configHash);
      await client.tasks.settled(t.key);
      const cached = await client.tasks.retrieve(t.key);
      expect(cached.config).toEqual(stored.config);
      expect(cached.configHash).toBe(stored.configHash);
    });

    it("should apply list actions to the config", async () => {
      const t = await testRack.createTask({
        name: `dispatch-list-${id.create()}`,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      await client.tasks.dispatch(t.key, [
        task.insert({
          path: ["config", "alerts"],
          value: { key: "a", status: "st-a", errors_critical: true },
        }),
        task.insert({
          path: ["config", "alerts"],
          value: { key: "b", status: "st-b" },
        }),
        task.move({ path: ["config", "alerts"], key: "b" }),
      ]);
      const stored = await createTestClient().tasks.retrieve(t.key);
      expect(stored.config.alerts).toEqual([
        expect.objectContaining({ key: "b", status: "st-b" }),
        expect.objectContaining({
          key: "a",
          status: "st-a",
          errorsCritical: true,
        }),
      ]);
    });

    it("should return false without a send when the task is not cached", async () => {
      const t = await testRack.createTask({
        name: `dispatch-uncached-${id.create()}`,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const fresh = createTestClient();
      const send = spyOnSend(fresh);
      expect(await fresh.tasks.dispatch(t.key, routingKey("cat"))).toBe(false);
      expect(send).not.toHaveBeenCalled();
    });

    it("should adopt the config the Core stores when it fills a default", async () => {
      await waitForStreamLive(client.connection);
      const a = await client.arcs.create({
        name: `dispatch-${id.create()}`,
        type: "automation",
        source: { mode: "text" },
      });
      const t = await testRack.createTask({
        name: `dispatch-default-${id.create()}`,
        config: { arcKey: a.key, performance: "high" },
        type: "arc",
      });
      expect(
        await client.tasks.dispatch(
          t.key,
          task.remove({ path: ["config"], key: "performance" }),
        ),
      ).toBe(true);
      await client.tasks.settled(t.key);
      const stored = await createTestClient().tasks.retrieve(t.key);
      expect(stored.config).toMatchObject({ performance: "auto" });
      const cached = await client.tasks.retrieve(t.key);
      expect(cached.config).toEqual(stored.config);
      expect(cached.configHash).toBe(stored.configHash);
    });

    it("should drop actions the Core refuses from the cached task", async () => {
      const t = await testRack.createTask({
        name: `dispatch-refused-${id.create()}`,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      await expect(
        client.tasks.dispatch(t.key, task.set({ path: ["type"], value: "arc" })),
      ).rejects.toThrow("[type]: the path names no editable field");
      const cached = await client.tasks.retrieve(t.key);
      expect(cached.type).toBe("pagerduty_alert");
      await client.tasks.settled(t.key);
    });
  });

  describe("references", () => {
    it("should clear a deleted status from the cached task's alerts", async () => {
      await waitForStreamLive(client.connection);
      const st = await client.statuses.set({
        key: uuid.create(),
        name: "detached",
        variant: "info",
        message: "detached",
      });
      const t = await testRack.createTask({
        name: `detach-${id.create()}`,
        config: { routingKey: "dog", alerts: [{ key: "a", status: st.key }] },
        type: "pagerduty_alert",
      });
      await client.statuses.delete(st.key);
      const stored = await createTestClient().tasks.retrieve(t.key);
      expect(stored.config.alerts).toEqual([
        expect.objectContaining({ key: "a", status: "" }),
      ]);
      expect(stored.configHash).not.toBe(t.configHash);
      await expect
        .poll(() => expectLive(client.tasks.watch(t.key).getCached()).configHash)
        .toBe(stored.configHash);
      expect(expectLive(client.tasks.watch(t.key).getCached()).config).toEqual(
        stored.config,
      );
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
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
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
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
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
    it("should send a pinned start with the given hash", async () => {
      const t = await testRack.createTask({
        name: "test",
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      const key = await client.tasks.executeCommand({
        task: t.key,
        type: "start",
        configHash: "seen",
        pinned: true,
      });
      await expect
        .poll<Promise<task.Command>>(async () => {
          const fr = await streamer.read();
          const sample = fr.at(-1)[task.COMMAND_CHANNEL_NAME];
          return task.commandZ.parse(sample);
        })
        .toMatchObject({ key, task: t.key, configHash: "seen", pinned: true });
      streamer.close();
    });
    it("should timeout on a synchronously executed command", async () => {
      const t = await testRack.createTask({
        name: "test",
        config: {},
        type: "pagerduty_alert",
      });
      await expect(
        t.executeCommandSync({ type: "test", timeout: TimeSpan.milliseconds(1) }),
      ).rejects.toThrow(new Error("Command to test timed out after 1ms"));
    });
    it("should name the task by key when its name lookup fails", async () => {
      const t = await testRack.createTask({
        name: "test",
        config: {},
        type: "pagerduty_alert",
      });
      await client.tasks.delete(t.key);
      const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
      onTestFinished(() => consoleError.mockRestore());
      await expect(
        client.tasks.executeCommandSync({
          task: t.key,
          type: "test",
          timeout: TimeSpan.milliseconds(1),
          statusDataZ: z.unknown(),
        }),
      ).rejects.toThrow(
        new Error(`Command to task with key ${t.key} timed out after 1ms`),
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

      const retrieved = await client.tasks.retrieve({ key: [t1.key, t2.key], schemas });
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

    it("should return one task for a filter that sets one", async () => {
      const prefix = `one-option-${id.create()}`;
      await testRack.createTask(
        {
          name: `${prefix}-a`,
          type: "pagerduty_alert",
          config: { routingKey: "rk-700", autoStart: false, alerts: [] },
        },
        schemas,
      );
      const retrieved = await client.tasks.retrieve({
        name: { startsWith: prefix },
        one: true,
        schemas,
      });
      expect(retrieved.config.routingKey).toBe("rk-700");
    });

    it("should return one task for a plain key beside other fields", async () => {
      const t = await testRack.createTask(
        {
          name: "one-schema",
          type: "pagerduty_alert",
          config: { routingKey: "rk-600", autoStart: false, alerts: [] },
        },
        schemas,
      );
      const retrieved = await client.tasks.retrieve({
        key: t.key,
        internal: false,
        schemas,
      });
      expect(retrieved.config.routingKey).toBe("rk-600");
      await expect(
        client.tasks.retrieve({ key: t.key, internal: true, schemas }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should reject a malformed key with or without schemas", async () => {
      const schemas = {
        type: z.literal("ni"),
        config: z.object({}),
        statusData: z.unknown(),
      };
      const err = new ValidationError(
        "task params address a single key but do not match the single-query " +
          "schema.\nFailed to parse task single query params (1 issue)\n\n  {\n" +
          '  ✗ "key": "not-a-uuid"  × expected uuid format\n  }',
      );
      await expect(client.tasks.retrieve({ key: "not-a-uuid" })).rejects.toThrow(err);
      await expect(
        client.tasks.retrieve({ key: "not-a-uuid", schemas }),
      ).rejects.toThrow(err);
    });

    it("should retrieve one task by type and rack when one is set", async () => {
      const schemas = {
        type: z.literal("modbus_scan"),
        config: z.object({ rate: z.number() }),
        statusData: z.unknown(),
      };
      const r = await client.racks.create({ name: "type-status" });
      const created = await r.createTask(
        { name: "type-status", type: "modbus_scan", config: { rate: 1 } },
        schemas,
      );
      const retrieved = await client.tasks.retrieve({
        type: "modbus_scan",
        rack: r.key,
        includeStatus: true,
        one: true,
        schemas,
      });
      expect(retrieved.key).toBe(created.key);
    });

    it("should retrieve the tasks of a type and rack as a list", async () => {
      const customSchemas = {
        type: z.literal("modbus_scan"),
        config: z.object({ rate: z.number() }),
        statusData: z.unknown(),
      };
      const r = await client.racks.create({ name: "type-list" });
      const created = await client.tasks.create(
        [42, 43].map((rate) => ({
          name: `type-list-${rate}`,
          type: "modbus_scan" as const,
          rack: r.key,
          config: { rate },
        })),
        customSchemas,
      );
      const params = { type: "modbus_scan", rack: r.key };
      const typed = await client.tasks.retrieve({ ...params, schemas: customSchemas });
      expect(typed.map((t) => t.config.rate).sort()).toEqual([42, 43]);
      const cached = await client.tasks.retrieve(params);
      expect(cached.map((t) => t.key).sort()).toEqual(created.map((t) => t.key).sort());
    });
  });

  // A rack with no Driver goes down within seconds of its creation, and the client
  // reads that state, so tests that need one state or the other must say which.
  const setRackStatus = async (
    key: number,
    variant: status.Variant,
    message: string,
    running: boolean = variant === "success",
  ) => {
    await client.statuses.set({
      owner: rack.resourceID(key),
      name: "Rack Status",
      variant,
      message,
      time: TimeStamp.now(),
      details: { running },
    });
  };

  // A task's creation placeholder echoes back over the status stream at its own pace.
  // Statuses stream in write order, so once a later write by another client lands here,
  // the echo cannot overwrite a status the spec sets after this returns.
  const setRackAliveAfterEcho = async (key: rack.Key) => {
    await waitForStreamLive(client.connection);
    const message = `alive-${id.create()}`;
    await writer.statuses.set({
      owner: rack.resourceID(key),
      name: "Rack Status",
      variant: "success",
      message,
      time: TimeStamp.now(),
      details: { running: true },
    });
    await expect
      .poll(() => client.statuses.ownedBy(rack.resourceID(key))?.message)
      .toBe(message);
  };

  // A Driver writes its statuses to the Core. A status set only in the local store
  // would lose to any refetch, which returns the Core's stored status.
  const writeAndSee = async (stat: task.Status) => {
    await writer.statuses.set(stat);
    await expect
      .poll(() => client.statuses.store.get(stat.key)?.message)
      .toBe(stat.message);
  };

  describe("executeCommandSync", () => {
    // The wait opens its streamer before it sends the command, so once the command
    // lands, the wait sees every status written after it.
    const readCommand = async (
      streamer: framer.Streamer,
      key: task.Key,
    ): Promise<task.Command> => {
      while (true) {
        const fr = await streamer.read();
        const commands = fr.get(task.COMMAND_CHANNEL_NAME).parseJSON(task.commandZ);
        const cmd = commands.find((c) => c.task === key);
        if (cmd != null) return cmd;
      }
    };

    const setDriverNotRunning = async (key: rack.Key) =>
      await writer.statuses.set({
        owner: rack.resourceID(key),
        name: "Rack Status",
        variant: "warning",
        message: "Driver not running",
        description: "The Core has not heard from the Driver.",
        time: TimeStamp.now(),
        details: { running: false },
      });

    it("should end the wait when the Driver of the task goes down", async () => {
      const dying = await client.racks.create({ name: `dying-${id.create()}` });
      const t = await dying.createTask({
        name: `dying-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      onTestFinished(() => streamer.close());
      const wait = t.executeCommandSync({ type: "scan", timeout: TimeSpan.minutes(1) });
      // Handled before the outage, which can reject the wait before its write returns.
      const ended = expect(wait).rejects.toMatchObject({
        message: "Driver not running",
        description: "The Core has not heard from the Driver.",
      });
      await readCommand(streamer, t.key);
      await setDriverNotRunning(dying.key);
      await ended;
    });

    it("should end the wait of a command sent through the client", async () => {
      const dying = await client.racks.create({ name: `dying-${id.create()}` });
      const t = await dying.createTask({
        name: `dying-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      onTestFinished(() => streamer.close());
      const wait = client.tasks.executeCommandSync({
        commands: [{ task: t.key, type: "scan" }],
        timeout: TimeSpan.minutes(1),
        statusDataZ: z.unknown(),
      });
      const ended = expect(wait).rejects.toThrow("Driver not running");
      await readCommand(streamer, t.key);
      await setDriverNotRunning(dying.key);
      await ended;
    });

    it("should end the wait of a batch that holds a deleted task", async () => {
      const dying = await client.racks.create({ name: `dying-${id.create()}` });
      const live = await dying.createTask({
        name: `live-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const gone = await testRack.createTask({
        name: `gone-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      await client.tasks.delete(gone.key);
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      onTestFinished(() => streamer.close());
      const wait = client.tasks.executeCommandSync({
        commands: [
          { task: gone.key, type: "scan" },
          { task: live.key, type: "scan" },
        ],
        timeout: TimeSpan.minutes(1),
        statusDataZ: z.unknown(),
      });
      const ended = expect(wait).rejects.toThrow("Driver not running");
      await readCommand(streamer, live.key);
      await setDriverNotRunning(dying.key);
      await ended;
    });

    it.each(["success", "warning"] as const)(
      "should keep waiting through a %s status from a running Driver",
      async (variant) => {
        const alive = await client.racks.create({ name: `alive-${id.create()}` });
        const t = await alive.createTask({
          name: `alive-${id.create()}`,
          config: {},
          type: "pagerduty_alert",
        });
        const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
        onTestFinished(() => streamer.close());
        const wait = t.executeCommandSync({
          type: "scan",
          timeout: TimeSpan.minutes(1),
        });
        const cmd = await readCommand(streamer, t.key);
        await writer.statuses.set({
          owner: rack.resourceID(alive.key),
          name: "Rack Status",
          variant,
          message: "Driver is running",
          time: TimeStamp.now(),
          details: { running: true },
        });
        const reply = {
          owner: task.resourceID(t.key),
          name: t.name,
          variant: "success" as const,
          message: "Scan complete",
          description: "",
          time: TimeStamp.now(),
          details: {
            running: true,
            cmd: cmd.key,
            configHash: "",
            rack: alive.key,
            data: {},
          },
        };
        await writer.statuses.set(reply);
        expect(await wait).toMatchObject({ message: "Scan complete" });
      },
    );

    const setReply = async (
      t: task.Task,
      cmd: task.Command,
      variant: "loading" | "success",
      message: string,
    ): Promise<void> => {
      await writer.statuses.set({
        owner: task.resourceID(t.key),
        name: t.name,
        variant,
        message,
        description: "",
        time: TimeStamp.now(),
        details: {
          running: variant === "success",
          cmd: cmd.key,
          configHash: "",
          rack: testRack.key,
          data: {},
        },
      });
    };

    it("should wait past the timeout for the final status after a loading reply", async () => {
      const t = await testRack.createTask({
        name: `slow-start-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      onTestFinished(() => streamer.close());
      const timeout = TimeSpan.milliseconds(500);
      const wait = t.executeCommandSync({ type: "start", timeout });
      const cmd = await readCommand(streamer, t.key);
      await setReply(t, cmd, "loading", "Starting task");
      await sleep.sleep(timeout.mult(2));
      await setReply(t, cmd, "success", "Started task");
      expect(await wait).toMatchObject({ variant: "success", message: "Started task" });
    });

    it("should give up two minutes after a loading reply with no final status", async () => {
      const t = await testRack.createTask({
        name: `lost-start-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      onTestFinished(() => streamer.close());
      const wait = t.executeCommandSync({
        type: "start",
        timeout: TimeSpan.minutes(1),
      });
      let settled = false;
      const markSettled = (): void => {
        settled = true;
      };
      wait.then(markSettled, markSettled);
      const cmd = await readCommand(streamer, t.key);
      vi.useFakeTimers({
        toFake: ["setTimeout", "clearTimeout"],
        shouldClearNativeTimers: true,
      });
      onTestFinished(() => {
        vi.useRealTimers();
      });
      await setReply(t, cmd, "loading", "Starting task");
      await expect
        .poll(() => client.statuses.ownedBy(task.resourceID(t.key))?.message)
        .toBe("Starting task");
      // The wait gets the reply at about the time the store does. Steps shorter than
      // the 1 minute acknowledgment limit cover the gap.
      await vi.waitFor(async () => {
        await vi.advanceTimersByTimeAsync(TimeSpan.seconds(30).milliseconds);
        expect(settled).toBe(true);
      });
      await expect(wait).rejects.toThrow(
        new Error(`Command to ${t.name} timed out after 2min`),
      );
    });
  });

  describe("onChange", () => {
    it("merges the status a task owns into a subscribed single query", async () => {
      const t = await testRack.createTask({
        name: `status-details-single-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const params = { key: t.key };
      const off = client.tasks.watch(params).onChange(vi.fn());
      try {
        await client.tasks.retrieve(params);
        await client.statuses.set({
          owner: task.resourceID(t.key),
          name: "Task Status",
          variant: "error",
          message: "task failed",
          time: TimeStamp.now(),
          details: { running: false, cmd: "", data: {} },
        });
        await expect
          .poll(() => {
            const cached = client.tasks.watch(params).getCached();
            if (!query.isLive(cached)) return undefined;
            return cached.status?.message;
          })
          .toBe("task failed");
      } finally {
        off();
      }
    });

    it("merges the status a task owns into a subscribed request query", async () => {
      const t = await testRack.createTask({
        name: `status-details-request-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const list = client.tasks.watch({ where: { key: { in: [t.key] } } });
      const off = list.onChange(vi.fn());
      try {
        await list.retrieve();
        await client.statuses.set({
          owner: task.resourceID(t.key),
          name: "Task Status",
          variant: "warning",
          message: "task degraded",
          time: TimeStamp.now(),
          details: { running: true, cmd: "", data: {} },
        });
        await expect
          .poll(() => {
            const records = list.getCached()?.records;
            return records?.find((v) => v.key === t.key)?.status?.message;
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
      // The optimistic status is transient: a real one from the cluster replaces it, so
      // it has to be captured as it is delivered rather than read back later.
      const loading: Array<NonNullable<task.Task["status"]>> = [];
      const off = client.tasks.watch(params).onChange((cached) => {
        if (!query.isLive(cached)) return;
        const { status } = cached;
        if (status?.variant === "loading") loading.push(status);
      });
      try {
        await client.tasks.retrieve(params);
        await client.statuses.set({
          owner: task.resourceID(t.key),
          name: "Task Status",
          variant: "success",
          message: "task started",
          time: TimeStamp.now(),
          details: {
            running: true,
            cmd: "",
            configHash: "deadbeef",
            rack: testRack.key,
            data: {},
          },
        });
        await expect
          .poll(() => {
            const cached = client.tasks.watch(params).getCached();
            if (!query.isLive(cached)) return undefined;
            return cached.status?.details.configHash;
          })
          .toBe("deadbeef");
        await setRackStatus(testRack.key, "success", "Driver is running");
        await client.tasks.executeCommand({ task: t.key, type: "stop" });
        await expect.poll(() => loading.length).toBeGreaterThan(0);
        expect(loading[0].name).toBe(t.name);
        expect(loading[0].message).toBe("Sending stop command");
        expect(loading[0].details.configHash).toBe("deadbeef");
        expect(loading[0].details.rack).toBe(testRack.key);
      } finally {
        off();
      }
    });

    it("writes no optimistic status for a task whose status the cache does not hold", async () => {
      const create = async (): Promise<task.Task> =>
        await testRack.createTask({
          name: `unheld-${id.create()}`,
          config: {},
          type: "pagerduty_alert",
        });
      const held = await create();
      const unheld = await create();
      const heldOwner = task.resourceID(held.key);
      const unheldOwner = task.resourceID(unheld.key);
      const local = createTestClient();
      await local.connect();
      await waitForStreamLive(local.connection);
      await local.tasks.retrieve({ key: [held.key, unheld.key] });
      // A reconnect empties the status table while the task stays known.
      local.statuses.store.delete(statusKeyOf(unheldOwner));
      const heldKey = statusKeyOf(heldOwner);
      // One frame carries both commands, so the client handles them together.
      await client.tasks.executeCommand({
        commands: [
          { task: held.key, type: "stop" },
          { task: unheld.key, type: "stop" },
        ],
      });
      // A rack with no Driver goes down at its own pace, so the stand-in is a loading
      // status or the rack's warning.
      const standIn = (): boolean =>
        ["loading", "warning"].includes(
          local.statuses.ownedBy(heldOwner)?.variant ?? "",
        );
      await expect.poll(standIn).toBe(true);
      expect(local.statuses.ownedBy(heldOwner)?.key).toEqual(heldKey);
      expect(local.statuses.ownedBy(unheldOwner)).toBeUndefined();
    });

    it("stands the rack's problem in for a wait its Driver cannot answer", async () => {
      const down = await client.racks.create({ name: `down-${id.create()}` });
      const t = await down.createTask({
        name: `down-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      await client.tasks.retrieve({ key: t.key });
      await setRackStatus(down.key, "warning", "no Driver here");
      // The store is watched directly so the assertion reads the write the command
      // makes. A subscribed query would let a refetch restore the stored status
      // first, and a poll would only ever see whichever landed last.
      let seeWrite = (_: status.Status): void => {};
      const written = new Promise<status.Status>((resolve) => (seeWrite = resolve));
      const off = client.statuses.store.subscribe((event) => {
        // The creation-time "has not been deployed" placeholder echoes back over the
        // status stream at its own pace, so a disabled event is never the command's.
        if (event.variant === "set" && event.value.variant !== "disabled")
          seeWrite(event.value);
      }, key);
      try {
        await client.tasks.executeCommand({ task: t.key, type: "start" });
        expect(await written).toMatchObject({
          variant: "warning",
          message: "no Driver here",
          details: { running: false },
        });
      } finally {
        off();
      }
    });

    it("sends a loading status for a rack whose running Driver warns", async () => {
      const warned = await client.racks.create({ name: `warned-${id.create()}` });
      const t = await warned.createTask({
        name: `warned-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      await client.tasks.retrieve({ key: t.key });
      await setRackStatus(warned.key, "warning", "Driver is degraded", true);
      let seeWrite = (_: status.Status): void => {};
      const written = new Promise<status.Status>((resolve) => (seeWrite = resolve));
      const off = client.statuses.store.subscribe((event) => {
        if (event.variant === "set" && event.value.variant !== "disabled")
          seeWrite(event.value);
      }, key);
      try {
        await client.tasks.executeCommand({ task: t.key, type: "start" });
        expect(await written).toMatchObject({
          variant: "loading",
          message: "Sending start command",
        });
      } finally {
        off();
      }
    });

    it("moves a stand-in to the next problem of its rack", async () => {
      const dying = await client.racks.create({ name: `dying-${id.create()}` });
      const t = await dying.createTask({
        name: `follow-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      await client.tasks.retrieve({ key: t.key });
      await setRackAliveAfterEcho(dying.key);
      await setRackStatus(dying.key, "warning", "Rack status unknown");
      await expect
        .poll(() => client.statuses.ownedBy(rack.resourceID(dying.key))?.message)
        .toBe("Rack status unknown");
      await client.tasks.executeCommand({ task: t.key, type: "start" });
      await expect
        .poll(() => client.statuses.store.get(key)?.message)
        .toBe("Rack status unknown");
      await setRackStatus(dying.key, "warning", "Driver not running");
      await expect
        .poll(() => client.statuses.store.get(key)?.message)
        .toBe("Driver not running");
    });

    it("stands the rack's problem in once its Driver goes down during a wait", async () => {
      const dying = await client.racks.create({ name: `dying-${id.create()}` });
      const t = await dying.createTask({
        name: `dying-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      let seeLoading = (): void => {};
      const loading = new Promise<void>((resolve) => (seeLoading = resolve));
      const off = client.statuses.store.subscribe((event) => {
        if (event.variant === "set" && event.value.variant === "loading") seeLoading();
      }, key);
      try {
        await client.tasks.retrieve({ key: t.key });
        await setRackAliveAfterEcho(dying.key);
        vi.useFakeTimers({
          toFake: ["setTimeout", "clearTimeout"],
          shouldAdvanceTime: true,
        });
        await client.tasks.executeCommand({ task: t.key, type: "start" });
        await loading;
        await setRackStatus(dying.key, "warning", "no Driver here");
        await expect
          .poll(() => client.statuses.store.get(key)?.message)
          .toBe("no Driver here");
        await vi.advanceTimersByTimeAsync(COMMAND_DEADLINE.milliseconds);
        expect(client.statuses.store.get(key)).toMatchObject({
          variant: "warning",
          message: "no Driver here",
          details: { running: false },
        });
      } finally {
        vi.useRealTimers();
        off();
      }
    });

    it("keeps a reply stored in the same batch as its rack's outage", async () => {
      const dying = await client.racks.create({ name: `dying-${id.create()}` });
      const t = await dying.createTask({
        name: `batched-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      let seeLoading = (): void => {};
      const loading = new Promise<void>((resolve) => (seeLoading = resolve));
      const off = client.statuses.store.subscribe((event) => {
        if (event.variant === "set" && event.value.variant === "loading") seeLoading();
      }, key);
      try {
        await client.tasks.retrieve({ key: t.key });
        await setRackAliveAfterEcho(dying.key);
        const cmdKey = await client.tasks.executeCommand({
          task: t.key,
          type: "start",
        });
        await loading;
        const outage: rack.Status = {
          key: statusKeyOf(rack.resourceID(dying.key)),
          owner: rack.resourceID(dying.key),
          name: "Rack Status",
          variant: "warning",
          message: "no Driver here",
          description: "",
          time: TimeStamp.now(),
          labels: [],
          details: { running: false },
        };
        const reply: task.Status = {
          key,
          owner: task.resourceID(t.key),
          labels: [],
          name: "Task Status",
          variant: "success",
          message: "Started task",
          description: "",
          time: TimeStamp.now(),
          details: {
            running: true,
            cmd: cmdKey,
            configHash: "",
            rack: dying.key,
            data: {},
          },
        };
        // The outage's event is delivered first, after both values are stored.
        client.statuses.store.batch(() => {
          client.statuses.store.set(outage);
          client.statuses.store.set(reply);
        });
        expect(client.statuses.store.get(key)).toMatchObject({
          variant: "success",
          message: "Started task",
        });
      } finally {
        off();
      }
    });

    it("gives up on a command no Driver answers", async () => {
      const alive = await client.racks.create({ name: `alive-${id.create()}` });
      const t = await alive.createTask({
        name: `deadline-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      // The status store is watched directly, leaving no subscribed query behind:
      // winding the clock forward would otherwise reach the cache streamer's
      // reconcile, whose refetch replaces the very status under test.
      let seeLoading = (): void => {};
      const loading = new Promise<void>((resolve) => (seeLoading = resolve));
      const off = client.statuses.store.subscribe((event) => {
        if (event.variant === "set" && event.value.variant === "loading") seeLoading();
      }, key);
      try {
        await client.tasks.retrieve({ key: t.key });
        await setRackAliveAfterEcho(alive.key);
        vi.useFakeTimers({
          toFake: ["setTimeout", "clearTimeout"],
          shouldAdvanceTime: true,
        });
        const placeholder = client.statuses.store.get(key);
        if (placeholder == null)
          throw new Error("no creation placeholder in the store");
        expect(placeholder.variant).toBe("disabled");
        await client.tasks.executeCommand({ task: t.key, type: "start" });
        await loading;
        // The creation placeholder echoes back over the stream at its own pace; a stale
        // echo landing after the command must not cancel the deadline.
        client.statuses.store.set(key, { ...placeholder });
        await vi.advanceTimersByTimeAsync(COMMAND_DEADLINE.milliseconds);
        expect(client.statuses.store.get(key)).toMatchObject({
          variant: "warning",
          message: "No response to the start command",
          details: { running: false },
        });
      } finally {
        vi.useRealTimers();
        off();
      }
    });

    it("hands the rack's stand-in reply to write handlers", async () => {
      const down = await client.racks.create({ name: `down-${id.create()}` });
      const t = await down.createTask({
        name: `down-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      await client.tasks.retrieve({ key: t.key });
      await setRackStatus(down.key, "warning", "no Driver here");
      let seeWrite = (_: status.Status): void => {};
      const written = new Promise<status.Status>((resolve) => (seeWrite = resolve));
      const off = client.statuses.onWrite((s) => {
        if (s.key === key && s.variant === "warning") seeWrite(s);
      });
      try {
        const cmd = await client.tasks.executeCommand({ task: t.key, type: "start" });
        expect(await written).toMatchObject({
          message: "no Driver here",
          details: { cmd, running: false },
        });
      } finally {
        off();
      }
    });

    it("hands the loading status of a command to write handlers", async () => {
      const alive = await client.racks.create({ name: `alive-${id.create()}` });
      const t = await alive.createTask({
        name: `loading-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      let seeWrite = (_: status.Status): void => {};
      const written = new Promise<status.Status>((resolve) => (seeWrite = resolve));
      const off = client.statuses.onWrite((s) => {
        if (s.key === key && s.variant === "loading") seeWrite(s);
      });
      try {
        await client.tasks.retrieve({ key: t.key });
        await setRackAliveAfterEcho(alive.key);
        await client.tasks.executeCommand({ task: t.key, type: "start" });
        expect(await written).toMatchObject({ message: "Sending start command" });
      } finally {
        off();
      }
    });

    it("hands the no-response warning to write handlers as the reply", async () => {
      const alive = await client.racks.create({ name: `alive-${id.create()}` });
      const t = await alive.createTask({
        name: `deadline-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      let seeLoading = (): void => {};
      const loading = new Promise<void>((resolve) => (seeLoading = resolve));
      const offLoading = client.statuses.store.subscribe((event) => {
        if (event.variant === "set" && event.value.variant === "loading") seeLoading();
      }, key);
      const written: status.Status[] = [];
      const offWrite = client.statuses.onWrite((s) => {
        if (s.key === key) written.push(s);
      });
      try {
        await client.tasks.retrieve({ key: t.key });
        await setRackAliveAfterEcho(alive.key);
        vi.useFakeTimers({
          toFake: ["setTimeout", "clearTimeout"],
          shouldAdvanceTime: true,
        });
        const cmd = await client.tasks.executeCommand({ task: t.key, type: "start" });
        await loading;
        await vi.advanceTimersByTimeAsync(COMMAND_DEADLINE.milliseconds);
        expect(written.at(-1)).toMatchObject({
          variant: "warning",
          message: "No response to the start command",
          details: { cmd, running: false },
        });
      } finally {
        vi.useRealTimers();
        offLoading();
        offWrite();
      }
    });

    it("keeps a reply stamped by a Driver clock behind this client's", async () => {
      const alive = await client.racks.create({ name: `alive-${id.create()}` });
      const t = await alive.createTask({
        name: `skew-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      let seeLoading = (): void => {};
      const loading = new Promise<void>((resolve) => (seeLoading = resolve));
      const off = client.statuses.store.subscribe((event) => {
        if (event.variant === "set" && event.value.variant === "loading") seeLoading();
      }, key);
      try {
        await client.tasks.retrieve({ key: t.key });
        await setRackAliveAfterEcho(alive.key);
        vi.useFakeTimers({
          toFake: ["setTimeout", "clearTimeout"],
          shouldAdvanceTime: true,
        });
        const cmdKey = await client.tasks.executeCommand({
          task: t.key,
          type: "start",
        });
        await loading;
        // The Driver stamps its reply with its own clock, which here runs a minute
        // behind. The reply names the command, so it answers it all the same.
        const reply: task.Status = {
          key,
          owner: task.resourceID(t.key),
          labels: [],
          name: "Task Status",
          variant: "success",
          message: "Started task",
          description: "",
          time: TimeStamp.now().sub(TimeSpan.minutes(1)),
          details: {
            running: true,
            cmd: cmdKey,
            configHash: "",
            rack: alive.key,
            data: {},
          },
        };
        await writeAndSee(reply);
        await vi.advanceTimersByTimeAsync(COMMAND_DEADLINE.milliseconds);
        expect(client.statuses.store.get(key)).toMatchObject({
          variant: "success",
          message: "Started task",
          details: { running: true, cmd: cmdKey },
        });
      } finally {
        vi.useRealTimers();
        off();
      }
    });

    it("keeps a status that follows the reply before the deadline", async () => {
      const alive = await client.racks.create({ name: `alive-${id.create()}` });
      const t = await alive.createTask({
        name: `follow-up-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      let seeLoading = (): void => {};
      const loading = new Promise<void>((resolve) => (seeLoading = resolve));
      const off = client.statuses.store.subscribe((event) => {
        if (event.variant === "set" && event.value.variant === "loading") seeLoading();
      }, key);
      try {
        await client.tasks.retrieve({ key: t.key });
        await setRackAliveAfterEcho(alive.key);
        vi.useFakeTimers({
          toFake: ["setTimeout", "clearTimeout"],
          shouldAdvanceTime: true,
        });
        const cmdKey = await client.tasks.executeCommand({
          task: t.key,
          type: "start",
        });
        await loading;
        const details = {
          running: true,
          cmd: cmdKey,
          configHash: "",
          rack: alive.key,
          data: {},
        };
        const reply: task.Status = {
          key,
          owner: task.resourceID(t.key),
          labels: [],
          name: "Task Status",
          variant: "success",
          message: "Started task",
          description: "",
          time: TimeStamp.now(),
          details,
        };
        await writeAndSee(reply);
        // The Driver clears cmd on the statuses that follow its reply, so a hardware
        // warning before the deadline carries none. The reply already answered.
        const warning: task.Status = {
          ...reply,
          variant: "warning",
          message: "Sensor out of range",
          time: TimeStamp.now(),
          details: { ...details, cmd: "" },
        };
        await writeAndSee(warning);
        await vi.advanceTimersByTimeAsync(COMMAND_DEADLINE.milliseconds);
        expect(client.statuses.store.get(key)).toMatchObject({
          variant: "warning",
          message: "Sensor out of range",
          details: { running: true, cmd: "" },
        });
      } finally {
        vi.useRealTimers();
        off();
      }
    });

    it("stands the rack's problem in after the Driver acknowledges a start", async () => {
      const dying = await client.racks.create({ name: `dying-${id.create()}` });
      const t = await dying.createTask({
        name: `acknowledged-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      let seeLoading = (): void => {};
      const loading = new Promise<void>((resolve) => (seeLoading = resolve));
      const off = client.statuses.store.subscribe((event) => {
        if (event.variant === "set" && event.value.variant === "loading") seeLoading();
      }, key);
      try {
        await client.tasks.retrieve({ key: t.key });
        await setRackAliveAfterEcho(dying.key);
        vi.useFakeTimers({
          toFake: ["setTimeout", "clearTimeout"],
          shouldAdvanceTime: true,
        });
        const cmdKey = await client.tasks.executeCommand({
          task: t.key,
          type: "start",
        });
        await loading;
        await writeAndSee({
          key,
          owner: task.resourceID(t.key),
          labels: [],
          name: "Task Status",
          variant: "loading",
          message: "Starting task",
          description: "",
          time: TimeStamp.now(),
          details: {
            running: false,
            cmd: cmdKey,
            configHash: "",
            rack: dying.key,
            data: {},
          },
        });
        await vi.advanceTimersByTimeAsync(COMMAND_DEADLINE.milliseconds);
        expect(client.statuses.store.get(key)).toMatchObject({
          message: "Starting task",
        });
        await setRackStatus(dying.key, "warning", "no Driver here");
        await expect
          .poll(() => client.statuses.store.get(key)?.message)
          .toBe("no Driver here");
      } finally {
        vi.useRealTimers();
        off();
      }
    });

    it("stops standing the rack's problem in two minutes after a loading reply", async () => {
      const dying = await client.racks.create({ name: `dying-${id.create()}` });
      const t = await dying.createTask({
        name: `lost-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      let seeLoading = (): void => {};
      const loading = new Promise<void>((resolve) => (seeLoading = resolve));
      const off = client.statuses.store.subscribe((event) => {
        if (event.variant === "set" && event.value.variant === "loading") seeLoading();
      }, key);
      try {
        await client.tasks.retrieve({ key: t.key });
        await setRackAliveAfterEcho(dying.key);
        vi.useFakeTimers({
          toFake: ["setTimeout", "clearTimeout"],
          shouldAdvanceTime: true,
        });
        const cmdKey = await client.tasks.executeCommand({
          task: t.key,
          type: "start",
        });
        await loading;
        await writeAndSee({
          key,
          owner: task.resourceID(t.key),
          labels: [],
          name: "Task Status",
          variant: "loading",
          message: "Starting task",
          description: "",
          time: TimeStamp.now(),
          details: {
            running: false,
            cmd: cmdKey,
            configHash: "",
            rack: dying.key,
            data: {},
          },
        });
        await vi.advanceTimersByTimeAsync(TimeSpan.minutes(2).milliseconds);
        await setRackStatus(dying.key, "warning", "no Driver here");
        await expect
          .poll(() => client.statuses.ownedBy(rack.resourceID(dying.key))?.message)
          .toBe("no Driver here");
        expect(client.statuses.store.get(key)?.message).toBe("Starting task");
      } finally {
        vi.useRealTimers();
        off();
      }
    });

    it("keeps a status that streams in during a fetch, whatever its time", async () => {
      const t = await testRack.createTask({
        name: `fence-${id.create()}`,
        config: {},
        type: "pagerduty_alert",
      });
      const key = statusKeyOf(task.resourceID(t.key));
      const local = createTestClient();
      await local.connect();
      await waitForStreamLive(local.connection);
      const { responded, release } = holdResponses(local, "/task/retrieve");
      const fetching = local.tasks.retrieve({ key: t.key });
      await responded;
      // Stamped by a Driver clock a minute behind the Core's, so only arrival order
      // makes it the newer status.
      const streamed: task.Status = {
        key,
        owner: task.resourceID(t.key),
        labels: [],
        name: "Task Status",
        variant: "success",
        message: "Started task",
        description: "",
        time: TimeStamp.now().sub(TimeSpan.minutes(1)),
        details: {
          running: true,
          cmd: "",
          configHash: "",
          rack: testRack.key,
          data: {},
        },
      };
      await writer.statuses.set(streamed);
      await expect
        .poll(() => local.statuses.store.get(key)?.message)
        .toBe(streamed.message);
      release();
      await fetching;
      expect(local.statuses.store.get(key)).toMatchObject({
        variant: "success",
        message: "Started task",
      });
    });

    it("applies another client's config set to a cached task", async () => {
      const t = await testRack.createTask({
        name: `set-config-${id.create()}`,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const remote = createTestClient();
      // Actions are the only thing that refreshes a reading client's cache, so a write
      // before its stream goes live is never seen.
      await waitForStreamLive(remote.connection);
      const params = { key: t.key };
      const off = remote.tasks.watch(params).onChange(vi.fn());
      try {
        expect((await remote.tasks.retrieve(params)).config).toMatchObject({
          routingKey: "dog",
        });
        expect(
          await client.tasks.dispatch(
            t.key,
            task.set({
              path: ["config"],
              value: { ...(t.payload.config as object), routingKey: "cat" },
            }),
          ),
        ).toBe(true);
        await expect
          .poll(() => {
            const cached = remote.tasks.watch(params).getCached();
            return query.isLive(cached) ? cached.config : undefined;
          })
          .toMatchObject({ routingKey: "cat" });
      } finally {
        off();
      }
    });

    it("applies another client's dispatch to a cached task with its hash", async () => {
      const remote = createTestClient();
      await waitForStreamLive(remote.connection);
      const t = await testRack.createTask({
        name: `dispatch-remote-${id.create()}`,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const params = { key: t.key };
      const off = remote.tasks.watch(params).onChange(vi.fn());
      try {
        await remote.tasks.retrieve(params);
        expect(
          await client.tasks.dispatch(
            t.key,
            task.set({ path: ["config", "routingKey"], value: "cat" }),
          ),
        ).toBe(true);
        const stored = await createTestClient().tasks.retrieve(t.key);
        await expect
          .poll(() => {
            const cached = remote.tasks.watch(params).getCached();
            if (!query.isLive(cached)) return undefined;
            return { config: cached.config, configHash: cached.configHash };
          })
          .toEqual({ config: stored.config, configHash: stored.configHash });
      } finally {
        off();
      }
    });

    it("ingests an uncached task moved onto a subscribed rack", async () => {
      const remote = createTestClient();
      await waitForStreamLive(remote.connection);
      const source = await client.racks.create({ name: `set-move-src-${id.create()}` });
      const dest = await client.racks.create({ name: `set-move-dest-${id.create()}` });
      const t = await source.createTask({
        name: `set-move-${id.create()}`,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const list = remote.tasks.watch({ where: { rack: { eq: dest.key } } });
      const off = list.onChange(vi.fn());
      try {
        expect((await list.retrieve()).records).toHaveLength(0);
        expect(
          await client.tasks.dispatch(
            t.key,
            task.set({ path: ["rack"], value: dest.key }),
          ),
        ).toBe(true);
        await expect
          .poll(() => {
            const records = list.getCached()?.records;
            return records?.find((v) => v.key === t.key)?.config;
          })
          .toMatchObject({ routingKey: "dog" });
      } finally {
        off();
      }
    });

    it("takes a change to a task the user cannot read with no fetch or error", async () => {
      const readable = await testRack.createTask({
        name: `set-readable-${id.create()}`,
        config: { routingKey: "dog" },
        type: "pagerduty_alert",
      });
      const denied = await testRack.createTask({
        name: `set-denied-${id.create()}`,
        config: { routingKey: "cat" },
        type: "pagerduty_alert",
      });
      const onError = vi.fn();
      const remote = await createTestClientWithPolicy(
        client,
        {
          name: "set-readable",
          objects: [
            channel.TYPE_RESOURCE_ID,
            framer.TYPE_RESOURCE_ID,
            task.resourceID(readable.key),
          ],
          actions: ["retrieve"],
        },
        onError,
      );
      await waitForStreamLive(remote.connection);
      const params = { key: readable.key };
      const off = remote.tasks.watch(params).onChange(vi.fn());
      try {
        await remote.tasks.retrieve(params);
        const fetches = recordResponses(remote, "/task/retrieve");
        await client.tasks.rename(denied.key, `${denied.name}-renamed`);
        // Both changes stream on one channel, so the readable rename lands last.
        const name = `set-readable-renamed-${id.create()}`;
        await client.tasks.rename(readable.key, name);
        await expect
          .poll(() => {
            const cached = remote.tasks.watch(params).getCached();
            return query.isLive(cached) ? cached.name : undefined;
          })
          .toBe(name);
        expect(fetches).toHaveLength(0);
        expect(onError).not.toHaveBeenCalled();
        expect(remote.tasks.watch({ key: denied.key }).getCached()).toBeUndefined();
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
            details: { running, configHash: statusHash, rack: statusRack },
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

describe("status composition", () => {
  it("should compose without a predicate scan of the status table", async () => {
    const testRack = await client.racks.create({ name: "status-scan-pin" });
    const t = await testRack.createTask({
      name: "test",
      config: { routingKey: "dog" },
      type: "pagerduty_alert",
    });
    const communicatedStatus: task.Status = {
      key: statusKeyOf(task.resourceID(t.key)),
      owner: task.resourceID(t.key),
      name: "test",
      variant: "success",
      labels: [],
      details: {
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
    const spy = vi.spyOn(client.statuses.store, "get");
    onTestFinished(() => spy.mockRestore());
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
    for (const [arg] of spy.mock.calls) expect(typeof arg).not.toBe("function");
  });
});
