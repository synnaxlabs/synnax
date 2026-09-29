// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ADMIN_ROLE, memory } from "@/server/directory";
import { emails, resolve } from "@/server/session";
import { STAFF_ORG_ID } from "@/testutil";

const ADA = { email: "ada@example.com", name: "Ada Lovelace" };

describe("session", () => {
  describe("resolve", () => {
    it("should throw a 401 when nobody is signed in", async () => {
      await expect(resolve(memory(), null, STAFF_ORG_ID)).rejects.toMatchObject({
        status: 401,
        message: "Sign in first",
      });
    });

    it("should read the user and their teams", async () => {
      const directory = memory({
        people: { user_a: ADA },
        members: {
          user_a: [
            { clerkOrgID: "org_acme", name: "Acme", role: "org:member" },
            { clerkOrgID: "org_globex", name: "Globex", role: ADMIN_ROLE },
          ],
        },
      });
      expect(await resolve(directory, "user_a", STAFF_ORG_ID)).toEqual({
        userID: "user_a",
        email: "ada@example.com",
        name: "Ada Lovelace",
        teams: [
          { clerkOrgID: "org_acme", name: "Acme", role: "org:member" },
          { clerkOrgID: "org_globex", name: "Globex", role: ADMIN_ROLE },
        ],
        clerkOrgIDs: ["org_acme", "org_globex"],
        staff: false,
      });
    });

    it("should mark an admin of the staff organization as staff", async () => {
      const directory = memory({
        people: { user_a: ADA },
        members: {
          user_a: [{ clerkOrgID: STAFF_ORG_ID, name: "Synnax Labs", role: ADMIN_ROLE }],
        },
      });
      expect((await resolve(directory, "user_a", STAFF_ORG_ID)).staff).toBe(true);
    });

    it("should not mark a plain member of staff as staff", async () => {
      const directory = memory({
        people: { user_a: ADA },
        members: {
          user_a: [
            { clerkOrgID: STAFF_ORG_ID, name: "Synnax Labs", role: "org:member" },
          ],
        },
      });
      expect((await resolve(directory, "user_a", STAFF_ORG_ID)).staff).toBe(false);
    });

    it("should resolve a user with no teams", async () => {
      const directory = memory({ people: { user_a: ADA } });
      expect(await resolve(directory, "user_a", STAFF_ORG_ID)).toMatchObject({
        teams: [],
        clerkOrgIDs: [],
        staff: false,
      });
    });
  });

  describe("emails", () => {
    const directory = memory({
      people: {
        user_a: ADA,
        user_b: { email: "bob@example.com", name: "Bob" },
        user_c: { email: "", name: "No Mail" },
      },
      members: {
        user_a: [{ clerkOrgID: "org_acme", name: "Acme", role: ADMIN_ROLE }],
        user_b: [{ clerkOrgID: "org_acme", name: "Acme", role: "org:member" }],
      },
    });

    it("should mail the owner of a personal organization", async () => {
      expect(
        await emails(directory, {
          kind: "personal",
          ownerUserID: "user_a",
          clerkOrgID: null,
        }),
      ).toEqual(["ada@example.com"]);
    });

    it("should mail nobody for a personal owner without an address", async () => {
      expect(
        await emails(directory, {
          kind: "personal",
          ownerUserID: "user_c",
          clerkOrgID: null,
        }),
      ).toEqual([]);
    });

    it("should mail nobody for a personal organization without an owner", async () => {
      expect(
        await emails(directory, {
          kind: "personal",
          ownerUserID: null,
          clerkOrgID: null,
        }),
      ).toEqual([]);
    });

    it("should mail only the admins of a team", async () => {
      expect(
        await emails(directory, {
          kind: "team",
          ownerUserID: null,
          clerkOrgID: "org_acme",
        }),
      ).toEqual(["ada@example.com"]);
    });

    it("should mail nobody for a team without a Clerk id", async () => {
      expect(
        await emails(directory, { kind: "team", ownerUserID: null, clerkOrgID: null }),
      ).toEqual([]);
    });
  });
});
