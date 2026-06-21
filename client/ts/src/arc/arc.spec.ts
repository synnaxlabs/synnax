// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { crdt, id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { arc } from "@/arc";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

const newTextArc = (name: string): arc.New => ({
  name,
  mode: "text",
  graph: {
    nodes: [],
    edges: [],
    functions: [],
  },
  text: { raw: "" },
});

describe("arc", () => {
  describe("retrieve", () => {
    it("should retrieve arcs by search term", async () => {
      const prefix = `searchable-arc-${id.create()}`;
      const names = [`${prefix}-1`, `${prefix}-2`];
      await client.arcs.create(names.map((name) => newTextArc(name)));
      await expect
        .poll(async () => {
          const results = await client.arcs.retrieve({ searchTerm: prefix });
          return results.map((a) => a.name).sort();
        })
        .toEqual(names);
    });
  });

  describe("dispatch", () => {
    it("materializes insert_char ops into the document's raw text", async () => {
      const created = await client.arcs.create(newTextArc(`dispatch-${id.create()}`));
      const gen = new crdt.Text(2);
      const ops = gen.insert(0, "hello").map((op) => arc.insertChar(op));
      await client.arcs.dispatch(created.key, "sess-1", ops);
      const res = await client.arcs.retrieve({ key: created.key });
      expect(res.text.raw).toEqual("hello");
    });

    it("drops deleted characters from the materialized raw text", async () => {
      const created = await client.arcs.create(newTextArc(`dispatch-${id.create()}`));
      const gen = new crdt.Text(2);
      const insertOps = gen.insert(0, "hello").map((op) => arc.insertChar(op));
      const deleteOps = gen.delete(0, 1).map((op) => arc.deleteChar(op));
      await client.arcs.dispatch(created.key, "sess-1", [...insertOps, ...deleteOps]);
      const res = await client.arcs.retrieve({ key: created.key });
      expect(res.text.raw).toEqual("ello");
    });

    it("reclaims tombstoned characters via a forget_chars dispatch", async () => {
      const created = await client.arcs.create(newTextArc(`dispatch-${id.create()}`));
      const gen = new crdt.Text(2);
      const insertOps = gen.insert(0, "hello").map((op) => arc.insertChar(op));
      const deletes = gen.delete(4, 1); // delete the trailing "o" (a leaf tombstone)
      await client.arcs.dispatch(created.key, "sess-1", [
        ...insertOps,
        ...deletes.map((op) => arc.deleteChar(op)),
      ]);
      await client.arcs.dispatch(created.key, "sess-1", [
        arc.forgetChars({ ids: deletes.map((op) => op.id) }),
      ]);
      const res = await client.arcs.retrieve({ key: created.key });
      expect(res.text.raw).toEqual("hell");
      expect(res.text.doc.deletes).toHaveLength(0);
      expect(res.text.doc.inserts).toHaveLength(4);
    });
  });
});
