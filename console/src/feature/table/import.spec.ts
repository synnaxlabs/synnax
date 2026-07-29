// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it, vi } from "vitest";

import { Table } from "@/feature/table";
import { createFileIngesterContext } from "@/platform/import/testutil";
import { type Panel } from "@/platform/panel";
import { createGrantedFluxStore, uniqueName } from "@/testutil";

describe("ingest", () => {
  it("should create the table on the cluster and open it as a tab", async () => {
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const original = await client.tables.create(proj.key, {
      name: uniqueName("table"),
    });
    const stream = await client.imex.export(table.ontologyID(original.key), {
      encoding: "JSON",
    });
    const data = JSON.parse(await new Response(stream).text());
    const store = await createGrantedFluxStore(
      client,
      table.TYPE_ONTOLOGY_ID,
      "update",
    );
    const openTab = vi.fn<Panel.OpenTab>();
    const id = await Table.ingest(
      data,
      createFileIngesterContext({ openTab, store, client, projectKey: proj.key }),
    );
    if (id == null) throw new Error("ingest returned no id");
    expect(openTab).toHaveBeenCalledWith({ variant: "resource", resource: id });
    const created = await client.tables.retrieve({ key: id.key });
    expect(created.name).toBe(original.name);
  });

  // The matcher pins the frozen legacy Console state shapes: if it stops claiming
  // them, typeless legacy files become unimportable; if it over-claims, foreign
  // files import as blank tables.
  describe("match", () => {
    it("should claim legacy Console table states", () => {
      expect(Table.ingest.match?.({ layout: {}, cells: {} })).toBe(true);
      expect(Table.ingest.match?.({ selectedCells: [] })).toBe(true);
      expect(Table.ingest.match?.({ hideIndicators: false })).toBe(true);
    });

    it("should not claim other resources' states", () => {
      expect(Table.ingest.match?.({ nodes: [], edges: [], props: {} })).toBe(false);
      expect(Table.ingest.match?.({ axes: {}, channels: {} })).toBe(false);
    });
  });
});
