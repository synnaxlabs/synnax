// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { allowRequest } from "@/access/enforce";
import { type GrantedQuery } from "@/access/granted";
import { ranger } from "@/ranger";
import { createTestClient, type TestPolicy } from "@/testutil";
import { user } from "@/user";

vi.mock("@/access/enforce", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    allowRequest: vi.fn(actual.allowRequest as typeof allowRequest),
  };
});

const wait = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

const client = createTestClient();

const RETRIEVE_RANGES: GrantedQuery = {
  objects: ranger.TYPE_RESOURCE_ID,
  action: "retrieve",
};

const createSubjectWithPolicy = async (pol: TestPolicy) => {
  const r = await client.access.roles.create({
    name: id.create(),
    description: "test",
  });
  const p = await client.access.policies.create({ ...pol, role: r.key });
  const u = await client.users.create({
    username: id.create(),
    password: "test",
    name: "test",
    roles: [r.key],
  });
  return { subject: user.resourceID(u.key), policy: p, role: r };
};

describe("Granted", () => {
  it("answers a fetched question", async () => {
    const { subject } = await createSubjectWithPolicy({
      name: id.create(),
      objects: [ranger.TYPE_RESOURCE_ID],
      actions: ["retrieve"],
    });
    expect(await client.access.granted.retrieve(subject, RETRIEVE_RANGES)).toBe(true);
    expect(
      await client.access.granted.retrieve(subject, {
        objects: ranger.TYPE_RESOURCE_ID,
        action: "delete",
      }),
    ).toBe(false);
  });

  it("evaluates a repeated query object once until the policies change", async () => {
    const { subject } = await createSubjectWithPolicy({
      name: id.create(),
      objects: [ranger.TYPE_RESOURCE_ID],
      actions: ["retrieve"],
    });
    const q = { ...RETRIEVE_RANGES };
    const stop = client.access.granted.onChange(subject, q, () => {});
    await client.access.policies.retrieve({ where: { subjects: [subject] } });
    await expect
      .poll(() => client.access.granted.getCached(subject, q), { timeout: 5000 })
      .toBe(true);
    const spy = vi.mocked(allowRequest);
    spy.mockClear();
    expect(client.access.granted.getCached(subject, q)).toBe(true);
    expect(client.access.granted.getCached(subject, q)).toBe(true);
    expect(client.access.granted.getCached(subject, q)).toBe(true);
    expect(spy).not.toHaveBeenCalled();
    // A distinct query object holds its own memo entry, so it evaluates once.
    expect(client.access.granted.getCached(subject, { ...RETRIEVE_RANGES })).toBe(true);
    expect(spy).toHaveBeenCalledTimes(1);
    stop();
  });

  it("re-evaluates the verdict when a policy is removed", async () => {
    const { subject, policy: p } = await createSubjectWithPolicy({
      name: id.create(),
      objects: [ranger.TYPE_RESOURCE_ID],
      actions: ["retrieve"],
    });
    const q = { ...RETRIEVE_RANGES };
    const stop = client.access.granted.onChange(subject, q, () => {});
    await client.access.policies.retrieve({ where: { subjects: [subject] } });
    await expect
      .poll(() => client.access.granted.getCached(subject, q), { timeout: 5000 })
      .toBe(true);
    await client.access.policies.delete(p.key);
    await expect
      .poll(() => client.access.granted.getCached(subject, q), { timeout: 5000 })
      .toBe(false);
    stop();
  });

  it("notifies only when the verdict flips", async () => {
    const {
      subject,
      policy: p,
      role: r,
    } = await createSubjectWithPolicy({
      name: id.create(),
      objects: [ranger.TYPE_RESOURCE_ID],
      actions: ["retrieve"],
    });
    const handler = vi.fn();
    const stop = client.access.granted.onChange(subject, RETRIEVE_RANGES, handler);
    await client.access.policies.retrieve({ where: { subjects: [subject] } });
    await expect
      .poll(() => handler.mock.calls.length, { timeout: 5000 })
      .toBeGreaterThan(0);
    expect(handler).toHaveBeenLastCalledWith(true);
    const settled = handler.mock.calls.length;
    // Policy churn that cannot change the answer must not reach the handler.
    await client.access.policies.create({
      role: r.key,
      name: id.create(),
      objects: [user.TYPE_RESOURCE_ID],
      actions: ["create"],
    });
    await wait(500);
    expect(handler.mock.calls.length).toBe(settled);
    await client.access.policies.delete(p.key);
    await expect.poll(() => handler.mock.lastCall?.[0], { timeout: 5000 }).toBe(false);
    stop();
  });
});
