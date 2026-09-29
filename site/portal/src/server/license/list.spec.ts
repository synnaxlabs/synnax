// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { type Organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { listActivatable, listAll, listForOrganization } from "@/server/license/list";
import { HASH_A, HASH_B, HASH_C, NOW } from "@/server/license/testutil";
import { createActivation, createLicense, createOrganization } from "@/testutil";

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
        { license: b, organization: globex },
        { license: a, organization: acme },
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
      expect(await listActivatable(store.query, [acme.key])).toEqual([
        { license: older, organization: acme },
      ]);
      expect(await listActivatable(store.query, [acme.key, globex.key])).toEqual([
        { license: other, organization: globex },
        { license: older, organization: acme },
      ]);
    });

    it("should list nothing for no organizations", async () => {
      await createLicense(store, { organization: acme.key });
      expect(await listActivatable(store.query, [])).toEqual([]);
    });
  });
});
