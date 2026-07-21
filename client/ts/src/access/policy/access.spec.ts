// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { policy } from "@/access/policy";
import { type cache } from "@/cache";
import { AuthError, NotFoundError } from "@/errors";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const client = createTestClient();

describe("policy", () => {
  describe("retrieve", () => {
    it("should filter by internal flag when retrieving policies", async () => {
      // Create a non-internal policy
      const created = await client.access.policies.create({
        name: "test-non-internal",
        objects: [],
        actions: ["retrieve"],
      });

      // Retrieve only internal policies (built-in system policies)
      const internalPolicies = await client.access.policies.retrieve({
        internal: true,
      });
      expect(internalPolicies.length).toBeGreaterThan(0);
      expect(internalPolicies.every((p) => p.internal === true)).toBe(true);
      expect(internalPolicies.find((p) => p.key === created.key)).toBeUndefined();

      // Retrieve only non-internal policies
      const nonInternalPolicies = await client.access.policies.retrieve({
        internal: false,
      });
      expect(nonInternalPolicies.every((p) => p.internal !== true)).toBe(true);
      expect(nonInternalPolicies.find((p) => p.key === created.key)).toBeDefined();
    });
  });

  describe("access control", () => {
    it("should deny access when no matching policy exists", async () => {
      // Create a user with no policy for retrieving policies
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const randomPolicy = await client.access.policies.create({
        name: "test",
        objects: [],
        actions: ["retrieve"],
      });
      await expect(
        userClient.access.policies.retrieve({ key: randomPolicy.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve policies with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [policy.ontologyID("")],
        actions: ["retrieve"],
      });
      const randomPolicy = await client.access.policies.create({
        name: "test",
        objects: [],
        actions: ["retrieve"],
      });
      const retrieved = await userClient.access.policies.retrieve({
        key: randomPolicy.key,
      });
      expect(retrieved.key).toBe(randomPolicy.key);
      expect(retrieved.name).toBe(randomPolicy.name);
      expect(retrieved.objects).toEqual(randomPolicy.objects);
      expect(retrieved.actions).toEqual(randomPolicy.actions);
    });

    it("should allow the caller to create policies with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [policy.ontologyID("")],
        actions: ["create"],
      });
      await userClient.access.policies.create({
        name: "test",
        objects: [],
        actions: ["retrieve"],
      });
    });

    it("should deny access when no create policy exists", async () => {
      // Create a user with no create policy for policies
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [policy.ontologyID("")],
        actions: ["retrieve"],
      });
      await expect(
        userClient.access.policies.create({
          name: "test",
          objects: [],
          actions: ["retrieve"],
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete policies with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [policy.ontologyID("")],
        actions: ["delete"],
      });
      const randomPolicy = await client.access.policies.create({
        name: "test",
        objects: [],
        actions: ["retrieve"],
      });
      await userClient.access.policies.delete(randomPolicy.key);
      await expect(
        userClient.access.policies.retrieve({ key: randomPolicy.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should deny access when no delete policy exists", async () => {
      // Create a user with no delete policy for policies
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [policy.ontologyID("")],
        actions: ["retrieve"],
      });
      const randomPolicy = await client.access.policies.create({
        name: "test",
        objects: [],
        actions: ["retrieve"],
      });
      await expect(userClient.access.policies.delete(randomPolicy.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});

// A second client with its own engine: its writes reach the first client only
// through the cluster's change streams, never through a shared cache.
const remote = createTestClient();

const createPolicy = async (c = client) =>
  await c.access.policies.create({
    name: `qry-${id.create()}`,
    objects: [],
    actions: ["retrieve"],
  });

describe("cached reads", () => {
  // Changes made while the stream is still opening are lost (epoch
  // reconciliation repairs them in production); open it up front so remote
  // writes in these specs are always observed.
  beforeAll(async () => await client.connect());

  describe("retrieve", () => {
    it("reflects remote changes on an unsubscribed repeat retrieve", async () => {
      const p = await createPolicy();
      const first = await client.access.policies.retrieve({ key: p.key });
      expect(first.name).toEqual(p.name);
      const renamed = `qry-renamed-${id.create()}`;
      await remote.access.policies.create({ ...p, name: renamed });
      // Unsubscribed queries hold no frozen answer: repeat retrieves refetch
      // and converge on the remote change once it streams in.
      await expect
        .poll(async () => (await client.access.policies.retrieve({ key: p.key })).name)
        .toBe(renamed);
    });

    it("preserves key order across cached and fetched entries", async () => {
      const a = await createPolicy();
      const b = await createPolicy();
      await client.access.policies.retrieve({ key: b.key });
      const res = await client.access.policies.retrieve({ keys: [a.key, b.key] });
      expect(res.map((p) => p.key)).toEqual([a.key, b.key]);
    });
  });

  describe("getCached", () => {
    it("serves a key query straight from the record written by create", async () => {
      const p = await createPolicy();
      const cached = client.access.policies.getCached({ key: p.key });
      expect(cached?.variant).toEqual("changed");
      if (cached?.variant === "changed") expect(cached.data.name).toEqual(p.name);
    });
  });

  describe("onChange", () => {
    it("delivers a remote rename to a subscribed single query", async () => {
      const p = await createPolicy();
      const handler = vi.fn();
      const off = client.access.policies.onChange({ key: p.key }, handler);
      try {
        await client.access.policies.retrieve({ key: p.key });
        const renamed = `qry-renamed-${id.create()}`;
        await remote.access.policies.create({ ...p, name: renamed });
        await expect
          .poll(() => {
            const cached = client.access.policies.getCached({ key: p.key });
            return cached?.variant === "changed" && cached.data.name === renamed;
          })
          .toBe(true);
        expect(handler).toHaveBeenCalled();
      } finally {
        off();
      }
    });

    it("delivers a remote delete as a deleted result carrying the corpse", async () => {
      const p = await createPolicy();
      const results: Array<cache.Cached<policy.Policy> | undefined> = [];
      const off = client.access.policies.onChange({ key: p.key }, (r) =>
        results.push(r),
      );
      try {
        await client.access.policies.retrieve({ key: p.key });
        await remote.access.policies.delete(p.key);
        await expect
          .poll(() => client.access.policies.getCached({ key: p.key })?.variant)
          .toBe("deleted");
        const last = results.at(-1);
        expect(last?.variant).toEqual("deleted");
        if (last?.variant === "deleted") expect(last.corpse.name).toEqual(p.name);
        await expect(client.access.policies.retrieve({ key: p.key })).rejects.toThrow(
          "deleted",
        );
      } finally {
        off();
      }
    });
  });
});
