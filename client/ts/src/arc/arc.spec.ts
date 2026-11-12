// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { crdt, id, uuid } from "@synnaxlabs/x";
import { assert, beforeAll, describe, expect, it, vi } from "vitest";

import { arc } from "@/arc";
import { AccessDeniedError, NotFoundError, ValidationError } from "@/index";
import { query } from "@/query";
import { status } from "@/status";
import { task } from "@/task";
import {
  createTestClient,
  createTestClientWithPolicy,
  expectLive,
  waitForStreamLive,
} from "@/testutil";

const client = createTestClient();

const textOf = (a: arc.Arc): arc.text.Text => {
  if (a.source.mode !== "text") throw new Error("expected a text-mode Arc");
  return a.source.text;
};

const newTextArc = (name: string): arc.New => ({
  name,
  type: "automation",
  source: { mode: "text" },
});

describe("arc", () => {
  describe("retrieve", () => {
    it("should retrieve an arc by its name", async () => {
      const created = await client.arcs.create(newTextArc(`named-arc-${id.create()}`));
      const retrieved = await client.arcs.retrieve(created.name);
      expect(retrieved.key).toEqual(created.key);
    });

    it("should retrieve arcs by search term", async () => {
      const prefix = `searchable-arc-${id.create()}`;
      const names = [`${prefix}-1`, `${prefix}-2`];
      await client.arcs.create(names.map((name) => newTextArc(name)));
      await expect
        .poll(async () => {
          const results = await client.arcs.retrieve({ searchTerm: prefix });
          return results.map((a) => a.name).sort();
        })
        .toEqual(names);
    });

    it("should retrieve arcs by where", async () => {
      const prefix = `where-arc-${id.create()}`;
      const [a, b] = await client.arcs.create([
        newTextArc(`${prefix}-a`),
        newTextArc(`${prefix}-b`),
      ]);
      const res = await client.arcs.retrieve({
        where: { name: { regex: `^${prefix}` }, not: { key: { eq: a.key } } },
      });
      expect(res.map(({ key }) => key)).toEqual([b.key]);
    });

    it("should store a module's compile status under its status key", async () => {
      // The Core writes the status after the create commits, and the cached answer
      // reaches it through the change stream.
      await waitForStreamLive(client.connection);
      const created = await client.arcs.create({
        name: `status_module_${id.create()}`,
        type: "module",
        source: { mode: "text", text: { raw: "export func f() i64 { return 1 }" } },
      });
      await expect
        .poll(async () => await client.arcs.status.retrieve(created.key))
        .toMatchObject({ message: "Compiled", name: created.name });
    });

    it("should throw a NotFoundError for a missing key in a key list", async () => {
      const created = await client.arcs.create(newTextArc(`missing-${id.create()}`));
      const missing = uuid.create();
      await expect(client.arcs.retrieve([created.key, missing])).rejects.toThrow(
        new NotFoundError(`Arc with key ${missing} not found`),
      );
    });

    it("should retrieve Arcs by a list of names in a flat filter", async () => {
      const names = [`flat-${id.create()}`, `flat-${id.create()}`];
      const created = await Promise.all(
        names.map(async (n) => await client.arcs.create(newTextArc(n))),
      );
      const res = await client.arcs.retrieve({ name: names });
      expect(res.map(({ key }) => key).sort()).toEqual(
        created.map(({ key }) => key).sort(),
      );
    });

    it("should omit missing keys from a filter on keys", async () => {
      const created = await client.arcs.create(newTextArc(`missing-${id.create()}`));
      const res = await client.arcs.retrieve({ key: [created.key, uuid.create()] });
      expect(res.map(({ key }) => key)).toEqual([created.key]);
    });
  });

  // A dispatch posts its ops to the Core, and the materialized text reaches this
  // client's cache through the change stream, so reads after one poll. The stream must
  // be live before the first dispatch: a missed echo leaves the dispatch outstanding
  // and the cache pinned to the pre-dispatch document.
  describe("dispatch", () => {
    beforeAll(async () => {
      await waitForStreamLive(client.connection);
    });

    it("materializes insert_char ops into the document's raw text", async () => {
      const created = await client.arcs.create(newTextArc(`dispatch-${id.create()}`));
      const gen = new crdt.Text(2);
      const ops = gen.insert(0, "hello").map((op) => arc.insertChar(op));
      await client.arcs.dispatch(created.key, ops);
      await expect
        .poll(async () => {
          const res = await client.arcs.retrieve(created.key);
          return textOf(res).raw;
        })
        .toEqual("hello");
    });

    it("drops deleted characters from the materialized raw text", async () => {
      const created = await client.arcs.create(newTextArc(`dispatch-${id.create()}`));
      const gen = new crdt.Text(2);
      const insertOps = gen.insert(0, "hello").map((op) => arc.insertChar(op));
      const deleteOps = gen.delete(0, 1).map((op) => arc.deleteChar(op));
      await client.arcs.dispatch(created.key, [...insertOps, ...deleteOps]);
      await expect
        .poll(async () => textOf(await client.arcs.retrieve(created.key)).raw)
        .toEqual("ello");
    });

    it("reclaims tombstoned characters via a forget_chars dispatch", async () => {
      const created = await client.arcs.create(newTextArc(`dispatch-${id.create()}`));
      const gen = new crdt.Text(2);
      const insertOps = gen.insert(0, "hello").map((op) => arc.insertChar(op));
      const deletes = gen.delete(4, 1); // delete the trailing "o" (a leaf tombstone)
      await client.arcs.dispatch(created.key, [
        ...insertOps,
        ...deletes.map((op) => arc.deleteChar(op)),
      ]);
      await client.arcs.dispatch(created.key, [
        arc.forgetChars({ ids: deletes.map((op) => op.id) }),
      ]);
      await expect
        .poll(async () => {
          const text = textOf(await client.arcs.retrieve(created.key));
          return {
            raw: text.raw,
            deletes: text.doc.deletes.length,
            inserts: text.doc.inserts.length,
          };
        })
        .toEqual({ raw: "hell", deletes: 0, inserts: 4 });
    });
  });

  describe("task", () => {
    const attachTask = async (arcKey: arc.Key): Promise<task.Task> => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      return await rack.createTask({
        name: `arc-task-${id.create()}`,
        type: "arc",
        config: { arcKey },
      });
    };

    it("resolves null for an arc with no task", async () => {
      const created = await client.arcs.create(newTextArc(`task-none-${id.create()}`));
      expect(await client.arcs.task.retrieve(created.key)).toBeNull();
    });

    it("resolves null for an arc that does not exist", async () => {
      expect(await client.arcs.task.retrieve(uuid.create())).toBeNull();
    });

    it("resolves null for an arc whose task runs another arc", async () => {
      const owner = await client.arcs.create(newTextArc(`task-own-${id.create()}`));
      const other = await client.arcs.create(newTextArc(`task-other-${id.create()}`));
      await attachTask(owner.key);
      expect(await client.arcs.task.retrieve(other.key)).toBeNull();
    });

    it("retrieves the task attached to an arc and caches the answer", async () => {
      const created = await client.arcs.create(newTextArc(`task-${id.create()}`));
      const tsk = await attachTask(created.key);
      const res = await client.arcs.task.retrieve(created.key);
      expect(res?.key).toEqual(tsk.key);
      expect(res?.config).toMatchObject({ key: expect.any(String) });
      const cached = client.arcs.task.watch(created.key).getCached();
      expect(expectLive(cached)?.key).toEqual(tsk.key);
    });

    it("delivers a task attached while subscribed", async () => {
      const created = await client.arcs.create(newTextArc(`task-sub-${id.create()}`));
      const off = client.arcs.task.watch(created.key).onChange(vi.fn());
      try {
        expect(await client.arcs.task.retrieve(created.key)).toBeNull();
        const tsk = await attachTask(created.key);
        await expect
          .poll(() => {
            const cached = client.arcs.task.watch(created.key).getCached();
            return query.isLive(cached) && cached?.key === tsk.key;
          })
          .toBe(true);
      } finally {
        off();
      }
    });

    it("recomposes the cached answer when the task's status changes", async () => {
      const created = await client.arcs.create(
        newTextArc(`task-status-${id.create()}`),
      );
      const tsk = await attachTask(created.key);
      const off = client.arcs.task.watch(created.key).onChange(vi.fn());
      try {
        const res = await client.arcs.task.retrieve(created.key);
        expect(res?.key).toEqual(tsk.key);
        await client.statuses.set(
          status.create<ReturnType<typeof task.statusDetailsZ>>({
            owner: task.resourceID(tsk.key),
            variant: "error",
            message: "Task failed",
            details: {
              running: false,
              cmd: "",
              configHash: "",
              rack: 0,
              data: {},
            },
          }),
        );
        await expect
          .poll(() => {
            const cached = client.arcs.task.watch(created.key).getCached();
            return (
              query.isLive(cached) &&
              cached?.status?.variant === "error" &&
              cached.status.message === "Task failed"
            );
          })
          .toBe(true);
      } finally {
        off();
      }
    });
  });

  describe("retrieveChanges", () => {
    it("should return nothing before the task is deployed", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`changes-${id.create()}`));
      await client.arcs.updateTask(created.key, { rack: rack.key });
      expect(await client.arcs.retrieveChanges(created.key)).toEqual([]);
    });

    it("should name the automation edited after its deploy", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`changes-${id.create()}`));
      const tsk = await client.arcs.updateTask(created.key, { rack: rack.key });
      assert(tsk != null);
      await client.statuses.set(
        status.create<ReturnType<typeof task.statusDetailsZ>>({
          owner: task.resourceID(tsk.key),
          variant: "success",
          message: "Task running",
          details: {
            running: true,
            cmd: "",
            configHash: tsk.configHash,
            rack: rack.key,
            data: {},
          },
        }),
      );
      const gen = new crdt.Text(2);
      const ops = gen.insert(0, "on -> off").map((op) => arc.insertChar(op));
      await client.arcs.dispatch(created.key, ops);
      await expect
        .poll(async () => await client.arcs.retrieveChanges(created.key))
        .toEqual([{ type: "automation", path: "", kind: "changed" }]);
    });

    it("should return nothing for a module", async () => {
      const created = await client.arcs.create({
        name: `changes_module_${id.create()}`,
        type: "module",
        source: { mode: "text", text: { raw: "export func f() i64 { return 1 }" } },
      });
      expect(await client.arcs.retrieveChanges(created.key)).toEqual([]);
    });
  });

  describe("updateTask", () => {
    it("creates the task stamped with the arc's hash", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`set-rack-${id.create()}`));
      const tsk = await client.arcs.updateTask(created.key, { rack: rack.key });
      expect(tsk).not.toBeNull();
      expect(tsk?.rack).toEqual(rack.key);
      expect(tsk?.type).toEqual("arc");
      expect(tsk?.config.arcKey).toEqual(created.key);
      expect(tsk?.config.hash).not.toEqual("");
      expect(await client.arcs.task.retrieve(created.key)).not.toBeNull();
    });

    it("finds the task through the task client by its Arc", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const deployed = await client.arcs.create(newTextArc(`by-arc-${id.create()}`));
      const other = await client.arcs.create(newTextArc(`by-arc-${id.create()}`));
      const tsk = await client.arcs.updateTask(deployed.key, { rack: rack.key });
      await client.arcs.updateTask(other.key, { rack: rack.key });
      const found = await client.tasks.retrieve({ arcs: [deployed.key] });
      expect(found.map(({ key }) => key)).toEqual([tsk?.key]);
    });

    it("reuses the task across rebinds and rack moves", async () => {
      const rackA = await client.racks.create({ name: `rack-${id.create()}` });
      const rackB = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`move-${id.create()}`));
      const first = await client.arcs.updateTask(created.key, { rack: rackA.key });
      const moved = await client.arcs.updateTask(created.key, { rack: rackB.key });
      expect(moved?.key).toEqual(first?.key);
      expect(moved?.rack).toEqual(rackB.key);
    });

    it("restamps the hash when the program changed", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`restamp-${id.create()}`));
      const first = await client.arcs.updateTask(created.key, { rack: rack.key });
      const gen = new crdt.Text(2);
      const ops = gen.insert(0, "on -> off").map((op) => arc.insertChar(op));
      await client.arcs.dispatch(created.key, ops);
      const second = await client.arcs.updateTask(created.key, { rack: rack.key });
      expect(second?.key).toEqual(first?.key);
      expect(second?.config.hash).not.toEqual(first?.config.hash);
    });

    it("clears the rack by deleting the task", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`clear-rack-${id.create()}`));
      const tsk = await client.arcs.updateTask(created.key, { rack: rack.key });
      if (tsk == null) throw new Error("expected a deployment task");
      await client.arcs.updateTask(created.key, { rack: 0 });
      // The task cache may briefly serve the deleted task until the delete signal
      // lands, so poll for the rejection.
      await expect
        .poll(async () => {
          try {
            await client.tasks.retrieve(tsk.key);
            return false;
          } catch {
            return true;
          }
        })
        .toBe(true);
      expect((await client.arcs.retrieve(created.key)).key).toEqual(created.key);
    });

    it("stops serving the unbound task from the cache", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`uncache-${id.create()}`));
      const tsk = await client.arcs.updateTask(created.key, { rack: rack.key });
      if (tsk == null) throw new Error("expected a deployment task");
      // Caching both the task and its status arms the cached fast-path, so a
      // retrieve after the rack is cleared answers locally instead of asking the Core.
      await client.tasks.retrieve(tsk.key);
      await client.arcs.updateTask(created.key, { rack: 0 });
      await expect(client.tasks.retrieve(tsk.key)).rejects.toThrow();
    });

    it("does not serve a snapshot of the task after an unbind", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`snap-${id.create()}`));
      const tsk = await client.arcs.updateTask(created.key, { rack: rack.key });
      if (tsk == null) throw new Error("expected a deployment task");
      expect((await client.arcs.task.retrieve(created.key))?.key).toEqual(tsk.key);
      await client.tasks.copy(tsk.key, "Snapshot", true);
      await client.arcs.updateTask(created.key, { rack: 0 });
      await expect
        .poll(async () => await client.arcs.task.retrieve(created.key))
        .toBeNull();
    });

    it("defaults a new task to auto performance", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`auto-${id.create()}`));
      const tsk = await client.arcs.updateTask(created.key, { rack: rack.key });
      expect(tsk?.config.performance).toEqual("auto");
    });

    it("sets the performance and keeps it across a rack move", async () => {
      const rackA = await client.racks.create({ name: `rack-${id.create()}` });
      const rackB = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`perf-${id.create()}`));
      const first = await client.arcs.updateTask(created.key, { rack: rackA.key });
      const set = await client.arcs.updateTask(created.key, { performance: "high" });
      expect(set?.key).toEqual(first?.key);
      expect(set?.config.performance).toEqual("high");
      expect(set?.configHash).not.toEqual(first?.configHash);
      const moved = await client.arcs.updateTask(created.key, { rack: rackB.key });
      expect(moved?.config.performance).toEqual("high");
    });

    it("rejects the performance of an arc with no rack", async () => {
      const created = await client.arcs.create(newTextArc(`no-rack-${id.create()}`));
      await expect(
        client.arcs.updateTask(created.key, { performance: "low" }),
      ).rejects.toThrow("cannot set the performance of an arc with no rack");
    });

    it("leaves the task untouched when the subject may not set the rack", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`denied-${id.create()}`));
      const tsk = await client.arcs.updateTask(created.key, { rack: rack.key });
      if (tsk == null) throw new Error("expected a deployment task");
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [arc.resourceID(""), task.resourceID("")],
        actions: ["retrieve"],
      });
      await expect(
        userClient.arcs.updateTask(created.key, { rack: rack.key }),
      ).rejects.toSatisfy(AccessDeniedError.matches);
      const surviving = await client.tasks.retrieve(tsk.key);
      expect(surviving.rack).toEqual(rack.key);
    });
  });

  describe("rename", () => {
    beforeAll(async () => {
      await waitForStreamLive(client.connection);
    });

    it("renames the arc and its task", async () => {
      const fresh = createTestClient();
      await waitForStreamLive(fresh.connection);
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`rename-${id.create()}`));
      const deployed = await client.arcs.updateTask(created.key, { rack: rack.key });
      if (deployed == null) throw new Error("expected a deployment task");
      const name = `renamed-${id.create()}`;
      await client.arcs.rename(created.key, name);
      expect((await fresh.arcs.retrieve(created.key)).name).toEqual(name);
      await expect
        .poll(async () => (await fresh.tasks.retrieve(deployed.key)).name)
        .toEqual(name);
    });

    it("leaves the arc and its task unchanged when the name is used", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const taken = await client.arcs.create(newTextArc(`taken-${id.create()}`));
      const created = await client.arcs.create(newTextArc(`rename-${id.create()}`));
      const deployed = await client.arcs.updateTask(created.key, { rack: rack.key });
      if (deployed == null) throw new Error("expected a deployment task");
      await expect(client.arcs.rename(created.key, taken.name)).rejects.toThrow(
        "is already used in this directory",
      );
      const fresh = createTestClient();
      expect((await fresh.arcs.retrieve(created.key)).name).toEqual(created.name);
      expect((await fresh.tasks.retrieve(deployed.key)).name).toEqual(deployed.name);
    });
  });

  describe("module name", () => {
    const newModule = (name: string): arc.New => ({
      name,
      type: "module",
      source: { mode: "text" },
    });

    const rejection = 'Name "bad name" must start with a letter or underscore';

    it("should reject a module whose name is not an identifier", async () => {
      const created = client.arcs.create(newModule("bad name"));
      await expect(created).rejects.toSatisfy(ValidationError.matches);
      await expect(created).rejects.toThrow(rejection);
    });

    it("should create an automation whose name is not an identifier", async () => {
      const name = `bad name ${id.create()}`;
      expect((await client.arcs.create(newTextArc(name))).name).toEqual(name);
    });

    it("should reject a module rename to a name that is not an identifier", async () => {
      const name = `m_${id.create().replace(/-/g, "_")}`;
      const created = await client.arcs.create(newModule(name));
      const renamed = client.arcs.rename(created.key, "bad name");
      await expect(renamed).rejects.toSatisfy(ValidationError.matches);
      await expect(renamed).rejects.toThrow(rejection);
      expect((await client.arcs.retrieve(created.key)).name).toEqual(created.name);
    });
  });

  describe("task sync", () => {
    // The writing client's task cache still holds the pre-dispatch copy, so assertions
    // read through a fresh client to reach the Core. That client learns of a rewrite
    // through the task set signal, so its stream must be live first.
    it("rewrites the task config when a dispatch changes the content", async () => {
      const fresh = createTestClient();
      await waitForStreamLive(fresh.connection);
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`sync-${id.create()}`));
      const deployed = await client.arcs.updateTask(created.key, { rack: rack.key });
      if (deployed == null) throw new Error("expected a deployment task");
      const gen = new crdt.Text(2);
      const ops = gen.insert(0, "x -> y").map((op) => arc.insertChar(op));
      await client.arcs.dispatch(created.key, ops);
      await expect
        .poll(async () => (await fresh.tasks.retrieve(deployed.key)).configHash)
        .not.toEqual(deployed.configHash);
      const synced = await fresh.tasks.retrieve(deployed.key);
      expect(synced.config.hash).not.toEqual(deployed.config.hash);
    });

    it("restores the deployed config when an edit is undone", async () => {
      const fresh = createTestClient();
      await waitForStreamLive(fresh.connection);
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`undo-${id.create()}`));
      const deployed = await client.arcs.updateTask(created.key, { rack: rack.key });
      if (deployed == null) throw new Error("expected a deployment task");
      const gen = new crdt.Text(2);
      const [op] = gen.insert(0, "x");
      await client.arcs.dispatch(created.key, [arc.insertChar(op)]);
      // Waiting for the edit to reach the task keeps the undo assertion from passing
      // against a config the insert has not touched yet.
      await expect
        .poll(async () => (await fresh.tasks.retrieve(deployed.key)).configHash)
        .not.toEqual(deployed.configHash);
      await client.arcs.dispatch(created.key, [arc.deleteChar({ id: op.id })]);
      await expect
        .poll(async () => (await fresh.tasks.retrieve(deployed.key)).configHash)
        .toEqual(deployed.configHash);
    });
  });
});
