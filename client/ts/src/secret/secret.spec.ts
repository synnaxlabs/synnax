// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError, PathError } from "@/index";
import { rack } from "@/rack";
import { secret } from "@/secret";
import { createTestClient, spyOnSend, waitForStreamLive } from "@/testutil";

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
});
