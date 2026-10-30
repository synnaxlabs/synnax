// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { type schema } from "@/server/db";
import { type Memory, openMemory } from "@/server/db/testutil";
import { addressedTo, linked, list, retrieve } from "@/server/invitation";
import { NOW } from "@/server/license/testutil";
import { createInvitation, createOrganization } from "@/testutil";

const at = (iso: string): Date => new Date(iso);

describe("invitation", () => {
  let store: Memory;
  let acme: schema.Organization;
  let globex: schema.Organization;

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    acme = await createOrganization(store, { kind: "team", clerkOrgID: "org_acme" });
    globex = await createOrganization(store, {
      kind: "team",
      name: "Globex",
      clerkOrgID: "org_globex",
    });
  });

  describe("retrieve", () => {
    it("should return an invitation with its team", async () => {
      const sent = await createInvitation(store, {
        organization: acme.key,
        email: "cy@acme.com",
      });
      expect(await retrieve(store.query, sent.key)).toEqual({
        invitation: sent,
        team: { ...acme, clerkOrgID: "org_acme" },
      });
    });

    it("should return null for an unknown or malformed key", async () => {
      expect(await retrieve(store.query, crypto.randomUUID())).toBeNull();
      expect(await retrieve(store.query, "inv_1")).toBeNull();
    });
  });

  describe("linked", () => {
    it("should find the invitation behind a page path and its query", async () => {
      const sent = await createInvitation(store, {
        organization: acme.key,
        email: "cy@acme.com",
      });
      const found = await linked(store.query, `/invitations/${sent.key}?join`);
      expect(found?.invitation).toEqual(sent);
    });

    it("should return null for any other path", async () => {
      const { key } = await createInvitation(store, {
        organization: acme.key,
        email: "cy@acme.com",
      });
      for (const path of [
        "/",
        `/licenses/${key}`,
        `/invitations/${key}/accept`,
        `/admin/invitations/${key}`,
      ])
        expect(await linked(store.query, path)).toBeNull();
    });
  });

  describe("addressedTo", () => {
    it("should return open invitations to the addresses, newest first", async () => {
      const older = await createInvitation(store, {
        organization: acme.key,
        email: "cy@acme.com",
        sentAt: at("2026-09-01T00:00:00Z"),
      });
      const newer = await createInvitation(store, {
        organization: globex.key,
        email: "cy@home.com",
      });
      await createInvitation(store, { organization: acme.key, email: "dee@acme.com" });
      await createInvitation(store, {
        organization: globex.key,
        email: "cy@acme.com",
        expiresAt: NOW,
      });
      const found = await addressedTo(store.query, ["cy@acme.com", "cy@home.com"], NOW);
      expect(found.map((i) => i.invitation)).toEqual([newer, older]);
      expect(found.map((i) => i.team.name)).toEqual(["Globex", "Acme"]);
    });

    it("should return none for a user with no verified address", async () => {
      await createInvitation(store, { organization: acme.key, email: "cy@acme.com" });
      expect(await addressedTo(store.query, [], NOW)).toEqual([]);
    });
  });

  describe("list", () => {
    it("should return a team's invitations, expired ones included", async () => {
      const older = await createInvitation(store, {
        organization: acme.key,
        email: "cy@acme.com",
        sentAt: at("2026-09-01T00:00:00Z"),
        expiresAt: at("2026-09-02T00:00:00Z"),
      });
      const newer = await createInvitation(store, {
        organization: acme.key,
        email: "dee@acme.com",
      });
      await createInvitation(store, { organization: globex.key, email: "cy@acme.com" });
      expect(await list(store.query, acme.key)).toEqual([newer, older]);
    });
  });
});
