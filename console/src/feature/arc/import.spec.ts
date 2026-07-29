// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it, vi } from "vitest";

import { Arc } from "@/feature/arc";
import { createFileIngesterContext } from "@/platform/import/testutil";
import { type Panel } from "@/platform/panel";
import { createGrantedFluxStore, uniqueName } from "@/testutil";

describe("ingest", () => {
  it("should create the arc on the cluster and open it as a tab", async () => {
    const client = createTestClient();
    const original = await client.arcs.create({
      name: uniqueName("arc"),
      mode: "graph",
    });
    const stream = await client.imex.export(arc.ontologyID(original.key), {
      encoding: "JSON",
    });
    const data = JSON.parse(await new Response(stream).text());
    const store = await createGrantedFluxStore(client, arc.TYPE_ONTOLOGY_ID, "update");
    const openTab = vi.fn<Panel.OpenTab>();
    const id = await Arc.ingest(
      data,
      createFileIngesterContext({ openTab, store, client }),
    );
    if (id == null) throw new Error("ingest returned no id");
    expect(openTab).toHaveBeenCalledWith({ variant: "resource", resource: id });
    const created = await client.arcs.retrieve({ key: id.key });
    expect(created.name).toBe(original.name);
  });

  // The matcher pins the frozen legacy Console state shapes: if it stops claiming
  // them, typeless legacy files become unimportable; if it over-claims, foreign
  // files import as blank arcs.
  describe("match", () => {
    it("should claim legacy Console arc states", () => {
      expect(Arc.ingest.match?.({ graph: {}, text: { raw: "" }, mode: "graph" })).toBe(
        true,
      );
      expect(Arc.ingest.match?.({ graph: {}, pendingUpload: {} })).toBe(true);
    });

    it("should not claim other resources' states", () => {
      expect(Arc.ingest.match?.({ nodes: [], edges: [], props: {} })).toBe(false);
      expect(Arc.ingest.match?.({ graph: {} })).toBe(false);
    });
  });
});
