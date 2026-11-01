// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createHmac } from "node:crypto";

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { POST } from "@/pages/api/webhooks/clerk";
import { activation, event, license, organization, referral } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { HASH_A, NOW } from "@/server/license/testutil";
import { joined } from "@/server/slack";
import {
  createActivation,
  createAPIContext,
  createHarness,
  createLicense,
  createOrganization,
  createReferral,
  type Harness,
  WEBHOOK_SECRET,
} from "@/testutil";

interface Signing {
  secret?: string;
  timestamp?: number;
}

const signature = (
  payload: unknown,
  id: string,
  timestamp: string,
  secret: string,
): string => {
  const key = Buffer.from(secret.slice("whsec_".length), "base64");
  const signed = `${id}.${timestamp}.${JSON.stringify(payload)}`;
  return `v1,${createHmac("sha256", key).update(signed).digest("base64")}`;
};

const userCreated = (data: Record<string, unknown>) => ({
  type: "user.created",
  data: {
    id: "user_ada",
    first_name: null,
    last_name: null,
    username: null,
    email_addresses: [],
    ...data,
  },
});

describe("POST /api/webhooks/clerk", () => {
  let store: Memory;
  let h: Harness;

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    h = createHarness(store);
  });

  const post = async (
    payload: unknown,
    { secret = WEBHOOK_SECRET, timestamp = Date.now() }: Signing = {},
  ): Promise<Response> => {
    const id = "msg_1";
    const seconds = String(Math.floor(timestamp / 1000));
    return await POST(
      createAPIContext(h.portal, {
        body: payload,
        headers: {
          "svix-id": id,
          "svix-timestamp": seconds,
          "svix-signature": signature(payload, id, seconds, secret),
        },
      }),
    );
  };

  const organizations = async () => await store.query.select().from(organization);

  describe("verification", () => {
    beforeEach(() => {
      vi.spyOn(console, "error").mockImplementation(() => {});
    });

    afterEach(() => {
      vi.restoreAllMocks();
    });

    const expectRejected = async (res: Response): Promise<void> => {
      expect(res.status).toBe(400);
      expect(await res.text()).toBe("Bad signature");
      expect(await organizations()).toEqual([]);
    };

    it("should reject a request without the Svix headers", async () => {
      await expectRejected(
        await POST(createAPIContext(h.portal, { body: userCreated({}) })),
      );
    });

    it("should reject a request signed with another secret", async () => {
      await expectRejected(
        await post(userCreated({}), {
          secret: `whsec_${Buffer.from("other-secret").toString("base64")}`,
        }),
      );
    });

    it("should reject a request signed long ago", async () => {
      await expectRejected(
        await post(userCreated({}), { timestamp: Date.now() - 60 * 60 * 1000 }),
      );
    });
  });

  describe("user.created", () => {
    it("should create a personal organization named for the user", async () => {
      const res = await post(userCreated({ first_name: "Ada", last_name: "Lovelace" }));
      expect(res.status).toBe(204);
      expect(await organizations()).toEqual([
        expect.objectContaining({
          kind: "personal",
          name: "Ada Lovelace",
          ownerUserID: "user_ada",
          clerkOrgID: null,
        }),
      ]);
    });

    it("should use the first name alone when there is no last name", async () => {
      await post(userCreated({ first_name: "Ada", last_name: "" }));
      expect((await organizations())[0].name).toBe("Ada");
    });

    it("should fall back to the username", async () => {
      await post(userCreated({ username: "ada" }));
      expect((await organizations())[0].name).toBe("ada");
    });

    it("should fall back to the first email address", async () => {
      await post(
        userCreated({
          email_addresses: [
            { email_address: "ada@example.com" },
            { email_address: "other@example.com" },
          ],
        }),
      );
      expect((await organizations())[0].name).toBe("ada@example.com");
    });

    it("should fall back to the user id", async () => {
      await post(userCreated({}));
      expect((await organizations())[0].name).toBe("user_ada");
    });

    it("should post the new account to Slack", async () => {
      await post(
        userCreated({
          first_name: "Ada",
          last_name: "Lovelace",
          primary_email_address_id: "email_2",
          email_addresses: [
            { id: "email_1", email_address: "old@example.com" },
            { id: "email_2", email_address: "ada@example.com" },
          ],
        }),
      );
      expect(h.slack.posted).toEqual([
        joined({ userID: "user_ada", name: "Ada Lovelace", email: "ada@example.com" }),
      ]);
    });

    it("should not post a Synnax Labs account to Slack", async () => {
      await post(
        userCreated({ email_addresses: [{ email_address: "em@synnaxlabs.com" }] }),
      );
      expect(await organizations()).toHaveLength(1);
      expect(h.slack.posted).toEqual([]);
    });

    it("should create the organization only once", async () => {
      await post(userCreated({ first_name: "Ada" }));
      await post(userCreated({ first_name: "Ada" }));
      expect(await organizations()).toHaveLength(1);
    });
  });

  describe("user.deleted", () => {
    const userDeleted = {
      type: "user.deleted",
      data: { id: "user_ada", deleted: true },
    };

    it("should delete the user's personal organization with its machines", async () => {
      const personal = await createOrganization(store, {
        kind: "personal",
        ownerUserID: "user_ada",
      });
      const lic = await createLicense(store, {
        organization: personal.key,
        edition: "desktop",
      });
      const act = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
      });
      await store.query.insert(event).values({
        at: NOW,
        kind: "link",
        actor: "user_ada",
        organization: personal.key,
        license: lic.key,
        activation: act.key,
      });
      expect((await post(userDeleted)).status).toBe(204);
      expect(await organizations()).toEqual([]);
      expect(await store.query.select().from(license)).toEqual([]);
      expect(await store.query.select().from(activation)).toEqual([]);
      expect(await store.query.select().from(event)).toEqual([]);
    });

    it("should delete the referral the user claimed", async () => {
      await createReferral(store, { userID: "user_ada", claimedAt: NOW });
      await post(userDeleted);
      expect(await store.query.select().from(referral)).toEqual([]);
    });

    it("should keep the personal organizations of other users", async () => {
      await createOrganization(store, { kind: "personal", ownerUserID: "user_bob" });
      await post(userDeleted);
      expect(await organizations()).toEqual([
        expect.objectContaining({ ownerUserID: "user_bob" }),
      ]);
    });

    it("should acknowledge a user the portal never saw", async () => {
      expect((await post(userDeleted)).status).toBe(204);
    });
  });

  describe("organization events", () => {
    it("should mirror a created Clerk organization as a team", async () => {
      h.directory.organizations.push({ clerkOrgID: "org_acme", name: "Acme" });
      const res = await post({
        type: "organization.created",
        data: { id: "org_acme", name: "Acme" },
      });
      expect(res.status).toBe(204);
      expect(await organizations()).toEqual([
        expect.objectContaining({
          kind: "team",
          name: "Acme",
          clerkOrgID: "org_acme",
          ownerUserID: null,
        }),
      ]);
    });

    it("should rename the team when the Clerk organization is updated", async () => {
      h.directory.organizations.push({ clerkOrgID: "org_acme", name: "Acme" });
      await post({
        type: "organization.created",
        data: { id: "org_acme", name: "Acme" },
      });
      await h.directory.rename("org_acme", "Acme Rockets");
      await post({
        type: "organization.updated",
        data: { id: "org_acme", name: "Acme Rockets" },
      });
      expect(await organizations()).toEqual([
        expect.objectContaining({ clerkOrgID: "org_acme", name: "Acme Rockets" }),
      ]);
    });

    it("should mirror an updated organization it has not seen", async () => {
      h.directory.organizations.push({ clerkOrgID: "org_new", name: "Newco" });
      await post({
        type: "organization.updated",
        data: { id: "org_new", name: "Newco" },
      });
      expect(await organizations()).toEqual([
        expect.objectContaining({ kind: "team", clerkOrgID: "org_new", name: "Newco" }),
      ]);
    });

    it("should mirror the name Clerk holds over a late event's", async () => {
      h.directory.organizations.push({ clerkOrgID: "org_acme", name: "Acme Rockets" });
      await post({
        type: "organization.updated",
        data: { id: "org_acme", name: "Acme" },
      });
      expect(await organizations()).toEqual([
        expect.objectContaining({ clerkOrgID: "org_acme", name: "Acme Rockets" }),
      ]);
    });

    it("should not mirror a late event for a deleted organization", async () => {
      const res = await post({
        type: "organization.updated",
        data: { id: "org_gone", name: "Gone" },
      });
      expect(res.status).toBe(204);
      expect(await organizations()).toEqual([]);
    });
  });

  it("should acknowledge an event it does not handle", async () => {
    const res = await post({ type: "session.created", data: { id: "sess_1" } });
    expect(res.status).toBe(204);
    expect(await organizations()).toEqual([]);
  });
});
