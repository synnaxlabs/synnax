// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { eq } from "drizzle-orm";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { POST as unlinkRoute } from "@/pages/api/activations/[key]/unlink";
import { POST as linkRoute } from "@/pages/api/desktop/link";
import { OPTIONS, POST as renewRoute } from "@/pages/api/desktop/renew";
import { POST as desktopUnlinkRoute } from "@/pages/api/desktop/unlink";
import { activation, event, license } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { hashSecret } from "@/server/license/desktop";
import { HASH_A, HASH_B, NOW } from "@/server/license/testutil";
import { PER_ACTOR } from "@/server/ratelimit";
import { CONVERTED } from "@/server/referral";
import { active, firstLogin } from "@/server/slack";
import { DAY_MS } from "@/server/usage";
import {
  body,
  createActivation,
  createAPIContext,
  createHarness,
  createLicense,
  createOrganization,
  createReferral,
  type Harness,
  readKey,
} from "@/testutil";

const TOMORROW = new Date(NOW.getTime() + DAY_MS);

const ADA = {
  userID: "user_a",
  name: "Ada Lovelace",
  email: "ada@example.com",
  joinedAt: new Date(0),
};

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
    harness.directory.people.user_em = { email: "em@synnaxlabs.com", name: "Emiliano" };
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const CLERK_DOWN = new Error("Clerk is down");

  /** breakNotes makes every Clerk account lookup fail, and catches what is logged. */
  const breakNotes = () => {
    harness.directory.account = async () => {
      throw CLERK_DOWN;
    };
    return vi.spyOn(console, "error").mockImplementation(() => {});
  };

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

  const postDesktopUnlink = async (authorization?: string) =>
    await desktopUnlinkRoute(
      createAPIContext(harness.portal, {
        headers: authorization == null ? {} : { authorization },
      }),
    );

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

    it("should refuse a fingerprint with a value that is not a hash", async () => {
      harness.signIn("user_a");
      const res = await postLink({ fingerprint: "nope", name: "stand-1" });
      expect(res.status).toBe(400);
      expect(await body(res)).toEqual({ error: '"nope" is not part of a fingerprint' });
    });

    it("should refuse an empty fingerprint", async () => {
      harness.signIn("user_a");
      const res = await postLink({ fingerprint: "", name: "stand-1" });
      expect(res.status).toBe(400);
      expect(await body(res)).toEqual({ error: "Enter the machine's fingerprint" });
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

    it("should post only the first Desktop login of each user to Slack", async () => {
      await linkAs("user_a");
      await linkAs("user_a", HASH_B);
      await linkAs("user_b");
      expect(harness.slack.posted).toEqual([
        firstLogin(
          { userID: "user_a", name: "Ada Lovelace", email: "ada@example.com" },
          "stand-1",
        ),
        firstLogin(
          { userID: "user_b", name: "Bob", email: "bob@example.com" },
          "stand-1",
        ),
      ]);
    });

    it("should not post the logins of a Synnax Labs user", async () => {
      await linkAs("user_em");
      harness.portal.now = () => TOMORROW;
      await linkAs("user_em", HASH_B);
      expect(harness.slack.posted).toEqual([]);
    });

    describe("referred user", () => {
      beforeEach(async () => {
        harness.directory.people.user_staff = {
          email: "em@synnaxlabs.com",
          name: "Emiliano",
        };
        await createReferral(store, { userID: "user_a", claimedAt: NOW });
      });

      const linkWith = async (analytics: boolean, fingerprint = HASH_A) => {
        harness.signIn("user_a");
        const res = await postLink({
          fingerprint,
          name: "stand-1",
          analytics: String(analytics),
        });
        expect(res.status).toBe(200);
      };

      it("should tell PostHog who referred the user at their first login", async () => {
        await linkWith(true);
        const props = {
          referral: "Ada-link-001",
          referred_by: "user_staff",
          referred_by_name: "Emiliano",
        };
        expect(harness.analytics.captured).toEqual([
          {
            userID: "user_a",
            event: CONVERTED,
            at: NOW,
            properties: props,
            once: { ...props, referred_at: NOW.toISOString() },
          },
        ]);
      });

      it("should tell PostHog nothing when the user refuses usage data", async () => {
        await linkWith(false);
        expect(harness.analytics.captured).toEqual([]);
      });

      it("should tell PostHog only about the first login", async () => {
        await linkWith(true);
        await linkWith(true, HASH_B);
        expect(harness.analytics.captured).toHaveLength(1);
      });

      it("should log the user in when the referral cannot be read", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {});
        harness.directory.names = async () => {
          throw CLERK_DOWN;
        };
        await linkWith(true);
        expect(harness.analytics.captured).toEqual([]);
        expect(harness.slack.posted).toEqual([
          firstLogin(
            { userID: "user_a", name: "Ada Lovelace", email: "ada@example.com" },
            "stand-1",
          ),
        ]);
        expect(logged).toHaveBeenCalledWith(
          "failed to read the referral of a first login",
          { userID: "user_a" },
          CLERK_DOWN,
        );
      });

      it("should name the referrer in the first login note", async () => {
        await linkWith(false);
        expect(harness.slack.posted).toEqual([
          firstLogin(
            { userID: "user_a", name: "Ada Lovelace", email: "ada@example.com" },
            "stand-1",
            "Emiliano",
          ),
        ]);
      });
    });

    it("should tell PostHog nothing about a user nobody referred", async () => {
      harness.signIn("user_a");
      await postLink({ fingerprint: HASH_A, name: "stand-1", analytics: "true" });
      expect(harness.analytics.captured).toEqual([]);
    });

    it("should not post a login after the user logged out", async () => {
      const linked = await linkAs("user_a");
      expect((await postDesktopUnlink(`Bearer ${linked.secret}`)).status).toBe(204);
      await linkAs("user_a");
      expect(harness.slack.posted).toHaveLength(1);
    });

    it("should post a returning user's first use of a day to Slack", async () => {
      await linkAs("user_a");
      harness.portal.now = () => TOMORROW;
      await linkAs("user_a", HASH_B);
      expect(harness.slack.posted.at(-1)).toEqual(
        active({
          account: ADA,
          machine: "stand-1",
          previous: NOW,
          at: TOMORROW,
          days: 2,
        }),
      );
    });

    it("should answer the new secret when the note about the login fails", async () => {
      const first = await linkAs("user_a");
      harness.portal.now = () => TOMORROW;
      const error = breakNotes();
      const again = await linkAs("user_a");
      expect(again.secret).not.toBe(first.secret);
      expect((await postRenew(`Bearer ${again.secret}`)).status).toBe(200);
      expect(error).toHaveBeenCalledWith("slack note failed", CLERK_DOWN);
      expect(harness.slack.posted).toHaveLength(1);
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

    it("should post the owner's first use of a day to Slack", async () => {
      const linked = await linkAs("user_a");
      harness.signIn(null);
      harness.portal.now = () => TOMORROW;
      expect((await postRenew(`Bearer ${linked.secret}`)).status).toBe(200);
      expect(harness.slack.posted).toEqual([
        firstLogin(ADA, "stand-1"),
        active({
          account: ADA,
          machine: "stand-1",
          previous: NOW,
          at: TOMORROW,
          days: 2,
        }),
      ]);
    });

    it("should not post the use of a Synnax Labs user", async () => {
      const linked = await linkAs("user_em");
      harness.signIn(null);
      harness.portal.now = () => TOMORROW;
      expect((await postRenew(`Bearer ${linked.secret}`)).status).toBe(200);
      expect(harness.slack.posted).toEqual([]);
    });

    it("should answer the key when the note about the renewal fails", async () => {
      const linked = await linkAs("user_a");
      harness.portal.now = () => TOMORROW;
      const error = breakNotes();
      const res = await postRenew(`Bearer ${linked.secret}`);
      expect(res.status).toBe(200);
      const { key } = await body<{ key: string }>(res);
      expect(readKey(key)).toMatchObject({ fingerprints: [HASH_A] });
      expect(error).toHaveBeenCalledWith("slack note failed", CLERK_DOWN);
      expect(harness.slack.posted).toHaveLength(1);
    });

    it("should not post a later use on the same day", async () => {
      const linked = await linkAs("user_a");
      harness.portal.now = () => new Date(NOW.getTime() + 6 * 60 * 60 * 1000);
      expect((await postRenew(`Bearer ${linked.secret}`)).status).toBe(200);
      expect(harness.slack.posted).toHaveLength(1);
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
      expect(await body(res)).toEqual({
        error: "This machine is not logged in",
        code: "unlinked",
      });
    });

    it("should refuse a machine that was logged out", async () => {
      const linked = await linkAs("user_a");
      expect((await postUnlink(linked.activation)).status).toBe(204);
      const res = await postRenew(`Bearer ${linked.secret}`);
      expect(res.status).toBe(403);
      expect(await body(res)).toEqual({
        error: "This machine is not logged in",
        code: "unlinked",
      });
    });
  });

  describe("POST /api/desktop/unlink", () => {
    it("should log out the machine the bearer secret belongs to", async () => {
      const linked = await linkAs("user_a");
      harness.signIn(null);
      const res = await postDesktopUnlink(`Bearer ${linked.secret}`);
      expect(res.status).toBe(204);
      expect(res.headers.get("access-control-allow-origin")).toBe("*");
      const [act] = await store.query
        .select()
        .from(activation)
        .where(eq(activation.key, linked.activation));
      expect(act.releasedAt).not.toBeNull();
      const unlinks = (await store.query.select().from(event)).filter(
        (e) => e.kind === "unlink",
      );
      expect(unlinks).toEqual([expect.objectContaining({ actor: linked.activation })]);
      expect((await postRenew(`Bearer ${linked.secret}`)).status).toBe(403);
    });

    it("should refuse a request without a bearer secret", async () => {
      const res = await postDesktopUnlink();
      expect(res.status).toBe(401);
      expect(res.headers.get("access-control-allow-origin")).toBe("*");
      expect(await body(res)).toEqual({ error: "Log in first" });
    });

    it("should refuse a machine that was logged out", async () => {
      const linked = await linkAs("user_a");
      expect((await postDesktopUnlink(`Bearer ${linked.secret}`)).status).toBe(204);
      const res = await postDesktopUnlink(`Bearer ${linked.secret}`);
      expect(res.status).toBe(403);
      expect(await body(res)).toEqual({
        error: "This machine is not logged in",
        code: "unlinked",
      });
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
