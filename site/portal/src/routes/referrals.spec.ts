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
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { POST as createRoute } from "@/pages/api/referrals";
import { POST as mailRoute } from "@/pages/api/referrals/[key]/mail";
import { POST as visitRoute } from "@/pages/api/referrals/[key]/visit";
import { referral } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { ADMIN_ROLE, type Team } from "@/server/directory";
import { NOW } from "@/server/license/testutil";
import { MAIL_WAIT_MS } from "@/server/referral";
import { claimed } from "@/server/slack";
import {
  body,
  type ContextArgs,
  createAPIContext,
  createHarness,
  createReferral,
  type Harness,
  STAFF_ORG_ID,
} from "@/testutil";

const ADA = "user_ada";
const BOB = "user_bob";
const STAFF = "user_staff";

const STAFF_TEAM: Team = {
  clerkOrgID: STAFF_ORG_ID,
  name: "Synnax Labs",
  role: ADMIN_ROLE,
};

describe("referral routes", () => {
  let store: Memory;
  let h: Harness;

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    h = createHarness(store);
    addPerson(ADA, "Ada Lovelace");
    addPerson(BOB, "Bob");
    addPerson(STAFF, "Emiliano", [STAFF_TEAM]);
  });

  const addPerson = (userID: string, name: string, teams: Team[] = []): void => {
    h.directory.people[userID] = { email: `${userID}@example.com`, name };
    h.directory.members[userID] = teams;
  };

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const call = async (route: APIRoute, args: ContextArgs = {}): Promise<Response> =>
    await route(createAPIContext(h.portal, args));

  const stored = async (key: string) => {
    const [row] = await store.query
      .select()
      .from(referral)
      .where(eq(referral.key, key));
    return row;
  };

  describe("POST /api/referrals", () => {
    it("should refuse a visitor who is not logged in", async () => {
      const res = await call(createRoute, { body: { recipient: "Ada" } });
      expect(res.status).toBe(401);
    });

    it("should refuse a user who is not staff", async () => {
      h.signIn(ADA);
      const res = await call(createRoute, { body: { recipient: "Bob" } });
      expect(res.status).toBe(403);
      expect(await body(res)).toEqual({ error: "Staff only" });
    });

    it("should refuse a link for nobody", async () => {
      h.signIn(STAFF);
      const res = await call(createRoute, { body: { recipient: "  " } });
      expect(res.status).toBe(400);
      expect(await body(res)).toEqual({ error: "Say who the link is for" });
    });

    it("should make a link with a short random key", async () => {
      h.signIn(STAFF);
      const res = await call(createRoute, { body: { recipient: " Ada " } });
      expect(res.status).toBe(200);
      const { key } = await body<{ key: string }>(res);
      expect(key).toMatch(/^[A-Za-z0-9_-]{12}$/);
      expect(await stored(key)).toMatchObject({
        recipient: "Ada",
        createdBy: STAFF,
        createdAt: NOW,
        openedAt: null,
        userID: null,
      });
    });
  });

  describe("POST /api/referrals/:key/visit", () => {
    const visit = async (key: string) => await call(visitRoute, { params: { key } });

    it("should refuse an unknown link", async () => {
      const res = await visit("nope");
      expect(res.status).toBe(404);
      expect(await body(res)).toEqual({ error: "Referral not found" });
    });

    it("should record when a visitor who is not logged in opens the link", async () => {
      const ref = await createReferral(store);
      expect((await visit(ref.key)).status).toBe(204);
      expect(await stored(ref.key)).toMatchObject({ openedAt: NOW, userID: null });
    });

    it("should keep the time the link was first opened", async () => {
      const first = new Date(NOW.getTime() - 60_000);
      const ref = await createReferral(store, { openedAt: first });
      await visit(ref.key);
      expect((await stored(ref.key)).openedAt).toEqual(first);
    });

    it("should claim the link for a logged-in visitor and tell staff", async () => {
      const ref = await createReferral(store);
      h.signIn(ADA);
      expect((await visit(ref.key)).status).toBe(204);
      const after = await stored(ref.key);
      expect(after).toMatchObject({ openedAt: NOW, userID: ADA, claimedAt: NOW });
      expect(h.slack.posted).toEqual([
        claimed(
          { userID: ADA, name: "Ada Lovelace", email: "user_ada@example.com" },
          after,
          "Emiliano",
        ),
      ]);
    });

    it("should not tell staff about a claim by a Synnax Labs user", async () => {
      const ref = await createReferral(store);
      h.directory.people[ADA].email = "ada@synnaxlabs.com";
      h.signIn(ADA);
      await visit(ref.key);
      expect((await stored(ref.key)).userID).toBe(ADA);
      expect(h.slack.posted).toEqual([]);
    });

    it("should not claim a link that another user claimed", async () => {
      const ref = await createReferral(store, { userID: BOB, claimedAt: NOW });
      h.signIn(ADA);
      await visit(ref.key);
      expect((await stored(ref.key)).userID).toBe(BOB);
      expect(h.slack.posted).toEqual([]);
    });

    it("should not claim a second link for a user who claimed one", async () => {
      await createReferral(store, { key: "first-link-1", userID: ADA, claimedAt: NOW });
      const second = await createReferral(store, { key: "second-link1" });
      h.signIn(ADA);
      await visit(second.key);
      expect((await stored(second.key)).userID).toBeNull();
    });

    it("should count a staff visit as no visit", async () => {
      const ref = await createReferral(store);
      h.signIn(STAFF);
      expect((await visit(ref.key)).status).toBe(204);
      expect(await stored(ref.key)).toMatchObject({ openedAt: null, userID: null });
    });
  });

  describe("POST /api/referrals/:key/mail", () => {
    const mail = async (key: string) => await call(mailRoute, { params: { key } });

    it("should refuse a visitor who is not logged in", async () => {
      const ref = await createReferral(store);
      expect((await mail(ref.key)).status).toBe(401);
      expect(h.mail.sent).toEqual([]);
    });

    it("should refuse an unknown link", async () => {
      h.signIn(ADA);
      const res = await mail("nope");
      expect(res.status).toBe(404);
      expect(await body(res)).toEqual({ error: "Referral not found" });
    });

    it("should email the link to the logged-in user", async () => {
      const ref = await createReferral(store);
      h.signIn(ADA);
      expect((await mail(ref.key)).status).toBe(204);
      expect(h.mail.sent).toHaveLength(1);
      expect(h.mail.sent[0]).toMatchObject({
        to: ["user_ada@example.com"],
        subject: "Install Synnax Desktop on your computer",
        facts: [
          { label: "Account", value: "user_ada@example.com" },
          { label: "Runs on", value: "macOS on Apple silicon and Windows" },
        ],
        action: { label: "Continue setup", href: "/invite/Ada-link-001" },
      });
      expect((await stored(ref.key)).mailedAt).toEqual(NOW);
    });

    it("should refuse to email the link again within the wait", async () => {
      const ref = await createReferral(store, {
        mailedAt: new Date(NOW.getTime() - MAIL_WAIT_MS + 1000),
      });
      h.signIn(ADA);
      const res = await mail(ref.key);
      expect(res.status).toBe(429);
      expect(await body(res)).toEqual({
        error:
          "We emailed the link a moment ago. Check your inbox, or try again in a few " +
          "minutes.",
      });
      expect(h.mail.sent).toEqual([]);
    });

    it("should email the link again after the wait", async () => {
      const ref = await createReferral(store, {
        mailedAt: new Date(NOW.getTime() - MAIL_WAIT_MS - 1000),
      });
      h.signIn(ADA);
      expect((await mail(ref.key)).status).toBe(204);
      expect(h.mail.sent).toHaveLength(1);
    });

    it("should record nothing when the email fails", async () => {
      vi.spyOn(console, "error").mockImplementation(() => {});
      const ref = await createReferral(store);
      h.mail.send = async () => {
        throw new Error("Resend is down");
      };
      h.signIn(ADA);
      expect((await mail(ref.key)).status).toBe(500);
      expect((await stored(ref.key)).mailedAt).toBeNull();
    });
  });
});
