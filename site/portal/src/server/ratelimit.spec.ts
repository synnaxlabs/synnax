// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { event, type EventKind, type Organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { NOW } from "@/server/license/testutil";
import { check, PER_ACTOR, PER_ORGANIZATION, WINDOW_MS } from "@/server/ratelimit";
import { createOrganization } from "@/testutil";

const MINUTE = 60 * 1000;

describe("ratelimit.check", () => {
  let store: Memory;
  let org: Organization;
  let other: Organization;
  beforeAll(async () => {
    store = await openMemory();
  });
  beforeEach(async () => {
    await store.clear();
    org = await createOrganization(store, { kind: "team", clerkOrgID: "org_a" });
    other = await createOrganization(store, { kind: "team", clerkOrgID: "org_b" });
  });

  const record = async (
    n: number,
    {
      actor = "user_a",
      organization = org.key,
      kind = "activate",
      at = new Date(NOW.getTime() - MINUTE),
    }: { actor?: string; organization?: string; kind?: EventKind; at?: Date } = {},
  ) => {
    if (n === 0) return;
    await store.query
      .insert(event)
      .values(Array.from({ length: n }, () => ({ kind, actor, organization, at })));
  };

  const run = async (actor = "user_a", organization = org.key) =>
    await check(store, { actor, organization, now: NOW });

  const TOO_MANY = { status: 429, message: "Too many activations. Try again later." };

  it("should pass a caller under the limit", async () => {
    await record(PER_ACTOR - 1);
    await expect(run()).resolves.toBeUndefined();
  });

  it("should refuse a caller at the limit", async () => {
    await record(PER_ACTOR);
    await expect(run()).rejects.toMatchObject(TOO_MANY);
  });

  it("should count activations, denials, and downloads alike", async () => {
    await record(10, { kind: "activate" });
    await record(10, { kind: "activate_denied" });
    await record(PER_ACTOR - 20, { kind: "download" });
    await expect(run()).rejects.toMatchObject(TOO_MANY);
  });

  it("should not count other kinds of events", async () => {
    for (const kind of [
      "issue",
      "amend",
      "release",
      "rename",
      "revoke",
      "expiry_notice",
    ] as const)
      await record(PER_ACTOR, { kind });
    await expect(run()).resolves.toBeUndefined();
  });

  it("should not count events older than the window", async () => {
    await record(PER_ACTOR, { at: new Date(NOW.getTime() - WINDOW_MS - MINUTE) });
    await record(PER_ACTOR, { at: new Date(NOW.getTime() - WINDOW_MS) });
    await expect(run()).resolves.toBeUndefined();
  });

  it("should count a caller's events in every organization", async () => {
    await record(PER_ACTOR, { organization: other.key });
    await expect(run()).rejects.toMatchObject(TOO_MANY);
  });

  it("should not count other callers against the caller", async () => {
    await record(PER_ACTOR, { actor: "user_b" });
    await expect(run()).resolves.toBeUndefined();
  });

  it("should refuse an organization at its limit across callers", async () => {
    const callers = Math.ceil(PER_ORGANIZATION / (PER_ACTOR - 1));
    for (let i = 0; i < callers; i++)
      await record(Math.min(PER_ACTOR - 1, PER_ORGANIZATION - i * (PER_ACTOR - 1)), {
        actor: `user_${i}`,
      });
    await expect(run("user_new")).rejects.toMatchObject(TOO_MANY);
  });

  it("should not count another organization's events", async () => {
    const callers = Math.ceil(PER_ORGANIZATION / (PER_ACTOR - 1));
    for (let i = 0; i < callers; i++)
      await record(PER_ACTOR - 1, { actor: `user_${i}`, organization: other.key });
    await expect(run("user_new")).resolves.toBeUndefined();
  });
});
