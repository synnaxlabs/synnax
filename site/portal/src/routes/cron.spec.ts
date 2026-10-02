// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { GET } from "@/pages/api/cron/expiry";
import { event, type License, type Organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { ADMIN_ROLE } from "@/server/directory";
import { NOW } from "@/server/license/testutil";
import {
  body,
  createAPIContext,
  createHarness,
  createLicense,
  createOrganization,
  CRON_SECRET,
  type Harness,
} from "@/testutil";

const ACME = "org_acme";
const ADMIN = "user_admin";
const DAY_MS = 24 * 60 * 60 * 1000;

describe("GET /api/cron/expiry", () => {
  let store: Memory;
  let h: Harness;
  let acme: Organization;
  let lic: License;

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    h = createHarness(store);
    h.directory.people[ADMIN] = { email: "admin@acme.com", name: "Admin" };
    h.directory.members[ADMIN] = [{ clerkOrgID: ACME, name: "Acme", role: ADMIN_ROLE }];
    acme = await createOrganization(store, { kind: "team", clerkOrgID: ACME });
    lic = await createLicense(store, {
      organization: acme.key,
      expiresAt: new Date(NOW.getTime() + 5 * DAY_MS),
    });
  });

  const get = async (authorization?: string): Promise<Response> =>
    await GET(
      createAPIContext(h.portal, {
        method: "GET",
        headers: authorization == null ? {} : { authorization },
      }),
    );

  it("should refuse a call without the cron secret", async () => {
    const res = await get();
    expect(res.status).toBe(401);
    expect(await res.text()).toBe("Unauthorized");
  });

  it("should refuse a call with the wrong secret", async () => {
    expect((await get("Bearer wrong")).status).toBe(401);
    expect(h.mail.sent).toEqual([]);
  });

  it("should warn the team's admins of a license that expires soon", async () => {
    const res = await get(`Bearer ${CRON_SECRET}`);
    expect(res.status).toBe(200);
    expect(await body(res)).toEqual({
      sent: [{ license: lic.key, days: 7, to: ["admin@acme.com"] }],
    });
    expect(h.mail.sent).toEqual([
      {
        to: ["admin@acme.com"],
        subject: "Your Synnax license expires in 7 days",
        text: expect.stringContaining('The Synnax license "Test rig" for Acme'),
      },
    ]);
    expect(await store.query.select().from(event)).toEqual([
      expect.objectContaining({
        kind: "expiry_notice",
        actor: "system",
        organization: acme.key,
        license: lic.key,
        detail: { days: 7, to: ["admin@acme.com"] },
      }),
    ]);
  });

  it("should not warn twice for the same window", async () => {
    await get(`Bearer ${CRON_SECRET}`);
    const res = await get(`Bearer ${CRON_SECRET}`);
    expect(await body(res)).toEqual({ sent: [] });
    expect(h.mail.sent).toHaveLength(1);
  });

  it("should warn the owner of a personal organization", async () => {
    h.directory.people.user_owner = { email: "owner@example.com", name: "Owner" };
    const personal = await createOrganization(store, {
      kind: "personal",
      ownerUserID: "user_owner",
    });
    const owned = await createLicense(store, {
      organization: personal.key,
      expiresAt: new Date(NOW.getTime() + 20 * DAY_MS),
    });
    const { sent } = await body<{ sent: unknown[] }>(
      await get(`Bearer ${CRON_SECRET}`),
    );
    expect(sent).toContainEqual({
      license: owned.key,
      days: 30,
      to: ["owner@example.com"],
    });
  });

  it("should skip a team with nobody to warn", async () => {
    h.directory.members[ADMIN] = [];
    const res = await get(`Bearer ${CRON_SECRET}`);
    expect(await body(res)).toEqual({ sent: [] });
    expect(await store.query.select().from(event)).toEqual([]);
  });
});
