// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { query } from "@/query";
import { bindDerived, deriveWatch } from "@/query/derived";

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
  const tasks = new query.Table<string, Task>({ onError: () => {} });
  const statuses = new query.Table<string, Status>({ onError: () => {} });
  const composed = new query.Table<string, ComposedTask>({ onError: () => {} });
  const compose = (t: Task): ComposedTask => ({
    ...t,
    status: statuses.get((s) => s.task === t.key)[0]?.message,
  });
  return { tasks, statuses, composed, compose };
};

const bindAll = ({
  tasks,
  statuses,
  composed,
  compose,
}: ReturnType<typeof newTables>) =>
  bindDerived(composed, {
    source: tasks,
    compose,
    watch: [
      deriveWatch(statuses, (event) =>
        event.variant === "set" ? [event.value.task] : "all",
      ),
    ],
  });

describe("derived", () => {
  it("composes existing source rows on bind", () => {
    const tables = newTables();
    tables.tasks.set(task("t1", "one"));
    bindAll(tables);
    expect(tables.composed.get("t1")).toEqual({ key: "t1", name: "one" });
  });

  it("materializes a composed row when the source row is set", () => {
    const tables = newTables();
    bindAll(tables);
    tables.tasks.set(task("t1", "one"));
    expect(tables.composed.get("t1")).toEqual({ key: "t1", name: "one" });
  });

  it("recomposes when a watched table event affects the row", () => {
    const tables = newTables();
    bindAll(tables);
    tables.tasks.set(task("t1", "one"));
    tables.statuses.set({ key: "s1", task: "t1", message: "running" });
    expect(tables.composed.get("t1")).toEqual({
      key: "t1",
      name: "one",
      status: "running",
    });
  });

  it("replaces row identity exactly when composition changes", () => {
    const tables = newTables();
    bindAll(tables);
    tables.tasks.set(task("t1", "one"));
    const first = tables.composed.get("t1");
    // A watch event that leaves the composition unchanged must be silenced.
    tables.statuses.set({ key: "s1", task: "t2", message: "elsewhere" });
    expect(tables.composed.get("t1")).toBe(first);
    tables.statuses.set({ key: "s2", task: "t1", message: "running" });
    const second = tables.composed.get("t1");
    expect(second).not.toBe(first);
    expect(second?.status).toEqual("running");
  });

  it("recomposes every row on an all projection", () => {
    const tables = newTables();
    bindDerived(tables.composed, {
      source: tables.tasks,
      compose: tables.compose,
      watch: [deriveWatch(tables.statuses, () => "all")],
    });
    tables.tasks.set([task("t1", "one"), task("t2", "two")]);
    tables.statuses.set({ key: "s1", task: "t2", message: "done" });
    expect(tables.composed.get("t2")?.status).toEqual("done");
  });

  it("tombstones the last composed row when the source row is deleted", () => {
    const tables = newTables();
    bindAll(tables);
    tables.tasks.set(task("t1", "one"));
    tables.statuses.set({ key: "s1", task: "t1", message: "running" });
    tables.tasks.delete("t1");
    expect(tables.composed.get("t1")).toBeUndefined();
    expect(tables.composed.getTombstone("t1")?.corpse).toEqual({
      key: "t1",
      name: "one",
      status: "running",
    });
  });

  it("detaches every subscription through the returned destructor", () => {
    const tables = newTables();
    const detach = bindAll(tables);
    detach();
    tables.tasks.set(task("t1", "one"));
    expect(tables.composed.get("t1")).toBeUndefined();
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
    const rows = await composed.retrieve(["t1"]);
    expect(rows).toEqual([{ key: "t1", name: "fetched", status: undefined }]);
    expect(fetch).toHaveBeenCalledWith(["t1"]);
    expect(composed.get("t1")?.name).toEqual("fetched");
  });

  it("clears derived rows on cache reset", async () => {
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
