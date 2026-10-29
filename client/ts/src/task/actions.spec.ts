// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { task } from "@/task";

const KEY = "5a7d3e1c-2f4b-4c6d-8e9f-0a1b2c3d4e5f";

const createTask = (): task.Payload =>
  task.payloadZ().parse({
    key: KEY,
    name: "Read",
    type: "ni_analog_read",
    rack: 1,
    configHash: "h1",
    config: {
      sampleRate: 10,
      channels: [
        { key: "a", port: 0, minVal: 0 },
        { key: "b", port: 1, minVal: 0 },
      ],
    },
  });

const channelKeys = (t: task.Payload): string[] =>
  (t.config.channels as Array<{ key: string }>).map(({ key }) => key);

describe("task.reduceAll", () => {
  describe("set", () => {
    it("should set a field at a snake_case path in the camelCase config", () => {
      const { next } = task.reduceAll(createTask(), [
        task.set({ path: ["config", "sample_rate"], value: 20 }),
      ]);
      expect(next.config.sampleRate).toBe(20);
    });

    it("should convert the keys of the value to camelCase", () => {
      const { next } = task.reduceAll(createTask(), [
        task.set({
          path: ["config", "channels", "a"],
          value: { key: "a", port: 3, min_val: 5 },
        }),
      ]);
      expect(next.config.channels).toEqual([
        { key: "a", port: 3, minVal: 5 },
        { key: "b", port: 1, minVal: 0 },
      ]);
    });

    it("should set the name", () => {
      const { next } = task.reduceAll(createTask(), [
        task.set({ path: ["name"], value: "Renamed" }),
      ]);
      expect(next.name).toBe("Renamed");
    });

    it("should set the config hash that the Core broadcasts", () => {
      const { next } = task.reduceAll(createTask(), [
        task.set({ path: ["config_hash"], value: "h2" }),
      ]);
      expect(next.configHash).toBe("h2");
    });

    it("should return a snake_case inverse that restores the task", () => {
      const prev = createTask();
      const { next, inverse } = task.reduceAll(prev, [
        task.set({ path: ["config", "channels", "a"], value: { key: "a", port: 3 } }),
      ]);
      expect(inverse).toEqual([
        task.set({
          path: ["config", "channels", "a"],
          value: { key: "a", port: 0, min_val: 0 },
        }),
      ]);
      expect(task.reduceAll(next, inverse).next).toEqual(prev);
    });

    it("should target the path of the set", () => {
      const { targets } = task.reduceAll(createTask(), [
        task.set({ path: ["config", "sample_rate"], value: 20 }),
      ]);
      expect(targets).toEqual(["config.sample_rate"]);
    });

    it("should do nothing when a guard does not hold", () => {
      const prev = createTask();
      const res = task.reduceAll(prev, [
        task.set({
          path: ["config", "sample_rate"],
          value: 20,
          guards: [{ path: ["type"], value: "ni_digital_read" }],
        }),
      ]);
      expect(res).toEqual({ next: prev, inverse: [], targets: [] });
      expect(res.next).toBe(prev);
    });

    it("should throw when the task cannot hold the value", () => {
      expect(() =>
        task.reduceAll(createTask(), [
          task.set({ path: ["config", "sample_rate", "hz"], value: 20 }),
        ]),
      ).toThrow("[config, sampleRate]: the value is not an object or a list");
    });
  });

  describe("insert", () => {
    it("should insert a camelCase entry after the given entry", () => {
      const { next, targets } = task.reduceAll(createTask(), [
        task.insert({
          path: ["config", "channels"],
          value: { key: "c", port: 2, min_val: 1 },
          after: "a",
        }),
      ]);
      expect(channelKeys(next)).toEqual(["a", "c", "b"]);
      expect(next.config.channels).toContainEqual({ key: "c", port: 2, minVal: 1 });
      expect(targets).toEqual(["config.channels.c"]);
    });
  });

  describe("remove", () => {
    it("should remove an entry with an inverse that inserts it again", () => {
      const prev = createTask();
      const { next, inverse } = task.reduceAll(prev, [
        task.remove({ path: ["config", "channels"], key: "a" }),
      ]);
      expect(channelKeys(next)).toEqual(["b"]);
      expect(task.reduceAll(next, inverse).next).toEqual(prev);
    });
  });

  describe("move", () => {
    it("should move an entry to the front", () => {
      const { next } = task.reduceAll(createTask(), [
        task.move({ path: ["config", "channels"], key: "b" }),
      ]);
      expect(channelKeys(next)).toEqual(["b", "a"]);
    });
  });

  describe("create", () => {
    it("should replace the task with the created task", () => {
      const created = { ...createTask(), name: "Created", configHash: "h3" };
      const { next } = task.reduceAll(createTask(), [task.create({ task: created })]);
      expect(next).toEqual(created);
    });
  });
});

describe("task.createOf", () => {
  it("should return the created task without its status", () => {
    const created = task.payloadZ().parse({
      ...createTask(),
      status: {
        key: "task:1",
        variant: "success",
        message: "Running",
        time: 0,
        details: { task: KEY, running: true },
      },
    });
    const { status: _, ...expected } = created;
    expect(task.createOf(task.create({ task: created }))).toEqual(expected);
  });

  it("should return undefined for another action", () => {
    expect(
      task.createOf(task.set({ path: ["name"], value: "Renamed" })),
    ).toBeUndefined();
  });
});
