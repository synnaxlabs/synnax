// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { eq } from "drizzle-orm";
import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { activation, event, license, organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import {
  expiryFrom,
  hashSecret,
  link,
  type Machine,
  machinesFor,
  mintSecret,
  renew,
  resolve,
  superseded,
  TERM_MS,
  unlink,
} from "@/server/license/desktop";
import {
  activationOf,
  HASH_A,
  HASH_B,
  HASH_C,
  LICENSE,
  NOW,
} from "@/server/license/testutil";
import {
  createActivation,
  createHarness,
  createLicense,
  createOrganization,
  readKey,
} from "@/testutil";

describe("desktop", () => {
  describe("mintSecret", () => {
    it("should mint a URL-safe secret that differs each time", () => {
      const a = mintSecret();
      const b = mintSecret();
      expect(a).toMatch(/^[A-Za-z0-9_-]{43}$/);
      expect(a).not.toBe(b);
    });
  });
  describe("hashSecret", () => {
    it("should hash the same secret to the same digest", () => {
      expect(hashSecret("s")).toBe(hashSecret("s"));
      expect(hashSecret("s")).toMatch(/^[0-9a-f]{64}$/);
      expect(hashSecret("s")).not.toBe(hashSecret("t"));
    });
  });
  describe("expiryFrom", () => {
    it("should put the expiry one term after now", () => {
      expect(expiryFrom(NOW).getTime() - NOW.getTime()).toBe(TERM_MS);
    });
  });
  describe("superseded", () => {
    const machineOf = (fingerprint: string[]): Machine => ({
      activation: activationOf(fingerprint),
      license: LICENSE,
    });
    it("should pick the machines that share any hash", () => {
      const same = machineOf([HASH_A, HASH_B]);
      const other = machineOf([HASH_C]);
      expect(superseded([same, other], [HASH_B])).toEqual([same]);
    });
    it("should pick nothing for a new machine", () => {
      expect(superseded([machineOf([HASH_A])], [HASH_C])).toEqual([]);
    });
  });
});

describe("desktop ledger", () => {
  let store: Memory;
  beforeAll(async () => {
    store = await openMemory();
  });
  beforeEach(async () => await store.clear());

  const signer = () => createHarness(store).portal.signer;
  const LATER = new Date(NOW.getTime() + 60_000);

  const linkMachine = async (fingerprint = [HASH_A], now = NOW) =>
    await link(store, signer(), {
      userID: "user_a",
      userName: "Ada Lovelace",
      fingerprint,
      machineName: "stand-1",
      now,
    });

  const licenseRow = async (key: string) =>
    (await store.query.select().from(license).where(eq(license.key, key)))[0];

  const activationRow = async (key: string) =>
    (await store.query.select().from(activation).where(eq(activation.key, key)))[0];

  const eventsOf = async (kind: string) =>
    (await store.query.select().from(event)).filter((e) => e.kind === kind);

  describe("link", () => {
    it("should issue a one-machine license to the personal organization", async () => {
      const linked = await linkMachine();
      const [org] = await store.query.select().from(organization);
      expect(org).toMatchObject({
        kind: "personal",
        name: "Ada Lovelace",
        ownerUserID: "user_a",
      });
      expect(linked.license).toMatchObject({
        organization: org.key,
        edition: "desktop",
        term: "subscription",
        nodes: 1,
        channels: 0,
        label: "stand-1",
        expiresAt: expiryFrom(NOW),
        maxVersion: null,
        issuedBy: "user_a",
        issuedAt: NOW,
        revokedAt: null,
      });
      expect(linked.activation).toMatchObject({
        license: linked.license.key,
        fingerprint: [HASH_A],
        name: "stand-1",
        renewalSecretHash: hashSecret(linked.secret),
        releasedAt: null,
      });
    });

    it("should sign a license key bound to the machine", async () => {
      const { key, license: lic } = await linkMachine([HASH_A, HASH_B]);
      expect(readKey(key)).toMatchObject({
        jti: lic.key,
        edition: "d",
        machines: 1,
        fingerprints: [HASH_A, HASH_B],
        exp: expiryFrom(NOW).getTime() / 1000,
      });
    });

    it("should record the link", async () => {
      const linked = await linkMachine();
      expect(await eventsOf("link")).toEqual([
        expect.objectContaining({
          at: NOW,
          actor: "user_a",
          organization: linked.license.organization,
          license: linked.license.key,
          activation: linked.activation.key,
          detail: { name: "stand-1", fingerprint: [HASH_A] },
        }),
      ]);
    });

    it("should replace the earlier link of a machine that logs in again", async () => {
      const first = await linkMachine([HASH_A, HASH_B]);
      const second = await linkMachine([HASH_B], LATER);
      expect(await activationRow(first.activation.key)).toMatchObject({
        releasedAt: LATER,
        renewalSecretHash: null,
      });
      expect((await licenseRow(first.license.key)).revokedAt).toEqual(LATER);
      expect(await eventsOf("unlink")).toEqual([
        expect.objectContaining({
          at: LATER,
          actor: "user_a",
          license: first.license.key,
          activation: first.activation.key,
          detail: { reason: "superseded" },
        }),
      ]);
      const machines = await machinesFor(store.query, second.license.organization);
      expect(machines.map((m) => m.activation.key)).toEqual([second.activation.key]);
    });

    it("should keep the link of a different machine", async () => {
      const first = await linkMachine([HASH_A]);
      await linkMachine([HASH_C], LATER);
      const machines = await machinesFor(store.query, first.license.organization);
      expect(machines).toHaveLength(2);
      expect(await eventsOf("unlink")).toHaveLength(0);
    });

    it("should mint a different secret for every link", async () => {
      const first = await linkMachine([HASH_A]);
      const second = await linkMachine([HASH_C]);
      expect(first.secret).not.toBe(second.secret);
    });
  });

  describe("machinesFor", () => {
    it("should list linked desktop machines, most recently seen first", async () => {
      const older = await linkMachine([HASH_A]);
      const newer = await linkMachine([HASH_C], LATER);
      const machines = await machinesFor(store.query, older.license.organization);
      expect(machines.map((m) => m.activation.key)).toEqual([
        newer.activation.key,
        older.activation.key,
      ]);
    });

    it("should leave out enterprise machines and released ones", async () => {
      const linked = await linkMachine([HASH_A]);
      const org = linked.license.organization;
      const enterprise = await createLicense(store, { organization: org });
      await createActivation(store, { license: enterprise.key, fingerprint: [HASH_B] });
      await unlink(store, {
        activationKey: linked.activation.key,
        actor: "user_a",
        now: LATER,
      });
      expect(await machinesFor(store.query, org)).toEqual([]);
    });
  });

  describe("resolve", () => {
    it("should find the machine a secret belongs to", async () => {
      const linked = await linkMachine();
      const machine = await resolve(store, linked.secret);
      expect(machine.activation.key).toBe(linked.activation.key);
      expect(machine.license.key).toBe(linked.license.key);
    });

    it("should refuse an unknown secret", async () => {
      await expect(resolve(store, mintSecret())).rejects.toMatchObject({
        status: 403,
        message: "This machine is not logged in",
      });
    });

    it("should refuse the secret of a superseded link", async () => {
      const first = await linkMachine([HASH_A]);
      await linkMachine([HASH_A], LATER);
      await expect(resolve(store, first.secret)).rejects.toMatchObject({
        status: 403,
        message: "This machine is not logged in",
      });
    });

    it("should refuse a machine whose license was revoked", async () => {
      const linked = await linkMachine();
      await store.query
        .update(license)
        .set({ revokedAt: LATER })
        .where(eq(license.key, linked.license.key));
      await expect(resolve(store, linked.secret)).rejects.toMatchObject({
        status: 403,
        message: "This machine was logged out. Log in again.",
      });
    });

    it("should refuse a machine whose seat was released", async () => {
      const linked = await linkMachine();
      await store.query
        .update(activation)
        .set({ releasedAt: LATER })
        .where(eq(activation.key, linked.activation.key));
      await expect(resolve(store, linked.secret)).rejects.toMatchObject({
        status: 403,
        message: "This machine was logged out. Log in again.",
      });
    });
  });

  describe("renew", () => {
    const RENEWED = new Date(NOW.getTime() + 20 * 24 * 60 * 60 * 1000);

    it("should slide the expiry one term past now and sign a fresh key", async () => {
      const linked = await linkMachine();
      const machine = await resolve(store, linked.secret);
      const { key, license: lic } = await renew(store, signer(), {
        machine,
        now: RENEWED,
      });
      expect(lic.expiresAt).toEqual(expiryFrom(RENEWED));
      expect((await licenseRow(lic.key)).expiresAt).toEqual(expiryFrom(RENEWED));
      expect((await activationRow(linked.activation.key)).lastSeen).toEqual(RENEWED);
      expect(readKey(key)).toMatchObject({
        jti: lic.key,
        fingerprints: [HASH_A],
        exp: expiryFrom(RENEWED).getTime() / 1000,
      });
    });

    it("should record the renewal with the machine as its actor", async () => {
      const linked = await linkMachine();
      const machine = await resolve(store, linked.secret);
      await renew(store, signer(), { machine, now: RENEWED });
      expect(await eventsOf("renew")).toEqual([
        expect.objectContaining({
          at: RENEWED,
          actor: linked.activation.key,
          organization: linked.license.organization,
          license: linked.license.key,
          activation: linked.activation.key,
          detail: {},
        }),
      ]);
    });

    it("should refuse a machine unlinked after it was resolved", async () => {
      const linked = await linkMachine();
      const machine = await resolve(store, linked.secret);
      await unlink(store, {
        activationKey: linked.activation.key,
        actor: "user_a",
        now: LATER,
      });
      await expect(
        renew(store, signer(), { machine, now: RENEWED }),
      ).rejects.toMatchObject({
        status: 403,
        message: "This machine was logged out. Log in again.",
      });
      expect(await eventsOf("renew")).toHaveLength(0);
    });
  });

  describe("unlink", () => {
    it("should release the seat, drop the secret, and revoke the license", async () => {
      const linked = await linkMachine();
      await unlink(store, {
        activationKey: linked.activation.key,
        actor: "user_a",
        now: LATER,
      });
      expect(await activationRow(linked.activation.key)).toMatchObject({
        releasedAt: LATER,
        renewalSecretHash: null,
      });
      expect((await licenseRow(linked.license.key)).revokedAt).toEqual(LATER);
      expect(await eventsOf("unlink")).toEqual([
        expect.objectContaining({
          at: LATER,
          actor: "user_a",
          license: linked.license.key,
          activation: linked.activation.key,
          detail: {},
        }),
      ]);
    });

    it("should throw a 404 for an unknown machine", async () => {
      await expect(
        unlink(store, {
          activationKey: crypto.randomUUID(),
          actor: "user_a",
          now: NOW,
        }),
      ).rejects.toMatchObject({ status: 404, message: "Activation not found" });
    });

    it("should refuse an enterprise machine and leave its license", async () => {
      const org = await createOrganization(store, {
        kind: "team",
        clerkOrgID: "org_a",
      });
      const lic = await createLicense(store, { organization: org.key });
      const act = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
      });
      await expect(
        unlink(store, { activationKey: act.key, actor: "user_a", now: NOW }),
      ).rejects.toMatchObject({
        status: 400,
        message: "Only a Synnax Desktop machine can be logged out",
      });
      expect((await licenseRow(lic.key)).revokedAt).toBeNull();
      expect((await activationRow(act.key)).releasedAt).toBeNull();
    });
  });
});
