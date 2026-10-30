// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";
import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { POST as acceptRoute } from "@/pages/api/invitations/[key]/accept";
import { schema } from "@/server/db";
import { failingEvents, type Memory, openMemory } from "@/server/db/testutil";
import { ADMIN_ROLE, MEMBER_ROLE, type Team } from "@/server/directory";
import { badRequest } from "@/server/errors";
import { NOW } from "@/server/license/testutil";
import {
  body,
  type ContextArgs,
  createAPIContext,
  createHarness,
  createInvitation,
  createOrganization,
  type Harness,
} from "@/testutil";

const ACME = "org_acme";
const CY = "user_cy";
const MEMBER = "user_member";
const OTHER = "user_other";

describe("invitation routes", () => {
  let store: Memory;
  let h: Harness;
  let acme: schema.Organization;

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    h = createHarness(store);
    h.directory.organizations.push({ clerkOrgID: ACME, name: "Acme" });
    addPerson(CY);
    addPerson(MEMBER, [{ clerkOrgID: ACME, name: "Acme", role: MEMBER_ROLE }]);
    addPerson(OTHER);
    acme = await createOrganization(store, { kind: "team", clerkOrgID: ACME });
  });

  const addPerson = (userID: string, teams: Team[] = []): void => {
    h.directory.people[userID] = { email: `${userID}@acme.com`, name: userID };
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

  const invitations = async () => await store.query.select().from(schema.invitation);

  const roleOf = (userID: string): string | undefined =>
    h.directory.members[userID]?.find((t) => t.clerkOrgID === ACME)?.role;

  describe("POST /api/invitations/[key]/accept", () => {
    const accept = async (key: string) => await call(acceptRoute, { params: { key } });

    const invite = async (role = MEMBER_ROLE, email = "user_cy@acme.com") =>
      await createInvitation(store, { organization: acme.key, email, role });

    it("should put the invitee on the team with the invitation's role", async () => {
      const sent = await invite(ADMIN_ROLE);
      h.signIn(CY);
      expect((await accept(sent.key)).status).toBe(204);
      expect(roleOf(CY)).toBe(ADMIN_ROLE);
      expect(await invitations()).toEqual([]);
    });

    it("should record the acceptance", async () => {
      const sent = await invite();
      h.signIn(CY);
      await accept(sent.key);
      expect(await store.query.select().from(schema.event)).toEqual([
        expect.objectContaining({
          kind: "accept_invitation",
          at: NOW,
          actor: CY,
          organization: acme.key,
          detail: { email: "user_cy@acme.com", role: MEMBER_ROLE },
        }),
      ]);
    });

    it("should accept an invitation only once", async () => {
      const sent = await invite();
      h.signIn(CY);
      await accept(sent.key);
      await expectError(await accept(sent.key), 404, "Invitation not found");
    });

    it("should refuse a user logged in with another address", async () => {
      const sent = await invite();
      h.signIn(OTHER);
      await expectError(
        await accept(sent.key),
        403,
        "This invitation is for user_cy@acme.com. Log in with that email to accept it.",
      );
      expect(roleOf(OTHER)).toBeUndefined();
      expect(await invitations()).toEqual([sent]);
    });

    it("should refuse an expired invitation", async () => {
      const sent = await createInvitation(store, {
        organization: acme.key,
        email: "user_cy@acme.com",
        expiresAt: NOW,
      });
      h.signIn(CY);
      await expectError(
        await accept(sent.key),
        400,
        "This invitation expired. Ask a team admin to send it again.",
      );
      expect(roleOf(CY)).toBeUndefined();
    });

    it("should answer 404 for an unknown or malformed key", async () => {
      h.signIn(CY);
      for (const key of [crypto.randomUUID(), "inv_1"])
        await expectError(await accept(key), 404, "Invitation not found");
    });

    it("should keep the role of someone already on the team", async () => {
      const sent = await invite(ADMIN_ROLE, "user_member@acme.com");
      h.signIn(MEMBER);
      expect((await accept(sent.key)).status).toBe(204);
      expect(roleOf(MEMBER)).toBe(MEMBER_ROLE);
      expect(await invitations()).toEqual([]);
    });

    it("should keep the invitation when Clerk refuses the membership", async () => {
      const sent = await invite();
      h.directory.add = async () => {
        throw badRequest("The team is full.");
      };
      h.signIn(CY);
      await expectError(await accept(sent.key), 400, "The team is full.");
      expect(await invitations()).toEqual([sent]);
      expect(await store.query.select().from(schema.event)).toEqual([]);
    });

    it("should keep the invitation when the event cannot be recorded", async () => {
      const sent = await invite();
      h.signIn(CY);
      const res = await failingEvents(
        store,
        "accept_invitation",
        async () => await accept(sent.key),
      );
      await expectError(res, 500, "Internal error");
      expect(roleOf(CY)).toBeUndefined();
      expect(await invitations()).toEqual([sent]);
    });

    it("should refuse a visitor who is not logged in", async () => {
      const sent = await invite();
      await expectError(await accept(sent.key), 401, "Log in first");
    });
  });
});
