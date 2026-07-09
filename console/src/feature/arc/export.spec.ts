// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc as clientArc, DisconnectedError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { createTestStore, uniqueName } from "@/testutil";

const client = createTestClient();

const createArc = async () =>
  await client.arcs.create({
    name: uniqueName("arc"),
    mode: "graph",
    graph: { nodes: [], edges: [] },
  });

describe("arc extract", () => {
  it("should serialize the retrieved arc tagged with the editor layout type", async () => {
    const arc = await createArc();
    const store = await createTestStore();
    const result = await Arc.extract(arc.key, { store, client });
    expect(result.name).toBe(arc.name);
    const data = JSON.parse(result.data);
    expect(data.type).toBe(clientArc.TYPE_ONTOLOGY_ID.type);
    expect(data.key).toBe(arc.key);
    expect(data.name).toBe(arc.name);
  });

  it("should throw a DisconnectedError without a client", async () => {
    const store = await createTestStore();
    await expect(Arc.extract("some-key", { store, client: null })).rejects.toSatisfy(
      (e) => DisconnectedError.matches(e),
    );
  });
});

describe("arc ingest", () => {
  it("should round-trip an exported arc back through ingest", async () => {
    const arc = await createArc();
    const store = await createTestStore();
    const { data } = await Arc.extract(arc.key, { store, client });
    const parsed = Arc.parseImport(JSON.parse(data));
    expect(parsed.name).toBe(arc.name);
    expect(parsed.mode).toBe("graph");
  });
});
