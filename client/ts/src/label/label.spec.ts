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

import { label } from "@/label";
import { createTestClient } from "@/testutil";

const client = createTestClient();

describe("Label", () => {
  describe("create", () => {
    it("should create a label", async () => {
      const v = await client.labels.create({
        name: "Label",
        color: "#E774D0",
      });
      expect(v.key).not.toHaveLength(0);
    });
  });

  describe("retrieve", () => {
    it("should retrieve a label by its key", async () => {
      const v = await client.labels.create({
        name: "Label",
        color: "#E774D0",
      });
      const retrieved = await client.labels.retrieve({ key: v.key });
      expect(retrieved).toEqual(v);
    });
    it("should retrieve labels by search term", async () => {
      const prefix = `searchable-label-${id.create()}`;
      const names = [`${prefix}-1`, `${prefix}-2`];
      await client.labels.create(names.map((name) => ({ name, color: "#E774D0" })));
      await expect
        .poll(async () => {
          const results = await client.labels.retrieve({ searchTerm: prefix });
          return results.map((l) => l.name).sort();
        })
        .toEqual(names);
    });
  });

  describe("delete", () => {
    it("should delete a label by its key", async () => {
      const v = await client.labels.create({
        name: "Label",
        color: "#E774D0",
      });
      await client.labels.delete(v.key);
      await expect(
        async () => await client.labels.retrieve({ key: v.key }),
      ).rejects.toThrow();
    });
  });

  describe("label", () => {
    it("should set a label on an item", async () => {
      const l1 = await client.labels.create({ name: "Label One", color: "#E774D0" });
      const l2 = await client.labels.create({ name: "Label Two", color: "#E774D0" });
      await client.labels.label(label.ontologyID(l1.key), [l2.key]);
      const labels = await client.labels.retrieve({ for: label.ontologyID(l1.key) });
      expect(labels).toHaveLength(1);
      expect(labels[0].key).toEqual(l2.key);
    });
    it("should replace the labels on an item", async () => {
      const l1 = await client.labels.create({ name: "Label One", color: "#E774D0" });
      const l2 = await client.labels.create({ name: "Label Two", color: "#E774D0" });
      await client.labels.label(label.ontologyID(l1.key), [l2.key]);
      await client.labels.label(label.ontologyID(l1.key), [l1.key], { replace: true });
      const labels = await client.labels.retrieve({ for: label.ontologyID(l1.key) });
      expect(labels).toHaveLength(1);
      expect(labels[0].key).toEqual(l1.key);
    });
  });
});

// A second client with its own engine: its writes reach the first client only
// through the cluster's change streams, never through a shared cache.
const remote = createTestClient();

describe("cached reads", () => {
  // Changes made while the stream is still opening are lost (epoch
  // reconciliation repairs them in production); open it up front so remote
  // writes in these specs are always observed.
  beforeAll(async () => await client.cache.ensureStreaming());

  it("serves a key query straight from the record written by create", async () => {
    const lbl = await client.labels.create({
      name: `qry-${id.create()}`,
      color: "#E774D0",
    });
    const cached = client.labels.getCached({ key: lbl.key });
    expect(cached?.variant).toEqual("changed");
    if (cached?.variant === "changed") expect(cached.data.name).toEqual(lbl.name);
  });

  it("delivers a remote delete as a deleted result carrying the corpse", async () => {
    const lbl = await client.labels.create({
      name: `qry-${id.create()}`,
      color: "#E774D0",
    });
    const off = client.labels.onChange({ key: lbl.key }, vi.fn());
    try {
      await client.labels.retrieve({ key: lbl.key });
      await remote.labels.delete(lbl.key);
      await expect
        .poll(() => client.labels.getCached({ key: lbl.key })?.variant)
        .toBe("deleted");
      const cached = client.labels.getCached({ key: lbl.key });
      if (cached?.variant === "deleted") expect(cached.corpse.name).toEqual(lbl.name);
      await expect(client.labels.retrieve({ key: lbl.key })).rejects.toThrow("deleted");
    } finally {
      off();
    }
  });

  it("tracks remote label attachment and removal on a subscribed for: query", async () => {
    const item = await client.labels.create({
      name: `qry-item-${id.create()}`,
      color: "#E774D0",
    });
    const lbl = await remote.labels.create({
      name: `qry-attached-${id.create()}`,
      color: "#E774D0",
    });
    const query = { for: label.ontologyID(item.key) };
    const off = client.labels.onChange(query, vi.fn());
    try {
      await client.labels.retrieve(query);
      await remote.labels.label(label.ontologyID(item.key), [lbl.key]);
      await expect
        .poll(() => {
          const cached = client.labels.getCached(query);
          return (
            cached?.variant === "changed" && cached.data.some((l) => l.key === lbl.key)
          );
        })
        .toBe(true);
      await remote.labels.remove(label.ontologyID(item.key), [lbl.key]);
      await expect
        .poll(() => {
          const cached = client.labels.getCached(query);
          return (
            cached?.variant === "changed" && cached.data.every((l) => l.key !== lbl.key)
          );
        })
        .toBe(true);
    } finally {
      off();
    }
  });
});

describe("store", () => {
  it("caches sets and corpses deletes from live signals", async () => {
    await client.cache.ensureStreaming();
    const created = await client.labels.create({
      name: `label-${id.create()}`,
      color: "#12E774",
    });
    await expect
      .poll(
        () => {
          const cached = client.labels.getCached({ key: created.key });
          return cached?.variant === "changed" ? cached.data.name : undefined;
        },
        { timeout: 5000 },
      )
      .toEqual(created.name);
    await client.labels.delete(created.key);
    await expect
      .poll(() => client.labels.getCached({ key: created.key })?.variant, {
        timeout: 5000,
      })
      .toBe("deleted");
    const cached = client.labels.getCached({ key: created.key });
    if (cached?.variant === "deleted") expect(cached.corpse.name).toEqual(created.name);
  }, 20000);
});
