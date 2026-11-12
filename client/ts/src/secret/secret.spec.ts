// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { id, Series, sleep, TimeSpan, uuid } from "@synnaxlabs/x";
import { assert, describe, expect, it, onTestFinished, vi } from "vitest";
import { type z } from "zod";

import { framer } from "@/framer";
import { NotFoundError, PathError } from "@/index";
import { query } from "@/query";
import { rack } from "@/rack";
import { secret } from "@/secret";
import { task } from "@/task";
import {
  CERTIFICATE,
  createTestClient,
  expectDeleted,
  expectLive,
  holdResponses,
  PushStreamer,
  spyOnSend,
  waitForStreamLive,
} from "@/testutil";

const { pem: PEM, privateKeyPem: PRIVATE_KEY } = CERTIFICATE;

describe("Secret", () => {
  describe("set", () => {
    it("should create a stored secret", async () => {
      const client = createTestClient();
      const s = await client.secrets.set({
        name: "Broker password",
        source: "stored",
        value: "hunter2",
      });
      expect(s).toMatchObject({ name: "Broker password", source: "stored" });
      expect(s.updated.valueOf()).toBeGreaterThan(0n);
    });

    it("should create a file secret", async () => {
      const client = createTestClient();
      const s = await client.secrets.set({
        name: "Client key",
        source: "file",
        fileName: "client.key",
      });
      expect(s).toMatchObject({
        name: "Client key",
        source: "file",
        fileName: "client.key",
      });
    });

    it("should create multiple secrets", async () => {
      const client = createTestClient();
      const secrets = await client.secrets.set([
        { name: "Token", source: "stored", value: "abc" },
        { name: "Key file", source: "file", fileName: "key.pem" },
      ]);
      expect(secrets).toHaveLength(2);
      expect(secrets[0]).toMatchObject({ name: "Token", source: "stored" });
      expect(secrets[1]).toMatchObject({ name: "Key file", source: "file" });
    });

    it("should replace a secret that has the same key", async () => {
      const client = createTestClient();
      const first = await client.secrets.set({
        name: "API key",
        source: "stored",
        value: "one",
      });
      const second = await client.secrets.set({
        key: first.key,
        name: "API key v2",
        source: "stored",
        value: "two",
      });
      expect(second.key).toEqual(first.key);
      expect(second.updated.after(first.updated)).toBe(true);
      const retrieved = await createTestClient().secrets.retrieve({ key: first.key });
      expect(retrieved).toEqual(second);
    });
  });

  describe("retrieve", () => {
    it("should retrieve a secret by key", async () => {
      const client = createTestClient();
      const created = await client.secrets.set({
        name: "Retrieve",
        source: "stored",
        value: "abc",
      });
      const retrieved = await createTestClient().secrets.retrieve(created.key);
      expect(retrieved).toEqual(created);
    });

    it("should retrieve multiple secrets", async () => {
      const client = createTestClient();
      const created = await client.secrets.set([
        { name: "Multi 1", source: "stored", value: "a" },
        { name: "Multi 2", source: "stored", value: "b" },
      ]);
      const keys = created.map((s) => s.key);
      const retrieved = await createTestClient().secrets.retrieve(keys);
      expect(retrieved.map((s) => s.name).sort()).toEqual(["Multi 1", "Multi 2"]);
    });

    it("should leave out a key that no secret has in a filter", async () => {
      const client = createTestClient();
      const s = await client.secrets.set({
        name: "Kept",
        source: "stored",
        value: "a",
      });
      const retrieved = await createTestClient().secrets.retrieve({
        key: [s.key, uuid.create()],
      });
      expect(retrieved.map((r) => r.key)).toEqual([s.key]);
    });

    it("should retrieve empty uses for a secret that nothing uses", async () => {
      const client = createTestClient();
      const s = await client.secrets.set({
        name: "Unused",
        source: "stored",
        value: "abc",
      });
      const [retrieved] = await client.secrets.retrieve({
        key: [s.key],
        includeUses: true,
      });
      expect(retrieved.uses).toEqual([]);
      expect((await client.secrets.retrieve(s.key)).uses).toBeUndefined();
    });

    it("should retrieve the uses of a secret by key", async () => {
      const client = createTestClient();
      const s = await client.secrets.set({
        name: "Unused by key",
        source: "stored",
        value: "abc",
      });
      const retrieved = await client.secrets.retrieve({
        key: s.key,
        includeUses: true,
      });
      expect(retrieved).toEqual({ ...s, uses: [] });
      expect((await client.secrets.retrieve({ key: s.key })).uses).toBeUndefined();
    });

    it("should read the uses of a secret created after a watched list", async () => {
      const client = createTestClient();
      await waitForStreamLive(client.connection);
      const list = client.secrets.watch({ includeUses: true });
      const off = list.onChange(() => {});
      await list.retrieve();
      try {
        const c = await createTestClient().certificates.set({
          name: "Watched uses",
          source: "stored",
          pem: PEM,
          privateKeyPem: PRIVATE_KEY,
        });
        assert(c.source === "stored" && c.privateKey != null);
        const { privateKey } = c;
        await expect
          .poll(() => list.getCached()?.records.find((r) => r.key === privateKey)?.uses)
          .toEqual([{ type: "certificate", key: c.key }]);
      } finally {
        off();
      }
    });

    it("should read fresh uses in a list for a secret created again", async () => {
      const client = createTestClient();
      await waitForStreamLive(client.connection);
      const list = client.secrets.watch({ includeUses: true });
      const off = list.onChange(() => {});
      await list.retrieve();
      const usesOf = (key: secret.Key) =>
        list.getCached()?.records.find((r) => r.key === key)?.uses;
      try {
        const c = await client.certificates.set({
          name: "Recreated key",
          source: "stored",
          pem: PEM,
          privateKeyPem: PRIVATE_KEY,
        });
        assert(c.source === "stored" && c.privateKey != null);
        const { privateKey } = c;
        await expect
          .poll(() => usesOf(privateKey))
          .toEqual([{ type: "certificate", key: c.key }]);
        await client.certificates.delete(c.key);
        await expect.poll(() => usesOf(privateKey)).toBeUndefined();
        await client.secrets.set({
          key: privateKey,
          name: "Recreated key",
          source: "stored",
          value: "abc",
        });
        await expect.poll(() => usesOf(privateKey)).toEqual([]);
      } finally {
        off();
      }
    });

    it("should drop the uses a list reads before a delete of the secret", async () => {
      const client = createTestClient();
      await waitForStreamLive(client.connection);
      const c = await client.certificates.set({
        name: "Overtaken key",
        source: "stored",
        pem: PEM,
        privateKeyPem: PRIVATE_KEY,
      });
      assert(c.source === "stored" && c.privateKey != null);
      const { privateKey } = c;
      await client.secrets.retrieve(privateKey);
      const { responded, release } = holdResponses(client, "/secret/retrieve");
      const list = client.secrets.watch({ includeUses: true });
      const off = list.onChange(() => {});
      try {
        const fetched = list.retrieve();
        await responded;
        await client.certificates.delete(c.key);
        await vi.waitFor(() =>
          expectDeleted(client.secrets.watch(privateKey).getCached()),
        );
        release();
        await fetched;
        await client.secrets.set({
          key: privateKey,
          name: "Overtaken key",
          source: "stored",
          value: "abc",
        });
        await expect
          .poll(() => list.getCached()?.records.find((r) => r.key === privateKey)?.uses)
          .toEqual([]);
      } finally {
        off();
      }
    });

    it("should drop the uses a list reads before a secret is deleted and set again", async () => {
      const client = createTestClient();
      await waitForStreamLive(client.connection);
      const c = await client.certificates.set({
        name: "Overtaken key",
        source: "stored",
        pem: PEM,
        privateKeyPem: PRIVATE_KEY,
      });
      assert(c.source === "stored" && c.privateKey != null);
      const { privateKey } = c;
      await client.secrets.retrieve(privateKey);
      const { responded, release } = holdResponses(client, "/secret/retrieve");
      const list = client.secrets.watch({ includeUses: true });
      const off = list.onChange(() => {});
      try {
        const fetched = list.retrieve();
        await responded;
        await client.certificates.delete(c.key);
        await vi.waitFor(() =>
          expectDeleted(client.secrets.watch(privateKey).getCached()),
        );
        await client.secrets.set({
          key: privateKey,
          name: "Overtaken key",
          source: "stored",
          value: "abc",
        });
        release();
        await fetched;
        await expect
          .poll(() => list.getCached()?.records.find((r) => r.key === privateKey)?.uses)
          .toEqual([]);
      } finally {
        off();
      }
    });

    it("should page a list that reads uses with a cursor", async () => {
      const client = createTestClient();
      const prefix = id.create();
      for (const i of [0, 1])
        await client.secrets.set({
          name: `${prefix}-${i}`,
          source: "stored",
          value: "abc",
        });
      const send = spyOnSend(client);
      const list = client.secrets.watch({
        name: { startsWith: prefix },
        includeUses: true,
        limit: 1,
      });
      const off = list.onChange(() => {});
      try {
        await list.retrieve();
        const window = await list.more(1);
        expect(window.records.map(({ uses }) => uses)).toEqual([[], []]);
        const afters = send.mock.calls
          .filter(([target]) => target === "/secret/retrieve")
          .map(([, req]) => (req as { after?: unknown }).after);
        expect(afters).toEqual([undefined, expect.anything()]);
      } finally {
        off();
      }
    });

    it("should refetch the uses of a secret for a new subscriber", async () => {
      const client = createTestClient();
      await waitForStreamLive(client.connection);
      const c = await client.certificates.set({
        name: "Gap",
        source: "stored",
        pem: PEM,
        privateKeyPem: PRIVATE_KEY,
      });
      assert(c.source === "stored" && c.privateKey != null);
      const { privateKey } = c;
      const q = { key: privateKey, includeUses: true };
      const off = client.secrets.watch(q).onChange(() => {});
      const before = await client.secrets.retrieve(q);
      expect(before.uses).toEqual([{ type: "certificate", key: c.key }]);
      vi.useFakeTimers({
        toFake: ["setTimeout", "clearTimeout", "Date"],
        shouldAdvanceTime: true,
      });
      try {
        off();
        // Past the teardown grace window, so nothing maintains the answer.
        await vi.advanceTimersByTimeAsync(TimeSpan.seconds(6).milliseconds);
      } finally {
        vi.useRealTimers();
      }
      await client.certificates.delete(c.key);
      await vi.waitFor(() =>
        expectDeleted(client.secrets.watch(privateKey).getCached()),
      );
      await client.secrets.set({
        key: privateKey,
        name: "Gap",
        source: "stored",
        value: "abc",
      });
      const offAgain = client.secrets.watch(q).onChange(() => {});
      try {
        await vi.waitFor(() =>
          expect(expectLive(client.secrets.watch(q).getCached()).uses).toEqual([]),
        );
      } finally {
        offAgain();
      }
    });

    it("should update the uses of a secret when a task starts using it", async () => {
      const client = createTestClient();
      await waitForStreamLive(client.connection);
      const s = await client.secrets.set({
        name: "Routing key",
        source: "stored",
        value: "abc",
      });
      const q = { key: s.key, includeUses: true };
      const off = client.secrets.watch(q).onChange(() => {});
      try {
        expect((await client.secrets.retrieve(q)).uses).toEqual([]);
        const r = await client.racks.create({ name: "Uses" });
        const t = await r.createTask({
          name: "Alerts",
          type: "pagerduty_alert",
          config: { routingKey: s.key },
        });
        await vi.waitFor(() =>
          expect(expectLive(client.secrets.watch(q).getCached()).uses).toEqual([
            task.resourceID(t.key),
          ]),
        );
        await client.tasks.delete(t.key);
        await vi.waitFor(() =>
          expect(expectLive(client.secrets.watch(q).getCached()).uses).toEqual([]),
        );
      } finally {
        off();
      }
    });

    it("should refetch uses that change while a list reads them", async () => {
      const client = createTestClient();
      await waitForStreamLive(client.connection);
      const s = await client.secrets.set({
        name: "Listed routing key",
        source: "stored",
        value: "abc",
      });
      const held = holdResponses(client, "/secret/retrieve");
      const list = client.secrets.watch({ includeUses: true });
      const off = list.onChange(() => {});
      const usesOf = () => list.getCached()?.records.find((r) => r.key === s.key)?.uses;
      try {
        const fetched = list.retrieve();
        await held.responded;
        const r = await client.racks.create({ name: "Listed uses" });
        const t = await r.createTask({
          name: "Alerts",
          type: "pagerduty_alert",
          config: { routingKey: s.key },
        });
        // Long enough for the task change to reach the client.
        await sleep.sleep(TimeSpan.seconds(1));
        held.release();
        await fetched;
        await vi.waitFor(() => expect(usesOf()).toEqual([task.resourceID(t.key)]));
      } finally {
        off();
      }
    });

    it("should apply refetched uses in the order of the changes", async () => {
      const client = createTestClient();
      await waitForStreamLive(client.connection);
      const s = await client.secrets.set({
        name: "Ordered routing key",
        source: "stored",
        value: "abc",
      });
      const q = { key: s.key, includeUses: true };
      const off = client.secrets.watch(q).onChange(() => {});
      try {
        expect((await client.secrets.retrieve(q)).uses).toEqual([]);
        const held = holdResponses(client, "/secret/retrieve");
        const r = await client.racks.create({ name: "Ordered uses" });
        const t = await r.createTask({
          name: "Alerts",
          type: "pagerduty_alert",
          config: { routingKey: s.key },
        });
        await held.responded;
        await client.tasks.delete(t.key);
        // Long enough for the deletion to reach the client.
        await sleep.sleep(TimeSpan.seconds(1));
        expect(held.sent()).toBe(1);
        held.release();
        await vi.waitFor(() =>
          expect(expectLive(client.secrets.watch(q).getCached()).uses).toEqual([]),
        );
      } finally {
        off();
      }
    });

    it("should apply other changes while it refetches uses", async () => {
      const client = createTestClient();
      await waitForStreamLive(client.connection);
      const s = await client.secrets.set({
        name: "Waiting routing key",
        source: "stored",
        value: "abc",
      });
      const q = { key: s.key, includeUses: true };
      const offUses = client.secrets.watch(q).onChange(() => {});
      const off = client.secrets.watch(s.key).onChange(() => {});
      try {
        await client.secrets.retrieve(q);
        const held = holdResponses(client, "/secret/retrieve");
        const r = await client.racks.create({ name: "Waiting uses" });
        await r.createTask({
          name: "Alerts",
          type: "pagerduty_alert",
          config: { routingKey: s.key },
        });
        await held.responded;
        await createTestClient().secrets.rename(s.key, "Renamed routing key");
        await expect
          .poll(() => expectLive(client.secrets.watch(s.key).getCached()).name)
          .toBe("Renamed routing key");
        held.release();
      } finally {
        off();
        offUses();
      }
    });

    it("should report no error for a uses refetch that a close stops", async () => {
      const internal: Error[] = [];
      const client = createTestClient({
        onInternalError: (error) => internal.push(error),
      });
      await waitForStreamLive(client.connection);
      const s = await client.secrets.set({
        name: "Closed routing key",
        source: "stored",
        value: "abc",
      });
      const q = { key: s.key, includeUses: true };
      client.secrets.watch(q).onChange(() => {});
      await client.secrets.retrieve(q);
      const held = holdResponses(client, "/secret/retrieve");
      const r = await client.racks.create({ name: "Closed uses" });
      await r.createTask({
        name: "Alerts",
        type: "pagerduty_alert",
        config: { routingKey: s.key },
      });
      await held.responded;
      await client.close();
      held.release();
      // Long enough for the refetch to fail.
      await sleep.sleep(TimeSpan.milliseconds(500));
      expect(internal).toEqual([]);
    });

    it("should search for secrets by name", async () => {
      const client = createTestClient();
      const name = `${id.create()} password`;
      const created = await client.secrets.set({
        name,
        source: "stored",
        value: "abc",
      });
      await expect
        .poll(async () => await client.secrets.retrieve({ searchTerm: name }))
        .toContainEqual(created);
    });

    it("should retrieve a secret by name", async () => {
      const name = `${id.create()} token`;
      const created = await createTestClient().secrets.set({
        name,
        source: "stored",
        value: "abc",
      });
      expect(await createTestClient().secrets.retrieve(name)).toEqual(created);
    });

    it("should page secrets in name order with a cursor", async () => {
      const prefix = id.create();
      const created = await createTestClient().secrets.set(
        ["C", "A", "B"].map((n) => ({
          name: `${prefix} ${n}`,
          source: "stored" as const,
          value: "v",
        })),
      );
      const keys = created.map((s) => s.key);
      const order = { field: "name" as const };
      const client = createTestClient();
      const first = await client.secrets.retrieve({ key: keys, order, limit: 2 });
      expect(first.map((s) => s.name)).toEqual([`${prefix} A`, `${prefix} B`]);
      const last = first[1];
      const second = await client.secrets.retrieve({
        key: keys,
        order,
        after: { key: last.key, name: last.name },
      });
      expect(second.map((s) => s.name)).toEqual([`${prefix} C`]);
    });

    it("should add a new secret to a watched list", async () => {
      const prefix = id.create();
      const client = createTestClient();
      const list = client.secrets.watch({ searchTerm: prefix });
      await list.retrieve();
      const seen: string[][] = [];
      const stop = list.onChange((w) => {
        if (w != null) seen.push(w.records.map((s) => s.name));
      });
      try {
        await createTestClient().secrets.set({
          name: `${prefix} watched`,
          source: "stored",
          value: "v",
        });
        await expect.poll(() => seen.at(-1)).toContain(`${prefix} watched`);
      } finally {
        stop();
      }
    });

    it("should throw a NotFoundError for a missing key", async () => {
      const key = uuid.create();
      const res = createTestClient().secrets.retrieve(key);
      await expect(res).rejects.toThrow(NotFoundError);
      await expect(res).rejects.toThrow(`secret with key ${key} not found`);
    });
  });

  describe("rename", () => {
    it("should rename a secret", async () => {
      const client = createTestClient();
      const s = await client.secrets.set({
        name: "Before",
        source: "stored",
        value: "abc",
      });
      await client.secrets.rename(s.key, "After");
      const retrieved = await createTestClient().secrets.retrieve({ key: s.key });
      expect(retrieved).toMatchObject({ name: "After", source: "stored" });
    });

    it("should update the cache of another client", async () => {
      const client = createTestClient();
      const remote = createTestClient();
      const s = await client.secrets.set({
        name: "Remote",
        source: "stored",
        value: "abc",
      });
      expect((await client.secrets.retrieve({ key: s.key })).name).toBe("Remote");
      const name = `renamed-${id.create()}`;
      await waitForStreamLive(client.connection);
      await remote.secrets.rename(s.key, name);
      await expect
        .poll(async () => (await client.secrets.retrieve({ key: s.key })).name)
        .toBe(name);
    });

    it("should reject an empty name", async () => {
      const client = createTestClient();
      const s = await client.secrets.set({
        name: "Keep",
        source: "stored",
        value: "abc",
      });
      const rename = client.secrets.rename(s.key, "");
      await expect(rename).rejects.toThrow(PathError);
      await expect(rename).rejects.toThrow("name: required");
      expect((await client.secrets.retrieve({ key: s.key })).name).toBe("Keep");
    });
  });

  describe("dispatch", () => {
    it("should refuse a set of the value before it sends", async () => {
      const client = createTestClient();
      const s = await client.secrets.set({
        name: "Hidden",
        source: "stored",
        value: "abc",
      });
      const send = spyOnSend(client);
      await expect(
        client.secrets.dispatch(s.key, secret.set({ path: ["value"], value: "x" })),
      ).rejects.toThrow('no editable field at "value"');
      expect(send).not.toHaveBeenCalled();
    });
  });

  describe("delete", () => {
    it("should delete secrets", async () => {
      const client = createTestClient();
      const secrets = await client.secrets.set([
        { name: "Delete 1", source: "stored", value: "a" },
        { name: "Delete 2", source: "file", fileName: "b" },
      ]);
      const keys = secrets.map((s) => s.key);
      await client.secrets.delete(keys);
      const res = createTestClient().secrets.retrieve(keys[0]);
      await expect(res).rejects.toThrow(NotFoundError);
      await expect(res).rejects.toThrow(`secret with key ${keys[0]} not found`);
    });

    it("should deliver the deletion of a secret read with its uses", async () => {
      const internal: Error[] = [];
      const client = createTestClient({ onInternalError: (e) => internal.push(e) });
      const s = await client.secrets.set({
        name: "Deleted with uses",
        source: "stored",
        value: "abc",
      });
      const q = { key: s.key, includeUses: true };
      const results: Array<query.Cached<secret.Secret> | undefined> = [];
      const off = client.secrets.watch(q).onChange((res) => results.push(res));
      try {
        // The read joins the fetch of the subscription, so that fetch has settled once
        // the read rejects.
        const reading = expect(client.secrets.retrieve(q)).rejects.toThrow(
          NotFoundError,
        );
        await client.secrets.delete(s.key);
        await reading;
        expect(expectDeleted(results.at(-1)).corpse.key).toEqual(s.key);
        expect(internal).toEqual([]);
      } finally {
        off();
      }
    });
  });

  describe("grant", () => {
    it("should return the key of a new grant", async () => {
      const client = createTestClient();
      const r = await client.racks.create({ name: "Grant rack" });
      const s = await client.secrets.set({
        name: "Grant",
        source: "stored",
        value: "abc",
      });
      const params = { subject: rack.resourceID(r.key), secrets: [s.key] };
      const first = await client.secrets.grant(params);
      const second = await client.secrets.grant(params);
      expect(first).toHaveLength(36);
      expect(second).not.toEqual(first);
    });

    it("should throw a NotFoundError for a missing secret", async () => {
      const client = createTestClient();
      const r = await client.racks.create({ name: "Grant rack" });
      const key = uuid.create();
      const grant = client.secrets.grant({
        subject: rack.resourceID(r.key),
        secrets: [key],
      });
      await expect(grant).rejects.toThrow(NotFoundError);
      await expect(grant).rejects.toThrow(`Secrets with keys [${key}] not found`);
    });
  });

  describe("record cap", () => {
    it("should keep the uses of a held secret when it trims uses", async () => {
      const uses = [{ type: "certificate", key: "held" }];
      const send = async (
        target: string,
        req: { where: { key: { eq: secret.Key } } },
        _: z.ZodType,
        res: z.ZodType,
      ) => {
        if (target !== "/secret/retrieve")
          throw new Error(`unexpected request to ${target}`);
        const { eq: key } = req.where.key;
        return res.parse({
          secrets: [{ key, name: key, source: "stored", updated: 0, uses }],
        });
      };
      const unary = { use: () => {}, send } as unknown as UnaryClient;
      const stream = new PushStreamer();
      const cache = new query.Cache({
        openStreamer: async () => stream,
        breaker: { baseInterval: TimeSpan.milliseconds(1) },
        onError: () => {},
        recordCap: 4,
      });
      onTestFinished(async () => await cache.close());
      const secrets = new secret.Client({ unary, cache });
      await cache.ensureStreaming();
      const [held, ...others] = Array.from({ length: 6 }, () => uuid.create());
      const q = { key: held, includeUses: true };
      onTestFinished(secrets.watch(q).onChange(() => {}));
      await secrets.retrieve(q);
      // The fifth read trims once. The sixth trims again and drops the oldest uses.
      for (const key of others) await secrets.retrieve({ key, includeUses: true });
      stream.push(
        new framer.Frame({
          [secret.ACTION_CHANNEL_NAME]: new Series([
            {
              key: held,
              dispatchKey: "remote-dispatch",
              seq: 1,
              actions: [
                secret.create({
                  secret: { key: held, name: "Renamed", source: "stored", updated: 0 },
                }),
              ],
            },
          ]),
        }),
      );
      await expect
        .poll(() => expectLive(secrets.watch(q).getCached()).name)
        .toBe("Renamed");
      expect(expectLive(secrets.watch(q).getCached()).uses).toEqual(uses);
    });
  });
});
