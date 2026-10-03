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

import { POST as unlinkRoute } from "@/pages/api/activations/[key]/unlink";
import { POST as linkRoute } from "@/pages/api/desktop/link";
import { OPTIONS, POST as renewRoute } from "@/pages/api/desktop/renew";
import { activation, event, license } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { hashSecret } from "@/server/license/desktop";
import { HASH_A, HASH_B } from "@/server/license/testutil";
import { PER_ACTOR } from "@/server/ratelimit";
import {
  body,
  createActivation,
  createAPIContext,
  createHarness,
  createLicense,
  createOrganization,
  type Harness,
  readKey,
} from "@/testutil";

interface Linked {
  key: string;
  secret: string;
  activation: string;
  email: string;
  user: string;
}

describe("desktop routes", () => {
  let store: Memory;
  let harness: Harness;
  beforeAll(async () => {
    store = await openMemory();
  });
  beforeEach(async () => {
    await store.clear();
    harness = createHarness(store);
    harness.directory.people.user_a = {
      email: "ada@example.com",
      name: "Ada Lovelace",
    };
    harness.directory.people.user_b = { email: "bob@example.com", name: "Bob" };
  });

  const postLink = async (fields: Record<string, string>) =>
    await linkRoute(createAPIContext(harness.portal, { body: fields }));

  const linkAs = async (userID: string, fingerprint = HASH_A): Promise<Linked> => {
    harness.signIn(userID);
    const res = await postLink({ fingerprint, name: "stand-1" });
    expect(res.status).toBe(200);
    return await body<Linked>(res);
  };

  const postRenew = async (authorization?: string) =>
    await renewRoute(
      createAPIContext(harness.portal, {
        headers: authorization == null ? {} : { authorization },
      }),
    );

  const postUnlink = async (key: string) =>
    await unlinkRoute(createAPIContext(harness.portal, { params: { key } }));

  describe("POST /api/desktop/link", () => {
    it("should refuse a visitor who is not logged in", async () => {
      const res = await postLink({ fingerprint: HASH_A, name: "stand-1" });
      expect(res.status).toBe(401);
      expect(await body(res)).toEqual({ error: "Log in first" });
    });

    it("should answer the key, the secret, the activation, the email, and the user", async () => {
      const linked = await linkAs("user_a");
      expect(linked.email).toBe("ada@example.com");
      expect(linked.user).toBe("user_a");
      const [act] = await store.query
        .select()
        .from(activation)
        .where(eq(activation.key, linked.activation));
      expect(act.renewalSecretHash).toBe(hashSecret(linked.secret));
      expect(readKey(linked.key)).toMatchObject({
        edition: "d",
        machines: 1,
        fingerprints: [HASH_A],
      });
    });

    it("should refuse a fingerprint that is not a host hash", async () => {
      harness.signIn("user_a");
      const res = await postLink({ fingerprint: "nope", name: "stand-1" });
      expect(res.status).toBe(400);
      expect(await body(res)).toEqual({ error: '"nope" is not a host hash' });
    });

    it("should refuse an empty fingerprint", async () => {
      harness.signIn("user_a");
      const res = await postLink({ fingerprint: "", name: "stand-1" });
      expect(res.status).toBe(400);
      expect(await body(res)).toEqual({ error: "Paste at least one host hash" });
    });

    it("should refuse a machine without a name", async () => {
      harness.signIn("user_a");
      const res = await postLink({ fingerprint: HASH_A, name: "  " });
      expect(res.status).toBe(400);
      expect(await body(res)).toEqual({ error: "The machine needs a name" });
    });

    it("should refuse links past the hourly limit", async () => {
      harness.signIn("user_a");
      for (let i = 0; i < PER_ACTOR; i++)
        expect((await postLink({ fingerprint: HASH_A, name: "stand-1" })).status).toBe(
          200,
        );
      const res = await postLink({ fingerprint: HASH_A, name: "stand-1" });
      expect(res.status).toBe(429);
      expect(await body(res)).toEqual({
        error: "Too many activations. Try again later.",
      });
    });
  });

  describe("OPTIONS /api/desktop/renew", () => {
    it("should answer the preflight for any origin", async () => {
      const res = await OPTIONS(
        createAPIContext(harness.portal, { method: "OPTIONS" }),
      );
      expect(res.status).toBe(204);
      expect(Object.fromEntries(res.headers)).toMatchObject({
        "access-control-allow-origin": "*",
        "access-control-allow-methods": "POST, OPTIONS",
        "access-control-allow-headers": "authorization",
      });
    });
  });

  describe("POST /api/desktop/renew", () => {
    it("should renew the machine the bearer secret belongs to", async () => {
      const linked = await linkAs("user_a");
      harness.signIn(null);
      const res = await postRenew(`Bearer ${linked.secret}`);
      expect(res.status).toBe(200);
      expect(res.headers.get("access-control-allow-origin")).toBe("*");
      const { key } = await body<{ key: string }>(res);
      expect(readKey(key)).toMatchObject({ fingerprints: [HASH_A] });
      const renewals = (await store.query.select().from(event)).filter(
        (e) => e.kind === "renew",
      );
      expect(renewals).toHaveLength(1);
    });

    it("should refuse a request without a bearer secret", async () => {
      const res = await postRenew();
      expect(res.status).toBe(401);
      expect(res.headers.get("access-control-allow-origin")).toBe("*");
      expect(await body(res)).toEqual({ error: "Log in first" });
    });

    it("should refuse an unknown secret", async () => {
      const res = await postRenew("Bearer unknown");
      expect(res.status).toBe(403);
      expect(res.headers.get("access-control-allow-origin")).toBe("*");
      expect(await body(res)).toEqual({ error: "This machine is not logged in" });
    });

    it("should refuse a machine that was logged out", async () => {
      const linked = await linkAs("user_a");
      expect((await postUnlink(linked.activation)).status).toBe(204);
      const res = await postRenew(`Bearer ${linked.secret}`);
      expect(res.status).toBe(403);
      expect(await body(res)).toEqual({ error: "This machine is not logged in" });
    });
  });

  describe("POST /api/activations/[key]/unlink", () => {
    it("should refuse a visitor who is not logged in", async () => {
      const linked = await linkAs("user_a");
      harness.signIn(null);
      const res = await postUnlink(linked.activation);
      expect(res.status).toBe(401);
    });

    it("should answer someone else's machine as unknown", async () => {
      const linked = await linkAs("user_a");
      harness.signIn("user_b");
      const res = await postUnlink(linked.activation);
      expect(res.status).toBe(404);
      expect(await body(res)).toEqual({ error: "Activation not found" });
    });

    it("should answer 404 for an unknown machine", async () => {
      harness.signIn("user_a");
      const res = await postUnlink(crypto.randomUUID());
      expect(res.status).toBe(404);
      expect(await body(res)).toEqual({ error: "Activation not found" });
    });

    it("should log the machine out and revoke its license", async () => {
      const linked = await linkAs("user_a");
      expect((await postUnlink(linked.activation)).status).toBe(204);
      const [act] = await store.query
        .select()
        .from(activation)
        .where(eq(activation.key, linked.activation));
      expect(act.renewalSecretHash).toBeNull();
      const [lic] = await store.query
        .select()
        .from(license)
        .where(eq(license.key, act.license));
      expect(lic.revokedAt).not.toBeNull();
    });

    it("should refuse a team member unlinking an enterprise machine", async () => {
      const org = await createOrganization(store, {
        kind: "team",
        clerkOrgID: "org_a",
      });
      harness.directory.members.user_a = [
        { clerkOrgID: "org_a", name: "Acme", role: "org:member" },
      ];
      const lic = await createLicense(store, { organization: org.key });
      const act = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_B],
      });
      harness.signIn("user_a");
      const res = await postUnlink(act.key);
      expect(res.status).toBe(400);
      expect(await body(res)).toEqual({
        error: "Only a Synnax Desktop machine can be logged out",
      });
    });
  });
});
