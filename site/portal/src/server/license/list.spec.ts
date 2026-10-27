// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { type schema } from "@/server/db";
import { type Memory, openMemory } from "@/server/db/testutil";
import {
  findActivatable,
  findSeat,
  listActivatable,
  listAll,
  listAllHeld,
  listForOrganization,
} from "@/server/license/list";
import { HASH_A, HASH_B, HASH_C, NOW } from "@/server/license/testutil";
import { createActivation, createLicense, createOrganization } from "@/testutil";

describe("list", () => {
  let store: Memory;
  let acme: schema.Organization;
  let globex: schema.Organization;
  beforeAll(async () => {
    store = await openMemory();
  });
  beforeEach(async () => {
    await store.clear();
    acme = await createOrganization(store, {
      kind: "team",
      name: "Acme",
      clerkOrgID: "org_a",
    });
    globex = await createOrganization(store, {
      kind: "team",
      name: "Globex",
      clerkOrgID: "org_b",
    });
  });

  const issuedOn = (day: number) => new Date(`2026-09-0${day}T00:00:00Z`);

  describe("listForOrganization", () => {
    it("should list an organization's licenses newest first with seats", async () => {
      const older = await createLicense(store, {
        organization: acme.key,
        issuedAt: issuedOn(1),
      });
      const newer = await createLicense(store, {
        organization: acme.key,
        issuedAt: issuedOn(2),
      });
      await createLicense(store, { organization: globex.key, issuedAt: issuedOn(3) });
      await createActivation(store, { license: older.key, fingerprint: [HASH_A] });
      await createActivation(store, { license: older.key, fingerprint: [HASH_B] });
      await createActivation(store, {
        license: older.key,
        fingerprint: [HASH_C],
        releasedAt: NOW,
      });
      expect(await listForOrganization(store.query, acme.key)).toEqual([
        { license: newer, seats: 0 },
        { license: older, seats: 2 },
      ]);
    });

    it("should list nothing for an organization without licenses", async () => {
      expect(await listForOrganization(store.query, acme.key)).toEqual([]);
    });

    it("should read inside a transaction", async () => {
      const lic = await createLicense(store, { organization: acme.key });
      expect(
        await store.transact(async (tx) => await listForOrganization(tx, acme.key)),
      ).toEqual([{ license: lic, seats: 0 }]);
    });
  });

  describe("listAll", () => {
    it("should list every license with its organization, newest first", async () => {
      const a = await createLicense(store, {
        organization: acme.key,
        issuedAt: issuedOn(1),
      });
      const b = await createLicense(store, {
        organization: globex.key,
        issuedAt: issuedOn(2),
        revokedAt: NOW,
      });
      expect(await listAll(store.query)).toEqual([
        { license: b, organization: globex, seats: 0 },
        { license: a, organization: acme, seats: 0 },
      ]);
    });

    it("should count only the seats machines still hold", async () => {
      const lic = await createLicense(store, { organization: acme.key });
      await createActivation(store, { license: lic.key, fingerprint: [HASH_A] });
      await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_B],
        releasedAt: NOW,
      });
      expect((await listAll(store.query))[0].seats).toBe(1);
    });
  });

  describe("listAllHeld", () => {
    it("should list the licenses of every organization with their seats", async () => {
      const a = await createLicense(store, {
        organization: acme.key,
        issuedAt: issuedOn(1),
      });
      const b = await createLicense(store, {
        organization: globex.key,
        issuedAt: issuedOn(2),
      });
      await createActivation(store, { license: a.key, fingerprint: [HASH_C] });
      expect(await listAllHeld(store.query)).toEqual([
        { license: b, seats: 0 },
        { license: a, seats: 1 },
      ]);
    });
  });

  describe("listActivatable", () => {
    it("should list the unrevoked licenses of the given organizations", async () => {
      const older = await createLicense(store, {
        organization: acme.key,
        issuedAt: issuedOn(1),
      });
      await createLicense(store, {
        organization: acme.key,
        issuedAt: issuedOn(2),
        revokedAt: NOW,
      });
      const other = await createLicense(store, {
        organization: globex.key,
        issuedAt: issuedOn(3),
      });
      expect(await listActivatable(store.query, [acme.key], NOW)).toEqual([
        { license: older, organization: acme, seats: 0 },
      ]);
      expect(await listActivatable(store.query, [acme.key, globex.key], NOW)).toEqual([
        { license: other, organization: globex, seats: 0 },
        { license: older, organization: acme, seats: 0 },
      ]);
    });

    it("should leave out an expired license unless it has a fallback", async () => {
      await createLicense(store, {
        organization: acme.key,
        expiresAt: new Date(NOW.getTime() - 1),
      });
      const fallback = await createLicense(store, {
        organization: acme.key,
        expiresAt: new Date(NOW.getTime() - 1),
        maxVersion: "0.60",
      });
      expect(await listActivatable(store.query, [acme.key], NOW)).toEqual([
        { license: fallback, organization: acme, seats: 0 },
      ]);
    });

    it("should leave out a Synnax Desktop license", async () => {
      await createLicense(store, { organization: acme.key, edition: "desktop" });
      expect(await listActivatable(store.query, [acme.key], NOW)).toEqual([]);
    });

    it("should list nothing for no organizations", async () => {
      await createLicense(store, { organization: acme.key });
      expect(await listActivatable(store.query, [], NOW)).toEqual([]);
    });

    it("should count only the seats that machines hold", async () => {
      const lic = await createLicense(store, { organization: acme.key });
      await createActivation(store, { license: lic.key, fingerprint: [HASH_A] });
      await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_B],
        releasedAt: NOW,
      });
      expect(await listActivatable(store.query, [acme.key], NOW)).toEqual([
        { license: lic, organization: acme, seats: 1 },
      ]);
    });
  });

  describe("findActivatable", () => {
    it("should find a license of any organization with its seats", async () => {
      const lic = await createLicense(store, { organization: globex.key });
      await createActivation(store, { license: lic.key, fingerprint: [HASH_A] });
      expect(await findActivatable(store.query, lic.key, NOW)).toEqual({
        license: lic,
        organization: globex,
        seats: 1,
      });
    });

    it("should skip a revoked, Desktop, or unknown license", async () => {
      const revoked = await createLicense(store, {
        organization: acme.key,
        revokedAt: NOW,
      });
      const desktop = await createLicense(store, {
        organization: acme.key,
        edition: "desktop",
      });
      for (const key of [revoked.key, desktop.key, crypto.randomUUID()])
        expect(await findActivatable(store.query, key, NOW)).toBeUndefined();
    });
  });

  describe("findSeat", () => {
    it("should find the seat of a machine that shares any hash", async () => {
      const lic = await createLicense(store, { organization: acme.key });
      const seat = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A, HASH_B],
      });
      expect(await findSeat(store.query, [lic.key], [HASH_B, HASH_C])).toEqual(seat);
    });

    it("should skip a released seat", async () => {
      const lic = await createLicense(store, { organization: acme.key });
      await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
        releasedAt: NOW,
      });
      expect(await findSeat(store.query, [lic.key], [HASH_A])).toBeUndefined();
    });

    it("should skip a seat on another license", async () => {
      const lic = await createLicense(store, { organization: acme.key });
      const other = await createLicense(store, { organization: globex.key });
      await createActivation(store, { license: other.key, fingerprint: [HASH_A] });
      expect(await findSeat(store.query, [lic.key], [HASH_A])).toBeUndefined();
    });

    it("should find nothing among no licenses", async () => {
      expect(await findSeat(store.query, [], [HASH_A])).toBeUndefined();
    });
  });
});
