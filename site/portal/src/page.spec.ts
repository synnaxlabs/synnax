// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { failure, invitee, load, loadScoped, readActivity, type Scoped } from "@/page";
import { event, type EventKind, type Organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { HTTPError } from "@/server/errors";
import { HASH_A, NOW } from "@/server/license/testutil";
import {
  createActivation,
  createAPIContext,
  createHarness,
  createLicense,
  createOrganization,
  type Harness,
  STAFF_ORG_ID,
} from "@/testutil";

const ACME = { clerkOrgID: "org_acme", name: "Acme", role: "org:member" };

describe("page", () => {
  let store: Memory;
  let harness: Harness;
  beforeAll(async () => {
    store = await openMemory();
  });
  beforeEach(async () => {
    await store.clear();
    harness = createHarness(store);
    harness.directory.people.user_a = { email: "ada@example.com", name: "Ada" };
  });
  afterEach(() => vi.restoreAllMocks());

  const signInWithTeam = (): void => {
    harness.directory.members.user_a = [ACME];
    harness.signIn("user_a");
  };

  describe("load", () => {
    it("should return the portal and the logged-in session", async () => {
      harness.signIn("user_a");
      const loaded = await load(createAPIContext(harness.portal, { method: "GET" }));
      if (loaded instanceof Response) throw new Error("expected a session");
      expect(loaded.portal).toBe(harness.portal);
      expect(loaded.session).toMatchObject({ userID: "user_a", name: "Ada" });
    });

    it("should redirect a logged-out visitor to log in and back", async () => {
      const res = await load(
        createAPIContext(harness.portal, {
          method: "GET",
          path: "/licenses/abc?org=xyz",
        }),
      );
      if (!(res instanceof Response)) throw new Error("expected a redirect");
      expect(res.status).toBe(303);
      expect(res.headers.get("location")).toBe(
        "/login?redirect_url=%2Flicenses%2Fabc%3Forg%3Dxyz",
      );
    });

    it("should rethrow any failure other than a 401", async () => {
      harness.signIn("user_unknown");
      await expect(
        load(createAPIContext(harness.portal, { method: "GET" })),
      ).rejects.toThrow("no Clerk user user_unknown");
    });
  });

  describe("invitee", () => {
    it("should return a logged-in visitor's email and invitations", async () => {
      harness.directory.invited.user_a = [
        { invitationID: "orginv_1", clerkOrgID: "org_acme", name: "Acme" },
      ];
      harness.signIn("user_a");
      const invited = await invitee(
        createAPIContext(harness.portal, { method: "GET" }),
      );
      expect(invited?.email).toBe("ada@example.com");
      expect(invited?.invitations).toMatchObject([
        { invitationID: "orginv_1", team: { kind: "team", name: "Acme" } },
      ]);
    });

    it("should return null for a logged-out visitor", async () => {
      expect(
        await invitee(createAPIContext(harness.portal, { method: "GET" })),
      ).toBeNull();
    });
  });

  describe("loadScoped", () => {
    const scopedOf = (res: Scoped | Response): Scoped => {
      if (res instanceof Response) throw new Error("expected a scoped page");
      return res;
    };

    it("should act for the user's first team by default", async () => {
      signInWithTeam();
      const loaded = scopedOf(
        await loadScoped(createAPIContext(harness.portal, { method: "GET" })),
      );
      expect(loaded.organizations.map((o) => o.kind)).toEqual(["personal", "team"]);
      expect(loaded.scope).toMatchObject({ kind: "team", name: "Acme" });
    });

    it("should act for the requested organization and remember it", async () => {
      signInWithTeam();
      const first = scopedOf(
        await loadScoped(createAPIContext(harness.portal, { method: "GET" })),
      );
      const personal = first.organizations.find((o) => o.kind === "personal");
      if (personal == null) throw new Error("expected a personal organization");
      const context = createAPIContext(harness.portal, {
        method: "GET",
        path: `/?org=${personal.key}`,
      });
      const loaded = scopedOf(await loadScoped(context));
      expect(loaded.scope.key).toBe(personal.key);
      expect(context.jar.get("scope")).toBe(personal.key);
    });

    it("should act for the remembered organization without a request", async () => {
      signInWithTeam();
      const first = scopedOf(
        await loadScoped(createAPIContext(harness.portal, { method: "GET" })),
      );
      const personal = first.organizations.find((o) => o.kind === "personal");
      if (personal == null) throw new Error("expected a personal organization");
      const context = createAPIContext(harness.portal, {
        method: "GET",
        cookies: { scope: personal.key },
      });
      expect(scopedOf(await loadScoped(context)).scope.key).toBe(personal.key);
    });

    it("should redirect home for an organization the user is not in", async () => {
      signInWithTeam();
      const context = createAPIContext(harness.portal, {
        method: "GET",
        path: `/?org=${crypto.randomUUID()}`,
      });
      const res = await loadScoped(context);
      if (!(res instanceof Response)) throw new Error("expected a redirect");
      expect(res.status).toBe(303);
      expect(res.headers.get("location")).toBe("/");
      expect(context.jar.has("scope")).toBe(false);
    });

    it("should redirect a scope of the wrong kind to its overview", async () => {
      signInWithTeam();
      const res = await loadScoped(
        createAPIContext(harness.portal, { method: "GET", path: "/devices" }),
        "personal",
      );
      if (!(res instanceof Response)) throw new Error("expected a redirect");
      expect(res.status).toBe(303);
      const location = res.headers.get("location") ?? "";
      expect(location).toMatch(/^\/\?org=[0-9a-f-]{36}$/);
      const team = scopedOf(
        await loadScoped(createAPIContext(harness.portal, { method: "GET" })),
      ).scope;
      expect(location).toBe(`/?org=${team.key}`);
    });

    it("should pass a scope of the right kind", async () => {
      signInWithTeam();
      const loaded = scopedOf(
        await loadScoped(createAPIContext(harness.portal, { method: "GET" }), "team"),
      );
      expect(loaded.scope.kind).toBe("team");
    });

    it("should redirect a logged-out visitor to log in", async () => {
      const res = await loadScoped(
        createAPIContext(harness.portal, { method: "GET", path: "/members" }),
      );
      if (!(res instanceof Response)) throw new Error("expected a redirect");
      expect(res.headers.get("location")).toBe("/login?redirect_url=%2Fmembers");
    });
  });

  describe("failure", () => {
    it("should show the status and message of an HTTP error", () => {
      expect(failure(new HTTPError(403, "Staff only"))).toEqual({
        status: 403,
        message: "Staff only",
      });
    });

    it("should hide any other error behind a 500", () => {
      const log = vi.spyOn(console, "error").mockImplementation(() => {});
      expect(failure(new Error("database down"))).toEqual({
        status: 500,
        message: "Something went wrong. Try again.",
      });
      expect(log).toHaveBeenCalledOnce();
    });
  });

  describe("readActivity", () => {
    let org: Organization;
    beforeEach(async () => {
      org = await createOrganization(store, { kind: "team", clerkOrgID: "org_a" });
    });

    const record = async (kind: EventKind, actor: string, extra = {}) => {
      const [row] = await store.query
        .insert(event)
        .values({ kind, actor, organization: org.key, at: NOW, ...extra })
        .returning();
      return row;
    };

    const read = async (search = "") =>
      await readActivity(
        harness.portal,
        new URL(`https://p.test/admin/activity${search}`),
      );

    it("should name the people in the page and not the machines", async () => {
      const lic = await createLicense(store, { organization: org.key });
      const machine = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
      });
      await record("issue", "user_a");
      await record("renew", machine.key, { activation: machine.key });
      const activity = await read();
      expect(activity.actors).toEqual({ user_a: "Ada" });
      expect(activity.page.entries).toHaveLength(2);
      expect(activity.olderHref).toBeNull();
    });

    it("should keep only the category the URL asks for", async () => {
      await record("issue", "user_a");
      const invited = await record("invite", "user_a");
      const { page } = await read("?category=members");
      expect(page.entries.map((e) => e.event)).toEqual([invited]);
    });

    it("should keep only what staff did when the URL asks", async () => {
      harness.directory.people.user_staff = { email: "s@synnaxlabs.com", name: "Sam" };
      harness.directory.members.user_staff = [
        { clerkOrgID: STAFF_ORG_ID, name: "Synnax Labs", role: "org:admin" },
      ];
      const staff = await record("issue", "user_staff");
      await record("activate", "user_a");
      const { page } = await read("?staff=1");
      expect(page.entries.map((e) => e.event)).toEqual([staff]);
    });

    it("should link the older events, keeping the filter", async () => {
      for (let i = 0; i < 51; i++) await record("download", "user_a");
      const first = await read("?category=licenses");
      expect(first.page.entries).toHaveLength(50);
      const next = first.page.entries[49].event.key;
      expect(first.olderHref).toBe(`/admin/activity?category=licenses&before=${next}`);
      const older = await read(`?category=licenses&before=${next}`);
      expect(older.page.entries).toHaveLength(1);
      expect(older.olderHref).toBeNull();
    });

    it("should refuse a category it does not know", async () => {
      await expect(read("?category=nope")).rejects.toThrow(
        new HTTPError(400, "Invalid category"),
      );
    });

    it("should refuse a page that is not a number", async () => {
      await expect(read("?before=abc")).rejects.toThrow(
        new HTTPError(400, "Invalid page"),
      );
    });
  });
});
