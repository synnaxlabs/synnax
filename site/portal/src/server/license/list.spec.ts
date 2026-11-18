// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare, uuid } from "@synnaxlabs/x";
import { eq } from "drizzle-orm";
import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import {
  type License,
  license,
  type Organization,
  organization,
} from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import {
  type Cursor,
  findActivatable,
  findSeat,
  listActivatable,
  listAllHeld,
  listForOrganization,
  listPage,
  type Query,
} from "@/server/license/list";
import { STATUSES, statusOf } from "@/server/license/status";
import { HASH_A, HASH_B, HASH_C, NOW } from "@/server/license/testutil";
import { createActivation, createLicense, createOrganization } from "@/testutil";

const PAGE_SIZE = 50;

describe("list", () => {
  let store: Memory;
  let acme: Organization;
  let globex: Organization;
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

  describe("listPage", () => {
    const days = (n: number): Date => new Date(NOW.getTime() + n * 24 * 60 * 60 * 1000);

    /** createMany saves `n` licenses of acme, all issued on the same day. */
    const createMany = async (n: number, values: Partial<License> = {}) =>
      await store.query
        .insert(license)
        .values(
          Array.from({ length: n }, () => ({
            organization: acme.key,
            edition: "enterprise" as const,
            term: "subscription" as const,
            nodes: 2,
            issuedBy: "user_staff",
            issuedAt: issuedOn(1),
            ...values,
          })),
        )
        .returning();

    const readAll = async (query: Query = {}): Promise<License[]> => {
      const read: License[] = [];
      let after: Cursor | undefined;
      for (;;) {
        const page = await listPage(store.query, { ...query, after }, NOW);
        read.push(...page.entries.map((e) => e.license));
        if (page.next == null) return read;
        after = page.next;
      }
    };

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
      expect(await listPage(store.query, {}, NOW)).toEqual({
        entries: [
          { license: b, organization: globex, seats: 0 },
          { license: a, organization: acme, seats: 0 },
        ],
        total: 2,
        next: null,
      });
    });

    it("should count only the seats machines still hold", async () => {
      const lic = await createLicense(store, { organization: acme.key });
      await createActivation(store, { license: lic.key, fingerprint: [HASH_A] });
      await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_B],
        releasedAt: NOW,
      });
      expect((await listPage(store.query, {}, NOW)).entries[0].seats).toBe(1);
    });

    it("should read a full page, newest first, with ties broken by key", async () => {
      const older = await createMany(PAGE_SIZE + 1);
      const newer = await createLicense(store, {
        organization: globex.key,
        issuedAt: issuedOn(2),
      });
      const first = await listPage(store.query, {}, NOW);
      expect(first.entries).toHaveLength(PAGE_SIZE);
      expect(first.total).toBe(PAGE_SIZE + 2);
      const keys = older.map((l) => l.key).sort(compare.reverse(compare.text));
      expect(first.entries.map((e) => e.license.key)).toEqual([
        newer.key,
        ...keys.slice(0, PAGE_SIZE - 1),
      ]);
      expect(first.next).toEqual({ key: keys[PAGE_SIZE - 2] });
      const second = await listPage(
        store.query,
        { after: first.next ?? undefined },
        NOW,
      );
      expect(second.entries.map((e) => e.license.key)).toEqual(keys.slice(-2));
      expect(second.total).toBe(PAGE_SIZE + 2);
      expect(second.next).toBeNull();
    });

    it("should find a match on a later page and count only matches", async () => {
      const needle = await createLicense(store, {
        organization: globex.key,
        label: "North bench",
        issuedAt: new Date("2026-08-01T00:00:00Z"),
      });
      await createMany(PAGE_SIZE + 1);
      const unfiltered = await listPage(store.query, {}, NOW);
      expect(unfiltered.entries.map((e) => e.license)).not.toContainEqual(needle);
      await store.query
        .update(organization)
        .set({ name: "ÉQUIPE" })
        .where(eq(organization.key, globex.key));
      for (const search of ["north BENCH", "équipe", needle.key.slice(9, 23)]) {
        const page = await listPage(store.query, { search: ` ${search} ` }, NOW);
        expect(page.entries.map((e) => e.license)).toEqual([needle]);
        expect(page.total).toBe(1);
      }
    });

    it("should match each status exactly as statusOf reads it", async () => {
      const shapes: Partial<License>[] = [
        { revokedAt: NOW, expiresAt: days(90) },
        { revokedAt: NOW, expiresAt: days(-1) },
        { expiresAt: null },
        { expiresAt: NOW },
        { expiresAt: days(-1) },
        { expiresAt: days(1) },
        { expiresAt: days(30) },
        { expiresAt: new Date(days(30).getTime() + 1) },
        { expiresAt: days(31) },
        { edition: "desktop", expiresAt: days(2) },
        { edition: "desktop", expiresAt: days(-1) },
        { edition: "desktop", expiresAt: null },
      ];
      const created: License[] = [];
      for (const s of shapes)
        created.push(await createLicense(store, { organization: acme.key, ...s }));
      for (const status of STATUSES) {
        const want = created.filter((l) => statusOf(l, NOW) === status);
        expect(want.length).toBeGreaterThan(0);
        const page = await listPage(store.query, { status }, NOW);
        expect(page.total).toBe(want.length);
        const keys = (l: License[]): string[] => l.map((x) => x.key).sort();
        expect(keys(page.entries.map((e) => e.license))).toEqual(keys(want));
      }
    });

    it("should page every status filter across pages", async () => {
      await createMany(PAGE_SIZE + 1, { revokedAt: NOW });
      await createMany(PAGE_SIZE + 1, { expiresAt: days(-1) });
      await createMany(PAGE_SIZE + 1, { expiresAt: days(90) });
      await createMany(3, { expiresAt: days(5) });
      for (const status of STATUSES) {
        const read = await readAll({ status });
        expect(read.every((l) => statusOf(l, NOW) === status)).toBe(true);
        expect(new Set(read.map((l) => l.key)).size).toBe(read.length);
        const { total } = await listPage(store.query, { status }, NOW);
        expect(read).toHaveLength(total);
      }
    });

    it("should list expiring licenses soonest first across pages", async () => {
      await createMany(PAGE_SIZE - 1, { expiresAt: days(20) });
      const soonest = await createMany(1, { expiresAt: days(3) });
      const later = await createMany(2, { expiresAt: days(25) });
      const read = await readAll({ status: "expiring" });
      expect(read).toHaveLength(PAGE_SIZE + 2);
      expect(read[0]).toEqual(soonest[0]);
      expect(read.slice(-2).map((l) => l.key)).toEqual(later.map((l) => l.key).sort());
      const expiries = read.map((l) => l.expiresAt?.getTime() ?? 0);
      expect(expiries).toEqual([...expiries].sort((a, b) => a - b));
    });

    it("should page on from the expiry its cursor read, even after a renewal", async () => {
      const licenses = await createMany(PAGE_SIZE + 2, { expiresAt: days(20) });
      const first = await listPage(store.query, { status: "expiring" }, NOW);
      const anchor = first.next?.key ?? "";
      expect(first.next?.expiresAt).toEqual(days(20));
      await store.query
        .update(license)
        .set({ expiresAt: null })
        .where(eq(license.key, anchor));
      const second = await listPage(
        store.query,
        { status: "expiring", after: first.next ?? undefined },
        NOW,
      );
      const keys = licenses.map((l) => l.key).sort();
      expect(second.entries.map((e) => e.license.key)).toEqual(keys.slice(-2));
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
      for (const key of [revoked.key, desktop.key, uuid.create()])
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
