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

import { type Action } from "@/access/action/types.gen";
import { role } from "@/access/role";
import { channel } from "@/channel";
import { NotFoundError, ValidationError } from "@/index";
import { query } from "@/query";
import {
  type BuiltInRole,
  createTestClient,
  createTestClientWithRole,
  expectDeleted,
  expectLive,
  RoleClients,
} from "@/testutil";

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
      const retrieved = await client.access.roles.retrieve(created.key);
      expect(retrieved.key).toBe(created.key);
      expect(retrieved.name).toBe(created.name);
      expect(retrieved.description).toBe(created.description);
    });

    it("should retrieve a role by its name", async () => {
      const created = await client.access.roles.create({
        name: `named-role-${id.create()}`,
        description: "test",
      });
      const retrieved = await client.access.roles.retrieve(created.name);
      expect(retrieved.key).toBe(created.key);
    });

    it("should filter by internal flag when retrieving roles", async () => {
      const created = await client.access.roles.create({
        name: "test-non-internal",
        description: "test",
      });

      // Retrieve only internal roles (built-in system roles)
      const internalRoles = await client.access.roles.retrieve({ internal: true });
      expect(internalRoles.length).toBeGreaterThan(0);
      expect(internalRoles.every((r) => r.internal === true)).toBe(true);
      expect(internalRoles.find((r) => r.key === created.key)).toBeUndefined();

      const nonInternalRoles = await client.access.roles.retrieve({ internal: false });
      expect(nonInternalRoles.every((r) => r.internal !== true)).toBe(true);
      expect(nonInternalRoles.find((r) => r.key === created.key)).toBeDefined();
    });

    it("should retrieve roles by search term", async () => {
      const prefix = `searchable-role-${id.create()}`;
      const names = [`${prefix}-1`, `${prefix}-2`];
      await client.access.roles.create(names.map((name) => ({ name })));
      await expect
        .poll(async () => {
          const results = await client.access.roles.retrieve({ searchTerm: prefix });
          return results.map((r) => r.name).sort();
        })
        .toEqual(names);
    });
  });

  describe("rename", () => {
    it("should rename the role", async () => {
      const r = await createRole();
      const name = `renamed-${id.create()}`;
      await client.access.roles.rename(r.key, name);
      expect((await createTestClient().access.roles.retrieve(r.key)).name).toBe(name);
    });
  });

  describe("dispatch", () => {
    it("should apply the action to the role", async () => {
      const r = await createRole();
      await client.access.roles.dispatch(
        r.key,
        role.set({ path: ["description"], value: "dispatched" }),
      );
      const stored = await createTestClient().access.roles.retrieve(r.key);
      expect(stored.description).toBe("dispatched");
    });
  });

  describe("delete", () => {
    it("should allow the caller to delete a role", async () => {
      const created = await client.access.roles.create({
        name: "test",
        description: "test",
      });
      await client.access.roles.delete(created.key);
      await expect(client.access.roles.retrieve(created.key)).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should delete the role's policies", async () => {
      const doomed = await client.access.roles.create({
        name: `doomed-${id.create()}`,
      });
      const p = await client.access.policies.create({
        role: doomed.key,
        name: "doomed",
        objects: [],
        actions: ["retrieve"],
      });
      await client.access.roles.delete(doomed.key);
      // A client with an empty cache reads what the Core stores.
      const fresh = createTestClient();
      await expect(fresh.access.policies.retrieve(p.key)).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should refuse to delete a role that a user holds", async () => {
      const held = await client.access.roles.create({ name: `held-${id.create()}` });
      const u = await client.users.create({
        username: id.create(),
        password: "test",
        role: held.key,
      });
      await expect(client.access.roles.delete(held.key)).rejects.toThrow(
        new ValidationError(
          `cannot delete role ${held.name} because user ${u.name} ` +
            "uses it as its role: validation error",
        ),
      );
      expect(expectLive(client.access.roles.watch(held.key).getCached()).key).toEqual(
        held.key,
      );
      const fresh = createTestClient();
      expect((await fresh.access.roles.retrieve(held.key)).key).toEqual(held.key);
      expect((await fresh.users.retrieve(u.key)).role).toEqual(held.key);
    });
  });

  describe("built-in roles", () => {
    it("should connect a client holding a built-in role's permissions", async () => {
      const viewer = await createTestClientWithRole(client, "Viewer");
      await viewer.connect();
      const grantee = await viewer.users.current.retrieve();
      const granted = async (action: Action) =>
        await viewer.access.granted.retrieve(grantee, {
          objects: channel.TYPE_RESOURCE_ID,
          action,
        });
      await expect(granted("create")).resolves.toBe(false);
      await expect(granted("retrieve")).resolves.toBe(true);
    });

    it("should return the same client for a role it already created", async () => {
      const roles = new RoleClients(client);
      const [first, second] = await Promise.all([
        roles.get("Operator"),
        roles.get("Operator"),
      ]);
      expect(first).toBe(second);
    });

    it("should throw when no built-in role carries the name", async () => {
      await expect(
        createTestClientWithRole(client, "Nobody" as BuiltInRole),
      ).rejects.toThrow(NotFoundError);
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
  beforeAll(async () => await client.connect());

  describe("retrieve", () => {
    it("reflects remote changes on an unsubscribed repeat retrieve", async () => {
      const r = await createRole();
      const first = await client.access.roles.retrieve(r.key);
      expect(first.name).toEqual(r.name);
      const renamed = `qry-renamed-${id.create()}`;
      await remote.access.roles.rename(r.key, renamed);
      // Unsubscribed queries hold no frozen answer: repeat retrieves refetch
      // and converge on the remote change once it streams in.
      await expect
        .poll(async () => (await client.access.roles.retrieve(r.key)).name)
        .toBe(renamed);
    });

    it("preserves key order across cached and fetched entries", async () => {
      const a = await createRole();
      const b = await createRole();
      await client.access.roles.retrieve(b.key);
      const res = await client.access.roles.retrieve([a.key, b.key]);
      expect(res.map((r) => r.key)).toEqual([a.key, b.key]);
    });
  });

  describe("getCached", () => {
    it("serves a key query straight from the record written by create", async () => {
      const r = await createRole();
      const cached = expectLive(client.access.roles.watch(r.key).getCached());
      expect(cached.name).toEqual(r.name);
    });
  });

  describe("onChange", () => {
    it("delivers a remote rename to a subscribed single query", async () => {
      const r = await createRole();
      const handler = vi.fn();
      const off = client.access.roles.watch(r.key).onChange(handler);
      try {
        await client.access.roles.retrieve(r.key);
        const renamed = `qry-renamed-${id.create()}`;
        await remote.access.roles.rename(r.key, renamed);
        await expect
          .poll(() => {
            const cached = client.access.roles.watch(r.key).getCached();
            return query.isLive(cached) && cached.name === renamed;
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
      const off = client.access.roles.watch(r.key).onChange((res) => results.push(res));
      try {
        await client.access.roles.retrieve(r.key);
        await remote.access.roles.delete(r.key);
        await expect
          .poll(() =>
            query.Deleted.matches(client.access.roles.watch(r.key).getCached()),
          )
          .toBe(true);
        const last = expectDeleted(results.at(-1));
        expect(last.corpse.name).toEqual(r.name);
        await expect(client.access.roles.retrieve(r.key)).rejects.toThrow("deleted");
      } finally {
        off();
      }
    });
  });
});
