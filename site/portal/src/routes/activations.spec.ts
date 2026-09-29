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

import { POST as downloadRoute } from "@/pages/api/activations/[key]/download";
import { POST as nameRoute } from "@/pages/api/activations/[key]/name";
import { POST as releaseRoute } from "@/pages/api/activations/[key]/release";
import {
  type Activation,
  activation,
  event,
  type License,
  license,
  type Organization,
} from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { type Team } from "@/server/directory";
import { HASH_A, HASH_B, NOW } from "@/server/license/testutil";
import { PER_ACTOR } from "@/server/ratelimit";
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
} from "@/testutil";

const MEMBER = "user_member";
const OUTSIDER = "user_outsider";
const ACME = "org_acme";
const EARLIER = new Date("2026-09-10T00:00:00Z");

describe("activation routes", () => {
  let store: Memory;
  let h: Harness;
  let acme: Organization;
  let lic: License;
  let act: Activation;

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    h = createHarness(store);
    addPerson(MEMBER, [{ clerkOrgID: ACME, name: "Acme", role: "org:member" }]);
    addPerson(OUTSIDER);
    acme = await createOrganization(store, { kind: "team", clerkOrgID: ACME });
    lic = await createLicense(store, { organization: acme.key });
    act = await createActivation(store, {
      license: lic.key,
      fingerprint: [HASH_A, HASH_B],
      name: "Test stand",
      firstSeen: EARLIER,
      lastSeen: EARLIER,
    });
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

  const machine = async (): Promise<Activation> => {
    const [row] = await store.query
      .select()
      .from(activation)
      .where(eq(activation.key, act.key));
    return row;
  };

  const events = async () => await store.query.select().from(event);

  const describeAccess = (
    route: APIRoute,
    args: Omit<ContextArgs, "params"> = {},
  ): void => {
    it("should reject a visitor who is not signed in", async () => {
      await expectError(
        await call(route, { ...args, params: { key: act.key } }),
        401,
        "Sign in first",
      );
    });

    it("should reject a user outside the owning organization", async () => {
      h.signIn(OUTSIDER);
      await expectError(
        await call(route, { ...args, params: { key: act.key } }),
        404,
        "Activation not found",
      );
    });

    it("should answer 404 for an unknown machine", async () => {
      h.signIn(MEMBER);
      await expectError(
        await call(route, { ...args, params: { key: crypto.randomUUID() } }),
        404,
        "Activation not found",
      );
    });
  };

  describe("POST /api/activations/[key]/download", () => {
    const post = async (): Promise<Response> =>
      await call(downloadRoute, { params: { key: act.key } });

    describeAccess(downloadRoute);

    it("should download a fresh license key for the machine", async () => {
      h.signIn(MEMBER);
      const res = await post();
      expect(res.status).toBe(200);
      expect(res.headers.get("content-type")).toBe("text/plain; charset=utf-8");
      expect(res.headers.get("content-disposition")).toBe(
        'attachment; filename="Test-rig.license"',
      );
      expect(readKey(await res.text())).toMatchObject({
        jti: lic.key,
        iat: Math.floor(NOW.getTime() / 1000),
        fingerprints: [HASH_A, HASH_B],
      });
    });

    it("should mark the machine seen and record the download", async () => {
      h.signIn(MEMBER);
      await post();
      expect((await machine()).lastSeen).toEqual(NOW);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "download" as const,
          actor: MEMBER,
          organization: acme.key,
          license: lic.key,
          activation: act.key,
          detail: {},
        }),
      ]);
    });

    it("should answer 404 for a machine whose seat was released", async () => {
      h.signIn(MEMBER);
      await store.query
        .update(activation)
        .set({ releasedAt: NOW })
        .where(eq(activation.key, act.key));
      await expectError(await post(), 404, "Activation not found");
    });

    it("should refuse a machine on a revoked license", async () => {
      h.signIn(MEMBER);
      await store.query
        .update(license)
        .set({ revokedAt: NOW })
        .where(eq(license.key, lic.key));
      await expectError(await post(), 400, "This license has been revoked.");
    });

    it("should rate limit a user who downloads too often", async () => {
      h.signIn(MEMBER);
      await store.query.insert(event).values(
        Array.from({ length: PER_ACTOR }, () => ({
          kind: "download" as const,
          actor: MEMBER,
          organization: acme.key,
          license: lic.key,
        })),
      );
      await expectError(await post(), 429, "Too many activations. Try again later.");
    });
  });

  describe("POST /api/activations/[key]/name", () => {
    const post = async (name: string): Promise<Response> =>
      await call(nameRoute, { params: { key: act.key }, body: { name } });

    describeAccess(nameRoute, { body: { name: "New" } });

    it("should require a name", async () => {
      h.signIn(MEMBER);
      await expectError(await post("   "), 400, "The machine needs a name");
    });

    it("should rename the machine and record the change", async () => {
      h.signIn(MEMBER);
      const res = await post("  Bench two  ");
      expect(res.status).toBe(204);
      expect((await machine()).name).toBe("Bench two");
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "rename",
          actor: MEMBER,
          activation: act.key,
          detail: { from: "Test stand", to: "Bench two" },
        }),
      ]);
    });

    it("should record nothing when the name is unchanged", async () => {
      h.signIn(MEMBER);
      expect((await post("Test stand")).status).toBe(204);
      expect(await events()).toEqual([]);
    });
  });

  describe("POST /api/activations/[key]/release", () => {
    const post = async (): Promise<Response> =>
      await call(releaseRoute, { params: { key: act.key } });

    describeAccess(releaseRoute);

    it("should free the seat and record the release", async () => {
      h.signIn(MEMBER);
      const res = await post();
      expect(res.status).toBe(204);
      expect((await machine()).releasedAt).toEqual(NOW);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "release",
          actor: MEMBER,
          organization: acme.key,
          license: lic.key,
          activation: act.key,
        }),
      ]);
    });

    it("should do nothing for a seat already released", async () => {
      h.signIn(MEMBER);
      await post();
      expect((await post()).status).toBe(204);
      expect(await events()).toHaveLength(1);
    });
  });
});
