// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { HTTPClient, type UnaryClient, WebSocketClient } from "@synnaxlabs/freighter";
import {
  binary,
  type record,
  Series,
  TimeSpan,
  TimeStamp,
  url,
  uuid,
} from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";
import { z } from "zod";

import { framer } from "@/framer";
import { query } from "@/query";
import { status } from "@/status";
import { task } from "@/task";
import { PushStreamer, TEST_CORE_ADDRESS } from "@/testutil";

// Every request fails, so the cache holds only what the stream delivers.
const unary: UnaryClient = {
  use: () => {},
  send: async (target: string) => {
    throw new Error(`unexpected request to ${target}`);
  },
};

const createStreamedClient = async (recordCap?: number) => {
  const stream = new PushStreamer();
  const cache = new query.Cache({
    openStreamer: async () => stream,
    breaker: { baseInterval: TimeSpan.milliseconds(1) },
    onError: () => {},
    recordCap,
  });
  onTestFinished(async () => await cache.close());
  const coreURL = new url.URL(TEST_CORE_ADDRESS);
  const frames = new framer.Client({
    stream: new WebSocketClient(coreURL, binary.JSON_CODEC),
    unary,
    file: new HTTPClient(coreURL, binary.JSON_CODEC),
    retrieveChannels: async () => [],
  });
  const statuses = new status.Client({ unary, cache });
  const tasks = new task.Client({ unary, framer: frames, cache, statuses });
  await cache.ensureStreaming();
  return { stream, tasks, statuses };
};

const STATUS_KEY = "5c1e7a0b-2d3f-4e6a-8b9c-0d1e2f3a4b5c";

const createStatus = (
  key: task.Key,
  message: string,
  time: TimeStamp,
  statusKey: string = STATUS_KEY,
) => ({
  key: statusKey,
  name: "Task status",
  variant: "warning" as const,
  message,
  time,
  labels: [],
  owner: task.resourceID(key),
  details: { running: false, cmd: "", data: {} },
});

describe("task.Client", () => {
  describe("a task another client creates", () => {
    it("should hold the status its create carries before the status streams in", async () => {
      const { stream, tasks } = await createStreamedClient();
      const key = "0a7d3c2e-6f1b-4c5d-9e8f-1a2b3c4d5e6f";
      const cachedMessage = () => {
        const cached = tasks.watch(key).getCached();
        return query.isLive(cached) ? cached.status?.message : undefined;
      };
      stream.push(
        new framer.Frame({
          [task.ACTION_CHANNEL_NAME]: new Series([
            {
              key,
              dispatchKey: "remote-dispatch",
              seq: 1,
              actions: [
                task.create({
                  task: {
                    key,
                    name: "created remotely",
                    type: "test",
                    config: {},
                    status: createStatus(key, "Task not deployed", TimeStamp.now()),
                  },
                }),
              ],
            },
          ]),
        }),
      );
      await expect.poll(() => query.isLive(tasks.watch(key).getCached())).toBe(true);
      expect(cachedMessage()).toBe("Task not deployed");
      stream.push(
        new framer.Frame({
          [status.SET_CHANNEL_NAME]: new Series([
            createStatus(key, "Task configured", TimeStamp.now().add(TimeSpan.SECOND)),
          ]),
        }),
      );
      await expect.poll(cachedMessage).toBe("Task configured");
    });
  });

  describe("parse", () => {
    const schemas = {
      type: z.literal("pagerduty_alert"),
      config: z.object({
        routingKey: z.string(),
        severity: z.string().default("high"),
      }),
      statusData: z.any().optional(),
    };
    const KEY = "3f6b2a1c-7d8e-4f90-a1b2-c3d4e5f6a7b8";
    const untyped = (tasks: task.Client, config: record.Unknown): task.Task =>
      tasks.sugar({
        key: KEY,
        rack: 0,
        name: "alert",
        type: "pagerduty_alert",
        config,
        configHash: "",
        internal: false,
      });

    it("should fill schema defaults on an untyped task", async () => {
      const { tasks } = await createStreamedClient();
      const parsed = tasks.parse(untyped(tasks, { routingKey: "abc" }), schemas);
      expect(query.isLive(parsed) && parsed.config).toEqual({
        routingKey: "abc",
        severity: "high",
      });
    });

    it("should return the same instance for the same task and schemas", async () => {
      const { tasks } = await createStreamedClient();
      const cached = untyped(tasks, { routingKey: "abc" });
      expect(tasks.parse(cached, schemas)).toBe(tasks.parse(cached, schemas));
    });

    it("should return undefined for a task that fails to parse", async () => {
      const { tasks } = await createStreamedClient();
      expect(tasks.parse(untyped(tasks, { routingKey: 1 }), schemas)).toBeUndefined();
    });

    it("should parse the corpse of a deleted task into one stable wrapper", async () => {
      const { tasks } = await createStreamedClient();
      const deleted = new query.Deleted(
        untyped(tasks, { routingKey: "abc" }),
        TimeStamp.now(),
      );
      const parsed = tasks.parse(deleted, schemas);
      expect(query.Deleted.matches(parsed) && parsed.corpse.config.severity).toBe(
        "high",
      );
      expect(tasks.parse(deleted, schemas)).toBe(parsed);
    });
  });

  describe("record cap", () => {
    it("should keep the status of a held task when it trims statuses", async () => {
      const { stream, tasks, statuses } = await createStreamedClient(10);
      const key = "3b8e1f4a-7c2d-4e9b-a5f6-2c3d4e5f6a7b";
      onTestFinished(tasks.watch(key).onChange(() => {}));
      const cachedMessage = () => {
        const cached = tasks.watch(key).getCached();
        return query.isLive(cached) ? cached.status?.message : undefined;
      };
      stream.push(
        new framer.Frame({
          [task.ACTION_CHANNEL_NAME]: new Series([
            {
              key,
              dispatchKey: "remote-dispatch",
              seq: 1,
              actions: [
                task.create({
                  task: {
                    key,
                    name: "held",
                    type: "test",
                    config: {},
                    status: createStatus(key, "Task not deployed", TimeStamp.now()),
                  },
                }),
              ],
            },
          ]),
        }),
      );
      await expect.poll(cachedMessage).toBe("Task not deployed");
      const pushStatuses = (n: number) =>
        stream.push(
          new framer.Frame({
            [status.SET_CHANNEL_NAME]: new Series(
              Array.from({ length: n }, () =>
                createStatus(uuid.create(), "Other", TimeStamp.now(), uuid.create()),
              ),
            ),
          }),
        );
      pushStatuses(11);
      await expect.poll(() => statuses.store.size).toBe(12);
      pushStatuses(1);
      await expect.poll(() => statuses.store.size).toBe(9);
      expect(statuses.ownedBy(task.resourceID(key))?.message).toBe("Task not deployed");
    });
  });
});
