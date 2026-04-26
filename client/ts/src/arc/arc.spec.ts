// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type arc } from "@/arc";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

const newTextArc = (name: string): arc.New => ({
  name,
  mode: "text",
  graph: {
    nodes: [],
    edges: [],
    viewport: { position: { x: 0, y: 0 }, zoom: 1 },
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
});
