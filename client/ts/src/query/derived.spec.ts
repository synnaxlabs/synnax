// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { record } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { query } from "@/query";

interface Task extends record.Keyed<string> {
  key: string;
  name: string;
}

interface Status extends record.Keyed<string> {
  key: string;
  task: string;
  message: string;
}

interface ComposedTask extends record.Keyed<string> {
  key: string;
  name: string;
  status?: string;
}

const task = (key: string, name: string): Task => ({ key, name });

const newTables = () => {
  const cache = new query.Cache({ openStreamer: null });
  const tasks = cache.createTable<string, Task>({ name: "tasks" });
  const statuses = cache.createTable<string, Status>({ name: "statuses" });
  const compose = (t: Task): ComposedTask => ({
    ...t,
    status: statuses.get((s) => s.task === t.key)[0]?.message,
  });
  return { cache, tasks, statuses, compose };
};

const derive = (
  { cache, tasks, statuses, compose }: ReturnType<typeof newTables>,
  trigger: query.DeriveTrigger<string> = query.deriveTrigger(statuses, (event) =>
    event.variant === "set" ? [event.value.task] : "all",
  ),
) =>
  cache.derive<string, Task, ComposedTask>({
    name: "tasks.composed",
    source: tasks,
    compose,
    triggers: [trigger],
  });

describe("derived", () => {
  it("composes existing source entries on derive", () => {
    const tables = newTables();
    tables.tasks.set(task("t1", "one"));
    const composed = derive(tables);
    expect(composed.get("t1")).toEqual({ key: "t1", name: "one" });
  });

  it("drops a composed entry when its source trims the record", () => {
    const tables = newTables();
    const composed = derive(tables);
    tables.tasks.set(task("t1", "one"));
    tables.tasks.trim(0);
    tables.tasks.trim(0);
    query.partialUpdate(tables.tasks, "t1", { name: "two" });
    expect(composed.get("t1")).toBeUndefined();
  });

  it("drops a trimmed composed entry silently and tells its holders", () => {
    const tables = newTables();
    const composed = derive(tables);
    tables.tasks.set(task("t1", "one"));
    const events = vi.fn();
    composed.subscribeBatch(events);
    const drop = vi.fn();
    composed.hold({ held: () => [], drop });
    const version = composed.version();
    tables.tasks.trim(0);
    tables.tasks.trim(0);
    expect(composed.has("t1")).toBe(false);
    expect(drop).toHaveBeenCalledWith(new Set(["t1"]));
    expect(events).not.toHaveBeenCalled();
    expect(composed.version()).toBe(version);
  });

  it("materializes a composed entry when the source entry is set", () => {
    const tables = newTables();
    const composed = derive(tables);
    tables.tasks.set(task("t1", "one"));
    expect(composed.get("t1")).toEqual({ key: "t1", name: "one" });
  });

  it("recomposes when a watched table event affects the entry", () => {
    const tables = newTables();
    const composed = derive(tables);
    tables.tasks.set(task("t1", "one"));
    tables.statuses.set({ key: "s1", task: "t1", message: "running" });
    expect(composed.get("t1")).toEqual({
      key: "t1",
      name: "one",
      status: "running",
    });
  });

  it("keeps the source entries that the derived table needs through a trim", () => {
    const tables = newTables();
    const composed = derive(tables);
    tables.tasks.set([task("t1", "one"), task("t2", "two")]);
    composed.subscribe(vi.fn(), "t1");
    tables.tasks.trim(tables.tasks.size);
    expect(tables.tasks.trim(0)).toEqual(["t2"]);
  });

  it("stops keeping the source entries once closed", async () => {
    const tables = newTables();
    const composed = derive(tables);
    tables.tasks.set(task("t1", "one"));
    composed.subscribe(vi.fn(), "t1");
    await tables.cache.close();
    tables.tasks.trim(tables.tasks.size);
    expect(tables.tasks.trim(0)).toEqual(["t1"]);
  });

  it("replaces entry identity exactly when composition changes", () => {
    const tables = newTables();
    const composed = derive(tables);
    tables.tasks.set(task("t1", "one"));
    const first = composed.get("t1");
    // A watch event that leaves the composition unchanged must be silenced.
    tables.statuses.set({ key: "s1", task: "t2", message: "elsewhere" });
    expect(composed.get("t1")).toBe(first);
    tables.statuses.set({ key: "s2", task: "t1", message: "running" });
    const second = composed.get("t1");
    expect(second).not.toBe(first);
    expect(second?.status).toEqual("running");
  });

  it("recomposes every entry on an all projection", () => {
    const tables = newTables();
    const composed = derive(
      tables,
      query.deriveTrigger(tables.statuses, () => "all"),
    );
    tables.tasks.set([task("t1", "one"), task("t2", "two")]);
    tables.statuses.set({ key: "s1", task: "t2", message: "done" });
    expect(composed.get("t2")?.status).toEqual("done");
  });

  it("tombstones the last composed entry when the source entry is deleted", () => {
    const tables = newTables();
    const composed = derive(tables);
    tables.tasks.set(task("t1", "one"));
    tables.statuses.set({ key: "s1", task: "t1", message: "running" });
    tables.tasks.delete("t1");
    expect(composed.get("t1")).toBeUndefined();
    expect(composed.getTombstone("t1")?.corpse).toEqual({
      key: "t1",
      name: "one",
      status: "running",
    });
  });

  it("keeps no derived corpse when a rolled-back source set is undone", () => {
    const tables = newTables();
    const composed = derive(tables);
    const rollback = tables.tasks.set(task("t1", "one"));
    rollback();
    expect(composed.get("t1")).toBeUndefined();
    expect(composed.status("t1")).toEqual("unknown");
  });

  it("flushes a bulk source write into the derived table as one batch", () => {
    const tables = newTables();
    const composed = derive(tables);
    const batches = vi.fn();
    composed.subscribeBatch(batches);
    tables.tasks.set([task("t1", "one"), task("t2", "two"), task("t3", "three")]);
    expect(batches).toHaveBeenCalledTimes(1);
    expect(batches.mock.calls[0][0]).toHaveLength(3);
  });

  it("recomposes once for a bulk write to a watched table", () => {
    const tables = newTables();
    const composed = derive(tables);
    tables.tasks.set([task("t1", "one"), task("t2", "two")]);
    const batches = vi.fn();
    composed.subscribeBatch(batches);
    tables.statuses.set([
      { key: "s1", task: "t1", message: "running" },
      { key: "s2", task: "t2", message: "done" },
    ]);
    expect(batches).toHaveBeenCalledTimes(1);
    expect(batches.mock.calls[0][0]).toHaveLength(2);
    expect(composed.get("t1")?.status).toEqual("running");
    expect(composed.get("t2")?.status).toEqual("done");
  });
});

describe("Cache.derive", () => {
  const newCache = () => new query.Cache({ openStreamer: null });

  it("fetches misses through the source and composes them", async () => {
    const cache = newCache();
    const fetch = vi.fn(async (keys: string[]) => keys.map((k) => task(k, "fetched")));
    const tasks = cache.createTable<string, Task>({ name: "tasks", fetch });
    const composed = cache.derive<string, Task, ComposedTask>({
      name: "tasks.composed",
      source: tasks,
      compose: (t) => ({ ...t, status: undefined }),
    });
    const entries = await composed.retrieve(["t1"]);
    expect(entries).toEqual([{ key: "t1", name: "fetched", status: undefined }]);
    expect(fetch).toHaveBeenCalledWith(["t1"]);
    expect(composed.get("t1")?.name).toEqual("fetched");
  });

  it("clears derived entries on cache reset", async () => {
    const cache = newCache();
    const tasks = cache.createTable<string, Task>({ name: "tasks" });
    const composed = cache.derive<string, Task, ComposedTask>({
      name: "tasks.composed",
      source: tasks,
      compose: (t) => ({ ...t }),
    });
    tasks.set(task("t1", "one"));
    expect(composed.get("t1")).toBeDefined();
    await cache.reset();
    expect(composed.get("t1")).toBeUndefined();
    // Repopulation after reset re-materializes.
    tasks.set(task("t1", "fresh"));
    expect(composed.get("t1")?.name).toEqual("fresh");
  });

  it("stops materializing after close", async () => {
    const cache = newCache();
    const tasks = cache.createTable<string, Task>({ name: "tasks" });
    const composed = cache.derive<string, Task, ComposedTask>({
      name: "tasks.composed",
      source: tasks,
      compose: (t) => ({ ...t }),
    });
    await cache.close();
    tasks.set(task("t1", "one"));
    expect(composed.get("t1")).toBeUndefined();
  });
});
