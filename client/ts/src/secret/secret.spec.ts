// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, sleep, TimeSpan, uuid } from "@synnaxlabs/x";
import { assert, describe, expect, it, vi } from "vitest";

import { NotFoundError, PathError } from "@/index";
import { type query } from "@/query";
import { type secret } from "@/secret";
import {
  createTestClient,
  expectDeleted,
  expectLive,
  waitForStreamLive,
} from "@/testutil";

// A self-signed P-256 certificate valid until 2126, and its key.
const PEM = `-----BEGIN CERTIFICATE-----
MIIBnDCCAUKgAwIBAgIUTlojETDboAh4nmfahlzAemoZ4g4wCgYIKoZIzj0EAwIw
MzEbMBkGA1UEAwwSU3lubmF4IFRlc3QgQ2xpZW50MRQwEgYDVQQKDAtTeW5uYXgg
TGFiczAgFw0yNjExMDQwMjUyMTBaGA8yMTI2MTAxMTAyNTIxMFowMzEbMBkGA1UE
AwwSU3lubmF4IFRlc3QgQ2xpZW50MRQwEgYDVQQKDAtTeW5uYXggTGFiczBZMBMG
ByqGSM49AgEGCCqGSM49AwEHA0IABAttV7/ZZaP/ck5ZSEibyz8Bq/FOha2BqRFx
KcHLLjh+GByF19Xf9tZdf+9TfDlOPwyGPp+zArW6WOjeHdzzPv6jMjAwMB0GA1Ud
DgQWBBQnJ8GnAtkFwtBOUq0b4+pWyuXWQTAPBgNVHRMBAf8EBTADAQH/MAoGCCqG
SM49BAMCA0gAMEUCIQDucqkUixw8A44mdBmPFZIhYtq3tzNMzPF5zPD9ONNZPwIg
I3WFxQbCO86E+bgOOHH1NSjmhCFtORY5jSOGw6yL3m4=
-----END CERTIFICATE-----
`;
const PRIVATE_KEY = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgjJtOV99NCzRnmMoB
wui9Lv+9a8aqZ+LwPX2PyGfVEa+hRANCAAQLbVe/2WWj/3JOWUhIm8s/AavxToWt
gakRcSnByy44fhgchdfV3/bWXX/vU3w5Tj8Mhj6fswK1uljo3h3c8z7+
-----END PRIVATE KEY-----
`;

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

    it("should retrieve the uses of a secret by key", async () => {
      const client = createTestClient();
      const s = await client.secrets.create({
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

    it("should refetch the uses of a secret for a new subscriber", async () => {
      const client = createTestClient();
      await waitForStreamLive(client.connection);
      const c = await client.certificates.create({
        name: "Gap",
        source: "stored",
        pem: PEM,
        privateKey: PRIVATE_KEY,
      });
      assert(c.source === "stored" && c.privateKey != null);
      const { privateKey } = c;
      const q = { key: privateKey, includeUses: true };
      const off = client.secrets.onChange(q, () => {});
      const before = await client.secrets.retrieve(q);
      expect(before.uses).toEqual([{ type: "certificate", key: c.key }]);
      off();
      // Past the teardown grace window, so nothing maintains the answer.
      await sleep.sleep(TimeSpan.seconds(8));
      await client.certificates.delete(c.key);
      await vi.waitFor(() => expectDeleted(client.secrets.getCached(privateKey)));
      await client.secrets.create({
        key: privateKey,
        name: "Gap",
        source: "stored",
        value: "abc",
      });
      const offAgain = client.secrets.onChange(q, () => {});
      try {
        await vi.waitFor(() =>
          expect(expectLive(client.secrets.getCached(q)).uses).toEqual([]),
        );
      } finally {
        offAgain();
      }
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

    it("should deliver the deletion of a secret read with its uses", async () => {
      const internal: Error[] = [];
      const client = createTestClient({ onInternalError: (e) => internal.push(e) });
      const s = await client.secrets.create({
        name: "Deleted with uses",
        source: "stored",
        value: "abc",
      });
      const q = { key: s.key, includeUses: true };
      const results: Array<query.Cached<secret.Secret> | undefined> = [];
      const off = client.secrets.onChange(q, (res) => results.push(res));
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
