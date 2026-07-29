// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it, vi } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { createFileIngesterContext } from "@/platform/import/testutil";
import { type Panel } from "@/platform/panel";
import { createGrantedFluxStore, uniqueName } from "@/testutil";

describe("ingest", () => {
  it("should create the line plot on the cluster and open it as a tab", async () => {
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const original = await client.lineplots.create(proj.key, {
      name: uniqueName("lineplot"),
    });
    const stream = await client.imex.export(lineplot.ontologyID(original.key), {
      encoding: "JSON",
    });
    const data = JSON.parse(await new Response(stream).text());
    const store = await createGrantedFluxStore(
      client,
      lineplot.TYPE_ONTOLOGY_ID,
      "update",
    );
    const openTab = vi.fn<Panel.OpenTab>();
    const id = await LinePlot.ingest(
      data,
      createFileIngesterContext({ openTab, store, client, projectKey: proj.key }),
    );
    if (id == null) throw new Error("ingest returned no id");
    expect(openTab).toHaveBeenCalledWith({ variant: "resource", resource: id });
    const created = await client.lineplots.retrieve({ key: id.key });
    expect(created.name).toBe(original.name);
  });

  // The matcher pins the frozen legacy Console state shapes: if it stops claiming
  // them, typeless legacy files become unimportable; if it over-claims, foreign
  // files import as blank plots.
  describe("match", () => {
    it("should claim legacy Console line plot states", () => {
      expect(LinePlot.ingest.match?.({ axes: {}, channels: {} })).toBe(true);
      expect(LinePlot.ingest.match?.({ selectedRules: [] })).toBe(true);
      expect(LinePlot.ingest.match?.({ hiddenLines: [] })).toBe(true);
    });

    it("should not claim other resources' states", () => {
      expect(LinePlot.ingest.match?.({ channels: [1, 2, 3] })).toBe(false);
      expect(LinePlot.ingest.match?.({ nodes: [], edges: [], props: {} })).toBe(false);
    });
  });
});
