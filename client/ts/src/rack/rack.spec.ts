// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, sleep, TimeStamp, zod } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";

import { NotFoundError, ValidationError } from "@/index";
import { query } from "@/query";
import { rack } from "@/rack";
import { status } from "@/status";
import {
  createTestClient,
  expectLive,
  holdResponses,
  spyOnSend,
  waitForStreamLive,
} from "@/testutil";

const client = createTestClient();

describe("Rack", () => {
  describe("create", () => {
    it("should create a single rack", async () => {
      const r = await client.racks.create({ name: "test" });
      expect(r.key).toBeGreaterThan(0n);
    });
    it("should return an error if the rack doesn't have a name", async () => {
      // @ts-expect-error - Testing for error
      await expect(client.racks.create({})).rejects.toThrow(zod.ParseError);
    });
    it("should create a rack with a custom status", async () => {
      const customStatus: rack.Status = {
        key: "",
        name: "",
        variant: "success",
        message: "Custom rack status",
        description: "Rack is running",
        time: TimeStamp.now(),
        labels: [],
        details: { running: true },
      };
      const r = await client.racks.create({
        name: "rack-with-status",
        status: customStatus,
      });
      expect(r.key).toBeGreaterThan(0n);
      const retrieved = await client.racks.retrieve({
        key: r.key,
        includeStatus: true,
      });
      expect(retrieved.status).toBeDefined();
      expect(retrieved.status?.variant).toBe("success");
      expect(retrieved.status?.message).toBe("Custom rack status");
      expect(retrieved.status?.description).toBe("Rack is running");
      expect(retrieved.status?.owner).toEqual(rack.resourceID(r.key));
    });
    it("should return the status of a new rack", async () => {
      const r = await client.racks.create({ name: "created-status" });
      expect(r.status?.owner).toEqual(rack.resourceID(r.key));
    });
    it("should answer a cached status query for a new rack", async () => {
      const local = createTestClient();
      await local.connect();
      const r = await local.racks.create({ name: "cached-status" });
      const cached = local.racks.watch({ key: r.key, includeStatus: true }).getCached();
      expect(cached).toBeInstanceOf(rack.Rack);
      expect((cached as rack.Rack).status?.owner).toEqual(rack.resourceID(r.key));
    });
  });
  describe("update", () => {
    it("should refuse a rack whose key exists and keep the stored rack", async () => {
      const r = await client.racks.create({ name: "test" });
      const create = client.racks.create({ key: r.key, name: "updated" });
      await expect(create).rejects.toThrow(ValidationError);
      await expect(create).rejects.toThrow(`rack ${r.key} already exists`);
      const retrieved = await client.racks.retrieve(r.key);
      expect(retrieved.name).toBe("test");
    });
  });
  describe("rename", () => {
    it("should rename a rack and keep its integrations", async () => {
      const r = await client.racks.create({
        name: "rack-rename",
        integrations: ["ni", "opc"],
      });
      await client.racks.rename(r.key, "rack-renamed");
      const retrieved = await createTestClient().racks.retrieve(r.key);
      expect(retrieved.name).toBe("rack-renamed");
      expect(retrieved.integrations).toEqual(["ni", "opc"]);
    });
    it("should keep the embedded flag of the embedded rack", async () => {
      const racks = await client.racks.retrieve({});
      const embedded = racks.find((r) => r.embedded);
      expect(embedded).toBeDefined();
      if (embedded == null) return;
      onTestFinished(
        async () => await client.racks.rename(embedded.key, embedded.name),
      );
      await client.racks.rename(embedded.key, `embedded-${id.create()}`);
      const retrieved = await createTestClient().racks.retrieve(embedded.key);
      expect(retrieved.embedded).toBe(true);
    });
    it("should rename the status of the rack", async () => {
      const r = await client.racks.create({ name: "rack-rename-status" });
      await client.racks.rename(r.key, "rack-renamed-status");
      const retrieved = await createTestClient().racks.retrieve({
        key: r.key,
        includeStatus: true,
      });
      expect(retrieved.status?.name).toBe("rack-renamed-status");
    });
    it("should update the cache of another client", async () => {
      const remote = createTestClient();
      const r = await client.racks.create({ name: "rack-rename-remote" });
      expect((await client.racks.retrieve(r.key)).name).toBe("rack-rename-remote");
      const name = `rack-renamed-${id.create()}`;
      await waitForStreamLive(client.connection);
      await remote.racks.rename(r.key, name);
      await expect
        .poll(async () => (await client.racks.retrieve(r.key)).name)
        .toBe(name);
    });
    it("should keep a rename that lands before an older fetch", async () => {
      const r = await client.racks.create({ name: `rack-${id.create()}` });
      const local = createTestClient();
      await local.connect();
      await waitForStreamLive(local.connection);
      await local.racks.retrieve(r.key);
      const { responded, release } = holdResponses(local, "/rack/retrieve");
      const fetching = local.racks.retrieve({ name: [r.name] });
      await responded;
      const name = `rack-renamed-${id.create()}`;
      await client.racks.rename(r.key, name);
      await expect
        .poll(() => expectLive(local.racks.watch(r.key).getCached()).name)
        .toBe(name);
      release();
      await fetching;
      expect(expectLive(local.racks.watch(r.key).getCached()).name).toBe(name);
    });
    it("should reject an empty name", async () => {
      const r = await client.racks.create({ name: "rack-rename-empty" });
      await expect(client.racks.rename(r.key, "")).rejects.toThrow();
      expect((await createTestClient().racks.retrieve(r.key)).name).toBe(
        "rack-rename-empty",
      );
    });
  });
  describe("dispatch", () => {
    it("should apply the action to the rack", async () => {
      const r = await client.racks.create({ name: `rack-${id.create()}` });
      await client.racks.dispatch(
        r.key,
        rack.set({ path: ["integrations"], value: ["modbus"] }),
      );
      const stored = await createTestClient().racks.retrieve(r.key);
      expect(stored.integrations).toEqual(["modbus"]);
    });
  });
  describe("retrieve", () => {
    it("retrieves a rack by its name", async () => {
      const created = await client.racks.create({ name: `named-rack-${id.create()}` });
      const retrieved = await client.racks.retrieve(created.name);
      expect(retrieved.key).toEqual(created.key);
    });

    it("retrieves racks by a list of names", async () => {
      const prefix = `named-racks-${id.create()}`;
      const created = await client.racks.create([
        { name: `${prefix}-a` },
        { name: `${prefix}-b` },
      ]);
      const retrieved = await client.racks.retrieve(created.map((r) => r.name));
      expect(retrieved.map((r) => r.key).sort()).toEqual(
        created.map((r) => r.key).sort(),
      );
    });

    it("coalesces concurrent single retrieves into one request", async () => {
      const created = [
        await client.racks.create({ name: "coalesce-a" }),
        await client.racks.create({ name: "coalesce-b" }),
      ];
      const local = createTestClient();
      await local.connect();
      const send = spyOnSend(local);
      const res = await Promise.all(
        created.map(async ({ key }) => await local.racks.retrieve(key)),
      );
      expect(res.map(({ key }) => key)).toEqual(created.map(({ key }) => key));
      expect(
        send.mock.calls.filter(([target]) => target === "/rack/retrieve"),
      ).toHaveLength(1);
    });

    it("does not reject concurrent retrieves when a key in the window is missing", async () => {
      const created = await client.racks.create({ name: "isolation" });
      const local = createTestClient();
      await local.connect();
      const [ok, missing] = await Promise.allSettled([
        local.racks.retrieve(created.key),
        local.racks.retrieve(4123456789),
      ]);
      expect(ok.status).toEqual("fulfilled");
      expect(missing.status).toEqual("rejected");
      if (missing.status === "rejected")
        expect(NotFoundError.matches(missing.reason)).toBe(true);
    });

    it("should retrieve a rack by its key", async () => {
      const r = await client.racks.create({ name: "test" });
      const retrieved = await client.racks.retrieve(r.key);
      expect(retrieved.key).toBe(r.key);
      expect(retrieved.name).toBe("test");
    });
    it("should retrieve a rack by its name", async () => {
      const name = `${TimeStamp.now().toString()}-${Math.random()}`;
      const r = await client.racks.create({ name });
      const retrieved = await client.racks.retrieve({ name });
      expect(retrieved.key).toBe(r.key);
      expect(retrieved.name).toEqual(name);
    });
    it("should reject a page on a plain name", async () => {
      const name = `rack-page-${id.create()}`;
      await client.racks.create([{ name }, { name }]);
      await expect(
        // @ts-expect-error - a page could hide a second rack with the name.
        client.racks.retrieve({ name, limit: 1 }),
      ).rejects.toThrow(
        new ValidationError(
          "a query for one record takes no limit or cursor, since a page could hide a " +
            "second match",
        ),
      );
    });
    it("should name the operator at fault in a where tree", async () => {
      await expect(
        // @ts-expect-error - a name operator takes a string.
        client.racks.retrieve({ where: { name: { eq: 5 } } }),
      ).rejects.toThrow(/at where\.name\n.*\n\s*✗ "eq": 5 {2}× expected string/);
    });
    it("should name the filter field at fault", async () => {
      await expect(
        // @ts-expect-error - embedded takes a boolean.
        client.racks.retrieve({ embedded: "yes" }),
      ).rejects.toThrow('✗ "embedded": "yes"  × expected boolean');
    });
    it("should retrieve racks that match any branch of an or", async () => {
      const [a, b, c] = await client.racks.create([
        { name: `rack-or-${id.create()}` },
        { name: `rack-or-${id.create()}`, integrations: ["labjack"] },
        { name: `rack-or-${id.create()}` },
      ]);
      const results = await client.racks.retrieve({
        where: {
          key: { in: [a.key, b.key, c.key] },
          or: [{ name: { eq: a.name } }, { integrations: { any: ["labjack"] } }],
        },
      });
      expect(results.map((r) => r.key).sort()).toEqual([a.key, b.key].sort());
    });
    it("should treat a missing key as a non-match", async () => {
      const r = await client.racks.create({ name: "rack-missing-key" });
      const results = await client.racks.retrieve({ key: [r.key, 4123456789] });
      expect(results.map(({ key }) => key)).toEqual([r.key]);
    });
    it("should retrieve racks by search term", async () => {
      const prefix = `searchable-rack-${id.create()}`;
      const names = [`${prefix}-1`, `${prefix}-2`];
      await client.racks.create(names.map((name) => ({ name })));
      await expect
        .poll(async () => {
          const results = await client.racks.retrieve({ searchTerm: prefix });
          return results.map((r) => r.name).sort();
        })
        .toEqual(names);
    });
  });
  describe("integrations", () => {
    it("should create a rack with integrations and retrieve them", async () => {
      const r = await client.racks.create({
        name: "rack-with-integrations",
        integrations: ["ni", "opc", "modbus"],
      });
      const retrieved = await client.racks.retrieve(r.key);
      expect(retrieved.integrations).toEqual(["ni", "opc", "modbus"]);
    });
    it("should filter racks by integration", async () => {
      const r1 = await client.racks.create({
        name: "rack-ni-opc",
        integrations: ["ni", "opc"],
      });
      const r2 = await client.racks.create({
        name: "rack-modbus-only",
        integrations: ["modbus"],
      });
      const results = await client.racks.retrieve({ integrations: "ni" });
      const keys = results.map((r) => r.key);
      expect(keys).toContain(r1.key);
      expect(keys).not.toContain(r2.key);
    });
    it("should filter racks by integration in a flat filter", async () => {
      const [r1, r2] = await Promise.all(
        [["ni", "opc"], ["modbus"]].map(
          async (integrations) =>
            await client.racks.create({ name: "rack-flat", integrations }),
        ),
      );
      const results = await client.racks.retrieve({
        key: [r1.key, r2.key],
        integrations: "ni",
      });
      expect(results.map((r) => r.key)).toEqual([r1.key]);
    });
    it("should update integrations with a dispatch", async () => {
      const r = await client.racks.create({
        name: "rack-update-integrations",
        integrations: ["ni"],
      });
      await client.racks.dispatch(
        r.key,
        rack.set({ path: ["integrations"], value: ["ni", "opc", "arc"] }),
      );
      const retrieved = await createTestClient().racks.retrieve(r.key);
      expect(retrieved.integrations).toEqual(["ni", "opc", "arc"]);
    });
  });
  describe("version", () => {
    it("should keep the Driver version of a rack", async () => {
      const r = await client.racks.create({
        name: "rack-with-version",
        version: "0.61.3",
      });
      expect(r.version).toBe("0.61.3");
      const retrieved = await client.racks.retrieve({
        key: r.key,
        includeStatus: true,
      });
      expect(retrieved.version).toBe("0.61.3");
    });
  });
  describe("tasks", () => {
    it("should list the tasks on a rack", async () => {
      const r = await client.racks.create({ name: "test" });
      const tasks = await r.listTasks();
      expect(tasks).toHaveLength(0);
    });
  });
  describe("status", () => {
    it("should include the rack's status when includeStatus is true", async () => {
      const r = await client.racks.create({ name: "test" });
      let status: rack.Status | undefined;
      await expect
        .poll(async () => {
          const retrieved = await client.racks.retrieve({
            key: r.key,
            includeStatus: true,
          });
          status = retrieved.status;
          return status;
        })
        .toBeDefined();
      expect(status?.owner).toEqual(rack.resourceID(r.key));
    });
    it("should include the status for multiple racks", async () => {
      const r1 = await client.racks.create({ name: "test1" });
      const r2 = await client.racks.create({ name: "test2" });
      let statuses: (rack.Status | undefined)[] = [];
      await expect
        .poll(async () => {
          const retrieved = await client.racks.retrieve({
            key: [r1.key, r2.key],
            includeStatus: true,
          });
          statuses = retrieved.map((r) => r.status);
          return statuses.every((s) => s != null);
        })
        .toBe(true);
      expect(statuses).toHaveLength(2);
      expect(statuses[0]?.owner).toEqual(rack.resourceID(r1.key));
      expect(statuses[1]?.owner).toEqual(rack.resourceID(r2.key));
    });
    it("should compose streamed statuses onto a single query answer", async () => {
      const r = await client.racks.create({ name: "status-compose-single" });
      const params = { key: r.key, includeStatus: true };
      await client.racks.retrieve(params);
      const seen: rack.Rack[] = [];
      const stop = client.racks.watch(params).onChange((cached) => {
        if (query.isLive(cached)) seen.push(cached);
      });
      try {
        await client.statuses.set(
          status.create({
            owner: rack.resourceID(r.key),
            variant: "warning",
            message: "Rack needs attention",
          }),
        );
        await expect
          .poll(() => seen.at(-1)?.status?.message)
          .toBe("Rack needs attention");
        expect(seen.at(-1)?.status?.variant).toBe("warning");
      } finally {
        stop();
      }
    });
    it("should compose streamed statuses onto a request query answer", async () => {
      const r = await client.racks.create({ name: "status-compose-request" });
      const list = client.racks.watch({
        where: { key: { in: [r.key] } },
        includeStatus: true,
      });
      await list.retrieve();
      const seen: rack.Rack[][] = [];
      const stop = list.onChange((w) => {
        if (w != null) seen.push(w.records);
      });
      try {
        await client.statuses.set(
          status.create({
            owner: rack.resourceID(r.key),
            variant: "error",
            message: "Rack went dark",
          }),
        );
        await expect
          .poll(() => seen.at(-1)?.find((item) => item.key === r.key)?.status?.message)
          .toBe("Rack went dark");
      } finally {
        stop();
      }
    });
    describe("rack cached without a status", () => {
      // Outlasts the query layer's refetch debounce.
      const REFETCH_WINDOW = 150;
      const retrieves = (send: ReturnType<typeof spyOnSend>) =>
        send.mock.calls.filter(([target]) => target === "/rack/retrieve");

      it("should fetch the status after a status-free read", async () => {
        const r = await client.racks.create({ name: "status-free-read" });
        const local = createTestClient();
        await local.racks.retrieve(r.key);
        const params = { key: r.key, includeStatus: true };
        expect(local.racks.watch(params).getCached()).toBeUndefined();
        const retrieved = await local.racks.retrieve(params);
        expect(retrieved.status?.owner).toEqual(rack.resourceID(r.key));
      });
      it("should fetch the status after a rack arrives on the stream", async () => {
        const r = await client.racks.create({ name: "stream-cached" });
        const local = createTestClient();
        const stop = local.racks.watch(r.key).onChange(() => {});
        await waitForStreamLive(local.connection);
        try {
          // A rename would also stream the status, so this edit keeps the name.
          await client.racks.dispatch(
            r.key,
            rack.set({ path: ["integrations"], value: ["ni"] }),
          );
          await expect
            .poll(() => {
              const cached = local.racks.watch(r.key).getCached();
              return query.isLive(cached) ? cached.integrations : undefined;
            })
            .toEqual(["ni"]);
          const params = { key: r.key, includeStatus: true };
          expect(local.racks.watch(params).getCached()).toBeUndefined();
          const retrieved = await local.racks.retrieve(params);
          expect(retrieved.status?.owner).toEqual(rack.resourceID(r.key));
        } finally {
          stop();
        }
      });
      it("should fetch the status for a subscriber with no fetch", async () => {
        const r = await client.racks.create({ name: "subscribed" });
        const local = createTestClient();
        await local.racks.retrieve(r.key);
        const seen: (rack.Status | undefined)[] = [];
        const stop = local.racks
          .watch({ key: r.key, includeStatus: true })
          .onChange((cached) => {
            if (query.isLive(cached)) seen.push(cached.status);
          });
        try {
          await expect.poll(() => seen.at(-1)?.owner).toEqual(rack.resourceID(r.key));
          expect(seen.every((st) => st != null)).toBe(true);
        } finally {
          stop();
        }
      });
      it("should send one fetch when a read follows a subscribe", async () => {
        const r = await client.racks.create({ name: "subscribe-then-read" });
        const local = createTestClient();
        await local.racks.retrieve(r.key);
        await waitForStreamLive(local.connection);
        const send = spyOnSend(local);
        const params = { key: r.key, includeStatus: true };
        const stop = local.racks.watch(params).onChange(() => {});
        try {
          const retrieved = await local.racks.retrieve(params);
          expect(retrieved.status?.owner).toEqual(rack.resourceID(r.key));
          await sleep.sleep(REFETCH_WINDOW);
          expect(retrieves(send)).toHaveLength(1);
        } finally {
          stop();
        }
      });
      it("should join a subscriber's fetch that is in flight", async () => {
        const r = await client.racks.create({ name: "read-joins-subscriber" });
        const local = createTestClient();
        await local.racks.retrieve(r.key);
        await waitForStreamLive(local.connection);
        const { responded, release } = holdResponses(local, "/rack/retrieve");
        const send = spyOnSend(local);
        const params = { key: r.key, includeStatus: true };
        const stop = local.racks.watch(params).onChange(() => {});
        try {
          await responded;
          const reading = local.racks.retrieve(params);
          release();
          const retrieved = await reading;
          expect(retrieved.status?.owner).toEqual(rack.resourceID(r.key));
          await sleep.sleep(REFETCH_WINDOW);
          expect(retrieves(send)).toHaveLength(1);
        } finally {
          stop();
        }
      });
      it("should keep the answer of a fetch that found no status", async () => {
        const r = await client.racks.create({ name: "no-status-row" });
        const [st] = await client.statuses.retrieve({
          owners: [rack.resourceID(r.key)],
        });
        await client.statuses.delete(st.key);
        const local = createTestClient();
        await local.racks.retrieve(r.key);
        const params = { key: r.key, includeStatus: true };
        const retrieved = await local.racks.retrieve(params);
        expect(retrieved.status).toBeUndefined();
        const cached = local.racks.watch(params).getCached();
        expect(cached).toBeInstanceOf(rack.Rack);
        expect((cached as rack.Rack).status).toBeUndefined();
      });
    });
    it("should not adopt a status that the rack does not own", async () => {
      const r = await client.racks.create({ name: "status-impostor" });
      const params = { key: r.key, includeStatus: true };
      await client.racks.retrieve(params);
      const seen: (string | undefined)[] = [];
      const stop = client.racks.watch(params).onChange((cached) => {
        if (query.isLive(cached)) seen.push(cached.status?.message);
      });
      const impostor = (message: string) =>
        status.create({ key: id.create(), variant: "loading", message });
      try {
        await client.statuses.set(impostor("Sending start command"));
        await client.statuses.set(impostor("Device reconnecting..."));
        await client.statuses.set(
          status.create({
            owner: rack.resourceID(r.key),
            variant: "success",
            message: "Driver alive",
          }),
        );
        await expect.poll(() => seen.at(-1)).toBe("Driver alive");
        expect(seen).not.toContain("Sending start command");
        expect(seen).not.toContain("Device reconnecting...");
      } finally {
        stop();
      }
    });
  });
});
