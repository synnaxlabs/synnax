// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, id, sleep, TimeSpan, uuid } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { MultipleFoundError, NotFoundError, ValidationError } from "@/errors";
import { label } from "@/label";
import { query } from "@/query";
import {
  createProxiedTestClient,
  createTestClient,
  expectDeleted,
  expectLive,
  FAST_RETRY,
  holdResponses,
  spyOnSend,
  waitForStreamLive,
} from "@/testutil";

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
      const retrieved = await client.labels.retrieve(v.key);
      expect(retrieved).toEqual(v);
    });
    it("should retrieve a label by its name", async () => {
      const v = await client.labels.create({
        name: `named-label-${id.create()}`,
        color: "#E774D0",
      });
      expect(await client.labels.retrieve(v.name)).toEqual(v);
    });
    it("should retrieve labels by a list of names", async () => {
      const prefix = `named-labels-${id.create()}`;
      const created = await client.labels.create([
        { name: `${prefix}-1`, color: "#E774D0" },
        { name: `${prefix}-2`, color: "#E774D0" },
      ]);
      const retrieved = await client.labels.retrieve(created.map((l) => l.name));
      expect(retrieved.map((l) => l.name).sort()).toEqual([
        `${prefix}-1`,
        `${prefix}-2`,
      ]);
    });
    it("should throw a NotFoundError for a name that no label has", async () => {
      const name = `missing-label-${id.create()}`;
      await expect(client.labels.retrieve(name)).rejects.toThrow(
        new NotFoundError(
          `label not found matching {"where":{"name":{"eq":"${name}"}}}`,
        ),
      );
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
    it("should retrieve labels by where", async () => {
      const prefix = `where-label-${id.create()}`;
      const [item, attached] = await client.labels.create([
        { name: `${prefix}-item`, color: "#E774D0" },
        { name: `${prefix}-attached`, color: "#E774D0" },
      ]);
      await client.labels.label(label.ontologyID(item.key), [attached.key]);
      const labels = await client.labels.retrieve({
        where: {
          name: { regex: `^${prefix}` },
          not: { for: label.ontologyID(item.key) },
        },
      });
      expect(labels.map((l) => l.key)).toEqual([item.key]);
    });
    it("should retrieve labels by a flat filter", async () => {
      const prefix = `flat-label-${id.create()}`;
      const [item, attached] = await client.labels.create([
        { name: `${prefix}-item`, color: "#E774D0" },
        { name: `${prefix}-attached`, color: "#E774D0" },
      ]);
      await client.labels.label(label.ontologyID(item.key), [attached.key]);
      const params: label.RetrieveMultipleParams = {
        name: { regex: `^${prefix}` },
        not: { for: label.ontologyID(item.key) },
      };
      const labels = await client.labels.retrieve(params);
      expect(labels.map((l) => l.key)).toEqual([item.key]);
    });
    it("should drop the keys that do not exist from a flat key list", async () => {
      const lbl = await client.labels.create({ name: "Label", color: "#E774D0" });
      const labels = await client.labels.retrieve({ key: [lbl.key, uuid.create()] });
      expect(labels.map((l) => l.key)).toEqual([lbl.key]);
    });
    it("should return one label for a plain name beside other fields", async () => {
      const prefix = `one-label-${id.create()}`;
      const [item, attached] = await client.labels.create([
        { name: `${prefix}-item`, color: "#E774D0" },
        { name: `${prefix}-attached`, color: "#E774D0" },
      ]);
      await client.labels.label(label.ontologyID(item.key), [attached.key]);
      const retrieved: label.Label = await client.labels.retrieve({
        name: attached.name,
        for: label.ontologyID(item.key),
      });
      expect(retrieved.key).toEqual(attached.key);
    });
    it("should return one label for a filter that sets one", async () => {
      const prefix = `one-option-${id.create()}`;
      const lbl = await client.labels.create({ name: `${prefix}-a`, color: "#E774D0" });
      const retrieved: label.Label = await client.labels.retrieve({
        name: { startsWith: prefix },
        one: true,
      });
      expect(retrieved.key).toEqual(lbl.key);
    });
    it("should throw a MultipleFoundError when a filter that sets one matches two labels", async () => {
      const prefix = `one-option-${id.create()}`;
      await client.labels.create([
        { name: `${prefix}-a`, color: "#E774D0" },
        { name: `${prefix}-b`, color: "#E774D0" },
      ]);
      await expect(
        client.labels.retrieve({ name: { startsWith: prefix }, one: true }),
      ).rejects.toThrow(MultipleFoundError);
    });
    it("should name the operator at fault in a where tree", async () => {
      await expect(
        // @ts-expect-error - a name operator takes a string.
        client.labels.retrieve({ where: { name: { eq: 5 } } }),
      ).rejects.toThrow(/at where\.name\n.*\n\s*✗ "eq": 5 {2}× expected string/);
    });
    it("should reject a filter that sets one beside a limit", async () => {
      await expect(
        // @ts-expect-error - a page could hide a second match.
        client.labels.retrieve({ name: { startsWith: "x" }, one: true, limit: 1 }),
      ).rejects.toThrow(
        new ValidationError(
          "a query for one record takes no limit or offset, since a page could " +
            "hide a second match",
        ),
      );
    });
    it("should not take a one that widens to a boolean", async () => {
      const params = {
        name: { startsWith: `missing-${id.create()}` },
        one: true as boolean,
      };
      // @ts-expect-error - a widened one could return one label or a list.
      await expect(client.labels.retrieve(params)).rejects.toThrow(NotFoundError);
    });
    it("should throw a NotFoundError when a plain key and name match nothing", async () => {
      const lbl = await client.labels.create({ name: "Label", color: "#E774D0" });
      const name = `other-${id.create()}`;
      await expect(client.labels.retrieve({ key: lbl.key, name })).rejects.toThrow(
        new NotFoundError(
          `label not found matching {"where":{"key":{"eq":"${lbl.key}"},"name":{"eq":"${name}"}}}`,
        ),
      );
    });
    it("should throw a NotFoundError for a missing key in a key list", async () => {
      const lbl = await client.labels.create({ name: "Label", color: "#E774D0" });
      const missing = uuid.create();
      await expect(client.labels.retrieve([lbl.key, missing])).rejects.toThrow(
        new NotFoundError(`label with key ${missing} not found`),
      );
    });
    it("should drop the keys that do not exist from a filter on keys", async () => {
      const lbl = await client.labels.create({ name: "Label", color: "#E774D0" });
      const labels = await client.labels.retrieve({
        where: { key: { in: [lbl.key, uuid.create()] } },
      });
      expect(labels.map((l) => l.key)).toEqual([lbl.key]);
    });
  });

  describe("delete", () => {
    it("should delete a label by its key", async () => {
      const v = await client.labels.create({
        name: "Label",
        color: "#E774D0",
      });
      await client.labels.delete(v.key);
      await expect(async () => await client.labels.retrieve(v.key)).rejects.toThrow();
    });
  });

  describe("label", () => {
    it("should set a label on an item", async () => {
      const l1 = await client.labels.create({ name: "Label One", color: "#E774D0" });
      const l2 = await client.labels.create({ name: "Label Two", color: "#E774D0" });
      await client.labels.label(label.ontologyID(l1.key), [l2.key]);
      const labels = await client.labels.retrieve({
        where: { for: label.ontologyID(l1.key) },
      });
      expect(labels).toHaveLength(1);
      expect(labels[0].key).toEqual(l2.key);
    });
    it("should replace the labels on an item", async () => {
      const l1 = await client.labels.create({ name: "Label One", color: "#E774D0" });
      const l2 = await client.labels.create({ name: "Label Two", color: "#E774D0" });
      await client.labels.label(label.ontologyID(l1.key), [l2.key]);
      await client.labels.label(label.ontologyID(l1.key), [l1.key], { replace: true });
      const labels = await client.labels.retrieve({
        where: { for: label.ontologyID(l1.key) },
      });
      expect(labels).toHaveLength(1);
      expect(labels[0].key).toEqual(l1.key);
    });
  });
});

// A second client with its own cache: its writes reach the first client only
// through the cluster's change streams, never through a shared cache.
const remote = createTestClient();

describe("cached reads", () => {
  // Changes made while the stream is still opening are lost (epoch
  // reconciliation repairs them in production); open it up front so remote
  // writes in these specs are always observed.
  beforeAll(async () => await client.connect());

  it("serves a key query straight from the record written by create", async () => {
    const lbl = await client.labels.create({
      name: `qry-${id.create()}`,
      color: "#E774D0",
    });
    const cached = expectLive(client.labels.getCached(lbl.key));
    expect(cached.name).toEqual(lbl.name);
  });

  it("delivers a remote delete as a deleted result carrying the corpse", async () => {
    const lbl = await client.labels.create({
      name: `qry-${id.create()}`,
      color: "#E774D0",
    });
    const off = client.labels.onChange(lbl.key, vi.fn());
    try {
      await client.labels.retrieve(lbl.key);
      await remote.labels.delete(lbl.key);
      await expect
        .poll(() => query.Deleted.matches(client.labels.getCached(lbl.key)))
        .toBe(true);
      const cached = expectDeleted(client.labels.getCached(lbl.key));
      expect(cached.corpse.name).toEqual(lbl.name);
      await expect(client.labels.retrieve(lbl.key)).rejects.toThrow("deleted");
    } finally {
      off();
    }
  });

  it("tracks remote label attachment and removal on a subscribed for query", async () => {
    const item = await client.labels.create({
      name: `qry-item-${id.create()}`,
      color: "#E774D0",
    });
    const lbl = await remote.labels.create({
      name: `qry-attached-${id.create()}`,
      color: "#E774D0",
    });
    const params = { where: { for: label.ontologyID(item.key) } };
    const off = client.labels.onChange(params, vi.fn());
    try {
      await client.labels.retrieve(params);
      await remote.labels.label(label.ontologyID(item.key), [lbl.key]);
      await expect
        .poll(() => {
          const cached = client.labels.getCached(params);
          return query.isLive(cached) && cached.some((l) => l.key === lbl.key);
        })
        .toBe(true);
      await remote.labels.remove(label.ontologyID(item.key), [lbl.key]);
      await expect
        .poll(() => {
          const cached = client.labels.getCached(params);
          return query.isLive(cached) && cached.every((l) => l.key !== lbl.key);
        })
        .toBe(true);
    } finally {
      off();
    }
  });

  it("tracks remote attachment and removal on a subscribed not for query", async () => {
    const item = await client.labels.create({
      name: `qry-item-${id.create()}`,
      color: "#E774D0",
    });
    const lbl = await remote.labels.create({
      name: `qry-attached-${id.create()}`,
      color: "#E774D0",
    });
    const params = { where: { not: { for: label.ontologyID(item.key) } } };
    const has = () => {
      const cached = client.labels.getCached(params);
      return query.isLive(cached) && cached.some((l) => l.key === lbl.key);
    };
    const off = client.labels.onChange(params, vi.fn());
    try {
      await client.labels.retrieve(params);
      await expect.poll(has).toBe(true);
      await remote.labels.label(label.ontologyID(item.key), [lbl.key]);
      await expect.poll(has).toBe(false);
      await remote.labels.remove(label.ontologyID(item.key), [lbl.key]);
      await expect.poll(has).toBe(true);
    } finally {
      off();
    }
  });

  it("should update a for query from its own writes without a retrieve", async () => {
    const fresh = createTestClient();
    await fresh.connect();
    const item = await fresh.labels.create({
      name: `qry-item-${id.create()}`,
      color: "#E774D0",
    });
    const lbl = await fresh.labels.create({
      name: `qry-attached-${id.create()}`,
      color: "#E774D0",
    });
    const itemID = label.ontologyID(item.key);
    const params = { where: { for: itemID } };
    const off = fresh.labels.onChange(params, vi.fn());
    const send = spyOnSend(fresh);
    try {
      await fresh.labels.retrieve(params);
      send.mockClear();
      await fresh.labels.label(itemID, [lbl.key]);
      expect(expectLive(fresh.labels.getCached(params)).map((l) => l.key)).toEqual([
        lbl.key,
      ]);
      await fresh.labels.remove(itemID, [lbl.key]);
      expect(expectLive(fresh.labels.getCached(params))).toEqual([]);
      await sleep.sleep(TimeSpan.milliseconds(250));
      expect(send.mock.calls.map(([target]) => target)).toEqual([
        "/label/set",
        "/label/remove",
      ]);
    } finally {
      off();
    }
  });

  describe("a label attached before the client connected", () => {
    const attach = async (): Promise<{ item: label.Label; lbl: label.Label }> => {
      const item = await remote.labels.create({
        name: `qry-item-${id.create()}`,
        color: "#E774D0",
      });
      const lbl = await remote.labels.create({
        name: `qry-attached-${id.create()}`,
        color: "#E774D0",
      });
      await remote.labels.label(label.ontologyID(item.key), [lbl.key]);
      return { item, lbl };
    };

    it("should stay in a for query when it changes", async () => {
      const { item, lbl } = await attach();
      const fresh = createTestClient();
      await fresh.connect();
      const params = { where: { for: label.ontologyID(item.key) } };
      const off = fresh.labels.onChange(params, vi.fn());
      try {
        await fresh.labels.retrieve(params);
        const name = `qry-renamed-${id.create()}`;
        await remote.labels.create({ ...lbl, name });
        await expect
          .poll(() => {
            const cached = fresh.labels.getCached(params);
            if (!query.isLive(cached)) return undefined;
            return cached.find((l) => l.key === lbl.key)?.name;
          })
          .toEqual(name);
      } finally {
        off();
      }
    });

    it("should stay out of a not for query when it changes", async () => {
      const { item, lbl } = await attach();
      const fresh = createTestClient();
      await fresh.connect();
      const params = { where: { not: { for: label.ontologyID(item.key) } } };
      const off = fresh.labels.onChange(params, vi.fn());
      try {
        await fresh.labels.retrieve(params);
        const name = `qry-renamed-${id.create()}`;
        await remote.labels.create({ ...lbl, name });
        await expect
          .poll(() => {
            const cached = fresh.labels.getCached(lbl.key);
            return query.isLive(cached) ? cached.name : undefined;
          })
          .toEqual(name);
        const cached = expectLive(fresh.labels.getCached(params));
        expect(cached.map((l) => l.key)).not.toContain(lbl.key);
      } finally {
        off();
      }
    });
  });

  // The remote change lands while the proxy is severed, so only the refetch on
  // reconnect tells the client about it. A later rename makes the client check the
  // label against its cached relationships.
  describe("a label attached or removed during a change stream gap", () => {
    const setup = async () => {
      const { proxy, client: local } = await createProxiedTestClient({
        retry: FAST_RETRY,
        onRetry: () => {},
      });
      await local.connect();
      const [item, kept, changed] = await local.labels.create(
        [0, 1, 2].map(() => ({ name: `qry-gap-${id.create()}`, color: "#E774D0" })),
      );
      const itemID = label.ontologyID(item.key);
      const params = { where: { for: itemID } };
      const keys = () => {
        const cached = local.labels.getCached(params);
        return query.isLive(cached) ? cached.map((l) => l.key).sort() : undefined;
      };
      const duringGap = async (change: () => Promise<void>) => {
        await proxy.sever();
        await change();
        await proxy.restore();
        await waitForStreamLive(local.connection);
      };
      const rename = async () => {
        const name = `qry-renamed-${id.create()}`;
        await remote.labels.create({ ...changed, name });
        await expect
          .poll(() => {
            const cached = local.labels.getCached(changed.key);
            return query.isLive(cached) ? cached.name : undefined;
          })
          .toEqual(name);
      };
      return { local, itemID, params, kept, changed, keys, duringGap, rename };
    };

    it("should keep a label removed during the gap out of a for query", async () => {
      const { local, itemID, params, kept, changed, keys, duringGap, rename } =
        await setup();
      await local.labels.label(itemID, [kept.key, changed.key]);
      const off = local.labels.onChange(params, vi.fn());
      try {
        await local.labels.retrieve(params);
        await duringGap(async () => await remote.labels.remove(itemID, [changed.key]));
        await expect.poll(keys).toEqual([kept.key]);
        await rename();
        expect(keys()).toEqual([kept.key]);
      } finally {
        off();
      }
    }, 30000);

    it("should keep a label attached again during the gap in a for query", async () => {
      const { local, itemID, params, kept, changed, keys, duringGap, rename } =
        await setup();
      await local.labels.label(itemID, [kept.key, changed.key]);
      await local.labels.remove(itemID, [changed.key]);
      const off = local.labels.onChange(params, vi.fn());
      try {
        await local.labels.retrieve(params);
        await duringGap(async () => await remote.labels.label(itemID, [changed.key]));
        const both = [kept.key, changed.key].sort();
        await expect.poll(keys).toEqual(both);
        await rename();
        expect(keys()).toEqual(both);
      } finally {
        off();
      }
    }, 30000);

    it("should keep a label attached during the gap when an older fetch lands late", async () => {
      const { local, itemID, params, changed, keys, duringGap } = await setup();
      const { responded, release } = holdResponses(local, "/label/retrieve");
      const stale = local.labels.retrieve(params);
      const off = local.labels.onChange(params, vi.fn());
      try {
        await responded;
        spyOnSend(local).mockRestore();
        await duringGap(async () => await remote.labels.label(itemID, [changed.key]));
        await expect.poll(keys).toEqual([changed.key]);
        release();
        await stale;
        expect(keys()).toEqual([changed.key]);
      } finally {
        off();
      }
    }, 30000);
  });
});

describe("store", () => {
  it("caches sets and corpses deletes from live signals", async () => {
    await client.connect();
    const created = await client.labels.create({
      name: `label-${id.create()}`,
      color: "#12E774",
    });
    await expect
      .poll(() => {
        const cached = client.labels.getCached(created.key);
        return query.isLive(cached) ? cached.name : undefined;
      })
      .toEqual(created.name);
    await client.labels.delete(created.key);
    await expect
      .poll(() => query.Deleted.matches(client.labels.getCached(created.key)))
      .toBe(true);
    const cached = expectDeleted(client.labels.getCached(created.key));
    expect(cached.corpse.name).toEqual(created.name);
  });
});

describe("dispatch", () => {
  beforeAll(async () => {
    await client.connect();
    await remote.connect();
  });
  const createLabel = async (c = client) =>
    await c.labels.create({ name: `dispatch-${id.create()}`, color: "#00ff00" });
  const rename = (value: string) => label.set({ path: ["name"], value });

  it("should store the actions and keep them in the cached label", async () => {
    const lbl = await createLabel();
    expect(
      await client.labels.dispatch(lbl.key, [
        rename("Anomaly"),
        label.set({ path: ["color"], value: "#ff0000" }),
      ]),
    ).toBe(true);
    const stored = await createTestClient().labels.retrieve(lbl.key);
    expect(stored.name).toBe("Anomaly");
    expect(stored.color).toEqual(color.construct("#ff0000"));
    const cached = expectLive(client.labels.getCached(lbl.key));
    expect(cached.name).toBe("Anomaly");
    expect(cached.color).toEqual(color.construct("#ff0000"));
  });

  it("should return false without a send when the label is not cached", async () => {
    const lbl = await createLabel(remote);
    const fresh = createTestClient();
    const send = spyOnSend(fresh);
    expect(await fresh.labels.dispatch(lbl.key, rename("Anomaly"))).toBe(false);
    expect(send).not.toHaveBeenCalled();
  });

  it("should drop actions the Core refuses from the cached label", async () => {
    const lbl = await createLabel();
    await expect(client.labels.dispatch(lbl.key, rename(""))).rejects.toThrow(
      "name: required",
    );
    expect(expectLive(client.labels.getCached(lbl.key)).name).toBe(lbl.name);
  });

  it("should send a transaction as one dispatch", async () => {
    const lbl = await createLabel();
    const send = spyOnSend(client);
    const tx = client.labels.beginTransaction(lbl.key);
    tx.add(rename("Anomaly"));
    tx.add(label.set({ path: ["color"], value: "#ff0000" }));
    expect(expectLive(client.labels.getCached(lbl.key)).name).toBe("Anomaly");
    expect(await tx.commit()).toBe(true);
    expect(
      send.mock.calls.filter(([target]) => target === "/label/dispatch"),
    ).toHaveLength(1);
    const stored = await createTestClient().labels.retrieve(lbl.key);
    expect(stored.name).toBe("Anomaly");
    expect(stored.color).toEqual(color.construct("#ff0000"));
  });

  it("applies another client's dispatch to a cached label", async () => {
    const lbl = await createLabel();
    await remote.labels.retrieve(lbl.key);
    await client.labels.dispatch(lbl.key, rename("Anomaly"));
    await expect
      .poll(() => {
        const cached = remote.labels.getCached(lbl.key);
        return query.isLive(cached) ? cached.name : undefined;
      })
      .toBe("Anomaly");
  });

  it("ingests a label that another client creates", async () => {
    const lbl = await createLabel(remote);
    await expect
      .poll(() => {
        const cached = client.labels.getCached(lbl.key);
        return query.isLive(cached) ? cached.name : undefined;
      })
      .toBe(lbl.name);
  });
});
