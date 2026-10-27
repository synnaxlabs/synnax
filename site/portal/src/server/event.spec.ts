// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import {
  event,
  type EventKind,
  type License,
  type Organization,
} from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { listEvents, listForLicense } from "@/server/event";
import { HASH_A, NOW } from "@/server/license/testutil";
import { createActivation, createLicense, createOrganization } from "@/testutil";

const MINUTE = 60 * 1000;

describe("event.listForLicense", () => {
  let store: Memory;
  let lic: License;
  let other: License;
  beforeAll(async () => {
    store = await openMemory();
  });
  beforeEach(async () => {
    await store.clear();
    const org = await createOrganization(store, { kind: "team", clerkOrgID: "org_a" });
    lic = await createLicense(store, { organization: org.key });
    other = await createLicense(store, { organization: org.key });
  });

  const record = async (license: string, minutesAgo: number, actor = "user_a") => {
    const [row] = await store.query
      .insert(event)
      .values({
        kind: "download",
        actor,
        license,
        at: new Date(NOW.getTime() - minutesAgo * MINUTE),
      })
      .returning();
    return row;
  };

  it("should list a license's events newest first", async () => {
    const oldest = await record(lic.key, 30);
    const newest = await record(lic.key, 1);
    const middle = await record(lic.key, 10);
    await record(other.key, 0);
    expect(await listForLicense(store, lic.key)).toEqual([newest, middle, oldest]);
  });

  it("should keep the most recent events up to the limit", async () => {
    for (let i = 60; i > 0; i--) await record(lic.key, i, `user_${i}`);
    const page = await listForLicense(store, lic.key);
    expect(page).toHaveLength(50);
    expect(page[0].actor).toBe("user_1");
    expect(page[49].actor).toBe("user_50");
    expect((await listForLicense(store, lic.key, 3)).map((e) => e.actor)).toEqual([
      "user_1",
      "user_2",
      "user_3",
    ]);
  });

  it("should list nothing for a license without events", async () => {
    expect(await listForLicense(store, lic.key)).toEqual([]);
  });
});

describe("event.listEvents", () => {
  let store: Memory;
  let acme: Organization;
  let globex: Organization;
  let lic: License;
  beforeAll(async () => {
    store = await openMemory();
  });
  beforeEach(async () => {
    await store.clear();
    acme = await createOrganization(store, {
      kind: "team",
      name: "Acme",
      clerkOrgID: "org_a",
    });
    globex = await createOrganization(store, {
      kind: "team",
      name: "Globex",
      clerkOrgID: "org_b",
    });
    lic = await createLicense(store, { organization: acme.key, label: "Test stand" });
  });

  const record = async (
    kind: EventKind,
    organization: string,
    actor = "user_a",
    extra: { license?: string; activation?: string } = {},
  ) => {
    const [row] = await store.query
      .insert(event)
      .values({ kind, actor, organization, at: NOW, ...extra })
      .returning();
    return row;
  };

  it("should name the organization, license, and machine of each event", async () => {
    const machine = await createActivation(store, {
      license: lic.key,
      fingerprint: [HASH_A],
      name: "Rig 1",
    });
    const e = await record("activate", acme.key, "user_a", {
      license: lic.key,
      activation: machine.key,
    });
    expect(await listEvents(store, {})).toEqual({
      entries: [
        {
          event: e,
          organization: "Acme",
          license: "Test stand",
          machine: { name: "Rig 1", fingerprint: [HASH_A] },
        },
      ],
      next: null,
    });
  });

  it("should leave out what an event does not concern", async () => {
    await record("register", acme.key);
    expect((await listEvents(store, {})).entries[0]).toMatchObject({
      organization: "Acme",
      license: null,
      machine: null,
    });
  });

  it("should keep only one organization's events", async () => {
    const mine = await record("invite", acme.key);
    await record("invite", globex.key);
    const { entries } = await listEvents(store, { organization: acme.key });
    expect(entries.map((e) => e.event)).toEqual([mine]);
  });

  it("should keep only the events of a category", async () => {
    await record("issue", acme.key);
    const member = await record("change_role", acme.key);
    const { entries } = await listEvents(store, { category: "members" });
    expect(entries.map((e) => e.event)).toEqual([member]);
  });

  it("should keep only the events of the given actors", async () => {
    const staff = await record("issue", acme.key, "user_staff");
    await record("activate", acme.key, "user_member");
    const { entries } = await listEvents(store, { actors: ["user_staff"] });
    expect(entries.map((e) => e.event)).toEqual([staff]);
  });

  it("should keep nothing for no actors", async () => {
    await record("issue", acme.key);
    expect((await listEvents(store, { actors: [] })).entries).toEqual([]);
  });

  it("should page from newest to oldest", async () => {
    const recorded = [];
    for (let i = 0; i < 5; i++) recorded.push(await record("download", acme.key));
    const first = await listEvents(store, { limit: 2 });
    expect(first.entries.map((e) => e.event)).toEqual([recorded[4], recorded[3]]);
    expect(first.next).toBe(recorded[3].key);
    const second = await listEvents(store, { limit: 2, before: first.next ?? 0 });
    expect(second.entries.map((e) => e.event)).toEqual([recorded[2], recorded[1]]);
    const last = await listEvents(store, { limit: 2, before: second.next ?? 0 });
    expect(last).toEqual({ entries: [expect.anything()], next: null });
  });
});
