// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { activationFor, licenseFor, requireStaff } from "@/access";
import { type Portal } from "@/portal";
import { type License, type Organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { HASH_A, HASH_B, HASH_C } from "@/server/license/testutil";
import { type Session } from "@/server/session";
import {
  createActivation,
  createHarness,
  createLicense,
  createOrganization,
} from "@/testutil";

const session = (overrides: Partial<Session> = {}): Session => ({
  userID: "user_a",
  email: "ada@example.com",
  name: "Ada",
  teams: [],
  clerkOrgIDs: [],
  staff: false,
  ...overrides,
});

const MEMBER = session({ clerkOrgIDs: ["org_acme"] });
const OUTSIDER = session({ userID: "user_x" });
const STAFF = session({ userID: "user_staff", staff: true });

describe("access", () => {
  let store: Memory;
  let portal: Portal;
  let org: Organization;
  let lic: License;
  beforeAll(async () => {
    store = await openMemory();
    portal = createHarness(store).portal;
  });
  beforeEach(async () => {
    await store.clear();
    org = await createOrganization(store, { kind: "team", clerkOrgID: "org_acme" });
    lic = await createLicense(store, { organization: org.key });
  });

  describe("licenseFor", () => {
    it("should load a license with its organization for a member", async () => {
      const view = await licenseFor(portal, MEMBER, lic.key);
      expect(view.license).toEqual(lic);
      expect(view.organization).toEqual(org);
      expect(view.activations).toEqual([]);
    });

    it("should load any license for staff", async () => {
      const view = await licenseFor(portal, STAFF, lic.key);
      expect(view.license.key).toBe(lic.key);
    });

    it("should load a personal license for its owner", async () => {
      const personal = await createOrganization(store, {
        kind: "personal",
        ownerUserID: "user_a",
      });
      const own = await createLicense(store, { organization: personal.key });
      expect((await licenseFor(portal, session(), own.key)).organization).toEqual(
        personal,
      );
    });

    it("should list the machines holding seats, most recently seen first", async () => {
      const older = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
        lastSeen: new Date("2026-09-01T00:00:00Z"),
      });
      const newest = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_B],
        lastSeen: new Date("2026-09-15T00:00:00Z"),
      });
      const middle = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_C],
        lastSeen: new Date("2026-09-10T00:00:00Z"),
      });
      const view = await licenseFor(portal, MEMBER, lic.key);
      expect(view.activations.map((a) => a.key)).toEqual([
        newest.key,
        middle.key,
        older.key,
      ]);
    });

    it("should throw a 403 for someone outside the organization", async () => {
      await expect(licenseFor(portal, OUTSIDER, lic.key)).rejects.toMatchObject({
        status: 403,
        message: "You are not a member of the organization that owns this license",
      });
    });

    it("should throw a 404 for an unknown license", async () => {
      await expect(
        licenseFor(portal, STAFF, crypto.randomUUID()),
      ).rejects.toMatchObject({ status: 404, message: "License not found" });
    });
  });

  describe("activationFor", () => {
    it("should load a machine with its license for a member", async () => {
      const act = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
      });
      const view = await activationFor(portal, MEMBER, act.key);
      expect(view.activation).toEqual(act);
      expect(view.license.key).toBe(lic.key);
      expect(view.organization.key).toBe(org.key);
    });

    it("should throw a 403 for someone outside the organization", async () => {
      const act = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
      });
      await expect(activationFor(portal, OUTSIDER, act.key)).rejects.toMatchObject({
        status: 403,
      });
    });

    it("should throw a 404 for an unknown machine", async () => {
      await expect(
        activationFor(portal, STAFF, crypto.randomUUID()),
      ).rejects.toMatchObject({ status: 404, message: "Activation not found" });
    });
  });

  describe("requireStaff", () => {
    it("should let staff through", () => {
      expect(() => requireStaff(STAFF)).not.toThrow();
    });

    it("should throw a 403 for anyone else", () => {
      let thrown: unknown;
      try {
        requireStaff(MEMBER);
      } catch (err) {
        thrown = err;
      }
      expect(thrown).toMatchObject({ status: 403, message: "Staff only" });
    });
  });
});
