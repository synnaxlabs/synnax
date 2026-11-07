// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, id, uuid } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { MultipleFoundError, NotFoundError, ValidationError } from "@/errors";
import { label } from "@/label";
import { query } from "@/query";
import { status } from "@/status";
import { createTestClient, expectDeleted, expectLive, spyOnSend } from "@/testutil";

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
        new NotFoundError(`Found no label where name = "${name}"`),
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
      const labels = await client.labels.retrieve({
        where: {
          name: { regex: `^${prefix}` },
          not: { key: { eq: attached.key } },
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
      const params: label.RetrieveMultipleParams = {
        name: { regex: `^${prefix}` },
        not: { key: { eq: attached.key } },
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
      const retrieved: label.Label = await client.labels.retrieve({
        name: attached.name,
        key: [item.key, attached.key],
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
    it("should name the search term when a search that sets one matches nothing", async () => {
      const term = id.create();
      await expect(
        client.labels.retrieve({ searchTerm: term, one: true }),
      ).rejects.toThrow(new NotFoundError(`Found no label matching "${term}"`));
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
          `Found no label where key = "${lbl.key}" and name = "${name}"`,
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
    const createStatus = async () =>
      await client.statuses.set({ variant: "info", message: `st-${id.create()}` });
    const labelsOf = async (key: status.Key): Promise<label.Key[]> =>
      (await createTestClient().statuses.retrieve(key)).labels.toSorted();

    it("should add labels to a status and keep the ones it holds", async () => {
      const [l1, l2] = await client.labels.create([
        { name: "Label One", color: "#E774D0" },
        { name: "Label Two", color: "#E774D0" },
      ]);
      const st = await createStatus();
      await client.labels.label(status.resourceID(st.key), [l1.key]);
      await client.labels.label(status.resourceID(st.key), [l2.key]);
      expect(await labelsOf(st.key)).toEqual([l1.key, l2.key].toSorted());
    });

    it("should replace the labels on a status", async () => {
      const [l1, l2] = await client.labels.create([
        { name: "Label One", color: "#E774D0" },
        { name: "Label Two", color: "#E774D0" },
      ]);
      const st = await createStatus();
      await client.labels.label(status.resourceID(st.key), [l1.key]);
      await client.labels.label(status.resourceID(st.key), [l2.key], { replace: true });
      expect(await labelsOf(st.key)).toEqual([l2.key]);
    });

    it("should remove labels from a status", async () => {
      const [l1, l2] = await client.labels.create([
        { name: "Label One", color: "#E774D0" },
        { name: "Label Two", color: "#E774D0" },
      ]);
      const st = await createStatus();
      await client.labels.label(status.resourceID(st.key), [l1.key, l2.key]);
      await client.labels.remove(status.resourceID(st.key), [l1.key]);
      expect(await labelsOf(st.key)).toEqual([l2.key]);
    });

    it("should reject a resource that is not a range or a status", async () => {
      const [l1, l2] = await client.labels.create([
        { name: "Label One", color: "#E774D0" },
        { name: "Label Two", color: "#E774D0" },
      ]);
      await expect(
        client.labels.label(label.resourceID(l1.key), [l2.key]),
      ).rejects.toThrow("id: a label cannot have labels");
    });

    it("should detach a deleted label from the statuses that hold it", async () => {
      const [kept, deleted] = await client.labels.create([
        { name: "Kept", color: "#E774D0" },
        { name: "Deleted", color: "#E774D0" },
      ]);
      const st = await createStatus();
      await client.labels.label(status.resourceID(st.key), [kept.key, deleted.key]);
      await client.labels.delete(deleted.key);
      expect(await labelsOf(st.key)).toEqual([kept.key]);
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

  it("tracks a remote label attachment on a subscribed status", async () => {
    const st = await client.statuses.set({
      variant: "info",
      message: `qry-item-${id.create()}`,
    });
    const lbl = await remote.labels.create({
      name: `qry-attached-${id.create()}`,
      color: "#E774D0",
    });
    const off = client.statuses.onChange(st.key, vi.fn());
    try {
      await client.statuses.retrieve(st.key);
      await remote.labels.label(status.resourceID(st.key), [lbl.key]);
      await expect
        .poll(() => {
          const cached = client.statuses.getCached(st.key);
          return query.isLive(cached) && cached.labels.includes(lbl.key);
        })
        .toBe(true);
      await remote.labels.remove(status.resourceID(st.key), [lbl.key]);
      await expect
        .poll(() => {
          const cached = client.statuses.getCached(st.key);
          return query.isLive(cached) && !cached.labels.includes(lbl.key);
        })
        .toBe(true);
    } finally {
      off();
    }
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
