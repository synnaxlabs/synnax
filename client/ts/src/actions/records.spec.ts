// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, type patch, type record } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

import { access } from "@/access";
import { arc } from "@/arc";
import type Synnax from "@/client";
import { device } from "@/device";
import { group } from "@/group";
import { project } from "@/project";
import { query } from "@/query";
import { rack } from "@/rack";
import { schematic } from "@/schematic";
import { createTestClient, spyOnSend } from "@/testutil";
import { user } from "@/user";
import { view } from "@/view";

const client = createTestClient();
const remote = createTestClient();

interface Named {
  name: string;
}

interface Domain<K extends record.Key, R extends Named, A> {
  dispatch(key: K, actions: A | A[]): Promise<void>;
  rename(key: K, name: string): Promise<void>;
  watch(key: K): query.Live<R>;
  retrieve(params: Record<string, never>): Promise<R[]>;
  retrieve(key: K): Promise<R>;
}

/** A record that a case created, with each operation bound to its key. */
interface Created {
  name: string;
  rename: (c: Synnax, name: string) => Promise<void>;
  dispatch: (c: Synnax, set: patch.Set) => Promise<void>;
  cachedName: (c: Synnax) => string | undefined;
  storedName: (c: Synnax) => Promise<string>;
  /** The name that a fetch of every record of the type returns. */
  listedName: (c: Synnax) => Promise<string | undefined>;
}

interface Case {
  type: string;
  create: (c: Synnax) => Promise<Created>;
  /** The error of the Core when it refuses an empty name. */
  refusal: string;
}

const defineCase = <K extends record.Key, R extends Named & record.Keyed<K>, A>(
  type: string,
  domain: (c: Synnax) => Domain<K, R, A>,
  set: (payload: patch.Set) => A,
  create: (c: Synnax) => Promise<R>,
  refusal: string,
): Case => ({
  type,
  refusal,
  create: async (c) => {
    const { key, name } = await create(c);
    return {
      name,
      rename: async (c, next) => await domain(c).rename(key, next),
      dispatch: async (c, payload) => await domain(c).dispatch(key, set(payload)),
      cachedName: (c) => {
        const cached = domain(c).watch(key).getCached();
        return query.isLive(cached) ? cached.name : undefined;
      },
      storedName: async (c) => (await domain(c).retrieve(key)).name,
      listedName: async (c) =>
        (await domain(c).retrieve({})).find((r) => r.key === key)?.name,
    };
  },
});

const uniqueName = () => `dispatch_${id.create()}`;

const CASES: Case[] = [
  defineCase(
    "device",
    (c) => c.devices,
    device.set,
    async (c) => {
      const { key: r } = await c.racks.create({ name: uniqueName() });
      return await c.devices.create({
        key: id.create(),
        rack: r,
        location: "dev1",
        name: uniqueName(),
        make: "ni",
        model: "pxi",
        properties: {},
      });
    },
    "name: required",
  ),
  defineCase(
    "rack",
    (c) => c.racks,
    rack.set,
    async (c) => await c.racks.create({ name: uniqueName() }),
    "name: required",
  ),
  defineCase(
    "user",
    (c) => c.users,
    user.set,
    async (c) => {
      const role = await c.access.roles.create({ name: uniqueName() });
      return await c.users.create({
        username: id.create(),
        password: "test",
        name: uniqueName(),
        role: role.key,
      });
    },
    "name: required",
  ),
  defineCase(
    "group",
    (c) => c.groups,
    group.set,
    async (c) => await c.groups.create({ name: uniqueName(), scope: "channel" }),
    "name: required",
  ),
  defineCase(
    "project",
    (c) => c.projects,
    project.set,
    async (c) => await c.projects.create({ name: uniqueName() }),
    "name: required",
  ),
  defineCase(
    "view",
    (c) => c.views,
    view.set,
    async (c) => await c.views.create({ name: uniqueName(), type: "range" }),
    "name: required",
  ),
  defineCase(
    "policy",
    (c) => c.access.policies,
    access.policy.set,
    async (c) => {
      const role = await c.access.roles.create({ name: uniqueName() });
      return await c.access.policies.create({
        name: uniqueName(),
        role: role.key,
        objects: [],
        actions: ["retrieve"],
      });
    },
    "name: required",
  ),
  defineCase(
    "role",
    (c) => c.access.roles,
    access.role.set,
    async (c) => await c.access.roles.create({ name: uniqueName() }),
    "name: required",
  ),
  defineCase(
    "symbol",
    (c) => c.schematics.symbols,
    schematic.symbol.set,
    async (c) => {
      const { key } = await c.groups.create({
        name: uniqueName(),
        scope: "schematic_symbol",
      });
      return await c.schematics.symbols.create({
        name: uniqueName(),
        data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
        group: key,
      });
    },
    "name: required",
  ),
  defineCase(
    "Arc directory",
    (c) => c.arcs.directories,
    arc.directory.set,
    async (c) => {
      const root = await c.arcs.directories.retrieveRoot();
      return await c.arcs.directories.create({ parent: root.key, name: uniqueName() });
    },
    'name "" must contain only letters, digits, and underscores',
  ),
];

describe("record dispatch", () => {
  beforeAll(async () => {
    await client.connect();
    await remote.connect();
  });

  describe.each(CASES)("$type", ({ create, refusal }) => {
    it("should hold a dispatched edit in the cache once the dispatch resolves", async () => {
      const rec = await create(client);
      const name = uniqueName();
      await rec.dispatch(client, { path: ["name"], value: name, guards: [] });
      expect(rec.cachedName(client)).toBe(name);
      expect(await rec.storedName(createTestClient())).toBe(name);
    });

    it("should drop an edit that the Core refuses from the cache", async () => {
      const rec = await create(client);
      await expect(
        rec.dispatch(client, { path: ["name"], value: "", guards: [] }),
      ).rejects.toThrow(refusal);
      expect(rec.cachedName(client)).toBe(rec.name);
    });

    it("should reject a path outside the editable fields before any send", async () => {
      const rec = await create(client);
      const send = spyOnSend(client);
      await expect(
        rec.dispatch(client, { path: ["key"], value: "other", guards: [] }),
      ).rejects.toThrow('no editable field at "key"');
      expect(send).not.toHaveBeenCalled();
      send.mockRestore();
    });

    it("should rename a record that the client does not cache", async () => {
      const rec = await create(remote);
      const fresh = createTestClient();
      const name = uniqueName();
      await rec.rename(fresh, name);
      expect(await rec.storedName(fresh)).toBe(name);
    });

    it("should apply another client's dispatch to a cached record", async () => {
      const rec = await create(client);
      await rec.storedName(remote);
      const name = uniqueName();
      await rec.dispatch(client, { path: ["name"], value: name, guards: [] });
      await expect.poll(() => rec.cachedName(remote)).toBe(name);
    });

    it("should settle its own echo so a later remote edit shows", async () => {
      const rec = await create(client);
      await rec.storedName(remote);
      await rec.dispatch(client, { path: ["name"], value: uniqueName(), guards: [] });
      const name = uniqueName();
      await rec.dispatch(remote, { path: ["name"], value: name, guards: [] });
      await expect.poll(() => rec.cachedName(client)).toBe(name);
    });

    it("should read a later remote edit with no change stream", async () => {
      const unstreamed = createTestClient({ cache: false });
      const rec = await create(unstreamed);
      await rec.rename(unstreamed, uniqueName());
      const name = uniqueName();
      await rec.rename(remote, name);
      expect(await rec.listedName(unstreamed)).toBe(name);
    });

    it("should ingest a record that another client creates", async () => {
      const rec = await create(remote);
      await expect.poll(() => rec.cachedName(client)).toBe(rec.name);
    });
  });
});
