// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { failure, load, loadScoped, type Scoped } from "@/page";
import { type Memory, openMemory } from "@/server/db/testutil";
import { HTTPError } from "@/server/errors";
import { createAPIContext, createHarness, type Harness } from "@/testutil";

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
    it("should return the portal and the signed-in session", async () => {
      harness.signIn("user_a");
      const loaded = await load(createAPIContext(harness.portal, { method: "GET" }));
      if (loaded instanceof Response) throw new Error("expected a session");
      expect(loaded.portal).toBe(harness.portal);
      expect(loaded.session).toMatchObject({ userID: "user_a", name: "Ada" });
    });

    it("should redirect a signed-out visitor to sign in and back", async () => {
      const res = await load(
        createAPIContext(harness.portal, {
          method: "GET",
          path: "/licenses/abc?org=xyz",
        }),
      );
      if (!(res instanceof Response)) throw new Error("expected a redirect");
      expect(res.status).toBe(303);
      expect(res.headers.get("location")).toBe(
        "/sign-in?redirect_url=%2Flicenses%2Fabc%3Forg%3Dxyz",
      );
    });

    it("should rethrow any failure other than a 401", async () => {
      harness.signIn("user_unknown");
      await expect(
        load(createAPIContext(harness.portal, { method: "GET" })),
      ).rejects.toThrow("no Clerk user user_unknown");
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

    it("should redirect a signed-out visitor to sign in", async () => {
      const res = await loadScoped(
        createAPIContext(harness.portal, { method: "GET", path: "/members" }),
      );
      if (!(res instanceof Response)) throw new Error("expected a redirect");
      expect(res.headers.get("location")).toBe("/sign-in?redirect_url=%2Fmembers");
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
});
