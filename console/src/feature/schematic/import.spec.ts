// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it, vi } from "vitest";

import { Schematic } from "@/feature/schematic";
import { createFileIngesterContext } from "@/platform/import/testutil";
import { type Panel } from "@/platform/panel";
import { createGrantedFluxStore, uniqueName } from "@/testutil";

describe("ingest", () => {
  it("should create the schematic on the cluster and open it as a tab", async () => {
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const original = await client.schematics.create(proj.key, {
      name: uniqueName("schematic"),
    });
    const stream = await client.imex.export(schematic.ontologyID(original.key), {
      encoding: "JSON",
    });
    const data = JSON.parse(await new Response(stream).text());
    const store = await createGrantedFluxStore(
      client,
      schematic.TYPE_ONTOLOGY_ID,
      "update",
    );
    const openTab = vi.fn<Panel.OpenTab>();
    const id = await Schematic.ingest(
      data,
      createFileIngesterContext({ openTab, store, client, projectKey: proj.key }),
    );
    if (id == null) throw new Error("ingest returned no id");
    expect(openTab).toHaveBeenCalledWith({ variant: "resource", resource: id });
    const created = await client.schematics.retrieve({ key: id.key });
    expect(created.name).toBe(original.name);
  });

  // The matcher pins the frozen legacy Console state shapes: if it stops claiming
  // them, typeless legacy files become unimportable; if it over-claims, foreign
  // files import as blank schematics.
  describe("match", () => {
    it("should claim legacy Console schematic states", () => {
      expect(Schematic.ingest.match?.({ nodes: [], edges: [], props: {} })).toBe(true);
      expect(Schematic.ingest.match?.({ controlStatus: "released" })).toBe(true);
    });

    it("should not claim other resources' states", () => {
      expect(Schematic.ingest.match?.({ layout: {}, cells: {} })).toBe(false);
      expect(Schematic.ingest.match?.({ graph: {}, mode: "graph" })).toBe(false);
    });
  });
});
