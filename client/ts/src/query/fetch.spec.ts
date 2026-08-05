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

// Every document table declares a fetch that resolves keys against the
// cluster and omits rows the cluster no longer has. Reconciliation and watch
// backfill ride that contract (the reconcile pass itself is covered in
// cache.spec.ts); these tests pin it per domain through the keys-only
// retrieve path, which resolves through the same table fetch. The reader
// client stays cold: every document is created by the writer, so a hit
// proves a real cluster fetch.
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
  retrieveKeys: (keys: string[]) => Promise<Array<{ key: string }>>;
}

const DOMAINS: Domain[] = [
  {
    name: "schematic",
    create: async () =>
      (await writer.schematics.create(proj.key, { name: `s-${id.create()}` })).key,
    del: async (key) => await writer.schematics.delete(key),
    retrieveKeys: async (keys) => await client.schematics.retrieve({ keys }),
  },
  {
    name: "lineplot",
    create: async () =>
      (await writer.lineplots.create(proj.key, { name: `lp-${id.create()}` })).key,
    del: async (key) => await writer.lineplots.delete(key),
    retrieveKeys: async (keys) => await client.lineplots.retrieve({ keys }),
  },
  {
    name: "log",
    create: async () =>
      (await writer.logs.create(proj.key, { name: `log-${id.create()}` })).key,
    del: async (key) => await writer.logs.delete(key),
    retrieveKeys: async (keys) => await client.logs.retrieve({ keys }),
  },
  {
    name: "table",
    create: async () =>
      (await writer.tables.create(proj.key, { name: `t-${id.create()}` })).key,
    del: async (key) => await writer.tables.delete(key),
    retrieveKeys: async (keys) => await client.tables.retrieve({ keys }),
  },
  {
    name: "arc",
    create: async () =>
      (await writer.arcs.create({ name: `arc-${id.create()}`, mode: "graph" })).key,
    del: async (key) => await writer.arcs.delete(key),
    retrieveKeys: async (keys) => await client.arcs.retrieve({ keys }),
  },
  {
    name: "panel",
    create: async () => (await writer.panels.create({ name: `p-${id.create()}` })).key,
    del: async (key) => await writer.panels.delete(key),
    retrieveKeys: async (keys) => await client.panels.retrieve({ keys }),
  },
];

describe("table fetch", () => {
  DOMAINS.forEach(({ name, create, del, retrieveKeys }) => {
    describe(name, () => {
      it("resolves uncached keys against the cluster", async () => {
        const key = await create();
        const res = await retrieveKeys([key]);
        expect(res.map((r) => r.key)).toEqual([key]);
      });

      it("omits keys the cluster no longer has", async () => {
        const key = await create();
        await del(key);
        const res = await retrieveKeys([key]);
        expect(res).toHaveLength(0);
      });
    });
  });
});
