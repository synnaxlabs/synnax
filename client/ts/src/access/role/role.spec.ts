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

import { type role } from "@/access/role";
import { NotFoundError } from "@/errors";
import { query } from "@/query";
import { createTestClient, expectDeleted, expectLive, isLive } from "@/testutil";

const client = createTestClient();

describe("role", () => {
  describe("create", () => {
    it("should allow the caller to create a role", async () => {
      const role = await client.access.roles.create({
        name: "test",
        description: "test",
      });
      expect(role.key).toBeDefined();
      expect(role.name).toBe("test");
      expect(role.description).toBe("test");
    });
  });

  describe("retrieve", () => {
    it("should allow the caller to retrieve a role", async () => {
      const created = await client.access.roles.create({
        name: "test",
        description: "test",
      });
      const retrieved = await client.access.roles.retrieve({ key: created.key });
      expect(retrieved.key).toBe(created.key);
      expect(retrieved.name).toBe(created.name);
      expect(retrieved.description).toBe(created.description);
    });

    it("should filter by internal flag when retrieving roles", async () => {
      // Create a non-internal role
      const created = await client.access.roles.create({
        name: "test-non-internal",
        description: "test",
      });

      // Retrieve only internal roles (built-in system roles)
      const internalRoles = await client.access.roles.retrieve({ internal: true });
      expect(internalRoles.length).toBeGreaterThan(0);
      expect(internalRoles.every((r) => r.internal === true)).toBe(true);
      expect(internalRoles.find((r) => r.key === created.key)).toBeUndefined();

      // Retrieve only non-internal roles
      const nonInternalRoles = await client.access.roles.retrieve({ internal: false });
      expect(nonInternalRoles.every((r) => r.internal !== true)).toBe(true);
      expect(nonInternalRoles.find((r) => r.key === created.key)).toBeDefined();
    });
  });

  describe("delete", () => {
    it("should allow the caller to delete a role", async () => {
      const created = await client.access.roles.create({
        name: "test",
        description: "test",
      });
      await client.access.roles.delete(created.key);
      await expect(client.access.roles.retrieve({ key: created.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("assign", () => {
    it("should allow the caller to assign a role to a user", async () => {
      const role = await client.access.roles.create({
        name: "test",
        description: "test",
      });
      const username = id.create();
      const u = await client.users.create({
        username,
        password: "test",
        firstName: "test",
        lastName: "test",
      });
      await client.access.roles.assign({
        user: u.key,
        role: role.key,
      });
    });
  });
});

// A second client with its own cache: its writes reach the first client only
// through the cluster's change streams, never through a shared cache.
const remote = createTestClient();

const createRole = async (c = client) =>
  await c.access.roles.create({ name: `qry-${id.create()}`, description: "test" });

describe("cached reads", () => {
  // Changes made while the stream is still opening are lost (epoch
  // reconciliation repairs them in production); open it up front so remote
  // writes in these specs are always observed.
  beforeAll(async () => await client.cache.ensureStreaming());

  describe("retrieve", () => {
    it("reflects remote changes on an unsubscribed repeat retrieve", async () => {
      const r = await createRole();
      const first = await client.access.roles.retrieve({ key: r.key });
      expect(first.name).toEqual(r.name);
      const renamed = `qry-renamed-${id.create()}`;
      await remote.access.roles.create({ ...r, name: renamed });
      // Unsubscribed queries hold no frozen answer: repeat retrieves refetch
      // and converge on the remote change once it streams in.
      await expect
        .poll(async () => (await client.access.roles.retrieve({ key: r.key })).name)
        .toBe(renamed);
    });

    it("preserves key order across cached and fetched entries", async () => {
      const a = await createRole();
      const b = await createRole();
      await client.access.roles.retrieve({ key: b.key });
      const res = await client.access.roles.retrieve({ keys: [a.key, b.key] });
      expect(res.map((r) => r.key)).toEqual([a.key, b.key]);
    });
  });

  describe("getCached", () => {
    it("serves a key query straight from the record written by create", async () => {
      const r = await createRole();
      const cached = expectLive(client.access.roles.getCached({ key: r.key }));
      expect(cached.name).toEqual(r.name);
    });
  });

  describe("onChange", () => {
    it("delivers a remote rename to a subscribed single query", async () => {
      const r = await createRole();
      const handler = vi.fn();
      const off = client.access.roles.onChange({ key: r.key }, handler);
      try {
        await client.access.roles.retrieve({ key: r.key });
        const renamed = `qry-renamed-${id.create()}`;
        await remote.access.roles.create({ ...r, name: renamed });
        await expect
          .poll(() => {
            const cached = client.access.roles.getCached({ key: r.key });
            return isLive(cached) && cached.name === renamed;
          })
          .toBe(true);
        expect(handler).toHaveBeenCalled();
      } finally {
        off();
      }
    });

    it("delivers a remote delete as a deleted result carrying the corpse", async () => {
      const r = await createRole();
      const results: Array<query.Cached<role.Role> | undefined> = [];
      const off = client.access.roles.onChange({ key: r.key }, (res) =>
        results.push(res),
      );
      try {
        await client.access.roles.retrieve({ key: r.key });
        await remote.access.roles.delete(r.key);
        await expect
          .poll(() =>
            query.Deleted.matches(client.access.roles.getCached({ key: r.key })),
          )
          .toBe(true);
        const last = expectDeleted(results.at(-1));
        expect(last.corpse.name).toEqual(r.name);
        await expect(client.access.roles.retrieve({ key: r.key })).rejects.toThrow(
          "deleted",
        );
      } finally {
        off();
      }
    });
  });
});
