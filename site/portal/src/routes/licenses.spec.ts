// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";
import { eq } from "drizzle-orm";
import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { POST as activateRoute } from "@/pages/api/licenses/[key]/activate";
import { POST as floatingRoute } from "@/pages/api/licenses/[key]/floating";
import { POST as amendRoute } from "@/pages/api/licenses/[key]/index";
import { POST as revokeRoute } from "@/pages/api/licenses/[key]/revoke";
import { POST as issueRoute } from "@/pages/api/licenses/index";
import {
  activation,
  event,
  type License,
  license,
  type Organization,
  organization,
} from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { ADMIN_ROLE, type Team } from "@/server/directory";
import { HASH_A, HASH_B, NOW } from "@/server/license/testutil";
import { PER_ACTOR, PER_ORGANIZATION } from "@/server/ratelimit";
import {
  body,
  type ContextArgs,
  createActivation,
  createAPIContext,
  createHarness,
  createLicense,
  createOrganization,
  type Harness,
  readKey,
  STAFF_ORG_ID,
} from "@/testutil";
import { Licenses } from "@/ui/licenses";

const STAFF = "user_staff";
const MEMBER = "user_member";
const OUTSIDER = "user_outsider";
const ACME = "org_acme";

const STAFF_TEAM: Team = {
  clerkOrgID: STAFF_ORG_ID,
  name: "Synnax Labs",
  role: ADMIN_ROLE,
};
const ACME_MEMBER: Team = { clerkOrgID: ACME, name: "Acme", role: "org:member" };
const ACME_ADMIN: Team = { ...ACME_MEMBER, role: ADMIN_ROLE };

const seconds = (date: Date): number => Math.floor(date.getTime() / 1000);

describe("license routes", () => {
  let store: Memory;
  let h: Harness;
  let acme: Organization;

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    h = createHarness(store);
    h.directory.organizations.push({ clerkOrgID: ACME, name: "Acme" });
    addPerson(STAFF, [STAFF_TEAM]);
    addPerson(MEMBER, [ACME_MEMBER]);
    addPerson(OUTSIDER);
    acme = await createOrganization(store, { kind: "team", clerkOrgID: ACME });
  });

  const addPerson = (userID: string, teams: Team[] = []): void => {
    h.directory.people[userID] = { email: `${userID}@example.com`, name: userID };
    h.directory.members[userID] = teams;
  };

  const call = async (route: APIRoute, args: ContextArgs = {}): Promise<Response> =>
    await route(createAPIContext(h.portal, args));

  const expectError = async (
    res: Response,
    status: number,
    error: string,
  ): Promise<void> => {
    expect(res.status).toBe(status);
    expect(await body(res)).toEqual({ error });
  };

  const licenses = async (): Promise<License[]> =>
    await store.query.select().from(license);

  const events = async () => await store.query.select().from(event);

  describe("POST /api/licenses", () => {
    const post = async (form: Record<string, string>): Promise<Response> =>
      await call(issueRoute, { body: form });

    const SUBSCRIPTION = {
      organization: ACME,
      term: "subscription",
      nodes: "3",
      channels: "500",
      label: "  Test stand  ",
      expiresAt: "2027-01-01",
    };

    it("should reject a visitor who is not logged in", async () => {
      await expectError(await post(SUBSCRIPTION), 401, "Log in first");
    });

    it("should reject a user who is not staff", async () => {
      h.signIn(MEMBER);
      await expectError(await post(SUBSCRIPTION), 403, "Staff only");
    });

    describe("validation", () => {
      beforeEach(() => h.signIn(STAFF));

      it("should reject a seat count below 1", async () => {
        await expectError(
          await post({ ...SUBSCRIPTION, nodes: "0" }),
          400,
          "Nodes must be a whole number of at least 1",
        );
      });

      it("should reject a seat count that is not a whole number", async () => {
        await expectError(
          await post({ ...SUBSCRIPTION, nodes: "1.5" }),
          400,
          "Nodes must be a whole number of at least 1",
        );
      });

      it("should reject a negative channel cap", async () => {
        await expectError(
          await post({ ...SUBSCRIPTION, channels: "-1" }),
          400,
          "Channels must be a whole number, 0 for unlimited",
        );
      });

      it("should require a label", async () => {
        await expectError(
          await post({ ...SUBSCRIPTION, label: "   " }),
          400,
          "Give the license a label",
        );
      });

      it("should require an organization", async () => {
        await expectError(
          await post({ ...SUBSCRIPTION, organization: "" }),
          400,
          "Choose an organization",
        );
      });

      it("should refuse an organization Clerk does not know", async () => {
        await expectError(
          await post({ ...SUBSCRIPTION, organization: "org_missing" }),
          400,
          "Clerk has no organization org_missing",
        );
      });

      it("should require an expiry on a subscription", async () => {
        await expectError(
          await post({ ...SUBSCRIPTION, expiresAt: "" }),
          400,
          "A subscription needs an expiry",
        );
      });

      it("should reject an expiry that is not a date", async () => {
        await expectError(
          await post({ ...SUBSCRIPTION, expiresAt: "soon" }),
          400,
          "The expiry must be a valid date",
        );
      });

      it("should reject an expiry in the past", async () => {
        await expectError(
          await post({ ...SUBSCRIPTION, expiresAt: "2026-01-01" }),
          400,
          "The expiry must be in the future",
        );
      });

      it("should require a maximum version on a perpetual license", async () => {
        await expectError(
          await post({ ...SUBSCRIPTION, term: "perpetual" }),
          400,
          "A perpetual license needs a maximum version",
        );
      });

      it("should reject a maximum version that is not major.minor", async () => {
        await expectError(
          await post({ ...SUBSCRIPTION, maxVersion: "0.62.1" }),
          400,
          'Maximum version must look like "0.62"',
        );
      });

      it("should issue nothing when validation fails", async () => {
        await post({ ...SUBSCRIPTION, nodes: "0" });
        expect(await licenses()).toHaveLength(0);
        expect(await events()).toHaveLength(0);
      });
    });

    describe("issuing", () => {
      beforeEach(() => h.signIn(STAFF));

      it("should issue a subscription and answer its key", async () => {
        const res = await post(SUBSCRIPTION);
        expect(res.status).toBe(200);
        const { key } = await body<{ key: string }>(res);
        const [row] = await licenses();
        expect(row).toMatchObject({
          key,
          organization: acme.key,
          edition: "enterprise",
          term: "subscription",
          nodes: 3,
          channels: 500,
          label: "Test stand",
          expiresAt: new Date("2027-01-01T00:00:00Z"),
          maxVersion: null,
          issuedBy: STAFF,
          issuedAt: NOW,
          revokedAt: null,
        });
      });

      it("should record the issuance in the ledger", async () => {
        const { key } = await body<{ key: string }>(await post(SUBSCRIPTION));
        expect(await events()).toEqual([
          expect.objectContaining({
            kind: "issue",
            actor: STAFF,
            organization: acme.key,
            license: key,
            detail: { nodes: 3, channels: 500, term: "subscription" },
          }),
        ]);
      });

      it("should default the channel cap to unlimited", async () => {
        await post({ ...SUBSCRIPTION, channels: "" });
        expect((await licenses())[0].channels).toBe(0);
      });

      it("should issue a perpetual license without an expiry", async () => {
        const res = await post({
          ...SUBSCRIPTION,
          term: "perpetual",
          maxVersion: " 0.62 ",
        });
        expect(res.status).toBe(200);
        expect((await licenses())[0]).toMatchObject({
          term: "perpetual",
          expiresAt: null,
          maxVersion: "0.62",
        });
      });

      it("should mirror a Clerk organization the portal has not seen", async () => {
        h.directory.organizations.push({ clerkOrgID: "org_new", name: "Newco" });
        await post({ ...SUBSCRIPTION, organization: "org_new" });
        const [org] = await store.query
          .select()
          .from(organization)
          .where(eq(organization.clerkOrgID, "org_new"));
        expect(org).toMatchObject({ kind: "team", name: "Newco" });
        expect((await licenses())[0].organization).toBe(org.key);
      });

      it("should issue from the values the issue dialog posts", async () => {
        const values = {
          ...Licenses.ZERO_TERMS,
          organization: ACME,
          label: "Test stand",
          nodes: 3,
          channels: 500,
          expiresAt: "2027-01-01",
        };
        const res = await call(issueRoute, { body: values });
        expect(res.status).toBe(200);
        expect(await licenses()).toEqual([
          expect.objectContaining({
            organization: acme.key,
            term: "subscription",
            nodes: 3,
            channels: 500,
            label: "Test stand",
            expiresAt: new Date("2027-01-01T00:00:00Z"),
            maxVersion: null,
          }),
        ]);
      });

      it("should accept a posted form", async () => {
        const form = new FormData();
        for (const [k, v] of Object.entries(SUBSCRIPTION)) form.set(k, v);
        const res = await call(issueRoute, { body: form });
        expect(res.status).toBe(200);
        expect(await licenses()).toHaveLength(1);
      });
    });
  });

  describe("POST /api/licenses/[key]", () => {
    let lic: License;

    beforeEach(async () => {
      lic = await createLicense(store, { organization: acme.key });
    });

    const TERMS = {
      term: "subscription",
      nodes: "5",
      channels: "100",
      label: "Renamed rig",
      expiresAt: "2027-06-01",
    };

    const post = async (
      form: Record<string, string>,
      key: string = lic.key,
    ): Promise<Response> => await call(amendRoute, { params: { key }, body: form });

    it("should reject a visitor who is not logged in", async () => {
      await expectError(await post(TERMS), 401, "Log in first");
    });

    it("should reject a member who is not staff", async () => {
      h.signIn(MEMBER);
      await expectError(await post(TERMS), 403, "Staff only");
    });

    it("should answer 404 for an unknown license", async () => {
      h.signIn(STAFF);
      await expectError(
        await post(TERMS, crypto.randomUUID()),
        404,
        "License not found",
      );
    });

    it("should reject a seat count below 1", async () => {
      h.signIn(STAFF);
      await expectError(
        await post({ ...TERMS, nodes: "0" }),
        400,
        "Nodes must be a whole number of at least 1",
      );
    });

    it("should require a label", async () => {
      h.signIn(STAFF);
      await expectError(
        await post({ ...TERMS, label: "" }),
        400,
        "Give the license a label",
      );
    });

    it("should reject a maximum version that is not major.minor", async () => {
      h.signIn(STAFF);
      await expectError(
        await post({ ...TERMS, maxVersion: "0.62.1" }),
        400,
        'Maximum version must look like "0.62"',
      );
    });

    it("should reject a negative channel cap", async () => {
      h.signIn(STAFF);
      await expectError(
        await post({ ...TERMS, channels: "-5" }),
        400,
        "Channels must be a whole number, 0 for unlimited",
      );
    });

    it("should refuse to lower the seats below the machines holding one", async () => {
      h.signIn(STAFF);
      await createActivation(store, { license: lic.key, fingerprint: [HASH_A] });
      await createActivation(store, { license: lic.key, fingerprint: [HASH_B] });
      await expectError(
        await post({ ...TERMS, nodes: "1" }),
        400,
        "2 machines hold a seat. Release one before lowering the limit to 1.",
      );
    });

    it("should refuse to change a revoked license", async () => {
      h.signIn(STAFF);
      await store.query
        .update(license)
        .set({ revokedAt: NOW })
        .where(eq(license.key, lic.key));
      await expectError(await post(TERMS), 400, "A revoked license cannot be changed");
    });

    it("should change the terms and keep the key", async () => {
      h.signIn(STAFF);
      const res = await post(TERMS);
      expect(res.status).toBe(204);
      expect(await licenses()).toEqual([
        expect.objectContaining({
          key: lic.key,
          nodes: 5,
          channels: 100,
          label: "Renamed rig",
          expiresAt: new Date("2027-06-01T00:00:00Z"),
        }),
      ]);
    });

    it("should post a license back to the amend route unchanged", async () => {
      h.signIn(STAFF);
      const subscription = await createLicense(store, {
        organization: acme.key,
        maxVersion: "0.62",
      });
      const perpetual = await createLicense(store, {
        organization: acme.key,
        term: "perpetual",
        expiresAt: null,
        maxVersion: "0.62",
      });
      for (const unchanged of [subscription, perpetual]) {
        const res = await call(amendRoute, {
          params: { key: unchanged.key },
          body: Licenses.termsOf(unchanged),
        });
        expect(res.status).toBe(204);
      }
      expect(await licenses()).toEqual(
        expect.arrayContaining([subscription, perpetual]),
      );
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "amend",
          license: subscription.key,
          detail: {},
        }),
        expect.objectContaining({ kind: "amend", license: perpetual.key, detail: {} }),
      ]);
    });

    it("should record what changed in the ledger", async () => {
      h.signIn(STAFF);
      await post(TERMS);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "amend",
          actor: STAFF,
          organization: acme.key,
          license: lic.key,
          detail: {
            nodes: { from: 2, to: 5 },
            channels: { from: 0, to: 100 },
            label: { from: "Test rig", to: "Renamed rig" },
            expiresAt: {
              from: "2027-03-01T00:00:00.000Z",
              to: "2027-06-01T00:00:00.000Z",
            },
          },
        }),
      ]);
    });
  });

  describe("POST /api/licenses/[key]/activate", () => {
    let lic: License;

    beforeEach(async () => {
      lic = await createLicense(store, { organization: acme.key });
    });

    const post = async (
      form: Record<string, string>,
      key: string = lic.key,
    ): Promise<Response> => await call(activateRoute, { params: { key }, body: form });

    const MACHINE = { fingerprint: HASH_A, name: "Test stand" };

    it("should reject a visitor who is not logged in", async () => {
      await expectError(await post(MACHINE), 401, "Log in first");
    });

    it("should reject a user outside the owning organization", async () => {
      h.signIn(OUTSIDER);
      await expectError(await post(MACHINE), 404, "License not found");
    });

    it("should answer 404 for an unknown license", async () => {
      h.signIn(MEMBER);
      await expectError(
        await post(MACHINE, crypto.randomUUID()),
        404,
        "License not found",
      );
    });

    it("should require at least one host hash", async () => {
      h.signIn(MEMBER);
      await expectError(
        await post({ ...MACHINE, fingerprint: " " }),
        400,
        "Paste at least one host hash",
      );
    });

    it("should reject text that is not a host hash", async () => {
      h.signIn(MEMBER);
      await expectError(
        await post({ ...MACHINE, fingerprint: "xyz" }),
        400,
        '"xyz" is not a host hash',
      );
    });

    it("should require a machine name", async () => {
      h.signIn(MEMBER);
      await expectError(
        await post({ ...MACHINE, name: "  " }),
        400,
        "The machine needs a name",
      );
    });

    it("should answer the license key, the activation, and the file name", async () => {
      h.signIn(MEMBER);
      const res = await post(MACHINE);
      expect(res.status).toBe(200);
      const answer = await body<{ key: string; activation: string; filename: string }>(
        res,
      );
      const [act] = await store.query.select().from(activation);
      expect(answer.activation).toBe(act.key);
      expect(answer.filename).toBe("Test-rig.lic");
      expect(readKey(answer.key)).toEqual({
        jti: lic.key,
        iat: seconds(NOW),
        exp: seconds(new Date("2027-03-01T00:00:00Z")),
        claims_version: 1,
        organization: acme.key,
        edition: "e",
        fingerprints: [HASH_A],
        fingerprint_scheme: 1,
        machines: 2,
        channels: 0,
        required: [],
      });
    });

    it("should record the machine and the activation", async () => {
      h.signIn(MEMBER);
      await post({ ...MACHINE, fingerprint: `${HASH_A}, ${HASH_B}` });
      const [act] = await store.query.select().from(activation);
      expect(act).toMatchObject({
        license: lic.key,
        fingerprint: [HASH_A, HASH_B],
        name: "Test stand",
        firstSeen: NOW,
        lastSeen: NOW,
        releasedAt: null,
      });
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "activate" as const,
          actor: MEMBER,
          organization: acme.key,
          license: lic.key,
          activation: act.key,
          detail: { fingerprint: [HASH_A, HASH_B] },
        }),
      ]);
    });

    it("should let staff activate a license of any organization", async () => {
      h.signIn(STAFF);
      expect((await post(MACHINE)).status).toBe(200);
    });

    it("should keep the seat of a machine that activates again", async () => {
      h.signIn(MEMBER);
      const first = await body<{ activation: string }>(await post(MACHINE));
      const again = await body<{ activation: string }>(
        await post({ fingerprint: `${HASH_B} ${HASH_A}`, name: "Renamed" }),
      );
      expect(again.activation).toBe(first.activation);
      const rows = await store.query.select().from(activation);
      expect(rows).toHaveLength(1);
      expect(rows[0].name).toBe("Renamed");
    });

    it("should refuse a machine when every seat is taken", async () => {
      h.signIn(MEMBER);
      await store.query
        .update(license)
        .set({ nodes: 1 })
        .where(eq(license.key, lic.key));
      await createActivation(store, { license: lic.key, fingerprint: [HASH_B] });
      await expectError(
        await post(MACHINE),
        400,
        "Every seat on this license is taken. Release a machine to free one.",
      );
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "activate_denied",
          actor: MEMBER,
          license: lic.key,
          detail: { reason: "no_seats", fingerprint: [HASH_A] },
        }),
      ]);
    });

    it("should refuse a revoked license", async () => {
      h.signIn(MEMBER);
      await store.query
        .update(license)
        .set({ revokedAt: NOW })
        .where(eq(license.key, lic.key));
      await expectError(await post(MACHINE), 400, "This license has been revoked.");
    });

    it("should refuse an expired subscription", async () => {
      h.signIn(MEMBER);
      await store.query
        .update(license)
        .set({ expiresAt: new Date("2026-09-01T00:00:00Z") })
        .where(eq(license.key, lic.key));
      await expectError(await post(MACHINE), 400, "This license has expired.");
    });

    it("should rate limit a user who activates too often", async () => {
      h.signIn(MEMBER);
      await store.query.insert(event).values(
        Array.from({ length: PER_ACTOR }, () => ({
          kind: "activate" as const,
          actor: MEMBER,
          organization: acme.key,
          license: lic.key,
        })),
      );
      await expectError(
        await post(MACHINE),
        429,
        "Too many activations. Try again later.",
      );
      expect(await store.query.select().from(activation)).toHaveLength(0);
    });

    it("should rate limit an organization that activates too often", async () => {
      h.signIn(MEMBER);
      await store.query.insert(event).values(
        Array.from({ length: PER_ORGANIZATION }, (_, i) => ({
          kind: "download" as const,
          actor: `user_${i}`,
          organization: acme.key,
          license: lic.key,
        })),
      );
      await expectError(
        await post(MACHINE),
        429,
        "Too many activations. Try again later.",
      );
    });
  });

  describe("POST /api/licenses/[key]/floating", () => {
    let lic: License;

    beforeEach(async () => {
      lic = await createLicense(store, { organization: acme.key });
    });

    const post = async (key: string = lic.key): Promise<Response> =>
      await call(floatingRoute, { params: { key } });

    it("should reject a visitor who is not logged in", async () => {
      await expectError(await post(), 401, "Log in first");
    });

    it("should reject a member who is not staff", async () => {
      h.signIn(MEMBER);
      await expectError(await post(), 403, "Staff only");
    });

    it("should answer 404 for an unknown license", async () => {
      h.signIn(STAFF);
      await expectError(await post(crypto.randomUUID()), 404, "License not found");
    });

    it("should refuse a revoked license", async () => {
      h.signIn(STAFF);
      await store.query
        .update(license)
        .set({ revokedAt: NOW })
        .where(eq(license.key, lic.key));
      await expectError(await post(), 400, "This license has been revoked.");
    });

    it("should download a license key bound to no machine", async () => {
      h.signIn(STAFF);
      const res = await post();
      expect(res.status).toBe(200);
      expect(res.headers.get("content-type")).toBe("text/plain; charset=utf-8");
      expect(res.headers.get("content-disposition")).toBe(
        'attachment; filename="Test-rig.lic"',
      );
      const claims = readKey(await res.text());
      expect(claims).toMatchObject({ jti: lic.key, fingerprints: [], machines: 2 });
    });

    it("should record the download without taking a seat", async () => {
      h.signIn(STAFF);
      await post();
      expect(await store.query.select().from(activation)).toHaveLength(0);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "download" as const,
          actor: STAFF,
          license: lic.key,
          activation: null,
          detail: { floating: true },
        }),
      ]);
    });
  });

  describe("POST /api/licenses/[key]/revoke", () => {
    let lic: License;

    beforeEach(async () => {
      lic = await createLicense(store, { organization: acme.key });
    });

    const post = async (key: string = lic.key): Promise<Response> =>
      await call(revokeRoute, { params: { key } });

    it("should reject a visitor who is not logged in", async () => {
      await expectError(await post(), 401, "Log in first");
    });

    it("should reject a member who is not staff", async () => {
      h.signIn(MEMBER);
      await expectError(await post(), 403, "Staff only");
    });

    it("should answer 404 for an unknown license", async () => {
      h.signIn(STAFF);
      await expectError(await post(crypto.randomUUID()), 404, "License not found");
    });

    it("should revoke the license and record it", async () => {
      h.signIn(STAFF);
      const res = await post();
      expect(res.status).toBe(204);
      expect((await licenses())[0].revokedAt).toEqual(NOW);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "revoke",
          actor: STAFF,
          organization: acme.key,
          license: lic.key,
        }),
      ]);
    });

    it("should mail the team's admins", async () => {
      addPerson("user_admin", [ACME_ADMIN]);
      h.signIn(STAFF);
      await post();
      expect(h.mail.sent).toEqual([
        {
          to: ["user_admin@example.com"],
          subject: 'Your Synnax license "Test rig" was revoked',
          text: expect.stringContaining(
            'The Synnax license "Test rig" for Acme has been revoked.',
          ),
        },
      ]);
    });

    it("should mail the owner of a personal organization", async () => {
      const personal = await createOrganization(store, {
        kind: "personal",
        ownerUserID: MEMBER,
      });
      const owned = await createLicense(store, { organization: personal.key });
      h.signIn(STAFF);
      await post(owned.key);
      expect(h.mail.sent.map((m) => m.to)).toEqual([[`${MEMBER}@example.com`]]);
    });

    it("should mail nobody when the team has no admin", async () => {
      h.signIn(STAFF);
      await post();
      expect(h.mail.sent).toEqual([]);
    });

    it("should refuse a second revocation without mailing again", async () => {
      addPerson("user_admin", [ACME_ADMIN]);
      h.signIn(STAFF);
      expect((await post()).status).toBe(204);
      await expectError(await post(), 400, "This license is already revoked");
      expect(h.mail.sent).toHaveLength(1);
    });
  });
});
