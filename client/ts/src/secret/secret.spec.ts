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
import { createTestClient, waitForStreamLive } from "@/testutil";

describe("Secret", () => {
  describe("create", () => {
    it("should create a stored secret", async () => {
      const client = createTestClient();
      const s = await client.secrets.create({
        name: "Broker password",
        source: "stored",
        value: "hunter2",
      });
      expect(s).toMatchObject({ name: "Broker password", source: "stored" });
      expect(s.updated.valueOf()).toBeGreaterThan(0n);
    });

    it("should create a file secret", async () => {
      const client = createTestClient();
      const s = await client.secrets.create({
        name: "Client key",
        source: "file",
        path: "/etc/synnax/client.key",
      });
      expect(s).toMatchObject({
        name: "Client key",
        source: "file",
        path: "/etc/synnax/client.key",
      });
    });

    it("should create multiple secrets", async () => {
      const client = createTestClient();
      const secrets = await client.secrets.create([
        { name: "Token", source: "stored", value: "abc" },
        { name: "Key file", source: "file", path: "/tmp/key.pem" },
      ]);
      expect(secrets).toHaveLength(2);
      expect(secrets[0]).toMatchObject({ name: "Token", source: "stored" });
      expect(secrets[1]).toMatchObject({ name: "Key file", source: "file" });
    });

    it("should replace a secret that has the same key", async () => {
      const client = createTestClient();
      const first = await client.secrets.create({
        name: "API key",
        source: "stored",
        value: "one",
      });
      const second = await client.secrets.create({
        key: first.key,
        name: "API key v2",
        source: "file",
        path: "/opt/api.key",
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
      const created = await client.secrets.create({
        name: "Retrieve",
        source: "stored",
        value: "abc",
      });
      const retrieved = await createTestClient().secrets.retrieve(created.key);
      expect(retrieved).toEqual(created);
    });

    it("should retrieve multiple secrets", async () => {
      const client = createTestClient();
      const created = await client.secrets.create([
        { name: "Multi 1", source: "stored", value: "a" },
        { name: "Multi 2", source: "stored", value: "b" },
      ]);
      const keys = created.map((s) => s.key);
      const retrieved = await createTestClient().secrets.retrieve({ keys });
      expect(retrieved.map((s) => s.name).sort()).toEqual(["Multi 1", "Multi 2"]);
    });

    it("should retrieve empty uses for a secret that nothing uses", async () => {
      const client = createTestClient();
      const s = await client.secrets.create({
        name: "Unused",
        source: "stored",
        value: "abc",
      });
      const [retrieved] = await client.secrets.retrieve({
        keys: [s.key],
        includeUses: true,
      });
      expect(retrieved.uses).toEqual([]);
      expect((await client.secrets.retrieve(s.key)).uses).toBeUndefined();
    });

    it("should search for secrets by name", async () => {
      const client = createTestClient();
      const name = `${id.create()} password`;
      const created = await client.secrets.create({
        name,
        source: "stored",
        value: "abc",
      });
      await expect
        .poll(async () => await client.secrets.retrieve({ searchTerm: name }))
        .toContainEqual(created);
    });

    it("should throw a NotFoundError for a missing key", async () => {
      const key = uuid.create();
      const res = createTestClient().secrets.retrieve(key);
      await expect(res).rejects.toThrow(NotFoundError);
      await expect(res).rejects.toThrow(`Secrets with keys [${key}] not found`);
    });
  });

  describe("rename", () => {
    it("should rename a secret", async () => {
      const client = createTestClient();
      const s = await client.secrets.create({
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
      const s = await client.secrets.create({
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
      const s = await client.secrets.create({
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

  describe("delete", () => {
    it("should delete secrets", async () => {
      const client = createTestClient();
      const secrets = await client.secrets.create([
        { name: "Delete 1", source: "stored", value: "a" },
        { name: "Delete 2", source: "file", path: "/tmp/b" },
      ]);
      const keys = secrets.map((s) => s.key);
      await client.secrets.delete(keys);
      const res = createTestClient().secrets.retrieve(keys[0]);
      await expect(res).rejects.toThrow(NotFoundError);
      await expect(res).rejects.toThrow(`Secrets with keys [${keys[0]}] not found`);
    });
  });

  describe("grant", () => {
    it("should return the key of a new grant", async () => {
      const client = createTestClient();
      const rack = await client.racks.create({ name: "Grant rack" });
      const s = await client.secrets.create({
        name: "Grant",
        source: "stored",
        value: "abc",
      });
      const first = await client.secrets.grant({ rack: rack.key, secrets: [s.key] });
      const second = await client.secrets.grant({ rack: rack.key, secrets: [s.key] });
      expect(first).toHaveLength(36);
      expect(second).not.toEqual(first);
    });

    it("should throw a NotFoundError for a missing secret", async () => {
      const client = createTestClient();
      const rack = await client.racks.create({ name: "Grant rack" });
      const key = uuid.create();
      const grant = client.secrets.grant({ rack: rack.key, secrets: [key] });
      await expect(grant).rejects.toThrow(NotFoundError);
      await expect(grant).rejects.toThrow(`Secrets with keys [${key}] not found`);
    });
  });

  describe("revoke", () => {
    it("should revoke one grant or many, and ignore unknown keys", async () => {
      const client = createTestClient();
      const rack = await client.racks.create({ name: "Revoke rack" });
      const s = await client.secrets.create({
        name: "Revoke",
        source: "stored",
        value: "abc",
      });
      const first = await client.secrets.grant({ rack: rack.key, secrets: [s.key] });
      const second = await client.secrets.grant({ rack: rack.key, secrets: [s.key] });
      await expect(client.secrets.revoke(first)).resolves.toBeUndefined();
      await expect(
        client.secrets.revoke([first, second, uuid.create()]),
      ).resolves.toBeUndefined();
    });
  });
});
