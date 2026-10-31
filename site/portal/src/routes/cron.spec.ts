// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { GET as digestRoute } from "@/pages/api/cron/digest";
import { GET } from "@/pages/api/cron/expiry";
import { GET as sessionsRoute } from "@/pages/api/cron/sessions";
import { type Session, type Visit } from "@/server/analytics";
import { event, type License, type Organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { ADMIN_ROLE } from "@/server/directory";
import { NOW } from "@/server/license/testutil";
import { digest, recap } from "@/server/slack";
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
      expect.objectContaining({
        to: ["admin@acme.com"],
        subject: "Your Synnax license expires in 7 days",
        facts: expect.arrayContaining([{ label: "License", value: "Test rig" }]),
      }),
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

describe("GET /api/cron/digest", () => {
  let store: Memory;
  let h: Harness;

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    h = createHarness(store);
  });

  const get = async (authorization?: string): Promise<Response> =>
    await digestRoute(
      createAPIContext(h.portal, {
        method: "GET",
        headers: authorization == null ? {} : { authorization },
      }),
    );

  it("should refuse a call without the cron secret", async () => {
    const res = await get();
    expect(res.status).toBe(401);
    expect(h.slack.posted).toEqual([]);
  });

  it("should post the use of the day before to Slack", async () => {
    h.directory.people.user_a = {
      email: "ada@example.com",
      name: "Ada",
      joinedAt: new Date(NOW.getTime() - DAY_MS),
    };
    const ada = await createOrganization(store, {
      kind: "personal",
      ownerUserID: "user_a",
    });
    await store.query.insert(event).values({
      kind: "link",
      actor: "user_a",
      organization: ada.key,
      at: new Date(NOW.getTime() - DAY_MS),
    });
    const res = await get(`Bearer ${CRON_SECRET}`);
    expect(res.status).toBe(200);
    const summary = {
      day: new Date("2026-09-16T00:00:00Z"),
      active: { day: 1, week: 1, month: 1 },
      accounts: 1,
      firstLogins: 1,
    };
    expect(h.slack.posted).toEqual([digest(summary)]);
  });
});

describe("GET /api/cron/sessions", () => {
  let store: Memory;
  let h: Harness;

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    h = createHarness(store);
    h.portal.now = () => new Date("2026-09-17T12:07:00Z");
  });

  const get = async (authorization?: string): Promise<Response> =>
    await sessionsRoute(
      createAPIContext(h.portal, {
        method: "GET",
        headers: authorization == null ? {} : { authorization },
      }),
    );

  const sessionEnding = (end: string, distinctID = "user_a"): Session => ({
    id: `session-${end}`,
    distinctID,
    email: "ada@example.com",
    start: new Date("2026-09-17T10:00:00Z"),
    end: new Date(end),
    version: "1.0.0",
    os: "macos",
    screens: [],
    actions: {},
  });

  const visitOf = ({ id, ...session }: Session): Visit => ({ ...session, replay: id });

  it("should refuse a call without the cron secret", async () => {
    h.analytics.recorded.push(sessionEnding("2026-09-17T11:20:00Z"));
    const res = await get();
    expect(res.status).toBe(401);
    expect(h.slack.posted).toEqual([]);
  });

  const ada = { email: "ada@example.com", name: "Ada", joinedAt: new Date(0) };

  it("should recap the visits that ended a quarter hour before the idle wait", async () => {
    h.directory.people.user_a = ada;
    const inside = sessionEnding("2026-09-17T11:15:00Z");
    h.analytics.recorded.push(
      sessionEnding("2026-09-17T11:14:59Z", "user_early"),
      inside,
      sessionEnding("2026-09-17T11:30:00Z", "user_late"),
    );
    const res = await get(`Bearer ${CRON_SECRET}`);
    expect(res.status).toBe(200);
    expect(await body(res)).toEqual({ visits: 1 });
    expect(h.slack.posted).toEqual([
      recap(visitOf(inside), { userID: "user_a", ...ada }),
    ]);
  });

  it("should post one recap for the sessions of each window", async () => {
    h.directory.people.user_a = ada;
    h.analytics.recorded.push(
      sessionEnding("2026-09-17T11:20:00Z"),
      {
        ...sessionEnding("2026-09-17T11:21:00Z"),
        start: new Date("2026-09-17T11:20:00Z"),
      },
      {
        ...sessionEnding("2026-09-17T11:21:00Z"),
        id: "reset",
        start: new Date("2026-09-17T11:21:00Z"),
      },
    );
    await get(`Bearer ${CRON_SECRET}`);
    expect(h.slack.posted).toHaveLength(1);
  });

  it("should recap a visit without a Clerk user as anonymous", async () => {
    const anonymous = sessionEnding("2026-09-17T11:20:00Z", "install-1");
    const unknown = sessionEnding("2026-09-17T11:25:00Z", "user_gone");
    h.analytics.recorded.push(anonymous, unknown);
    await get(`Bearer ${CRON_SECRET}`);
    expect(h.slack.posted).toEqual([
      recap(visitOf(anonymous), null),
      recap(visitOf(unknown), null),
    ]);
  });

  it("should not recap the visits of Synnax Labs users", async () => {
    h.directory.people.user_em = { email: "em@synnaxlabs.com", name: "Emiliano" };
    h.directory.people.user_blank = { email: "", name: "Blank" };
    h.analytics.recorded.push(
      sessionEnding("2026-09-17T11:20:00Z", "user_em"),
      {
        ...sessionEnding("2026-09-17T11:25:00Z", "install-1"),
        email: "em@synnaxlabs.com",
      },
      {
        ...sessionEnding("2026-09-17T11:27:00Z", "user_blank"),
        email: "blank@synnaxlabs.com",
      },
    );
    await get(`Bearer ${CRON_SECRET}`);
    expect(h.slack.posted).toEqual([]);
  });
});
