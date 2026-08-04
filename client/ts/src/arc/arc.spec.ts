// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { crdt, id, uuid } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { arc } from "@/arc";
import { AuthError } from "@/errors";
import { query } from "@/query";
import { status } from "@/status";
import { task } from "@/task";
import { createTestClient, createTestClientWithPolicy, expectLive } from "@/testutil";

const client = createTestClient();

const newTextArc = (name: string): arc.New => ({ name, mode: "text" });

describe("arc", () => {
  describe("retrieve", () => {
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
  });

  describe("dispatch", () => {
    it("materializes insert_char ops into the document's raw text", async () => {
      const created = await client.arcs.create(newTextArc(`dispatch-${id.create()}`));
      const gen = new crdt.Text(2);
      const ops = gen.insert(0, "hello").map((op) => arc.insertChar(op));
      await client.arcs.dispatch(created.key, ops);
      await expect
        .poll(async () => {
          const res = await client.arcs.retrieve(created.key);
          return res.text.raw;
        })
        .toEqual("hello");
    });

    it("drops deleted characters from the materialized raw text", async () => {
      const created = await client.arcs.create(newTextArc(`dispatch-${id.create()}`));
      const gen = new crdt.Text(2);
      const insertOps = gen.insert(0, "hello").map((op) => arc.insertChar(op));
      const deleteOps = gen.delete(0, 1).map((op) => arc.deleteChar(op));
      await client.arcs.dispatch(created.key, [...insertOps, ...deleteOps]);
      const res = await client.arcs.retrieve(created.key);
      expect(res.text.raw).toEqual("ello");
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
      const res = await client.arcs.retrieve(created.key);
      expect(res.text.raw).toEqual("hell");
      expect(res.text.doc.deletes).toHaveLength(0);
      expect(res.text.doc.inserts).toHaveLength(4);
    });
  });

  describe("task", () => {
    const attachTask = async (arcKey: arc.Key): Promise<task.Task> => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const tsk = await rack.createTask({
        name: `arc-task-${id.create()}`,
        type: "testType",
        config: { value: "test" },
      });
      await client.ontology.addChildren(
        arc.ontologyID(arcKey),
        task.ontologyID(tsk.key),
      );
      return tsk;
    };

    it("resolves null for an arc with no task", async () => {
      const created = await client.arcs.create(newTextArc(`task-none-${id.create()}`));
      expect(await client.arcs.task.retrieve(created.key)).toBeNull();
    });

    it("resolves null for an arc that does not exist", async () => {
      expect(await client.arcs.task.retrieve(uuid.create())).toBeNull();
    });

    it("retrieves the task attached to an arc and caches the answer", async () => {
      const created = await client.arcs.create(newTextArc(`task-${id.create()}`));
      const tsk = await attachTask(created.key);
      const res = await client.arcs.task.retrieve(created.key);
      expect(res?.key).toEqual(tsk.key);
      expect(res?.config).toEqual({ value: "test" });
      const cached = client.arcs.task.getCached(created.key);
      expect(expectLive(cached)?.key).toEqual(tsk.key);
    });

    it("delivers a task attached while subscribed", async () => {
      const created = await client.arcs.create(newTextArc(`task-sub-${id.create()}`));
      const off = client.arcs.task.onChange(created.key, vi.fn());
      try {
        expect(await client.arcs.task.retrieve(created.key)).toBeNull();
        const tsk = await attachTask(created.key);
        await expect
          .poll(() => {
            const cached = client.arcs.task.getCached(created.key);
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
      const off = client.arcs.task.onChange(created.key, vi.fn());
      try {
        const res = await client.arcs.task.retrieve(created.key);
        expect(res?.key).toEqual(tsk.key);
        await client.statuses.set(
          status.create<ReturnType<typeof task.statusDetailsZ>>({
            key: task.statusKey(tsk.key),
            variant: "error",
            message: "Task failed",
            details: {
              task: tsk.key,
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
            const cached = client.arcs.task.getCached(created.key);
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

  describe("deploy", () => {
    it("creates the deployment task stamped with the arc's hash", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`deploy-${id.create()}`));
      const tsk = await client.arcs.deploy(created.key, rack.key);
      expect(tsk).not.toBeNull();
      expect(tsk?.rack).toEqual(rack.key);
      expect(tsk?.type).toEqual("arc");
      expect(tsk?.config.arcKey).toEqual(created.key);
      expect(tsk?.config.hash).not.toEqual("");
      expect(await client.arcs.task.retrieve(created.key)).not.toBeNull();
    });

    it("reuses the task across redeploys and rack moves", async () => {
      const rackA = await client.racks.create({ name: `rack-${id.create()}` });
      const rackB = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`move-${id.create()}`));
      const first = await client.arcs.deploy(created.key, rackA.key);
      const moved = await client.arcs.deploy(created.key, rackB.key);
      expect(moved?.key).toEqual(first?.key);
      expect(moved?.rack).toEqual(rackB.key);
    });

    it("restamps the hash when the program changed", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`restamp-${id.create()}`));
      const first = await client.arcs.deploy(created.key, rack.key);
      const gen = new crdt.Text(2);
      const ops = gen.insert(0, "on -> off").map((op) => arc.insertChar(op));
      await client.arcs.dispatch(created.key, ops);
      const second = await client.arcs.deploy(created.key, rack.key);
      expect(second?.key).toEqual(first?.key);
      expect(second?.config.hash).not.toEqual(first?.config.hash);
    });

    it("undeploys by deleting the task", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`undeploy-${id.create()}`));
      const tsk = await client.arcs.deploy(created.key, rack.key);
      if (tsk == null) throw new Error("expected a deployment task");
      await client.arcs.undeploy(created.key);
      await expect(client.tasks.retrieve(tsk.key)).rejects.toThrow();
      expect((await client.arcs.retrieve(created.key)).key).toEqual(created.key);
    });

    it("leaves the task untouched when the subject may not deploy", async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const created = await client.arcs.create(newTextArc(`denied-${id.create()}`));
      const tsk = await client.arcs.deploy(created.key, rack.key);
      if (tsk == null) throw new Error("expected a deployment task");
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [arc.ontologyID(""), task.ontologyID("")],
        actions: ["retrieve"],
      });
      await expect(userClient.arcs.deploy(created.key, rack.key)).rejects.toThrow(
        AuthError,
      );
      const surviving = await client.tasks.retrieve(tsk.key);
      expect(surviving.rack).toEqual(rack.key);
    });
  });

  describe("hash", () => {
    it("serves the semantic hash on retrieve", async () => {
      const created = await client.arcs.create(newTextArc(`hash-${id.create()}`));
      const res = await client.arcs.retrieve(created.key);
      expect(res.hash).toBeTypeOf("string");
      expect(res.hash).not.toEqual("");
    });

    it("updates the cached hash from the dispatch response", async () => {
      const created = await client.arcs.create(newTextArc(`hash-d-${id.create()}`));
      const before = (await client.arcs.retrieve(created.key)).hash;
      const gen = new crdt.Text(2);
      const ops = gen.insert(0, "x -> y").map((op) => arc.insertChar(op));
      await client.arcs.dispatch(created.key, ops);
      const cached = expectLive(client.arcs.getCached(created.key));
      expect(cached?.hash).toBeTypeOf("string");
      expect(cached?.hash).not.toEqual(before);
      expect((await client.arcs.retrieve(created.key)).hash).toEqual(cached?.hash);
    });
  });
});
