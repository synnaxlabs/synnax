// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

import { type project } from "@/project";
import { createTestClient } from "@/testutil/client";

// The reader client never opens its change stream, so the writer's deletes go
// unseen until reconcile() refetches each document table by key. A doc table
// without a refetch config would keep serving the stale record forever.
const client = createTestClient();
const writer = createTestClient();

let proj: project.Project;

beforeAll(async () => {
  proj = await writer.projects.create({ name: `proj-${id.create()}`, layout: {} });
});

interface Domain {
  name: string;
  create: () => Promise<string>;
  del: (key: string) => Promise<void>;
  getCached: (key: string) => { variant: string } | undefined;
}

const DOMAINS: Domain[] = [
  {
    name: "schematic",
    create: async () =>
      (await client.schematics.create(proj.key, { name: `s-${id.create()}` })).key,
    del: async (key) => await writer.schematics.delete(key),
    getCached: (key) => client.schematics.getCached({ key }),
  },
  {
    name: "lineplot",
    create: async () =>
      (await client.lineplots.create(proj.key, { name: `lp-${id.create()}` })).key,
    del: async (key) => await writer.lineplots.delete(key),
    getCached: (key) => client.lineplots.getCached({ key }),
  },
  {
    name: "log",
    create: async () =>
      (await client.logs.create(proj.key, { name: `log-${id.create()}` })).key,
    del: async (key) => await writer.logs.delete(key),
    getCached: (key) => client.logs.getCached({ key }),
  },
  {
    name: "table",
    create: async () =>
      (await client.tables.create(proj.key, { name: `t-${id.create()}` })).key,
    del: async (key) => await writer.tables.delete(key),
    getCached: (key) => client.tables.getCached({ key }),
  },
  {
    name: "arc",
    create: async () =>
      (await client.arcs.create({ name: `arc-${id.create()}`, mode: "graph" })).key,
    del: async (key) => await writer.arcs.delete(key),
    getCached: (key) => client.arcs.getCached({ key }),
  },
  {
    name: "panel",
    create: async () => (await client.panels.create({ name: `p-${id.create()}` })).key,
    del: async (key) => await writer.panels.delete(key),
    getCached: (key) => client.panels.getCached(key),
  },
];

describe("reconcile", () => {
  DOMAINS.forEach(({ name, create, del, getCached }) => {
    describe(name, () => {
      it("tombstones a cached document deleted out from under the cache", async () => {
        const key = await create();
        expect(getCached(key)?.variant).toEqual("changed");
        await del(key);
        await client.cache.reconcile();
        expect(getCached(key)?.variant).toEqual("deleted");
      });

      it("keeps a still-live cached document intact", async () => {
        const key = await create();
        await client.cache.reconcile();
        expect(getCached(key)?.variant).toEqual("changed");
      });
    });
  });
});
